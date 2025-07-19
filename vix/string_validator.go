package vix

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/c3p0-box/utils/erm"
)

// =============================================================================
// String Validator Type and Constructor
// =============================================================================

// StringValidator provides validation rules for string values.
// It supports method chaining for readable and maintainable validation.
type StringValidator struct {
	*BaseValidator
}

// String creates a new StringValidator for the given value and field name.
//
// Example:
//
//	err := validator.String("john@example.com", "email").
//		Required().
//		Email().
//		MaxLength(100).
//		Validate()
func String(value string, fieldName string) *StringValidator {
	return &StringValidator{
		BaseValidator: NewBaseValidator(value, fieldName),
	}
}

// =============================================================================
// Chain Methods
// =============================================================================
// Note: Chain methods (Not, When, Unless, Custom) are inherited from BaseValidator
// and automatically return the correct type through interface satisfaction.

// Not negates the next validation rule.
func (sv *StringValidator) Not() *StringValidator {
	sv.BaseValidator.Not()
	return sv
}

// When adds a condition that must be true for validation to run.
func (sv *StringValidator) When(condition func() bool) *StringValidator {
	sv.BaseValidator.When(condition)
	return sv
}

// Unless adds a condition that must be false for validation to run.
func (sv *StringValidator) Unless(condition func() bool) *StringValidator {
	sv.BaseValidator.Unless(condition)
	return sv
}

// Custom validates using a custom validation function.
func (sv *StringValidator) Custom(fn func(value interface{}) error) *StringValidator {
	sv.BaseValidator.Custom(fn)
	return sv
}

// =============================================================================
// Basic Validation
// =============================================================================

// Required validates that the string is not empty (after trimming whitespace).
func (sv *StringValidator) Required() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	isValid := !isEmpty(str)

	if !isValid && !sv.negated {
		sv.addValidationError("required", MsgRequired, nil)
	} else if isValid && sv.negated {
		sv.addValidationError("not_required", "validation.empty", nil)
	}

	sv.negated = false
	return sv
}

// Empty validates that the string is empty (exactly empty, not just whitespace).
func (sv *StringValidator) Empty() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	isValid := str == ""

	if !isValid && !sv.negated {
		sv.addValidationError("required", "validation.empty", nil)
	} else if isValid && sv.negated {
		sv.addValidationError("not_required", "validation.not_empty", nil)
	}

	sv.negated = false
	return sv
}

// =============================================================================
// Length/Range Validation
// =============================================================================

// MinLength validates that the string has at least the specified number of characters.
func (sv *StringValidator) MinLength(min int) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	length := getLength(str)
	isValid := length >= min

	if !isValid && !sv.negated {
		sv.addValidationError("min_length", MsgMinLength,
			map[string]interface{}{"min": min})
	} else if isValid && sv.negated {
		sv.addValidationError("not_min_length", "validation.not_min_length",
			map[string]interface{}{"min": min})
	}

	sv.negated = false
	return sv
}

// MaxLength validates that the string has at most the specified number of characters.
func (sv *StringValidator) MaxLength(max int) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	length := getLength(str)
	isValid := length <= max

	if !isValid && !sv.negated {
		sv.addValidationError("max_length", MsgMaxLength,
			map[string]interface{}{"max": max})
	} else if isValid && sv.negated {
		sv.addValidationError("not_max_length", "validation.not_max_length",
			map[string]interface{}{"max": max})
	}

	sv.negated = false
	return sv
}

// ExactLength validates that the string has exactly the specified number of characters.
func (sv *StringValidator) ExactLength(length int) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	actualLength := getLength(str)
	isValid := actualLength == length

	if !isValid && !sv.negated {
		sv.addValidationError("exact_length", MsgExactLength,
			map[string]interface{}{"length": length})
	} else if isValid && sv.negated {
		sv.addValidationError("not_exact_length", "validation.not_exact_length",
			map[string]interface{}{"length": length})
	}

	sv.negated = false
	return sv
}

// LengthBetween validates that the string length is between min and max (inclusive).
func (sv *StringValidator) LengthBetween(min, max int) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	length := getLength(str)
	isValid := length >= min && length <= max

	if !isValid && !sv.negated {
		sv.addValidationError("between", MsgBetween,
			map[string]interface{}{"min": min, "max": max})
	} else if isValid && sv.negated {
		sv.addValidationError("not_between", "validation.not_between",
			map[string]interface{}{"min": min, "max": max})
	}

	sv.negated = false
	return sv
}

// =============================================================================
// Format Validation
// =============================================================================

// Email validates that the string is a valid email address format.
func (sv *StringValidator) Email() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	isValid := EmailRegex.MatchString(str)

	if !isValid && !sv.negated {
		sv.addValidationError("email", MsgEmail, nil)
	} else if isValid && sv.negated {
		sv.addValidationError("not_email", "validation.not_email", nil)
	}

	sv.negated = false
	return sv
}

// URL validates that the string is a valid URL format.
func (sv *StringValidator) URL() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	isValid := URLRegex.MatchString(str)

	if !isValid && !sv.negated {
		sv.addValidationError("url", MsgURL, nil)
	} else if isValid && sv.negated {
		sv.addValidationError("not_url", "validation.not_url", nil)
	}

	sv.negated = false
	return sv
}

// Numeric validates that the string contains only numeric characters.
func (sv *StringValidator) Numeric() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := str != "" && NumericRegex.MatchString(str)

	if !valid && !sv.negated {
		sv.addValidationError("numeric", MsgNumeric, nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_numeric", "validation.not_numeric", nil)
	}

	sv.negated = false
	return sv
}

// Alpha validates that the string contains only alphabetic characters.
func (sv *StringValidator) Alpha() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := str != "" && AlphaRegex.MatchString(str)

	if !valid && !sv.negated {
		sv.addValidationError("alpha", MsgAlpha, nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_alpha", "validation.not_alpha", nil)
	}

	sv.negated = false
	return sv
}

// AlphaNumeric validates that the string contains only alphanumeric characters.
func (sv *StringValidator) AlphaNumeric() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := str != "" && AlphaNumericRegex.MatchString(str)

	if !valid && !sv.negated {
		sv.addValidationError("alpha_numeric", MsgAlphaNumeric, nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_alpha_numeric", "validation.not_alpha_numeric", nil)
	}

	sv.negated = false
	return sv
}

// Regex validates that the string matches the given regular expression.
func (sv *StringValidator) Regex(pattern *regexp.Regexp) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := pattern.MatchString(str)

	if !valid && !sv.negated {
		sv.addValidationError("regex", MsgRegex,
			map[string]interface{}{"pattern": pattern.String()})
	} else if valid && sv.negated {
		sv.addValidationError("not_regex", "validation.not_regex",
			map[string]interface{}{"pattern": pattern.String()})
	}

	sv.negated = false
	return sv
}

// =============================================================================
// In/NotIn Validation
// =============================================================================

// In validates that the string is one of the specified values.
func (sv *StringValidator) In(values ...string) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := false
	for _, v := range values {
		if str == v {
			valid = true
			break
		}
	}

	if !valid && !sv.negated {
		sv.addValidationError("in", "{{field}} must be one of: {{values}}",
			map[string]interface{}{"values": strings.Join(values, ", ")})
	} else if valid && sv.negated {
		sv.addValidationError("not_in", "{{field}} must not be one of: {{values}}",
			map[string]interface{}{"values": strings.Join(values, ", ")})
	}

	sv.negated = false
	return sv
}

// NotIn validates that the string is not one of the specified values.
func (sv *StringValidator) NotIn(values ...string) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := true
	for _, v := range values {
		if str == v {
			valid = false
			break
		}
	}

	if !valid && !sv.negated {
		sv.addValidationError("not_in", "{{field}} must not be one of: {{values}}",
			map[string]interface{}{"values": strings.Join(values, ", ")})
	} else if valid && sv.negated {
		sv.addValidationError("not_not_in", "{{field}} may be one of: {{values}}",
			map[string]interface{}{"values": strings.Join(values, ", ")})
	}

	sv.negated = false
	return sv
}

// =============================================================================
// Contains/StartsWith/EndsWith Validation
// =============================================================================

// Contains validates that the string contains the specified substring.
func (sv *StringValidator) Contains(substring string) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := strings.Contains(str, substring)

	if !valid && !sv.negated {
		sv.addValidationError("contains", "{{field}} must contain '{{substring}}'",
			map[string]interface{}{"substring": substring})
	} else if valid && sv.negated {
		sv.addValidationError("not_contains", "{{field}} must not contain '{{substring}}'",
			map[string]interface{}{"substring": substring})
	}

	sv.negated = false
	return sv
}

// StartsWith validates that the string starts with the specified prefix.
func (sv *StringValidator) StartsWith(prefix string) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := strings.HasPrefix(str, prefix)

	if !valid && !sv.negated {
		sv.addValidationError("starts_with", "{{field}} must start with '{{prefix}}'",
			map[string]interface{}{"prefix": prefix})
	} else if valid && sv.negated {
		sv.addValidationError("not_starts_with", "{{field}} must not start with '{{prefix}}'",
			map[string]interface{}{"prefix": prefix})
	}

	sv.negated = false
	return sv
}

// EndsWith validates that the string ends with the specified suffix.
func (sv *StringValidator) EndsWith(suffix string) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := strings.HasSuffix(str, suffix)

	if !valid && !sv.negated {
		sv.addValidationError("ends_with", "{{field}} must end with '{{suffix}}'",
			map[string]interface{}{"suffix": suffix})
	} else if valid && sv.negated {
		sv.addValidationError("not_ends_with", "{{field}} must not end with '{{suffix}}'",
			map[string]interface{}{"suffix": suffix})
	}

	sv.negated = false
	return sv
}

// =============================================================================
// Case Validation
// =============================================================================

// Lowercase validates that the string is in lowercase.
func (sv *StringValidator) Lowercase() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := str == strings.ToLower(str)

	if !valid && !sv.negated {
		sv.addValidationError("lowercase", "{{field}} must be in lowercase", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_lowercase", "{{field}} must not be in lowercase", nil)
	}

	sv.negated = false
	return sv
}

// Uppercase validates that the string is in uppercase.
func (sv *StringValidator) Uppercase() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := str == strings.ToUpper(str)

	if !valid && !sv.negated {
		sv.addValidationError("uppercase", "{{field}} must be in uppercase", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_uppercase", "{{field}} must not be in uppercase", nil)
	}

	sv.negated = false
	return sv
}

// =============================================================================
// Type Validation
// =============================================================================

// Integer validates that the string represents a valid integer.
func (sv *StringValidator) Integer() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	_, err := strconv.Atoi(str)
	valid := err == nil

	if !valid && !sv.negated {
		sv.addValidationError("integer", "{{field}} must be a valid integer", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_integer", "{{field}} must not be a valid integer", nil)
	}

	sv.negated = false
	return sv
}

// Float validates that the string represents a valid floating-point number.
func (sv *StringValidator) Float() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	_, err := strconv.ParseFloat(str, 64)
	valid := err == nil

	if !valid && !sv.negated {
		sv.addValidationError("float", "{{field}} must be a valid number", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_float", "{{field}} must not be a valid number", nil)
	}

	sv.negated = false
	return sv
}

// JSON validates that the string is valid JSON.
func (sv *StringValidator) JSON() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := isValidJSON(str)

	if !valid && !sv.negated {
		sv.addValidationError("json", "{{field}} must be valid JSON", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_json", "{{field}} must not be valid JSON", nil)
	}

	sv.negated = false
	return sv
}

// Base64 validates that the string is valid base64.
func (sv *StringValidator) Base64() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := isValidBase64(str)

	if !valid && !sv.negated {
		sv.addValidationError("base64", "{{field}} must be valid base64", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_base64", "{{field}} must not be valid base64", nil)
	}

	sv.negated = false
	return sv
}

// UUID validates that the string is a valid UUID.
func (sv *StringValidator) UUID() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := isValidUUID(str)

	if !valid && !sv.negated {
		sv.addValidationError("uuid", "{{field}} must be a valid UUID", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_uuid", "{{field}} must not be a valid UUID", nil)
	}

	sv.negated = false
	return sv
}

// Slug validates that the string is a valid URL slug.
func (sv *StringValidator) Slug() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := isValidSlug(str)

	if !valid && !sv.negated {
		sv.addValidationError("slug", "{{field}} must be a valid slug", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_slug", "{{field}} must not be a valid slug", nil)
	}

	sv.negated = false
	return sv
}

// validateStringFormat validates string against various format patterns.
// This function consolidates common string format validation logic.
func validateStringFormat(value, fieldName string, validators ...func(string) bool) error {
	str := toString(value)
	for _, validator := range validators {
		if !validator(str) {
			return erm.NewValidationError("{{field}} format is invalid", fieldName, value)
		}
	}
	return nil
}

// String format validation helper functions
// These functions are used internally and can be reused across different validators.

// isValidJSON checks if the string is valid JSON.
func isValidJSON(str string) bool {
	if strings.TrimSpace(str) == "" {
		return false
	}
	var js interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}

// isValidUUID checks if the string is a valid UUID (version 4 format).
func isValidUUID(str string) bool {
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(str) != 36 {
		return false
	}

	// Check format with regex for better validation
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(str)
}

// isValidSlug checks if the string is a valid URL slug.
// A valid slug contains only lowercase letters, numbers, and hyphens,
// and doesn't start or end with a hyphen.
func isValidSlug(str string) bool {
	if str == "" {
		return false
	}

	// Check for valid slug pattern
	slugRegex := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return slugRegex.MatchString(str)
}

// isValidBase64 checks if the string is valid base64 encoding.
func isValidBase64(str string) bool {
	if str == "" {
		return false
	}

	// Try to decode the string to validate it's proper base64
	// This handles both padded and unpadded base64 strings
	_, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		// If standard decoding fails, try with unpadded base64
		_, err = base64.RawStdEncoding.DecodeString(str)
	}

	return err == nil
}
