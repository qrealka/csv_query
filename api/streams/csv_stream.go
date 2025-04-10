package streams

// CsvStream represents a stream of CSV records.
type CsvStream interface {
	// ReadCsvRecord reads the next CSV record from the stream.
	// Returns the next record as a slice of strings and any error encountered.
	ReadCsvRecord() ([]string, error)

	// GetHeader returns the header row of the CSV file.
	// This is typically the first row with column names.
	GetHeader() []string
}
