package aggregators

import (
	"context"

	attr "propertytreeanalyzer/pkg/api/attribute"
)

// AverageByGroup represents aggregated values for a group
type AverageByGroup interface {
	GroupKey() attr.BaseAttribute
	AverageValue() attr.NumericAttribute
}

type AvgerageAggregator interface {
	Process(ctx context.Context, streets <-chan attr.StreetAttribute) ([]AverageByGroup, error)
}
