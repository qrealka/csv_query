package aggregators

import (
	apd "github.com/cockroachdb/apd/v3"
)

// AverageByGroup represents aggregated values for a group
type AverageByGroup interface {
	GroupKey() any
	AverageValue() apd.Decimal
}
