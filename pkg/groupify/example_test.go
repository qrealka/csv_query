package groupify_test

import (
	"context"
	"fmt"
	"strings"

	"propertytreeanalyzer/pkg/groupify"
	"propertytreeanalyzer/pkg/streams"
)

const data = `{"short":{"MainStreet":3,"SecondStreet":1},"tall":{"ElmStreet":5}}`

func ExampleNewTreesGrouper() {
	stream := streams.NewJsonStream(strings.NewReader(data))
	grouper, dst := groupify.NewTreesGrouper(stream)

	ctx := context.Background()
	go func() {
		if err := grouper.GroupStreets(ctx, dst); err != nil {
			fmt.Println("error:", err)
		}
	}()

	for item := range dst {
		// Print group size and street name
		fmt.Printf("%v: %v\n", item.Key(), item.StreetName())
	}
	// Output:
	// short: mainstreet
	// short: secondstreet
	// tall: elmstreet
}
