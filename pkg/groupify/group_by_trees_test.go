package groupify

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"testing"

	api "propertytreeanalyzer/pkg/api/groupify"
)

type mockJsonStream struct {
	tokens []any
	index  int
}

// ReadJsonToken implements the JsonStream interface
func (m *mockJsonStream) ReadJsonToken(_ context.Context) (json.Token, error) {
	if m.index >= len(m.tokens) {
		return nil, io.EOF
	}
	token := m.tokens[m.index]
	m.index++
	return token, nil
}

func createMockTreeStream() *mockJsonStream {
	// This mock simulates JSON structure like:
	// {
	//   "short": {
	//     "road": { "main": 5 },
	//     "avenue": { "oak": 10 }
	//   },
	//   "tall": {
	//     "boulevard": { "elm": 15 }
	//   }
	// }
	return &mockJsonStream{
		tokens: []any{
			json.Delim('{'),
			"short",
			json.Delim('{'),
			"road",
			json.Delim('{'),
			"main",
			json.Number("5"),
			json.Delim('}'),
			"avenue",
			json.Delim('{'),
			"oak",
			json.Number("10"),
			json.Delim('}'),
			json.Delim('}'),
			"tall",
			json.Delim('{'),
			"boulevard",
			json.Delim('{'),
			"elm",
			json.Number("15"),
			json.Delim('}'),
			json.Delim('}'),
			json.Delim('}'),
		},
	}
}

func collectGroupItems(items []api.StreetGroupItem) map[api.TreeSize][]string {
	result := make(map[api.TreeSize][]string)

	for _, item := range items {
		treeSize := item.Key().(api.TreeSize)
		streetName := item.StreetName().String()
		result[treeSize] = append(result[treeSize], streetName)
	}

	return result
}

func TestTreesGrouperGroupStreets(t *testing.T) {
	tests := []struct {
		name     string
		stream   *mockJsonStream
		expected map[api.TreeSize][]string
	}{
		{
			name:   "Basic tree grouping",
			stream: createMockTreeStream(),
			expected: map[api.TreeSize][]string{
				api.TreeSizeShort: {"main", "oak"},
				api.TreeSizeTall:  {"elm"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grouper := NewTreesGrouper(tt.stream)
			itemChan := make(chan api.StreetGroupItem)

			// Collect items from the channel in a separate goroutine
			var items []api.StreetGroupItem
			done := make(chan struct{})
			go func() {
				defer close(done)
				for item := range itemChan {
					items = append(items, item)
				}
			}()

			// Process the items
			ctx := context.Background()
			err := grouper.GroupStreets(ctx, itemChan)
			<-done

			if err != nil {
				t.Fatalf("GroupStreets returned error: %v", err)
			}
			result := collectGroupItems(items)

			// Verify all expected tree sizes are present
			for treeSize, expectedStreets := range tt.expected {
				streets, ok := result[treeSize]
				if !ok {
					t.Errorf("Expected tree size %v not found in results", treeSize)
					continue
				}

				if !reflect.DeepEqual(streets, expectedStreets) {
					t.Errorf("For tree size %v, expected streets %v, got %v",
						treeSize, expectedStreets, streets)
				}
			}
		})
	}
}

func TestTreesGrouperWithLargeJson(t *testing.T) {
	stream := &mockJsonStream{
		tokens: []any{
			json.Delim('{'),
			"short",
			json.Delim('{'),
			"drive",
			json.Delim('{'),
			"abbey",
			json.Delim('{'),
			"rothe",
			json.Delim('{'),
			"rothe abbey",
			json.Number("10"),
			json.Delim('}'),
			json.Delim('}'),
			json.Delim('}'),
			"park",
			json.Delim('{'),
			"markievicz",
			json.Delim('{'),
			"markievicz park",
			json.Number("5"),
			json.Delim('}'),
			json.Delim('}'),
			json.Delim('}'),
			"tall",
			json.Delim('{'),
			"road",
			json.Delim('{'),
			"finglas",
			json.Delim('{'),
			"finglas road",
			json.Number("10"),
			json.Delim('}'),
			json.Delim('}'),
			json.Delim('}'),
			json.Delim('}'),
		},
	}

	expected := map[api.TreeSize][]string{
		api.TreeSizeShort: {"rothe abbey", "markievicz park"},
		api.TreeSizeTall:  {"finglas road"},
	}

	grouper := NewTreesGrouper(stream)
	itemChan := make(chan api.StreetGroupItem)
	var items []api.StreetGroupItem
	done := make(chan struct{})
	go func() {
		defer close(done)
		for item := range itemChan {
			items = append(items, item)
		}
	}()

	ctx := context.Background()
	err := grouper.GroupStreets(ctx, itemChan)
	<-done

	if err != nil {
		t.Fatalf("GroupStreets returned error: %v", err)
	}

	result := collectGroupItems(items)

	// Verify all expected tree sizes are present with correct streets
	for treeSize, expectedStreets := range expected {
		streets, ok := result[treeSize]
		if !ok {
			t.Errorf("Expected tree size %v not found in results", treeSize)
			continue
		}

		if !reflect.DeepEqual(streets, expectedStreets) {
			t.Errorf("For tree size %v, expected streets %v, got %v",
				treeSize, expectedStreets, streets)
		}
	}
}

func TestEmptyJson(t *testing.T) {
	stream := &mockJsonStream{
		tokens: []interface{}{
			json.Delim('{'),
			json.Delim('}'),
		},
	}

	grouper := NewTreesGrouper(stream)
	itemChan := make(chan api.StreetGroupItem)

	var items []api.StreetGroupItem
	done := make(chan struct{})
	go func() {
		defer close(done)
		for item := range itemChan {
			items = append(items, item)
		}
	}()

	ctx := context.Background()
	err := grouper.GroupStreets(ctx, itemChan)
	<-done

	if err != nil {
		t.Fatalf("GroupStreets returned error: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items for empty JSON, got %d", len(items))
	}
}

func TestCancelledContext(t *testing.T) {
	stream := createMockTreeStream()

	grouper := NewTreesGrouper(stream)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	itemChan := make(chan api.StreetGroupItem)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range itemChan {
			// Just drain the channel
		}
	}()

	err := grouper.GroupStreets(ctx, itemChan)
	<-done

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
