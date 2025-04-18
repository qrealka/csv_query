package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	apd "github.com/cockroachdb/apd/v3"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func main() {
	// Define command line flags
	csvPath := flag.String("file", "", "Path to the CSV file to process")
	flag.Parse()

	// Check if file path was provided
	if *csvPath == "" {
		log.Fatal("Please provide a CSV file path using -file flag")
	}

	// Open the file
	file, err := os.Open(*csvPath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	// Create a transform reader for Windows-1252 encoding
	decoder := charmap.Windows1252.NewDecoder()
	reader := transform.NewReader(file, decoder)

	// Create a CSV reader
	csvReader := csv.NewReader(reader)

	// Read the header row
	header, err := csvReader.Read()
	if err != nil {
		log.Fatalf("Error reading CSV header: %v", err)
	}

	// Find the price column index (case-insensitive)
	priceIndex := -1
	for i, column := range header {
		if strings.EqualFold(column, "Price") {
			priceIndex = i
			break
		}
	}

	if priceIndex == -1 {
		log.Fatalf("Price column not found in CSV file")
	}

	fmt.Printf("Found Price column at index %d\n", priceIndex)

	// Variables for calculating average
	var count int64
	sum := apd.New(0, 0)
	sumCtx := apd.Context{
		Precision:   100,
		MaxExponent: apd.MaxExponent,
		MinExponent: apd.MinExponent,
		Traps:       apd.DefaultTraps,
		Rounding:    apd.RoundHalfEven, // Or RoundHalfUp, less critical here
	}

	// Process the file row by row
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: Error reading row: %v", err)
			continue
		}

		// Check if the record has enough fields
		if len(record) <= priceIndex {
			log.Printf("Warning: Row has insufficient columns")
			continue
		}

		// Extract price from the record and parse it
		priceStr := record[priceIndex]
		price, err := parsePrice(priceStr)
		if err != nil {
			log.Printf("Warning: Could not parse price '%s': %v", priceStr, err)
			continue
		}

		// Add to sum and increment count
		if _, err := sumCtx.Add(sum, sum, price); err != nil {
			log.Printf("Warning: Error adding price '%s' to sum: %v", priceStr, err)
			return
		}
		count++
	}

	// Calculate and print average
	if count > 0 {
		avgCtx := apd.Context{
			Precision:   50,
			MaxExponent: apd.MaxExponent,
			MinExponent: apd.MinExponent,
			Traps:       apd.DefaultTraps,
			Rounding:    apd.RoundHalfEven, // Banker's rounding for final result
		}
		cnt := apd.New(count, 0)
		avg := apd.New(0, 0)
		if _, err := avgCtx.Quo(avg, sum, cnt); err != nil {
			log.Printf("Warning: Error calculating average: %v", err)
			return
		}
		if _, err := avgCtx.Quantize(avg, avg, -2); err != nil {
			log.Printf("Warning: Error quantizing average: %v", err)
			return
		}
		fmt.Printf("Processed %d records\n", count)
		fmt.Println("Average price: ", avg)
	} else {
		fmt.Println("No valid price records found")
	}
}

// parsePrice extracts a float value from a price string by removing
// currency symbols, commas and other non-numeric characters
func parsePrice(price string) (d *apd.Decimal, err error) {
	// Remove currency symbol, commas, and spaces
	price = strings.ReplaceAll(price, "$", "")
	price = strings.ReplaceAll(price, "€", "")
	price = strings.ReplaceAll(price, "£", "")
	price = strings.Replace(price, ",", "", -1)
	price = strings.Replace(price, " ", "", -1)
	price = strings.TrimSpace(price)

	d, _, err = apd.NewFromString(price)
	return d, err
}
