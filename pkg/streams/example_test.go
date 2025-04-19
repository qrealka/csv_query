package streams_test

import (
	"context"
	"fmt"
	"strings"

	"propertytreeanalyzer/pkg/streams"
)

func ExampleNewCsvStream() {
	csvData := "col1,col2\nval1,val2\nval3,val4\n"
	s, err := streams.NewCsvStream(strings.NewReader(csvData))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Print header
	fmt.Println(s.GetHeader())

	// Read records
	rec1, _ := s.ReadCsvRecord(context.Background())
	fmt.Println(rec1)
	rec2, _ := s.ReadCsvRecord(context.Background())
	fmt.Println(rec2)

	// Output:
	// [col1 col2]
	// [val1 val2]
	// [val3 val4]
}

func ExampleNewJsonStream() {
	jsonData := `[{"foo":1},{"foo":2}]`
	s := streams.NewJsonStream(strings.NewReader(jsonData))
	ctx := context.Background()

	count := 0
	for {
		_, err := s.ReadJsonToken(ctx)
		if err != nil {
			break
		}
		count++
	}
	fmt.Println(count)
	// Output:
	// 10
}
