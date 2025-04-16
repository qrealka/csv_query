package attribute

// NumericType represents the type of numeric value
type NumericType int

const (
	// Float represents a float64 value
	Float = iota
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
