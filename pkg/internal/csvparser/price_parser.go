package csvparser

import (
	"errors"
	"io"
	"strconv"
	"strings"

	apiParsers "propertytreeanalyzer/pkg/api/parsers"
	apiStreams "propertytreeanalyzer/pkg/api/streams"
)

var (
	// Error definitions
	errNilCsvStream        = errors.New("csv stream cannot be nil")
	errNoHeader            = errors.New("csv stream has no header")
	errStreetColumnMissing = errors.New("street column not found in CSV header")
	errPriceColumnMissing  = errors.New("price column not found in CSV header")
	errNilParserOrStream   = errors.New("parser or stream is nil")
)

// streetPricePair represents a pair of street name and price
type streetPricePair struct {
	streetName string
	price      float64
}

// StreetName returns the name of the street
func (s streetPricePair) StreetName() string {
	return s.streetName
}

// AttributeValue returns the price as an 'any' type
func (s streetPricePair) AttributeValue() any {
	return s.price
}

// PriceParser parses CSV records and extracts street name and price pairs
// It implements the StreetAttributeParser interface
type PriceParser struct {
	stream        apiStreams.CsvStream
	streetColName string
	priceColName  string
	streetIdx     int
	priceIdx      int
}

// NewPriceParser creates a new price parser with the given CSV stream and column names
func NewPriceParser(stream apiStreams.CsvStream, streetColName, priceColName string) (*PriceParser, error) {
	if stream == nil {
		return nil, errNilCsvStream
	}

	header := stream.GetHeader()
	if len(header) == 0 {
		return nil, errNoHeader
	}

	streetIdx := -1
	priceIdx := -1

	for i, col := range header {
		if strings.EqualFold(col, streetColName) {
			streetIdx = i
		}
		if strings.EqualFold(col, priceColName) {
			priceIdx = i
		}
	}

	if streetIdx == -1 {
		return nil, errStreetColumnMissing
	}
	if priceIdx == -1 {
		return nil, errPriceColumnMissing
	}

	return &PriceParser{
		stream:        stream,
		streetColName: streetColName,
		priceColName:  priceColName,
		streetIdx:     streetIdx,
		priceIdx:      priceIdx,
	}, nil
}

// loadPrices reads the CSV stream and sends street name and price pairs to the provided channel
// It processes the street name to lowercase and converts price strings to float64
// The channel is closed when parsing is complete or an error occurs
func (p *PriceParser) loadPrices(out chan<- streetPricePair) error {
	if p == nil || p.stream == nil {
		close(out)
		return errNilParserOrStream
	}

	defer close(out)

	for {
		record, err := p.stream.ReadCsvRecord()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if len(record) <= p.streetIdx || len(record) <= p.priceIdx {
			continue
		}

		streetName := strings.ToLower(record[p.streetIdx])
		priceStr := record[p.priceIdx]
		price, err := parsePrice(priceStr)
		if err != nil {
			continue
		}

		out <- streetPricePair{
			streetName: streetName,
			price:      price,
		}
	}
}

// ParseAttributes reads the CSV stream and sends street attribute pairs to the provided channel
// It implements the StreetAttributeParser interface method
func (p *PriceParser) ParseAttributes(out chan<- apiParsers.StreetAttribute) error {
	if p == nil || p.stream == nil {
		close(out)
		return errNilParserOrStream
	}

	// Create an intermediary channel for StreetPricePair objects
	priceChan := make(chan streetPricePair)

	// Start a goroutine to process and forward the street price pairs
	go func() {
		for pair := range priceChan {
			out <- pair
		}
		close(out)
	}()

	// Use the existing ParsePrices method to handle the parsing logic
	return p.loadPrices(priceChan)
}

// parsePrice extracts a float value from a price string by removing
// currency symbols, commas and other non-numeric characters
func parsePrice(price string) (float64, error) {
	// Remove currency symbol, commas, and spaces
	price = strings.Replace(price, "$", "", -1)
	price = strings.Replace(price, "€", "", -1)
	price = strings.Replace(price, "£", "", -1)
	price = strings.Replace(price, ",", "", -1)
	price = strings.Replace(price, " ", "", -1)
	price = strings.TrimSpace(price)

	// Parse as float
	return strconv.ParseFloat(price, 64)
}
