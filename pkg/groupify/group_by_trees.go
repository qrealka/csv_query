package groupify

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"

	attr "propertytreeanalyzer/pkg/api/attribute"
	apiGroupify "propertytreeanalyzer/pkg/api/groupify"
	apiStreams "propertytreeanalyzer/pkg/api/streams"
)

// TreesGrouper contains channels for tree processing
type treesGrouper struct {
	source       apiStreams.JsonStream
	depth        int
	lastKey      string
	currentGroup apiGroupify.TreeSize
}

type streetsGroupsByTreeSize struct {
	groupKey apiGroupify.TreeSize
	street   apiGroupify.StreetName
}

var (
	_ apiGroupify.StreetGroups    = (*treesGrouper)(nil)
	_ apiGroupify.StreetGroupItem = (*streetsGroupsByTreeSize)(nil)
)

// Key implements StreetGroupItem.
func (s *streetsGroupsByTreeSize) Key() attr.BaseAttribute {
	return s.groupKey
}

// StreetName implements StreetGroupItem.
func (s *streetsGroupsByTreeSize) StreetName() apiGroupify.StreetName {
	return s.street
}

// NewTreesGrouper initializes a TreesGrouper with channels
func NewTreesGrouper(stream apiStreams.JsonStream) apiGroupify.StreetGroups {
	return &treesGrouper{source: stream}
}

func (t *treesGrouper) processJson(ctx context.Context, dst chan<- apiGroupify.StreetGroupItem) (bool, error) {
	tok, err := t.source.ReadJsonToken()
	if err == io.EOF {
		return true, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "Error reading JSON token", "error", err)
		return false, err
	}

	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{', '[':
			if t.depth == 1 {
				t.currentGroup = apiGroupify.ParseTreeSize(t.lastKey)
			}
			t.depth++

		case '}', ']':
			t.depth--
			if t.depth == 1 {
				if t.currentGroup != apiGroupify.TreeSizeNone {
					t.currentGroup = apiGroupify.TreeSizeNone
				}
			}
		}
		t.lastKey = "" // Reset key after exiting a scope

	case string:
		// This token is a key. Store it.
		t.lastKey = v

	case json.Number:
		if t.lastKey != "" {
			// We found a key followed by a number.
			// Add the key to the correct list based on the current section.
			dst <- &streetsGroupsByTreeSize{
				groupKey: t.currentGroup,
				street:   apiGroupify.ParseStreetName(t.lastKey),
			}
			t.lastKey = ""
		}

	default:
		// Other value types (boolean, null, string *value*).
	}
	return false, nil
}

// GroupStreets implements StreetsGrouper.
func (t *treesGrouper) GroupStreets(ctx context.Context, dst chan<- apiGroupify.StreetGroupItem) error {
	defer close(dst)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			done, err := t.processJson(ctx, dst)
			if done {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}
