package csvparser

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"unicode"

	attr "propertytreeanalyzer/pkg/api/attribute"
	apiParser "propertytreeanalyzer/pkg/api/parsers"
	apiStreams "propertytreeanalyzer/pkg/api/streams"
)

var (
	_ attr.StreetAttribute            = (*streetPricePair)(nil)
	_ apiParser.StreetAttributeParser = (*priceParser)(nil)
)

// streetPricePair represents a pair of street name and price
type streetPricePair struct {
	streetName string
	price      string
}

// StreetName returns the name of the street
func (s streetPricePair) StreetName() string {
	return s.streetName
}

// AttributeValue returns the value of the attribute associated with the street
func (s streetPricePair) AttributeValue() string {
	return s.price
}

// EqualTo checks if two street price pairs are equal
func (s streetPricePair) EqualTo(other attr.StreetAttribute) bool {
	if other == nil {
		return false
	}
	if !strings.EqualFold(s.streetName, other.StreetName()) {
		return false
	}
	return s.price == other.AttributeValue()
}

// priceParser parses CSV records and extracts street name and price pairs
// It implements the StreetAttributeParser interface
type priceParser struct {
	stream    apiStreams.CsvStream
	streetIdx int
	priceIdx  int
}

// NewPriceParser creates a new price parser with the given CSV stream and column names
func NewPriceParser(stream apiStreams.CsvStream, opts ...PriceParserOption) (apiParser.StreetAttributeParser, error) {
	if stream == nil {
		return nil, errNilCsvStream
	}
	p := &priceParser{
		stream:    stream,
		streetIdx: -1,
		priceIdx:  -1,
	}
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// loadPrices reads the CSV stream and sends street name and price pairs to the provided channel
// It processes the street name to lowercase and converts price strings to float64
// The channel is closed when parsing is complete or an error occurs
func (p *priceParser) loadPrices(ctx context.Context, out chan<- attr.StreetAttribute) error {
	if p == nil || p.stream == nil {
		close(out)
		return errNilParserOrStream
	}

	defer close(out)

	for {
		record, err := p.stream.ReadCsvRecord(ctx)
		if err == io.EOF {
			slog.InfoContext(ctx, "End of CSV stream")
			return nil
		}
		if err != nil {
			return err
		}

		if len(record) <= p.streetIdx || len(record) <= p.priceIdx {
			continue
		}

		// drop everything that is not a digit, dot or minus in one pass
		price := strings.Map(func(r rune) rune {
			if unicode.IsDigit(r) || r == '.' || r == '-' {
				return r
			}
			return -1
		}, record[p.priceIdx])

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if len(price) != 0 {
				out <- streetPricePair{
					streetName: strings.ToLower(record[p.streetIdx]),
					price:      price,
				}
			}
		}
	}
}

// ParseAttributes reads the CSV stream and sends street attribute pairs to the provided channel
// It implements the StreetAttributeParser interface method
func (p *priceParser) ParseAttributes(ctx context.Context, out chan<- attr.StreetAttribute) error {
	if p == nil || p.stream == nil {
		close(out)
		return errNilParserOrStream
	}
	return p.loadPrices(ctx, out)
}
