package vix

import (
	"fmt"
	"math"
	"strings"
)

// =============================================================================
// Number Type Constraint and Constructor
// =============================================================================

// Number defines a type constraint for all numeric types
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// NumberValidator provides validation rules for numeric values.
// It supports method chaining for readable and maintainable validation.
type NumberValidator[T Number] struct {
	*BaseValidator
	value T
}

// Numeric creates a new NumberValidator for the given value and field name.
// This function uses Go generics to support all numeric types.
//
// Example:
//
//	err := validator.Numeric(42, "age").
//		Min(18).
//		Max(100).
//		Validate()
func Numeric[T Number](value T, fieldName string) *NumberValidator[T] {
	return &NumberValidator[T]{
		BaseValidator: NewBaseValidator(value, fieldName),
		value:         value,
	}
}

// =============================================================================
// Chain Methods
// =============================================================================

// Not negates the next validation rule.
func (nv *NumberValidator[T]) Not() *NumberValidator[T] {
	nv.BaseValidator.Not()
	return nv
}

// When adds a condition that must be true for validation to run.
func (nv *NumberValidator[T]) When(condition func() bool) *NumberValidator[T] {
	nv.BaseValidator.When(condition)
	return nv
}

// Unless adds a condition that must be false for validation to run.
func (nv *NumberValidator[T]) Unless(condition func() bool) *NumberValidator[T] {
	nv.BaseValidator.Unless(condition)
	return nv
}

// Custom validates using a custom validation function.
// The function receives both the numeric value being validated and the field name,
// allowing for more contextual error messages.
//
// Example:
//
//	err := vix.Int(13, "age").
//		Custom(func(value interface{}, fieldName string) error {
//			age := value.(int)
//			if age == 13 {
//				return erm.NewValidationError("{{field}} cannot be unlucky number 13", fieldName, value)
//			}
//			return nil
//		}).
//		Validate()
func (nv *NumberValidator[T]) Custom(fn func(value interface{}, fieldName string) error) *NumberValidator[T] {
	nv.BaseValidator.Custom(fn)
	return nv
}

// =============================================================================
// Basic Validation
// =============================================================================

// Required validates that the number is not zero.
func (nv *NumberValidator[T]) Required() *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	isZero := nv.value == 0

	if isZero && !nv.negated {
		nv.addValidationError("required", MsgRequired, nil)
	} else if !isZero && nv.negated {
		nv.addValidationError("not_required", "validation.must_be_zero", nil)
	}

	nv.negated = false
	return nv
}

// Zero validates that the number is zero.
func (nv *NumberValidator[T]) Zero() *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	isZero := nv.value == 0

	if !isZero && !nv.negated {
		nv.addValidationError("zero", "{{field}} must be zero", nil)
	} else if isZero && nv.negated {
		nv.addValidationError("not_zero", "{{field}} must not be zero", nil)
	}

	nv.negated = false
	return nv
}

// Min validates that the number is greater than or equal to the minimum value.
func (nv *NumberValidator[T]) Min(min T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value >= min

	if !valid && !nv.negated {
		nv.addValidationError("min", MsgMin,
			map[string]interface{}{"min": min, "value": nv.value})
	} else if valid && nv.negated {
		nv.addValidationError("not_min", "validation.not_min_value",
			map[string]interface{}{"min": min, "value": nv.value})
	}

	nv.negated = false
	return nv
}

// Max validates that the number is less than or equal to the maximum value.
func (nv *NumberValidator[T]) Max(max T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value <= max

	if !valid && !nv.negated {
		nv.addValidationError("max", MsgMax,
			map[string]interface{}{"max": max, "value": nv.value})
	} else if valid && nv.negated {
		nv.addValidationError("not_max", "validation.not_max_value",
			map[string]interface{}{"max": max, "value": nv.value})
	}

	nv.negated = false
	return nv
}

// Between validates that the number is between min and max (inclusive).
func (nv *NumberValidator[T]) Between(min, max T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value >= min && nv.value <= max

	if !valid && !nv.negated {
		nv.addValidationError("between", "{{field}} must be between {{min}} and {{max}}",
			map[string]interface{}{"min": min, "max": max, "value": nv.value})
	} else if valid && nv.negated {
		nv.addValidationError("not_between", "{{field}} must not be between {{min}} and {{max}}",
			map[string]interface{}{"min": min, "max": max, "value": nv.value})
	}

	nv.negated = false
	return nv
}

// Equal validates that the number equals the specified value.
func (nv *NumberValidator[T]) Equal(expected T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value == expected

	if !valid && !nv.negated {
		nv.addValidationError("equal", "{{field}} must equal {{expected}}",
			map[string]interface{}{"expected": expected, "value": nv.value})
	} else if valid && nv.negated {
		nv.addValidationError("not_equal", "{{field}} must not equal {{expected}}",
			map[string]interface{}{"expected": expected, "value": nv.value})
	}

	nv.negated = false
	return nv
}

// EqualTo validates that the number equals the specified value.
// Optionally accepts a custom error message template as the second parameter.
// If no custom message is provided, uses the default localized message.
//
// Example:
//
//	// With default message
//	err := vix.Int(25, "age").EqualTo(18).Validate()
//
//	// With custom message template
//	err = vix.Int(25, "age").
//		EqualTo(18, "{{field}} must be exactly {{expected}} years old").
//		Validate()
//
//	// Works with negation and different numeric types
//	err = vix.Float64(3.14, "pi").
//		Not().EqualTo(2.71).
//		Validate()
func (nv *NumberValidator[T]) EqualTo(expected T, msgTemplate ...string) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value == expected

	// Use custom message template if provided, otherwise use default
	var messageKey string
	if len(msgTemplate) > 0 && msgTemplate[0] != "" {
		messageKey = msgTemplate[0]
	} else {
		messageKey = MsgEqualTo
	}

	if !valid && !nv.negated {
		nv.addValidationError("equal_to", messageKey,
			map[string]interface{}{"expected": expected, "value": nv.value})
	} else if valid && nv.negated {
		nv.addValidationError("not_equal_to", "validation.not_equal_to",
			map[string]interface{}{"expected": expected, "value": nv.value})
	}

	nv.negated = false
	return nv
}

// GreaterThan validates that the number is greater than the specified value.
func (nv *NumberValidator[T]) GreaterThan(value T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value > value

	if !valid && !nv.negated {
		nv.addValidationError("greater_than", "{{field}} must be greater than {{value}}",
			map[string]interface{}{"value": value, "actual": nv.value})
	} else if valid && nv.negated {
		nv.addValidationError("not_greater_than", "{{field}} must not be greater than {{value}}",
			map[string]interface{}{"value": value, "actual": nv.value})
	}

	nv.negated = false
	return nv
}

// LessThan validates that the number is less than the specified value.
func (nv *NumberValidator[T]) LessThan(value T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value < value

	if !valid && !nv.negated {
		nv.addValidationError("less_than", "{{field}} must be less than {{value}}",
			map[string]interface{}{"value": value, "actual": nv.value})
	} else if valid && nv.negated {
		nv.addValidationError("not_less_than", "{{field}} must not be less than {{value}}",
			map[string]interface{}{"value": value, "actual": nv.value})
	}

	nv.negated = false
	return nv
}

// Positive validates that the number is positive (greater than zero).
func (nv *NumberValidator[T]) Positive() *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value > 0

	if !valid && !nv.negated {
		nv.addValidationError("positive", "{{field}} must be positive", nil)
	} else if valid && nv.negated {
		nv.addValidationError("not_positive", "{{field}} must not be positive", nil)
	}

	nv.negated = false
	return nv
}

// Negative validates that the number is negative (less than zero).
func (nv *NumberValidator[T]) Negative() *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := nv.value < 0

	if !valid && !nv.negated {
		nv.addValidationError("negative", "{{field}} must be negative", nil)
	} else if valid && nv.negated {
		nv.addValidationError("not_negative", "{{field}} must not be negative", nil)
	}

	nv.negated = false
	return nv
}

// In validates that the number is one of the specified values.
func (nv *NumberValidator[T]) In(values ...T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := false
	for _, v := range values {
		if nv.value == v {
			valid = true
			break
		}
	}

	if !valid && !nv.negated {
		nv.addValidationError("in", "{{field}} must be one of: {{values}}",
			map[string]interface{}{"values": formatValues(values)})
	} else if valid && nv.negated {
		nv.addValidationError("not_in", "{{field}} must not be one of: {{values}}",
			map[string]interface{}{"values": formatValues(values)})
	}

	nv.negated = false
	return nv
}

// NotIn validates that the number is not one of the specified values.
func (nv *NumberValidator[T]) NotIn(values ...T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := true
	for _, v := range values {
		if nv.value == v {
			valid = false
			break
		}
	}

	if !valid && !nv.negated {
		nv.addValidationError("not_in", "{{field}} must not be one of: {{values}}",
			map[string]interface{}{"values": formatValues(values)})
	} else if valid && nv.negated {
		nv.addValidationError("not_not_in", "{{field}} may be one of: {{values}}",
			map[string]interface{}{"values": formatValues(values)})
	}

	nv.negated = false
	return nv
}

// MultipleOf validates that the number is a multiple of the specified value.
func (nv *NumberValidator[T]) MultipleOf(divisor T) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	if divisor == 0 {
		nv.addValidationError("invalid_divisor", "{{field}} divisor cannot be zero", nil)
		return nv
	}

	valid := false
	switch any(nv.value).(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// For integers, use modulo
		valid = int64(nv.value)%int64(divisor) == 0
	case float32, float64:
		// For floats, use math.Mod
		valid = math.Mod(float64(nv.value), float64(divisor)) == 0
	}

	if !valid && !nv.negated {
		nv.addValidationError("multiple_of", "{{field}} must be a multiple of {{divisor}}",
			map[string]interface{}{"divisor": divisor})
	} else if valid && nv.negated {
		nv.addValidationError("not_multiple_of", "{{field}} must not be a multiple of {{divisor}}",
			map[string]interface{}{"divisor": divisor})
	}

	nv.negated = false
	return nv
}

// Even validates that the number is even.
func (nv *NumberValidator[T]) Even() *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := int64(nv.value)%2 == 0

	if !valid && !nv.negated {
		nv.addValidationError("even", "{{field}} must be even", nil)
	} else if valid && nv.negated {
		nv.addValidationError("not_even", "{{field}} must not be even", nil)
	}

	nv.negated = false
	return nv
}

// Odd validates that the number is odd.
func (nv *NumberValidator[T]) Odd() *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	valid := int64(nv.value)%2 != 0

	if !valid && !nv.negated {
		nv.addValidationError("odd", "{{field}} must be odd", nil)
	} else if valid && nv.negated {
		nv.addValidationError("not_odd", "{{field}} must not be odd", nil)
	}

	nv.negated = false
	return nv
}

// Finite validates that the number is finite (for floating-point numbers).
func (nv *NumberValidator[T]) Finite() *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	var valid bool
	switch v := any(nv.value).(type) {
	case float32:
		valid = !math.IsInf(float64(v), 0) && !math.IsNaN(float64(v))
	case float64:
		valid = !math.IsInf(v, 0) && !math.IsNaN(v)
	default:
		// For integer types, always finite
		valid = true
	}

	if !valid && !nv.negated {
		nv.addValidationError("finite", "{{field}} must be finite", nil)
	} else if valid && nv.negated {
		nv.addValidationError("not_finite", "{{field}} must not be finite", nil)
	}

	nv.negated = false
	return nv
}

// Precision validates that a float has at most the specified number of decimal places.
func (nv *NumberValidator[T]) Precision(places int) *NumberValidator[T] {
	if !nv.shouldValidate() {
		return nv
	}

	var valid bool
	switch v := any(nv.value).(type) {
	case float32, float64:
		// Convert to string and count decimal places
		str := fmt.Sprintf("%g", v)
		parts := strings.Split(str, ".")

		if len(parts) == 1 {
			// No decimal point, so 0 decimal places
			valid = 0 <= places
		} else if len(parts) == 2 {
			// Count decimal places
			decimalPart := parts[1]
			// Remove trailing zeros
			decimalPart = strings.TrimRight(decimalPart, "0")
			actualPlaces := len(decimalPart)
			valid = actualPlaces <= places
		} else {
			// Invalid number format
			valid = false
		}
	default:
		// For integer types, always valid (0 decimal places)
		valid = true
	}

	if !valid && !nv.negated {
		nv.addValidationError("precision", "{{field}} must have at most {{places}} decimal places",
			map[string]interface{}{"places": places})
	} else if valid && nv.negated {
		nv.addValidationError("not_precision", "{{field}} must not have {{places}} decimal places",
			map[string]interface{}{"places": places})
	}

	nv.negated = false
	return nv
}

// Helper function to format values for error messages
func formatValues[T Number](values []T) string {
	if len(values) == 0 {
		return ""
	}

	result := fmt.Sprintf("%v", values[0])
	for i := 1; i < len(values); i++ {
		result += fmt.Sprintf(", %v", values[i])
	}
	return result
}

// =============================================================================
// Convenience Functions
// =============================================================================

// Int creates a NumberValidator for int values.
func Int(value int, fieldName string) *NumberValidator[int] {
	return Numeric(value, fieldName)
}

// Int8 creates a NumberValidator for int8 values.
func Int8(value int8, fieldName string) *NumberValidator[int8] {
	return Numeric(value, fieldName)
}

// Int16 creates a NumberValidator for int16 values.
func Int16(value int16, fieldName string) *NumberValidator[int16] {
	return Numeric(value, fieldName)
}

// Int32 creates a NumberValidator for int32 values.
func Int32(value int32, fieldName string) *NumberValidator[int32] {
	return Numeric(value, fieldName)
}

// Int64 creates a NumberValidator for int64 values.
func Int64(value int64, fieldName string) *NumberValidator[int64] {
	return Numeric(value, fieldName)
}

// Uint creates a NumberValidator for uint values.
func Uint(value uint, fieldName string) *NumberValidator[uint] {
	return Numeric(value, fieldName)
}

// Uint8 creates a NumberValidator for uint8 values.
func Uint8(value uint8, fieldName string) *NumberValidator[uint8] {
	return Numeric(value, fieldName)
}

// Uint16 creates a NumberValidator for uint16 values.
func Uint16(value uint16, fieldName string) *NumberValidator[uint16] {
	return Numeric(value, fieldName)
}

// Uint32 creates a NumberValidator for uint32 values.
func Uint32(value uint32, fieldName string) *NumberValidator[uint32] {
	return Numeric(value, fieldName)
}

// Uint64 creates a NumberValidator for uint64 values.
func Uint64(value uint64, fieldName string) *NumberValidator[uint64] {
	return Numeric(value, fieldName)
}

// Float32 creates a NumberValidator for float32 values.
func Float32(value float32, fieldName string) *NumberValidator[float32] {
	return Numeric(value, fieldName)
}

// Float64 creates a NumberValidator for float64 values.
func Float64(value float64, fieldName string) *NumberValidator[float64] {
	return Numeric(value, fieldName)
}
