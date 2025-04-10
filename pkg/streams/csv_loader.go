package streams

import (
	"encoding/csv"
	"io"

	iface "propertytreeanalyzer/api/streams"
)

type csvReader struct {
	reader *csv.Reader
	header []string
}

var _ iface.CsvStream = (*csvReader)(nil)

// NewCsvStream creates a new CSV stream from an io.Reader.
// It reads the header row immediately.
func NewCsvStream(reader io.Reader) (iface.CsvStream, error) {
	csvR := csv.NewReader(reader)

	// Read header row
	header, err := csvR.Read()
	if err != nil {
		return nil, err
	}

	return &csvReader{
		reader: csvR,
		header: header,
	}, nil
}

// ReadCsvRecord implements CsvStream.
func (c *csvReader) ReadCsvRecord() ([]string, error) {
	if c == nil || c.reader == nil {
		return nil, io.EOF
	}
	return c.reader.Read()
}

// GetHeader implements CsvStream.
func (c *csvReader) GetHeader() []string {
	if c == nil {
		return nil
	}
	return c.header
}
