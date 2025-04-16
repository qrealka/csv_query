package csvparser

import "errors"

var (
	// Error definitions
	errNilCsvStream                  = errors.New("csv stream cannot be nil")
	errNoHeader                      = errors.New("csv stream has no header")
	errStreetColumnNotSpecified      = errors.New("street column name not specified")
	errStreetColumnIndexNotSpecified = errors.New("street column index not specified")
	errPriceColumnNotSpecified       = errors.New("price column name not specified")
	errPriceColumnIndexNotSpecified  = errors.New("price column index not specified")
	errColumnNamesEqual              = errors.New("street and price column names are equal")
	errColumnIndexesEqual            = errors.New("street and price column indexes are equal")
	errStreetColumnMissing           = errors.New("street column not found in CSV header")
	errPriceColumnMissing            = errors.New("price column not found in CSV header")
	errNilParserOrStream             = errors.New("parser or stream is nil")
)
