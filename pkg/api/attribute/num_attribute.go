package attribute

// NumericType represents the type of numeric value
type NumericType int

const (
	// Nothing represents a non-numeric value
	// We need this, b/c zero initialized enums shouldn't be used
	Nothing NumericType = iota
	// Float represents a float64 value
	Float
	// Decimal represents an arbitrary decimal value with fixed precision
	Decimal
)

type NumericAttribute interface {
	BaseAttribute
	// GetNumericType returns the type of numeric value
	GetNumericType() NumericType
	// EqualTo checks if two numeric attributes are equal
	EqualTo(other NumericAttribute) bool
}
