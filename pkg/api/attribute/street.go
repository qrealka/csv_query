package attribute

// StreetAttribute represents a street name and an associated attribute value
type StreetAttribute interface {
	// StreetName returns the name of the street
	StreetName() string

	// AttributeValue returns the value of the attribute associated with the street
	AttributeValue() string

	// EqualTo checks if two street attributes are equal
	EqualTo(other StreetAttribute) bool
}
