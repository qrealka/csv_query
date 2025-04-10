# Property Tree Analyzer

A Go application for analyzing property values in relation to street tree density.

## Overview

This application processes:
- Property data from CSV files containing property details like addresses, prices, and sizes
- Street tree data from JSON files containing information about trees and their locations

It then performs analysis to determine if there's a correlation between street tree density and property values.

## Project Structure

```
propertytreeanalyzer/
├── cmd/
│   └── propertytreeanalyzer/
│       └── main.go         # Main application entry point, CLI handling
├── pkg/
│   ├── config/             # Configuration management
│   │   └── config.go
│   ├── streetclassifier/   # Logic for loading & classifying streets from JSON
│   │   ├── classifier.go
│   │   └── classifier_test.go
│   ├── csvparser/          # Logic for parsing the CSV property data
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── aggregator/         # Logic for aggregating property prices and statistics
│   │   ├── aggregator.go
│   │   └── aggregator_test.go
│   └── pipeline/           # Coordinates the concurrent processing workflow
│       └── pipeline.go
├── data/                   # Sample/test data files
│   ├── test_trees.json
│   └── test_properties.csv
```

## Installation

This project requires Go 1.24.1 or higher.

```bash
git clone <repository-url>
cd propertytreeanalyzer
go build ./cmd/propertytreeanalyzer
```

## Usage

### Basic Usage

```bash
./propertytreeanalyzer --trees path/to/trees.json --properties path/to/properties.csv --output results.csv
```

### Command-line Arguments

- `--trees`: Path to the JSON file containing street tree data (required)
- `--properties`: Path to the CSV file containing property data (required)
- `--output`: Path to output the results (default: results.csv)

## Example Output

The application will generate a CSV file with the following columns:
- Street: Street name
- Tree Count: Number of trees on the street
- Tree Density: Classification of tree density (Low, Medium, High)
- Property Count: Number of properties on the street
- Avg Price: Average property price on the street
- Median Price: Median property price on the street
- Min Price: Minimum property price on the street
- Max Price: Maximum property price on the street
- Avg Price Per SqFt: Average price per square foot on the street

## Running Tests

To run tests for all packages:

```bash
go test ./...
```

To run tests for a specific package:

```bash
go test ./pkg/aggregator
go test ./pkg/csvparser
go test ./pkg/streetclassifier
```

## License

[Specify your license here]