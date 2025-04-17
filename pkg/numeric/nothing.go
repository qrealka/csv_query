package numeric

import (
	attr "propertytreeanalyzer/pkg/api/attribute"
)

type nothingAttribute struct{}

// EqualTo implements attribute.NumericAttribute.
func (n nothingAttribute) EqualTo(other attr.NumericAttribute) bool {
	return false
}

// GetNumericType implements attribute.NumericAttribute.
func (n nothingAttribute) GetNumericType() attr.NumericType {
	return attr.Nothing
}

// String implements attribute.NumericAttribute.
func (n nothingAttribute) String() string {
	return ""
}

var (
	_           attr.NumericAttribute = nothingAttribute{}
	NoneNumeric                       = nothingAttribute{}
)
