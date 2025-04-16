package aggregators

import (
	attr "propertytreeanalyzer/pkg/api/attribute"
)

// AverageByGroup represents aggregated values for a group
type AverageByGroup interface {
	GroupKey() attr.BaseAttribute
	AverageValue() attr.NumericAttribute
}
