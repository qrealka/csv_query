package csvparser

import "strings"

// PriceParserOption configures a PriceParser
type PriceParserOption func(*PriceParser) error

// WithColNames sets street and price column names by header lookup
func WithColNames(streetColName, priceColName string) PriceParserOption {
	return func(p *PriceParser) error {
		streetColName = strings.TrimSpace(streetColName)
		priceColName = strings.TrimSpace(priceColName)

		if streetColName == "" {
			return errStreetColumnNotSpecified
		}
		if priceColName == "" {
			return errPriceColumnNotSpecified
		}
		if strings.EqualFold(streetColName, priceColName) {
			return errColumnNamesEqual
		}

		header := p.stream.GetHeader()
		if len(header) == 0 {
			return errNoHeader
		}
		for i, col := range header {
			col = strings.TrimSpace(col)
			if strings.EqualFold(col, streetColName) {
				p.streetIdx = i
			}
			if strings.EqualFold(col, priceColName) {
				p.priceIdx = i
			}
		}
		if p.streetIdx == -1 {
			return errStreetColumnMissing
		}
		if p.priceIdx == -1 {
			return errPriceColumnMissing
		}
		return nil
	}
}

// WithColIndexes sets street and price column indexes directly
func WithColIndexes(streetColIdx, priceColIdx int) PriceParserOption {
	return func(p *PriceParser) error {
		if streetColIdx < 0 {
			return errStreetColumnIndexNotSpecified
		}
		if priceColIdx < 0 {
			return errPriceColumnIndexNotSpecified
		}
		if streetColIdx == priceColIdx {
			return errColumnIndexesEqual
		}
		p.streetIdx = streetColIdx
		p.priceIdx = priceColIdx
		return nil
	}
}

// WithDecimals forces decimal parsing (default)
func WithDecimals() PriceParserOption {
	return func(p *PriceParser) error {
		p.useFloats = false
		return nil
	}
}

// WithFloats forces float64 parsing
func WithFloats() PriceParserOption {
	return func(p *PriceParser) error {
		p.useFloats = true
		return nil
	}
}
