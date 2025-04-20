# Property Tree Analyzer

A Go application for analyzing property values in relation to street tree density.

## Overview

This application processes:

- Property data from CSV files containing property details like addresses, prices, and sizes
- Street tree data from JSON files containing information about trees and their locations

It then performs analysis to determine if there's a correlation between street tree density and property values.

We have two files: dublin-trees.json and dublin-property.csv:

- dublin-trees.json contains a list of street names. Streets are split into two categories: `short` and `tall`, based on the median tree height as recorded by Dublin City Council.
- dublin-property.csv contains a subset of the Residential Property Price Register, with a list of property addresses, their street name and sale price in euro.

We've cleaned the datasets a little, so the street name in dublin-trees.json exactly matches the `Street Name` column in dublin-property.csv.

### dublin-trees.json structure

There are two top-level entries: short and tall.

Street names are in an arbitrarily nested structure, only the entry with a height is relevant and this entry contains the complete street name.

Here's an example:

```json
        {
            "short": {
                "drive": {
                    "abbey": {
                        "abbey drive": 0
                    },
                    "coolrua": {
                        "coolrua drive": 10
                    },
                    "coultry": {
                        "coultry drive": 5
                    },
                }
            },
            "tall": {
                "gardens": {
                    "temple": {
                        "temple gardens": 20
                    }
                },
                "bramblings": {
                    "the": {
                        "the bramblings": 20
                    }
                },
            }
        }
```

The "short tree" street names in this example are:

- abbey drive
- coolrua drive
- coultry drive

and the "tall tree" street names are:

- temple gardens
- the bramblings

## Building

### Requirements

Docker must be installed and running. The build process uses a Go container image to ensure a consistent build environment.

### How to build

To build the application binary:

1. Clone the repository.

2. Navigate to the project's root directory.

3. Run the following command:

   ```bash
   make build
   ```

This command will use Docker to compile the Go code and place the resulting binary (brightbeam) in the *buildDir* directory.

## Project Structure (for Developers)

The project follows a standard Go project layout:

- cmd/brightbeam: Contains the main application entry point (main.go).
- data: Contains sample data files for analysis.
  - dublin-trees.json, dublin-property.csv: Sample data files.
- pkg: Contains the core logic of the application, organized into sub-packages:
  - aggregator/: Logic for calculating average prices based on groups.
  - api/: Defines interfaces used throughout the application (e.g., for streams, parsers, attributes, grouping).
  - csvparser/: Logic for parsing the property CSV data.
  - groupify/: Logic for grouping streets based on the tree JSON data.
  - streams/: Implementations for reading data streams (CSV, JSON).
- Makefile: Defines build and test automation tasks.
- go.mod, go.sum: Go module dependency management files.
