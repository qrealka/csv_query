package csvparser

import (
	"context"
	"io"
	"log/slog"
	"strings"

	attr "propertytreeanalyzer/pkg/api/attribute"
	apiParser "propertytreeanalyzer/pkg/api/parsers"
	apiStreams "propertytreeanalyzer/pkg/api/streams"
	decimal "propertytreeanalyzer/pkg/numeric"
)

var (
	_ attr.StreetAttribute            = (*streetPricePair)(nil)
	_ apiParser.StreetAttributeParser = (*priceParser)(nil)
)

// streetPricePair represents a pair of street name and price
type streetPricePair struct {
	streetName string
	price      attr.NumericAttribute
}

// StreetName returns the name of the street
func (s streetPricePair) StreetName() string {
	return s.streetName
}

// AttributeValue returns the price as an 'any' type
func (s streetPricePair) AttributeValue() attr.NumericAttribute {
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
	return s.price.EqualTo(other.AttributeValue())
}

// priceParser parses CSV records and extracts street name and price pairs
// It implements the StreetAttributeParser interface
type priceParser struct {
	stream    apiStreams.CsvStream
	streetIdx int
	priceIdx  int
	useFloats bool
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
		useFloats: false,
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

		streetName := strings.ToLower(record[p.streetIdx])
		priceStr := record[p.priceIdx]

		var price attr.NumericAttribute
		if !p.useFloats {
			price, err = decimal.ParseDecimalAttribute(priceStr)
			if err != nil {
				slog.WarnContext(ctx, "Failed to parse price as decimal", slog.String("price", priceStr), slog.Any("error", err))
				continue
			}
		} else {
			price, err = decimal.ParseFloatAttribute(priceStr)
			if err != nil {
				slog.WarnContext(ctx, "Failed to parse price as float", slog.String("price", priceStr), slog.Any("error", err))
				continue
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			out <- streetPricePair{
				streetName: streetName,
				price:      price,
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
