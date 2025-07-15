package vix

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

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

// Required validates that the string is not empty (after trimming whitespace).
func (sv *StringValidator) Required() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	isEmpty := strings.TrimSpace(str) == ""

	if isEmpty && !sv.negated {
		sv.addValidationError(CodeRequired, "{{field}} is required", nil)
	} else if !isEmpty && sv.negated {
		sv.addValidationError("not_"+CodeRequired, "{{field}} must be empty", nil)
	}

	sv.negated = false
	return sv
}

// Empty validates that the string is empty.
func (sv *StringValidator) Empty() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	isEmpty := str == ""

	if !isEmpty && !sv.negated {
		sv.addValidationError(CodeRequired, "{{field}} must be empty", nil)
	} else if isEmpty && sv.negated {
		sv.addValidationError("not_"+CodeRequired, "{{field}} must not be empty", nil)
	}

	sv.negated = false
	return sv
}

// MinLength validates that the string length is at least the specified minimum.
func (sv *StringValidator) MinLength(min int) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	length := getLength(sv.value)
	valid := length >= min

	if !valid && !sv.negated {
		sv.addValidationError(CodeMinLength, "{{field}} must be at least {{min}} characters long",
			map[string]interface{}{"min": min, "length": length})
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeMinLength, "{{field}} must be less than {{min}} characters long",
			map[string]interface{}{"min": min, "length": length})
	}

	sv.negated = false
	return sv
}

// MaxLength validates that the string length is at most the specified maximum.
func (sv *StringValidator) MaxLength(max int) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	length := getLength(sv.value)
	valid := length <= max

	if !valid && !sv.negated {
		sv.addValidationError(CodeMaxLength, "{{field}} must be at most {{max}} characters long",
			map[string]interface{}{"max": max, "length": length})
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeMaxLength, "{{field}} must be more than {{max}} characters long",
			map[string]interface{}{"max": max, "length": length})
	}

	sv.negated = false
	return sv
}

// ExactLength validates that the string length is exactly the specified length.
func (sv *StringValidator) ExactLength(length int) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	actualLength := getLength(sv.value)
	valid := actualLength == length

	if !valid && !sv.negated {
		sv.addValidationError(CodeExactLength, "{{field}} must be exactly {{length}} characters long",
			map[string]interface{}{"length": length, "actual": actualLength})
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeExactLength, "{{field}} must not be exactly {{length}} characters long",
			map[string]interface{}{"length": length, "actual": actualLength})
	}

	sv.negated = false
	return sv
}

// LengthBetween validates that the string length is between min and max (inclusive).
func (sv *StringValidator) LengthBetween(min, max int) *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	length := getLength(sv.value)
	valid := length >= min && length <= max

	if !valid && !sv.negated {
		sv.addValidationError(CodeBetween, "{{field}} must be between {{min}} and {{max}} characters long",
			map[string]interface{}{"min": min, "max": max, "length": length})
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeBetween, "{{field}} must not be between {{min}} and {{max}} characters long",
			map[string]interface{}{"min": min, "max": max, "length": length})
	}

	sv.negated = false
	return sv
}

// Email validates that the string is a valid email address.
func (sv *StringValidator) Email() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := EmailRegex.MatchString(str)

	if !valid && !sv.negated {
		sv.addValidationError(CodeEmail, "{{field}} must be a valid email address", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeEmail, "{{field}} must not be a valid email address", nil)
	}

	sv.negated = false
	return sv
}

// URL validates that the string is a valid URL.
func (sv *StringValidator) URL() *StringValidator {
	if !sv.shouldValidate() {
		return sv
	}

	str := toString(sv.value)
	valid := URLRegex.MatchString(str)

	if !valid && !sv.negated {
		sv.addValidationError(CodeURL, "{{field}} must be a valid URL", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeURL, "{{field}} must not be a valid URL", nil)
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
		sv.addValidationError(CodeNumeric, "{{field}} must be numeric", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeNumeric, "{{field}} must not be numeric", nil)
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
		sv.addValidationError(CodeAlpha, "{{field}} must contain only letters", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeAlpha, "{{field}} must not contain only letters", nil)
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
		sv.addValidationError(CodeAlphaNumeric, "{{field}} must contain only letters and numbers", nil)
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeAlphaNumeric, "{{field}} must not contain only letters and numbers", nil)
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
		sv.addValidationError(CodeRegex, "{{field}} format is invalid",
			map[string]interface{}{"pattern": pattern.String()})
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeRegex, "{{field}} must not match the required format",
			map[string]interface{}{"pattern": pattern.String()})
	}

	sv.negated = false
	return sv
}

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
		sv.addValidationError(CodeIn, "{{field}} must be one of: {{values}}",
			map[string]interface{}{"values": strings.Join(values, ", ")})
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeIn, "{{field}} must not be one of: {{values}}",
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
		sv.addValidationError(CodeNotIn, "{{field}} must not be one of: {{values}}",
			map[string]interface{}{"values": strings.Join(values, ", ")})
	} else if valid && sv.negated {
		sv.addValidationError("not_"+CodeNotIn, "{{field}} may be one of: {{values}}",
			map[string]interface{}{"values": strings.Join(values, ", ")})
	}

	sv.negated = false
	return sv
}

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
			return NewValidationError("format", "{{field}} format is invalid", fieldName, value)
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
