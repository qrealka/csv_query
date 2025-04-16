package parsers

import (
	"context"
	attr "propertytreeanalyzer/pkg/api/attribute"
)

// StreetAttributeParser defines an interface for parsing street attributes
// and sending them to a channel
type StreetAttributeParser interface {
	// ParseAttributes reads data from a source and sends street attribute pairs
	// to the provided channel. The channel is closed when parsing is complete or an error occurs.
	ParseAttributes(ctx context.Context, out chan<- attr.StreetAttribute) error
}
