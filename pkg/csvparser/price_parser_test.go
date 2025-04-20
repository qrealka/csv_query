package csvparser

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	attr "propertytreeanalyzer/pkg/api/attribute"
	apiStreams "propertytreeanalyzer/pkg/api/streams"
)

// MockCsvStream is a mock implementation of the CsvStream interface for testing
type MockCsvStream struct {
	header     []string
	records    [][]string
	currentRow int
}

var _ apiStreams.CsvStream = (*MockCsvStream)(nil)

// NewMockCsvStream creates a new mock CSV stream with the given header and records
func NewMockCsvStream(header []string, records [][]string) *MockCsvStream {
	return &MockCsvStream{
		header:     header,
		records:    records,
		currentRow: 0,
	}
}

// GetHeader returns the CSV header
func (m *MockCsvStream) GetHeader() []string {
	return m.header
}

// ReadCsvRecord reads the next CSV record
func (m *MockCsvStream) ReadCsvRecord(_ context.Context) ([]string, error) {
	if m.currentRow >= len(m.records) {
		return nil, io.EOF
	}

	record := m.records[m.currentRow]
	m.currentRow++
	return record, nil
}

func TestNewPriceParser(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		stream        apiStreams.CsvStream
		streetColName string
		priceColName  string
		wantErr       error
	}{
		{
			name:          "Valid parser creation",
			stream:        NewMockCsvStream([]string{"Date", "Address", "Street Name", "Price"}, nil),
			streetColName: "Street Name",
			priceColName:  "Price",
			wantErr:       nil,
		},
		{
			name:          "Nil stream",
			stream:        nil,
			streetColName: "Street Name",
			priceColName:  "Price",
			wantErr:       errNilCsvStream,
		},
		{
			name:          "Empty header",
			stream:        NewMockCsvStream([]string{}, nil),
			streetColName: "Street Name",
			priceColName:  "Price",
			wantErr:       errNoHeader,
		},
		{
			name:          "Missing street column",
			stream:        NewMockCsvStream([]string{"Date", "Address", "Price"}, nil),
			streetColName: "Street Name",
			priceColName:  "Price",
			wantErr:       errStreetColumnMissing,
		},
		{
			name:          "Missing price column",
			stream:        NewMockCsvStream([]string{"Date", "Address", "Street Name"}, nil),
			streetColName: "Street Name",
			priceColName:  "Price",
			wantErr:       errPriceColumnMissing,
		},
		{
			name:          "Case insensitive column matching",
			stream:        NewMockCsvStream([]string{"Date", "Address", "STREET name", "price"}, nil),
			streetColName: "Street Name",
			priceColName:  "Price",
			wantErr:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewPriceParser(tt.stream, WithColNames(tt.streetColName, tt.priceColName))

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewPriceParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && parser == nil {
				t.Errorf("NewPriceParser() returned nil parser with no error")
			}

			if err != nil && parser != nil {
				t.Errorf("NewPriceParser() returned parser with error: %v", err)
			}
		})
	}
}

func TestParseAttributes(t *testing.T) {
	tests := []struct {
		name          string
		header        []string
		records       [][]string
		streetColName string
		priceColName  string
		expectedAttrs []attr.StreetAttribute
		expectedError error
	}{
		{
			name:          "Valid parsing",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St", "main street", "100,000.00"}, {"02/01/2023", "456 Oak Ave", "oak avenue", "200,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedAttrs: []attr.StreetAttribute{
				streetPricePair{streetName: "main street", price: "100000.00"},
				streetPricePair{streetName: "oak avenue", price: "200000.00"},
			},
			expectedError: nil,
		},
		{
			name:          "Converting street name to lowercase",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St", "Main Street", "100,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedAttrs: []attr.StreetAttribute{
				streetPricePair{streetName: "main street", price: "100000.00"},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := NewMockCsvStream(tt.header, tt.records)
			parser, perr := NewPriceParser(stream, WithColNames(tt.streetColName, tt.priceColName))
			if perr != nil {
				t.Fatalf("Failed to create parser: %v", perr)
			}

			out := make(chan attr.StreetAttribute)
			var results []attr.StreetAttribute

			done := make(chan struct{})
			go func() {
				for a := range out {
					results = append(results, a)
				}
				close(done)
			}()

			err := parser.ParseAttributes(t.Context(), out)
			<-done

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("ParseAttributes() error = %v, wantErr %v", err, tt.expectedError)
			}
			if len(results) != len(tt.expectedAttrs) {
				t.Errorf("ParseAttributes() got %d results, want %d", len(results), len(tt.expectedAttrs))
				return
			}
			for i, got := range results {
				want := tt.expectedAttrs[i]
				if got.StreetName() != want.StreetName() {
					t.Errorf("Result[%d] street name = %v, want %v", i, got.StreetName(), want.StreetName())
				}
				if got.AttributeValue() != want.AttributeValue() {
					t.Errorf("Result[%d] price = %v, want %v", i, got.AttributeValue(), want.AttributeValue())
				}
			}
		})
	}
}

func TestParsePrices(t *testing.T) {
	tests := []struct {
		name          string
		header        []string
		records       [][]string
		streetColName string
		priceColName  string
		expectedPairs []streetPricePair
		expectedError error
	}{
		{
			name:          "Valid parsing",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St", "main street", "100,000.00"}, {"02/01/2023", "456 Oak Ave", "oak avenue", "200,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedPairs: []streetPricePair{
				{streetName: "main street", price: "100000.00"},
				{streetName: "oak avenue", price: "200000.00"},
			},
			expectedError: nil,
		},
		{
			name:          "Converting street name to lowercase",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St", "Main Street", "100,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedPairs: []streetPricePair{
				{streetName: "main street", price: "100000.00"},
			},
			expectedError: nil,
		},
		{
			name:          "Invalid price format - skips record",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St", "main street", "invalid"}, {"02/01/2023", "456 Oak Ave", "oak avenue", "200,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedPairs: []streetPricePair{
				{streetName: "oak avenue", price: "200000.00"},
			},
			expectedError: nil,
		},
		{
			name:          "Missing field in record - skips record",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St"}, {"02/01/2023", "456 Oak Ave", "oak avenue", "200,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedPairs: []streetPricePair{
				{streetName: "oak avenue", price: "200000.00"},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := NewMockCsvStream(tt.header, tt.records)
			parser, err := NewPriceParser(stream, WithColNames(tt.streetColName, tt.priceColName))
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			out := make(chan attr.StreetAttribute)
			var results []attr.StreetAttribute

			done := make(chan struct{})
			go func() {
				for pair := range out {
					results = append(results, pair)
				}
				close(done)
			}()

			err = parser.ParseAttributes(t.Context(), out)
			<-done

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("ParsePrices() error = %v, wantErr %v", err, tt.expectedError)
			}

			if len(results) != len(tt.expectedPairs) {
				t.Errorf("ParsePrices() got %d results, want %d", len(results), len(tt.expectedPairs))
				return
			}
			for i, got := range results {
				want := tt.expectedPairs[i]
				if !got.EqualTo(want) {
					t.Errorf("ParsePrices()[%d] street name = %v, want %v", i, got.StreetName(), want.StreetName())
				}
			}
		})
	}
}

func TestErrorScenarios(t *testing.T) {
	// Test for stream that returns error on ReadCsvRecord
	t.Run("Stream read error", func(t *testing.T) {
		// Create a custom mock that returns an error
		mockStream := &MockErrorStream{
			header: []string{"Date", "Address", "Street Name", "Price"},
			err:    errors.New("read error"),
		}

		parser, err := NewPriceParser(mockStream, WithColNames("Street Name", "Price"))
		if err != nil {
			t.Fatalf("Failed to create parser: %v", err)
		}

		out := make(chan attr.StreetAttribute)
		go func() {
			for range out {
				// Just consume the channel
			}
		}()

		err = parser.ParseAttributes(t.Context(), out)
		if err == nil || err.Error() != "read error" {
			t.Errorf("Expected 'read error', got %v", err)
		}
	})
}

// MockErrorStream is a mock that returns an error when ReadCsvRecord is called
type MockErrorStream struct {
	header []string
	err    error
}

func (m *MockErrorStream) GetHeader() []string {
	return m.header
}

func (m *MockErrorStream) ReadCsvRecord(_ context.Context) ([]string, error) {
	return nil, m.err
}

func TestStreamIntegration(t *testing.T) {
	csvData := `Date,Address,Street Name,Price
01/01/2023,123 Main St,Main Street,100000
02/01/2023,456 Oak Ave,Oak Avenue,200000
`
	reader := strings.NewReader(csvData)
	csvStream := &testCsvStream{reader: reader}

	parser, err := NewPriceParser(csvStream, WithColNames("Street Name", "Price"))
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	out := make(chan attr.StreetAttribute)
	var results []attr.StreetAttribute

	done := make(chan struct{})
	go func() {
		for pair := range out {
			results = append(results, pair)
		}
		close(done)
	}()

	err = parser.ParseAttributes(t.Context(), out)
	if err != nil {
		t.Errorf("ParsePrices() error = %v", err)
	}
	<-done

	expected := []streetPricePair{
		{streetName: "main street", price: "100000"},
		{streetName: "oak avenue", price: "200000"},
	}

	if len(results) != len(expected) {
		t.Errorf("ParsePrices() got %d results, want %d", len(results), len(expected))
	} else {
		for i, got := range results {
			want := expected[i]
			if !got.EqualTo(want) {
				t.Errorf("Result[%d] = %v, want %v", i, got, want)
			}
		}
	}
}

// testCsvStream implements a simple CsvStream for testing
type testCsvStream struct {
	reader  *strings.Reader
	header  []string
	hasRead bool
}

func (t *testCsvStream) GetHeader() []string {
	if !t.hasRead {
		t.readHeader()
	}
	return t.header
}

func (t *testCsvStream) readHeader() {
	data, err := io.ReadAll(t.reader)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 {
		t.header = strings.Split(lines[0], ",")
		remaining := strings.Join(lines[1:], "\n")
		t.reader = strings.NewReader(remaining)
	}
	t.hasRead = true
}

func (t *testCsvStream) ReadCsvRecord(_ context.Context) ([]string, error) {
	if !t.hasRead {
		t.readHeader()
	}

	var line string
	for {
		r, _, err := t.reader.ReadRune()
		if err != nil {
			return nil, io.EOF
		}
		if r == '\n' {
			break
		}
		line += string(r)
	}

	if line == "" {
		return nil, io.EOF
	}

	return strings.Split(line, ","), nil
}
