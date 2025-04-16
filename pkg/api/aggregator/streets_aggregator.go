package aggregators

import (
	"context"

	attr "propertytreeanalyzer/pkg/api/attribute"
)

// StreetGroupAggregator joins street attributes with street groups
// and calculates statistics for each group
type StreetGroupAggregator interface {
	// Process reads attributes from a channel, groups them by street name,
	// and returns statistics for each group
	Process(ctx context.Context, attributes <-chan attr.StreetAttribute) error
}
