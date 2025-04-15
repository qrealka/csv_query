package parsers

// StreetAttribute represents a street name and an associated attribute value
type StreetAttribute interface {
	// StreetName returns the name of the street
	StreetName() string

	// AttributeValue returns the value of the attribute associated with the street
	AttributeValue() any
}

// StreetAttributeParser defines an interface for parsing street attributes
// and sending them to a channel
type StreetAttributeParser interface {
	// ParseAttributes reads data from a source and sends street attribute pairs
	// to the provided channel. The channel is closed when parsing is complete or an error occurs.
	ParseAttributes(out chan<- StreetAttribute) error
}
