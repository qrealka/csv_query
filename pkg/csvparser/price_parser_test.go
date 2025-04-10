package csvparser

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	apiParsers "propertytreeanalyzer/api/parsers"
	apiStreams "propertytreeanalyzer/api/streams"
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
func (m *MockCsvStream) ReadCsvRecord() ([]string, error) {
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
			parser, err := NewPriceParser(tt.stream, tt.streetColName, tt.priceColName)

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
	// Test cases
	tests := []struct {
		name           string
		header         []string
		records        [][]string
		streetColName  string
		priceColName   string
		expectedAttrs  []apiParsers.StreetAttribute
		expectedError  error
		nilParser      bool
		nilParserError bool
	}{
		{
			name:          "Valid parsing",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St", "main street", "100,000.00"}, {"02/01/2023", "456 Oak Ave", "oak avenue", "200,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedAttrs: []apiParsers.StreetAttribute{
				streetPricePair{streetName: "main street", price: 100000.00},
				streetPricePair{streetName: "oak avenue", price: 200000.00},
			},
			expectedError: nil,
		},
		{
			name:          "Converting street name to lowercase",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St", "Main Street", "100,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedAttrs: []apiParsers.StreetAttribute{
				streetPricePair{streetName: "main street", price: 100000.00},
			},
			expectedError: nil,
		},
		{
			name:           "Nil parser",
			nilParser:      true,
			nilParserError: true,
			expectedError:  errNilParserOrStream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var parser apiParsers.StreetAttributeParser
			var err error

			if !tt.nilParser {
				stream := NewMockCsvStream(tt.header, tt.records)
				priceParser, perr := NewPriceParser(stream, tt.streetColName, tt.priceColName)
				if perr != nil {
					t.Fatalf("Failed to create parser: %v", perr)
				}
				parser = priceParser // Use PriceParser as StreetAttributeParser
			}

			out := make(chan apiParsers.StreetAttribute)
			var results []apiParsers.StreetAttribute

			done := make(chan struct{})
			go func() {
				for attr := range out {
					results = append(results, attr)
				}
				close(done)
			}()

			if parser != nil {
				err = parser.ParseAttributes(out)
			} else {
				// Handle nil parser case
				var p *PriceParser
				err = p.ParseAttributes(out)
			}
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
				gotVal := got.AttributeValue().(float64)
				wantVal := want.AttributeValue().(float64)
				if gotVal != wantVal {
					t.Errorf("Result[%d] price = %v, want %v", i, gotVal, wantVal)
				}
			}
		})
	}
}

func TestParsePrices(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		header         []string
		records        [][]string
		streetColName  string
		priceColName   string
		expectedPairs  []streetPricePair
		expectedError  error
		nilParser      bool
		nilParserError bool
	}{
		{
			name:          "Valid parsing",
			header:        []string{"Date", "Address", "Street Name", "Price"},
			records:       [][]string{{"01/01/2023", "123 Main St", "main street", "100,000.00"}, {"02/01/2023", "456 Oak Ave", "oak avenue", "200,000.00"}},
			streetColName: "Street Name",
			priceColName:  "Price",
			expectedPairs: []streetPricePair{
				{streetName: "main street", price: 100000.00},
				{streetName: "oak avenue", price: 200000.00},
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
				{streetName: "main street", price: 100000.00},
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
				{streetName: "oak avenue", price: 200000.00},
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
				{streetName: "oak avenue", price: 200000.00},
			},
			expectedError: nil,
		},
		{
			name:           "Nil parser",
			nilParser:      true,
			nilParserError: true,
			expectedError:  errNilParserOrStream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var parser *PriceParser
			var err error

			if !tt.nilParser {
				stream := NewMockCsvStream(tt.header, tt.records)
				parser, err = NewPriceParser(stream, tt.streetColName, tt.priceColName)
				if err != nil {
					t.Fatalf("Failed to create parser: %v", err)
				}
			}

			out := make(chan streetPricePair)
			var results []streetPricePair

			done := make(chan struct{})
			go func() {
				for pair := range out {
					results = append(results, pair)
				}
				close(done)
			}()

			err = parser.loadPrices(out)
			<-done

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("ParsePrices() error = %v, wantErr %v", err, tt.expectedError)
			}

			if !reflect.DeepEqual(results, tt.expectedPairs) {
				t.Errorf("ParsePrices() got = %v, want %v", results, tt.expectedPairs)
			}
		})
	}
}

func TestParsePrice(t *testing.T) {
	tests := []struct {
		name      string
		priceStr  string
		want      float64
		wantError bool
	}{
		{"Simple number", "100", 100.0, false},
		{"Decimal number", "100.50", 100.50, false},
		{"With commas", "1,000,000.00", 1000000.00, false},
		{"With dollar sign", "$100.00", 100.00, false},
		{"With euro sign", "€100.00", 100.00, false},
		{"With pound sign", "£100.00", 100.00, false},
		{"With whitespace", " 100.00 ", 100.00, false},
		{"Combined formats", "€ 1,000,000.50", 1000000.50, false},
		{"Invalid format", "abc", 0, true},
		{"Empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePrice(tt.priceStr)

			if (err != nil) != tt.wantError {
				t.Errorf("parsePrice() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && got != tt.want {
				t.Errorf("parsePrice() = %v, want %v", got, tt.want)
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

		parser, err := NewPriceParser(mockStream, "Street Name", "Price")
		if err != nil {
			t.Fatalf("Failed to create parser: %v", err)
		}

		out := make(chan streetPricePair)
		go func() {
			for range out {
				// Just consume the channel
			}
		}()

		err = parser.loadPrices(out)
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

func (m *MockErrorStream) ReadCsvRecord() ([]string, error) {
	return nil, m.err
}

func TestStreamIntegration(t *testing.T) {
	csvData := `Date,Address,Street Name,Price
01/01/2023,123 Main St,Main Street,100000
02/01/2023,456 Oak Ave,Oak Avenue,200000
`
	reader := strings.NewReader(csvData)
	csvStream := &testCsvStream{reader: reader}

	parser, err := NewPriceParser(csvStream, "Street Name", "Price")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	out := make(chan streetPricePair)
	var results []streetPricePair

	done := make(chan struct{})
	go func() {
		for pair := range out {
			results = append(results, pair)
		}
		close(done)
	}()

	err = parser.loadPrices(out)
	if err != nil {
		t.Errorf("ParsePrices() error = %v", err)
	}
	<-done

	expected := []streetPricePair{
		{streetName: "main street", price: 100000},
		{streetName: "oak avenue", price: 200000},
	}

	if !reflect.DeepEqual(results, expected) {
		t.Errorf("ParsePrices() got = %v, want %v", results, expected)
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

func (t *testCsvStream) ReadCsvRecord() ([]string, error) {
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
