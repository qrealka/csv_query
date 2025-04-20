package csvparser_test

import (
	"context"
	"fmt"
	"strings"

	attr "propertytreeanalyzer/pkg/api/attribute"
	"propertytreeanalyzer/pkg/csvparser"
	"propertytreeanalyzer/pkg/streams"
)

// ExampleNewPriceParser demonstrates how to count all street–price pairs.
func ExampleNewPriceParser() {
	// prepare a CSV in memory
	data := `
street,price
Main St,10.5
Oak Ave,20.75
Elm Rd,15.00
`
	stream, _ := streams.NewCsvStream(strings.NewReader(data))
	// configure parser by column names and force float parsing
	parser, _ := csvparser.NewPriceParser(
		stream,
		csvparser.WithColNames("street", "price"),
	)

	out := make(chan attr.StreetAttribute)
	go func() {
		_ = parser.ParseAttributes(context.Background(), out)
	}()

	// count items
	count := 0
	for range out {
		count++
	}
	fmt.Println("total items:", count)
	// Output:
	// total items: 3
}
