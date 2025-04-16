package numeric

import (
	"strings"

	attr "propertytreeanalyzer/pkg/api/attribute"

	apd "github.com/cockroachdb/apd/v3"
)

var (
	_ attr.NumericAttribute = (*decimalAttribute)(nil)
	_ DecimalValue          = (*decimalAttribute)(nil)
)

type decimalAttribute struct {
	value apd.Decimal
}

type DecimalValue interface {
	// GetValue returns the decimal value
	GetDecimal() *apd.Decimal
}

// GetDecimal implements DecimalValue.
func (d *decimalAttribute) GetDecimal() *apd.Decimal {
	return &d.value
}

// GetNumericType implements attribute.NumericAttribute.
func (d *decimalAttribute) GetNumericType() attr.NumericType {
	return attr.Decimal
}

// String implements attribute.NumericAttribute.
func (d *decimalAttribute) String() string {
	return d.value.String()
}

// EqualTo implements attribute.NumericAttribute.
func (d *decimalAttribute) EqualTo(other attr.NumericAttribute) bool {
	if other == nil {
		return false
	}
	if d.GetNumericType() != other.GetNumericType() {
		return false
	}
	decimalValue, ok := other.(DecimalValue)
	if !ok {
		return false
	}
	return d.value.Cmp(decimalValue.GetDecimal()) == 0
}

// NewDecimalAttribute creates a new decimal attribute from an apd.Decimal value.
func NewDecimalAttribute(value *apd.Decimal) attr.NumericAttribute {
	return &decimalAttribute{
		value: *value,
	}
}

// NewDecimalAttributeFromString creates a new decimal attribute from a string value.
func ParseDecimalAttribute(value string) (attr.NumericAttribute, error) {
	value = strings.Replace(value, "$", "", -1)
	value = strings.Replace(value, "€", "", -1)
	value = strings.Replace(value, "£", "", -1)
	value = strings.Replace(value, ",", "", -1)
	value = strings.Replace(value, " ", "", -1)

	decimalValue, _, err := apd.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	return &decimalAttribute{
		value: *decimalValue,
	}, nil
}

// CastToDecimalAttribute attempts to cast a NumericAttribute to a DecimalValue.
func CastToDecimalAttribute(value attr.NumericAttribute) (DecimalValue, error) {
	valueType := value.GetNumericType()
	if valueType != attr.Decimal {
		return nil, ErrInvalidType
	}
	decimalValue, ok := value.(DecimalValue)
	if !ok {
		return nil, ErrInvalidType
	}
	return decimalValue, nil
}
