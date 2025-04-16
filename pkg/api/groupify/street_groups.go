package groupify

import (
	"context"

	attr "propertytreeanalyzer/pkg/api/attribute"
)

type StreetGroupItem interface {
	Key() attr.BaseAttribute
	StreetName() StreetName
}

// StreetGroups defines an interface for grouping street names
type StreetGroups interface {
	GroupStreets(context.Context, chan<- StreetGroupItem) error
}
