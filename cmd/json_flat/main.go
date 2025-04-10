package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// Section represents different sections in the JSON document
type Section int

const (
	SectionNone Section = iota
	SectionShort
	SectionTall
)

// String returns the string representation of a Section
func (s Section) String() string {
	switch s {
	case SectionShort:
		return "short"
	case SectionTall:
		return "tall"
	default:
		return ""
	}
}

// getSectionFromKey determines which section a key belongs to
func getSectionFromKey(key string) Section {
	if strings.EqualFold(key, "short") {
		return SectionShort
	} else if strings.EqualFold(key, "tall") {
		return SectionTall
	}
	return SectionNone
}

func main() {
	// Define a command-line flag for the file path
	var filePath string
	flag.StringVar(&filePath, "file", "", "Path to the JSON file to process")
	flag.Parse()

	// Check if a file path was provided
	if filePath == "" {
		log.Fatal("Please provide a file path using the -file flag")
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error opening file %s: %v", filePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	// Use json.Number to correctly identify numeric tokens without precision loss
	decoder.UseNumber()

	var shortStreets []string
	var tallStreets []string

	var depth int              // Current nesting depth (starts at 0)
	var lastKey string         // Stores the most recently read key
	var currentSection Section // Tracks if we are inside "short", "tall", or "" (neither)

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break // End of file reached successfully
		}
		if err != nil {
			log.Fatalf("Error decoding JSON token: %v", err)
		}

		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '{', '[':
				if depth == 1 {
					currentSection = getSectionFromKey(lastKey)
				}
				depth++

			case '}', ']':
				// Exiting an object or array
				depth--
				if depth == 1 {
					if currentSection != SectionNone {
						currentSection = SectionNone
					}
				}
			}
			lastKey = "" // Reset key after exiting a scope

		case string:
			// This token is a key. Store it.
			lastKey = v

		case json.Number:
			if lastKey != "" {
				// We found a key followed by a number.
				// Add the key to the correct list based on the current section.
				if currentSection == SectionShort {
					shortStreets = append(shortStreets, lastKey)
				} else if currentSection == SectionTall {
					tallStreets = append(tallStreets, lastKey)
				}
				lastKey = ""
			}

		default:
			// Other value types (boolean, null, string *value*).
		}
	}

	// --- Output the results ---
	fmt.Println("\n--- Short Streets ---")
	if len(shortStreets) > 0 {
		for _, street := range shortStreets {
			fmt.Println(street)
		}
		fmt.Printf("Total short streets: %d\n", len(shortStreets))
	} else {
		fmt.Println("(No short streets found)")
	}

	fmt.Println("\n--- Tall Streets ---")
	if len(tallStreets) > 0 {
		for _, street := range tallStreets {
			fmt.Println(street)
		}
		fmt.Printf("Total tall streets: %d\n", len(tallStreets))
	} else {
		fmt.Println("(No tall streets found)")
	}
}
