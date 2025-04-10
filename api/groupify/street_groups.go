package groupify

import "context"

type StreetGroupItem interface {
	Key() any
	StreetName() StreetName
}

// StreetGroups defines an interface for grouping street names
type StreetGroups interface {
	GroupStreets(context.Context, chan<- StreetGroupItem) error
}
