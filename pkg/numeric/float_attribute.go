package numeric

import (
	"math"
	"strconv"
	"strings"

	attr "propertytreeanalyzer/pkg/api/attribute"
)

const (
	epsilon = 1e-6
)

var (
	_ attr.NumericAttribute = (*floatAttribute)(nil)
	_ FloatValue            = (*floatAttribute)(nil)
)

type floatAttribute struct {
	value float64
}

type FloatValue interface {
	// GetFloat returns the float64 value
	GetFloat() float64
}

// GetFloat implements FloatValue.
func (f *floatAttribute) GetFloat() float64 {
	return f.value
}

// GetNumericType implements attribute.NumericAttribute.
func (f *floatAttribute) GetNumericType() attr.NumericType {
	return attr.Float
}

// String implements attribute.NumericAttribute.
func (f *floatAttribute) String() string {
	return strconv.FormatFloat(f.value, 'f', -1, 64)
}

// EqualTo implements attribute.NumericAttribute.
func (f *floatAttribute) EqualTo(other attr.NumericAttribute) bool {
	if other == nil {
		return false
	}
	if f.GetNumericType() != other.GetNumericType() {
		return false
	}
	floatValue, ok := other.(FloatValue)
	if !ok {
		return false
	}
	return math.Abs(f.value-floatValue.GetFloat()) <= epsilon
}

// NewFloatAttribute creates a new float attribute from a float64 value.
func NewFloatAttribute(value float64) attr.NumericAttribute {
	return &floatAttribute{value: value}
}

// ParseFloatAttribute creates a new float attribute from a string value.
func ParseFloatAttribute(value string) (attr.NumericAttribute, error) {
	value = strings.Replace(value, "$", "", -1)
	value = strings.Replace(value, "€", "", -1)
	value = strings.Replace(value, "£", "", -1)
	value = strings.Replace(value, ",", "", -1)
	value = strings.Replace(value, " ", "", -1)

	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return nil, err
	}
	return &floatAttribute{value: v}, nil
}

// CastToFloatAttribute attempts to cast a NumericAttribute to a FloatValue.
func CastToFloatAttribute(value attr.NumericAttribute) (FloatValue, error) {
	if value.GetNumericType() != attr.Float {
		return nil, ErrInvalidType
	}
	fv, ok := value.(FloatValue)
	if !ok {
		return nil, ErrInvalidType
	}
	return fv, nil
}
