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
