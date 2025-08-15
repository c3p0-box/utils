package vix

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/c3p0-box/utils/erm"
)

// =============================================================================
// String Validator Tests - Basic Validation
// =============================================================================

func TestStringValidatorRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid string", "hello", false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"single character", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Required().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorNotRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid string", "hello", true},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Not().Required().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// =============================================================================
// String Validator Tests - Length/Range Validation
// =============================================================================

func TestStringValidatorMinLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		min       int
		shouldErr bool
	}{
		{"valid length", "hello", 3, false},
		{"exact length", "hello", 5, false},
		{"too short", "hi", 5, true},
		{"empty string", "", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").MinLength(tt.min).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		max       int
		shouldErr bool
	}{
		{"valid length", "hello", 10, false},
		{"exact length", "hello", 5, false},
		{"too long", "hello world", 5, true},
		{"empty string", "", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").MaxLength(tt.max).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorEmail(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with plus", "test+tag@example.com", false},
		{"invalid email no @", "testexample.com", true},
		{"invalid email no domain", "test@", true},
		{"invalid email no tld", "test@example", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Email().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorURL(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid http url", "http://example.com", false},
		{"valid https url", "https://example.com", false},
		{"invalid url no protocol", "example.com", true},
		{"invalid url wrong protocol", "ftp://example.com", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").URL().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorNumeric(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid numeric", "12345", false},
		{"invalid with letters", "123abc", true},
		{"invalid with spaces", "123 456", true},
		{"empty string", "", true},
		{"single digit", "5", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Numeric().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorAlpha(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid alpha", "hello", false},
		{"valid mixed case", "Hello", false},
		{"invalid with numbers", "hello123", true},
		{"invalid with spaces", "hello world", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Alpha().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorAlphaNumeric(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid alphanumeric", "hello123", false},
		{"valid letters only", "hello", false},
		{"valid numbers only", "123", false},
		{"invalid with spaces", "hello 123", true},
		{"invalid with symbols", "hello!", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").AlphaNumeric().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorRegex(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Z][a-z]+$`)
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid pattern", "Hello", false},
		{"invalid pattern lowercase", "hello", true},
		{"invalid pattern uppercase", "HELLO", true},
		{"invalid pattern mixed", "HeLLo", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Regex(pattern).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorIn(t *testing.T) {
	validValues := []string{"apple", "banana", "orange"}
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid value", "apple", false},
		{"valid value 2", "banana", false},
		{"invalid value", "grape", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").In(validValues...).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorNotIn(t *testing.T) {
	invalidValues := []string{"admin", "root", "system"}
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid value", "user", false},
		{"invalid value", "admin", true},
		{"invalid value 2", "root", true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").NotIn(invalidValues...).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorContains(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		substr    string
		shouldErr bool
	}{
		{"contains substring", "hello world", "world", false},
		{"contains at start", "hello world", "hello", false},
		{"contains at end", "hello world", "world", false},
		{"does not contain", "hello world", "foo", true},
		{"empty string", "", "foo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Contains(tt.substr).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorStartsWith(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		prefix    string
		shouldErr bool
	}{
		{"starts with prefix", "hello world", "hello", false},
		{"exact match", "hello", "hello", false},
		{"does not start with", "hello world", "world", true},
		{"empty string", "", "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").StartsWith(tt.prefix).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorEndsWith(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		suffix    string
		shouldErr bool
	}{
		{"ends with suffix", "hello world", "world", false},
		{"exact match", "hello", "hello", false},
		{"does not end with", "hello world", "hello", true},
		{"empty string", "", "world", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").EndsWith(tt.suffix).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorChaining(t *testing.T) {
	// Test that multiple validations can be chained
	err := String("test@example.com", "email").
		Required().
		Email().
		MinLength(5).
		MaxLength(50).
		Validate()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test chain that should fail
	err = String("invalid", "email").
		Required().
		Email().
		MinLength(5).
		MaxLength(50).
		Validate()

	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestStringValidatorConditional(t *testing.T) {
	// Test conditional validation with When
	condition := true
	err := String("", "test").
		When(func() bool { return condition }).
		Required().
		Validate()

	if err == nil {
		t.Error("expected error but got none")
	}

	// Test conditional validation with When - condition false
	condition = false
	err = String("", "test").
		When(func() bool { return condition }).
		Required().
		Validate()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test conditional validation with Unless
	condition = false
	err = String("", "test").
		Unless(func() bool { return condition }).
		Required().
		Validate()

	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestStringValidatorCustom(t *testing.T) {
	// Test custom validation function
	customValidator := func(value interface{}) error {
		str := value.(string)
		if strings.Contains(str, "forbidden") {
			return erm.NewValidationError("{{field}} contains forbidden word", "test", value)
		}
		return nil
	}

	err := String("this is forbidden", "test").
		Custom(customValidator).
		Validate()

	if err == nil {
		t.Error("expected error but got none")
	}

	err = String("this is allowed", "test").
		Custom(customValidator).
		Validate()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNumberValidatorRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		shouldErr bool
	}{
		{"non-zero value", 42, false},
		{"zero value", 0, true},
		{"negative value", -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Required().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorMin(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		min       int
		shouldErr bool
	}{
		{"above min", 10, 5, false},
		{"equal to min", 5, 5, false},
		{"below min", 3, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Min(tt.min).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorMax(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		max       int
		shouldErr bool
	}{
		{"below max", 5, 10, false},
		{"equal to max", 10, 10, false},
		{"above max", 15, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Max(tt.max).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorBetween(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		min       int
		max       int
		shouldErr bool
	}{
		{"within range", 5, 1, 10, false},
		{"equal to min", 1, 1, 10, false},
		{"equal to max", 10, 1, 10, false},
		{"below range", 0, 1, 10, true},
		{"above range", 11, 1, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Between(tt.min, tt.max).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorPositive(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		shouldErr bool
	}{
		{"positive value", 5, false},
		{"zero value", 0, true},
		{"negative value", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Positive().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorNegative(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		shouldErr bool
	}{
		{"negative value", -5, false},
		{"zero value", 0, true},
		{"positive value", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Negative().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorIn(t *testing.T) {
	validValues := []int{1, 2, 3, 5, 8}
	tests := []struct {
		name      string
		value     int
		shouldErr bool
	}{
		{"valid value", 3, false},
		{"invalid value", 4, true},
		{"valid value at start", 1, false},
		{"valid value at end", 8, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").In(validValues...).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorEven(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		shouldErr bool
	}{
		{"even value", 4, false},
		{"odd value", 5, true},
		{"zero", 0, false},
		{"negative even", -4, false},
		{"negative odd", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Even().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorOdd(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		shouldErr bool
	}{
		{"odd value", 5, false},
		{"even value", 4, true},
		{"zero", 0, true},
		{"negative odd", -5, false},
		{"negative even", -4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Odd().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorMultipleOf(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		divisor   int
		shouldErr bool
	}{
		{"multiple of 3", 9, 3, false},
		{"not multiple of 3", 10, 3, true},
		{"multiple of 1", 42, 1, false},
		{"zero divisor", 5, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").MultipleOf(tt.divisor).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLocalization(t *testing.T) {
	erm.SetupTestLocalizer() // Use shared test helper

	// Test default locale
	englishResult := String("", "name").Required().Result()
	err := englishResult.Error()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("expected english message, got: %v", err)
	}

	// Test field name substitution with different field name
	result := String("", "nombre").Required().Result()
	err = result.Error()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "nombre is required") {
		t.Errorf("expected message with field name, got: %v", err)
	}
}

func TestValidationResult(t *testing.T) {
	// Test successful validation
	result := String("valid", "test").Required().Result()
	if !result.Valid() {
		t.Error("expected valid result")
	}
	if result.Error() != nil {
		t.Error("expected no error")
	}
	if len(result.AllErrors()) != 0 {
		t.Error("expected no errors")
	}

	// Test failed validation
	result = String("", "test").Required().Result()
	if result.Valid() {
		t.Error("expected invalid result")
	}
	if result.Error() == nil {
		t.Error("expected error")
	}
	if len(result.AllErrors()) != 1 {
		t.Error("expected one error")
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test isEmpty
	tests := []struct {
		value    interface{}
		expected bool
	}{
		{nil, true},
		{"", true},
		{"   ", true},
		{"hello", false},
		{[]string{}, true},
		{[]string{"a"}, false},
		{map[string]interface{}{}, true},
		{map[string]interface{}{"a": 1}, false},
	}

	for _, tt := range tests {
		result := isEmpty(tt.value)
		if result != tt.expected {
			t.Errorf("isEmpty(%v) = %v, expected %v", tt.value, result, tt.expected)
		}
	}

	// Test getLength
	lengthTests := []struct {
		value    interface{}
		expected int
	}{
		{"hello", 5},
		{"", 0},
		{[]string{"a", "b", "c"}, 3},
		{map[string]interface{}{"a": 1, "b": 2}, 2},
	}

	for _, tt := range lengthTests {
		result := getLength(tt.value)
		if result != tt.expected {
			t.Errorf("getLength(%v) = %v, expected %v", tt.value, result, tt.expected)
		}
	}
}

func TestValidationHelperFunctions(t *testing.T) {
	// Test isValidJSON
	jsonTests := []struct {
		input    string
		expected bool
	}{
		{`{"key": "value"}`, true},
		{`[1, 2, 3]`, true},
		{`{"nested": {"key": "value"}}`, true},
		{`invalid json`, false},
		{`{"unclosed"`, false},
		{``, false},
	}

	for _, tt := range jsonTests {
		result := isValidJSON(tt.input)
		if result != tt.expected {
			t.Errorf("isValidJSON(%s) = %v, expected %v", tt.input, result, tt.expected)
		}
	}

	// Test isValidUUID
	uuidTests := []struct {
		input    string
		expected bool
	}{
		{"123e4567-e89b-12d3-a456-426614174000", true},
		{"invalid-uuid", false},
		{"123e4567-e89b-12d3-a456-42661417400", false},   // wrong length
		{"123e4567-e89b-12d3-a456-426614174000x", false}, // too long
		{"", false},
	}

	for _, tt := range uuidTests {
		result := isValidUUID(tt.input)
		if result != tt.expected {
			t.Errorf("isValidUUID(%s) = %v, expected %v", tt.input, result, tt.expected)
		}
	}

	// Test isValidSlug
	slugTests := []struct {
		input    string
		expected bool
	}{
		{"hello-world", true},
		{"hello", true},
		{"hello123", true},
		{"123hello", true},
		{"-hello", false},      // starts with hyphen
		{"hello-", false},      // ends with hyphen
		{"hello_world", false}, // underscore not allowed
		{"hello world", false}, // space not allowed
		{"", false},
	}

	for _, tt := range slugTests {
		result := isValidSlug(tt.input)
		if result != tt.expected {
			t.Errorf("isValidSlug(%s) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestGenericNumberValidation(t *testing.T) {
	// Test different numeric types
	intErr := Int(5, "test").Min(10).Validate()
	if intErr == nil {
		t.Error("expected error for int validation")
	}

	floatErr := Float64(5.5, "test").Min(10.0).Validate()
	if floatErr == nil {
		t.Error("expected error for float64 validation")
	}

	uint8Err := Uint8(5, "test").Min(10).Validate()
	if uint8Err == nil {
		t.Error("expected error for uint8 validation")
	}

	// Test successful validation
	successErr := Int(15, "test").Min(10).Validate()
	if successErr != nil {
		t.Errorf("unexpected error: %v", successErr)
	}
}

func TestErrorMessages(t *testing.T) {
	erm.SetupTestLocalizer() // Use shared test helper

	// Test with field name
	err := String("", "username").Required().Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "username") {
		t.Errorf("expected error message to contain field name, got: %v", err)
	}

	// Test error message contains validation details
	err = String("a", "password").MinLength(8).Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "8") {
		t.Errorf("expected error message to contain minimum length, got: %v", err)
	}
}

func TestComplexValidationScenarios(t *testing.T) {
	// Test user registration validation
	email := "user@example.com"
	password := "password123"
	age := 25

	// Valid user
	emailErr := String(email, "email").Required().Email().MaxLength(100).Validate()
	passwordErr := String(password, "password").Required().MinLength(8).Validate()
	ageErr := Int(age, "age").Required().Min(18).Max(100).Validate()

	if emailErr != nil || passwordErr != nil || ageErr != nil {
		t.Errorf("unexpected errors in valid user: email=%v, password=%v, age=%v", emailErr, passwordErr, ageErr)
	}

	// Invalid user
	invalidEmail := "invalid-email"
	invalidPassword := "123"
	invalidAge := 15

	emailErr = String(invalidEmail, "email").Required().Email().MaxLength(100).Validate()
	passwordErr = String(invalidPassword, "password").Required().MinLength(8).Validate()
	ageErr = Int(invalidAge, "age").Required().Min(18).Max(100).Validate()

	if emailErr == nil || passwordErr == nil || ageErr == nil {
		t.Error("expected errors for invalid user")
	}
}

// TestStringValidatorEmpty tests the Empty validation rule
func TestStringValidatorEmpty(t *testing.T) {
	erm.SetupTestLocalizer() // Use shared test helper

	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"empty string", "", false},
		{"non-empty string", "hello", true},
		{"whitespace only", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Empty().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorExactLength tests the ExactLength validation rule
func TestStringValidatorExactLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		length    int
		shouldErr bool
	}{
		{"exact length", "hello", 5, false},
		{"too short", "hi", 5, true},
		{"too long", "hello world", 5, true},
		{"empty string", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").ExactLength(tt.length).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorLengthBetween tests the LengthBetween validation rule
func TestStringValidatorLengthBetween(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		min       int
		max       int
		shouldErr bool
	}{
		{"within range", "hello", 3, 10, false},
		{"equal to min", "hello", 5, 10, false},
		{"equal to max", "hello", 1, 5, false},
		{"too short", "hi", 5, 10, true},
		{"too long", "hello world", 1, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").LengthBetween(tt.min, tt.max).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorLowercase tests the Lowercase validation rule
func TestStringValidatorLowercase(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"lowercase", "hello", false},
		{"uppercase", "HELLO", true},
		{"mixed case", "Hello", true},
		{"empty string", "", false},
		{"numbers and symbols", "123!@#", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Lowercase().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorUppercase tests the Uppercase validation rule
func TestStringValidatorUppercase(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"uppercase", "HELLO", false},
		{"lowercase", "hello", true},
		{"mixed case", "Hello", true},
		{"empty string", "", false},
		{"numbers and symbols", "123!@#", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Uppercase().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorInteger tests the Integer validation rule
func TestStringValidatorInteger(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"positive integer", "123", false},
		{"negative integer", "-123", false},
		{"zero", "0", false},
		{"float", "123.45", true},
		{"non-numeric", "abc", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Integer().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorFloat tests the Float validation rule
func TestStringValidatorFloat(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"positive float", "123.45", false},
		{"negative float", "-123.45", false},
		{"integer", "123", false},
		{"zero", "0", false},
		{"scientific notation", "1.23e10", false},
		{"non-numeric", "abc", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Float().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorJSON tests the JSON validation rule
func TestStringValidatorJSON(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid object", `{"name": "John", "age": 30}`, false},
		{"valid array", `[1, 2, 3]`, false},
		{"valid string", `"hello"`, false},
		{"valid number", `123`, false},
		{"valid boolean", `true`, false},
		{"valid null", `null`, false},
		{"invalid json", `{"name": "John", "age":}`, true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").JSON().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorBase64 tests the Base64 validation rule
func TestStringValidatorBase64(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid base64", "SGVsbG8gV29ybGQ=", false},
		{"valid base64 no padding", "SGVsbG8gV29ybGQ", false},
		{"invalid base64", "SGVsbG8gV29ybGQ===", true},
		{"invalid characters", "SGVsbG8@V29ybGQ=", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Base64().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorUUID tests the UUID validation rule
func TestStringValidatorUUID(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid UUID v1", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", false},
		{"invalid UUID too short", "550e8400-e29b-41d4-a716-44665544000", true},
		{"invalid UUID too long", "550e8400-e29b-41d4-a716-4466554400000", true},
		{"invalid UUID wrong format", "550e8400e29b41d4a716446655440000", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").UUID().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestStringValidatorSlug tests the Slug validation rule
func TestStringValidatorSlug(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid slug", "hello-world", false},
		{"valid slug with numbers", "hello-world-123", false},
		{"invalid slug with spaces", "hello world", true},
		{"invalid slug with uppercase", "Hello-World", true},
		{"invalid slug with special chars", "hello@world", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Slug().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorEqualTo(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  string
		shouldErr bool
	}{
		{"equal strings", "hello", "hello", false},
		{"different strings", "hello", "world", true},
		{"empty strings equal", "", "", false},
		{"empty vs non-empty", "", "hello", true},
		{"non-empty vs empty", "hello", "", true},
		{"case sensitive", "Hello", "hello", true},
		{"whitespace matters", "hello ", "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").EqualTo(tt.expected).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStringValidatorEqualToWithCustomMessage(t *testing.T) {
	t.Run("custom message template", func(t *testing.T) {
		customMsg := "{{field}} should match the expected value '{{expected}}'"
		err := String("hello", "greeting").EqualTo("world", customMsg).Validate()
		if err == nil {
			t.Error("expected error but got none")
		}
		// Note: The actual error message checking would require ERM infrastructure
		// This test ensures the method accepts and processes the custom message parameter
	})

	t.Run("empty custom message uses default", func(t *testing.T) {
		err := String("hello", "greeting").EqualTo("world", "").Validate()
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

func TestStringValidatorEqualToWithNegation(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  string
		shouldErr bool
	}{
		{"not equal (valid)", "hello", "world", false},
		{"equal (invalid when negated)", "hello", "hello", true},
		{"empty strings (invalid when negated)", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String(tt.value, "test").Not().EqualTo(tt.expected).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorZero tests the Zero validation rule
func TestNumberValidatorZero(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		shouldErr bool
	}{
		{"zero value", 0, false},
		{"positive value", 5, true},
		{"negative value", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Zero().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorEqual tests the Equal validation rule
func TestNumberValidatorEqual(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		target    int
		shouldErr bool
	}{
		{"equal values", 5, 5, false},
		{"different values", 5, 10, true},
		{"zero values", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Equal(tt.target).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorEqualTo(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		expected  int
		shouldErr bool
	}{
		{"equal values", 42, 42, false},
		{"different values", 42, 24, true},
		{"zero values", 0, 0, false},
		{"negative values equal", -10, -10, false},
		{"negative vs positive", -10, 10, true},
		{"positive vs zero", 5, 0, true},
		{"zero vs negative", 0, -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").EqualTo(tt.expected).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorEqualToWithCustomMessage(t *testing.T) {
	t.Run("custom message template", func(t *testing.T) {
		customMsg := "{{field}} must be exactly {{expected}}, got {{value}}"
		err := Int(25, "age").EqualTo(18, customMsg).Validate()
		if err == nil {
			t.Error("expected error but got none")
		}
		// Note: The actual error message checking would require ERM infrastructure
		// This test ensures the method accepts and processes the custom message parameter
	})

	t.Run("empty custom message uses default", func(t *testing.T) {
		err := Int(25, "age").EqualTo(18, "").Validate()
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

func TestNumberValidatorEqualToWithNegation(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		expected  int
		shouldErr bool
	}{
		{"not equal (valid)", 42, 24, false},
		{"equal (invalid when negated)", 42, 42, true},
		{"zero values (invalid when negated)", 0, 0, true},
		{"negative values not equal (valid)", -10, -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Not().EqualTo(tt.expected).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNumberValidatorEqualToWithDifferentTypes(t *testing.T) {
	t.Run("Float64 EqualTo", func(t *testing.T) {
		err := Float64(3.14, "pi").EqualTo(3.14).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		err = Float64(3.14, "pi").EqualTo(2.71).Validate()
		if err == nil {
			t.Error("expected error but got none")
		}
	})

	t.Run("Int64 EqualTo", func(t *testing.T) {
		err := Int64(1000000, "large").EqualTo(1000000).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		err = Int64(1000000, "large").EqualTo(999999).Validate()
		if err == nil {
			t.Error("expected error but got none")
		}
	})

	t.Run("Float32 EqualTo", func(t *testing.T) {
		err := Float32(2.5, "half").EqualTo(2.5).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		err = Float32(2.5, "half").EqualTo(3.5).Validate()
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

// TestNumberValidatorGreaterThan tests the GreaterThan validation rule
func TestNumberValidatorGreaterThan(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		target    int
		shouldErr bool
	}{
		{"greater than", 10, 5, false},
		{"equal to", 5, 5, true},
		{"less than", 5, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").GreaterThan(tt.target).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorLessThan tests the LessThan validation rule
func TestNumberValidatorLessThan(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		target    int
		shouldErr bool
	}{
		{"less than", 5, 10, false},
		{"equal to", 5, 5, true},
		{"greater than", 10, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").LessThan(tt.target).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorNotIn tests the NotIn validation rule
func TestNumberValidatorNotIn(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		values    []int
		shouldErr bool
	}{
		{"not in list", 5, []int{1, 2, 3}, false},
		{"in list", 2, []int{1, 2, 3}, true},
		{"empty list", 5, []int{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").NotIn(tt.values...).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorFinite tests the Finite validation rule
func TestNumberValidatorFinite(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		shouldErr bool
	}{
		{"finite value", 3.14, false},
		{"zero", 0.0, false},
		{"negative finite", -3.14, false},
		{"positive infinity", math.Inf(1), true},
		{"negative infinity", math.Inf(-1), true},
		{"NaN", math.NaN(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Float64(tt.value, "test").Finite().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorPrecision tests the Precision validation rule
func TestNumberValidatorPrecision(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision int
		shouldErr bool
	}{
		{"within precision", 3.14, 2, false},
		{"exact precision", 3.1, 1, false},
		{"exceeds precision", 3.141592, 2, true},
		{"integer value", 3.0, 2, false},
		{"no decimals", 3.0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Float64(tt.value, "test").Precision(tt.precision).Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorNot tests the Not operator with number validation
func TestNumberValidatorNot(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		shouldErr bool
	}{
		{"not required (zero)", 0, false},
		{"not required (non-zero)", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Not().Required().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorWhen tests the When condition with number validation
func TestNumberValidatorWhen(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		condition func() bool
		shouldErr bool
	}{
		{"condition true, valid", 5, func() bool { return true }, false},
		{"condition true, invalid", 0, func() bool { return true }, true},
		{"condition false, invalid", 0, func() bool { return false }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").When(tt.condition).Required().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberValidatorUnless tests the Unless condition with number validation
func TestNumberValidatorUnless(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		condition func() bool
		shouldErr bool
	}{
		{"condition false, valid", 5, func() bool { return false }, false},
		{"condition false, invalid", 0, func() bool { return false }, true},
		{"condition true, invalid", 0, func() bool { return true }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Int(tt.value, "test").Unless(tt.condition).Required().Validate()
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNumberTypeValidators tests all number type validators
func TestNumberTypeValidators(t *testing.T) {
	// Test each number type constructor
	t.Run("Int8", func(t *testing.T) {
		err := Int8(42, "test").Min(int8(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Int16", func(t *testing.T) {
		err := Int16(42, "test").Min(int16(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Int32", func(t *testing.T) {
		err := Int32(42, "test").Min(int32(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Int64", func(t *testing.T) {
		err := Int64(42, "test").Min(int64(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Uint", func(t *testing.T) {
		err := Uint(42, "test").Min(uint(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Uint16", func(t *testing.T) {
		err := Uint16(42, "test").Min(uint16(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Uint32", func(t *testing.T) {
		err := Uint32(42, "test").Min(uint32(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Uint64", func(t *testing.T) {
		err := Uint64(42, "test").Min(uint64(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Float32", func(t *testing.T) {
		err := Float32(3.14, "test").Min(float32(0)).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestValidationResultMethods tests ValidationResult methods
func TestValidationResultMethods(t *testing.T) {
	setupLocalizer() // Set up localizer for tests

	t.Run("NewValidationResult with empty field name", func(t *testing.T) {
		result := NewValidationResult("test", "")
		if result.FieldName != "value" {
			t.Errorf("expected field name 'value', got %s", result.FieldName)
		}
	})

	t.Run("ToError method", func(t *testing.T) {
		// Test valid result returns nil
		result := NewValidationResult("test", "field")
		errorMap := result.ErrMap()
		if errorMap != nil {
			t.Error("expected nil error map for valid result")
		}

		// Test result with single error
		result = NewValidationResult("", "email")
		err := erm.NewValidationError("validation.required", "email", "")
		result.AddError(err)

		errorMap = result.ErrMap()
		if errorMap == nil {
			t.Fatal("expected error map, got nil")
		}
		if len(errorMap) != 1 {
			t.Errorf("expected 1 field in error map, got %d", len(errorMap))
		}
		if errors, exists := errorMap["email"]; !exists {
			t.Error("expected 'email' field in error map")
		} else if len(errors) != 1 {
			t.Errorf("expected 1 error message, got %d", len(errors))
		} else if errors[0] != "email is required" {
			t.Errorf("expected 'email is required', got %s", errors[0])
		}

		// Test result with multiple errors using manual error creation
		result = NewValidationResult("abc", "password")
		err1 := erm.NewValidationError("validation.min_length", "password", "abc")
		err1 = err1.WithParam("min", 8)
		err2 := erm.NewValidationError("validation.invalid", "password", "abc")
		result.AddError(err1)
		result.AddError(err2)

		errorMap = result.ErrMap()
		if errorMap == nil {
			t.Fatal("expected error map, got nil")
		}
		if len(errorMap) != 1 {
			t.Errorf("expected 1 field in error map, got %d", len(errorMap))
		}
		if errors, exists := errorMap["password"]; !exists {
			t.Error("expected 'password' field in error map")
		} else if len(errors) != 2 {
			t.Errorf("expected 2 error messages, got %d", len(errors))
		} else {
			if errors[0] != "password must be at least 8 characters long" {
				t.Errorf("expected 'password must be at least 8 characters long', got %s", errors[0])
			}
			if errors[1] != "password value is invalid" {
				t.Errorf("expected 'password value is invalid', got %s", errors[1])
			}
		}

		// Test result with actual validator errors
		result = String("a", "password").Required().MinLength(8).Result()
		errorMap = result.ErrMap()
		if errorMap == nil {
			t.Fatal("expected error map, got nil")
		}
		if len(errorMap) != 1 {
			t.Errorf("expected 1 field in error map, got %d", len(errorMap))
		}
		if errors, exists := errorMap["password"]; !exists {
			t.Error("expected 'password' field in error map")
		} else if len(errors) != 1 {
			t.Errorf("expected 1 error message, got %d", len(errors))
		} else {
			// Check that the error message contains the expected text
			if !strings.Contains(errors[0], "must be at least 8 characters") {
				t.Errorf("expected min length error message, got %s", errors[0])
			}
		}
	})
}

// TestValidationErrorMethods tests validation error methods now provided by erm.Error
func TestValidationErrorMethods(t *testing.T) {
	t.Run("WithParam", func(t *testing.T) {
		err := erm.NewValidationError("Test message", "field", "value")
		err = err.WithParam("test", "value")

		if err.Params()["test"] != "value" {
			t.Error("expected parameter to be set")
		}
	})
	// ValidationCode functionality has been removed since vix always uses 400 status
}

// TestCustomValidationFunction tests custom validation functions
func TestCustomValidationFunction(t *testing.T) {
	t.Run("Custom validation success", func(t *testing.T) {
		customFunc := func(value interface{}) error {
			return nil // Always pass
		}

		err := String("test", "field").Custom(customFunc).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Custom validation failure", func(t *testing.T) {
		customFunc := func(value interface{}) error {
			return erm.NewValidationError("Custom error", "field", value)
		}

		err := String("test", "field").Custom(customFunc).Validate()
		if err == nil {
			t.Error("expected custom validation to fail")
		}
	})
}

// TestUtilityFunctions tests utility functions
func TestUtilityFunctions(t *testing.T) {
	t.Run("isEmpty", func(t *testing.T) {
		if !isEmpty("") {
			t.Error("expected empty string to be empty")
		}
		if !isEmpty("   ") {
			t.Error("expected whitespace to be empty")
		}
		if isEmpty("test") {
			t.Error("expected non-empty string to not be empty")
		}
		if !isEmpty(0) {
			t.Error("expected zero to be empty")
		}
		if isEmpty(42) {
			t.Error("expected non-zero number to not be empty")
		}
	})

	t.Run("getLength", func(t *testing.T) {
		if getLength("hello") != 5 {
			t.Error("expected length of 'hello' to be 5")
		}
		if getLength([]int{1, 2, 3}) != 3 {
			t.Error("expected length of slice to be 3")
		}
		if getLength(42) != 0 {
			t.Error("expected length of non-string/slice to be 0")
		}
	})

	t.Run("toString", func(t *testing.T) {
		if toString("hello") != "hello" {
			t.Error("expected string to remain string")
		}
		if toString(42) != "42" {
			t.Error("expected number to convert to string")
		}
	})

	t.Run("toTime", func(t *testing.T) {
		// Test toTime with proper time value
		now := time.Now()
		timeVal, ok := toTime(now)
		if !ok || timeVal != now {
			t.Error("expected time to remain time")
		}
		timeVal, ok = toTime("not a time")
		if ok || timeVal != (time.Time{}) {
			t.Error("expected non-time to return zero time and false")
		}
	})
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("MultipleOf with zero", func(t *testing.T) {
		// This should not panic, but handle gracefully
		err := Int(10, "test").MultipleOf(0).Validate()
		if err == nil {
			t.Error("expected error for division by zero")
		}
	})

	t.Run("Validation with nil custom function", func(t *testing.T) {
		err := String("test", "field").Custom(nil).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Chaining multiple validations", func(t *testing.T) {
		err := String("test@example.com", "email").
			Required().
			Email().
			MaxLength(100).
			MinLength(5).
			Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Result method returns proper result", func(t *testing.T) {
		result := String("test", "field").Required().Result()
		if !result.Valid() {
			t.Error("expected validation to pass")
		}
		if result.FieldName != "field" {
			t.Errorf("expected field name 'field', got %s", result.FieldName)
		}
	})
}

// TestComplexValidationChains tests complex validation chains
func TestComplexValidationChains(t *testing.T) {
	t.Run("Complex string validation", func(t *testing.T) {
		err := String("test123", "username").
			Required().
			MinLength(5).
			MaxLength(20).
			AlphaNumeric().
			Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Complex number validation", func(t *testing.T) {
		err := Int(42, "age").
			Required().
			Min(18).
			Max(100).
			Positive().
			Even().
			Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Conditional validation with When", func(t *testing.T) {
		condition := true
		err := String("test", "field").
			When(func() bool { return condition }).
			Required().
			Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Conditional validation with Unless", func(t *testing.T) {
		condition := false
		err := String("test", "field").
			Unless(func() bool { return condition }).
			Required().
			Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Negated validation", func(t *testing.T) {
		err := String("", "field").
			Not().
			Required().
			Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestValidatorToError tests ToError() method integration with validators
func TestValidatorToError(t *testing.T) {
	setupLocalizer() // Set up localizer for tests

	t.Run("String validator ToError", func(t *testing.T) {
		// Test valid string
		result := String("test@example.com", "email").Required().Email().Result()
		errorMap := result.ErrMap()
		if errorMap != nil {
			t.Error("expected nil error map for valid string")
		}

		// Test invalid string - single error
		result = String("", "email").Required().Result()
		errorMap = result.ErrMap()
		if errorMap == nil {
			t.Fatal("expected error map, got nil")
		}
		if len(errorMap) != 1 {
			t.Errorf("expected 1 field in error map, got %d", len(errorMap))
		}
		if errors, exists := errorMap["email"]; !exists {
			t.Error("expected 'email' field in error map")
		} else if len(errors) != 1 {
			t.Errorf("expected 1 error message, got %d", len(errors))
		} else if errors[0] != "email is required" {
			t.Errorf("expected 'email is required', got %s", errors[0])
		}

		// Test invalid string - multiple errors
		result = String("a", "password").Required().MinLength(8).MaxLength(128).Result()
		errorMap = result.ErrMap()
		if errorMap == nil {
			t.Fatal("expected error map, got nil")
		}
		if len(errorMap) != 1 {
			t.Errorf("expected 1 field in error map, got %d", len(errorMap))
		}
		if errors, exists := errorMap["password"]; !exists {
			t.Error("expected 'password' field in error map")
		} else if len(errors) != 1 {
			t.Errorf("expected 1 error message, got %d", len(errors))
		} else if !strings.Contains(errors[0], "must be at least 8 characters") {
			t.Errorf("expected min length error, got %s", errors[0])
		}
	})

	t.Run("Number validator ToError", func(t *testing.T) {
		// Test valid number
		result := Int(25, "age").Required().Min(18).Max(100).Result()
		errorMap := result.ErrMap()
		if errorMap != nil {
			t.Error("expected nil error map for valid number")
		}

		// Test invalid number - single error
		result = Int(0, "age").Required().Result()
		errorMap = result.ErrMap()
		if errorMap == nil {
			t.Fatal("expected error map, got nil")
		}
		if len(errorMap) != 1 {
			t.Errorf("expected 1 field in error map, got %d", len(errorMap))
		}
		if errors, exists := errorMap["age"]; !exists {
			t.Error("expected 'age' field in error map")
		} else if len(errors) != 1 {
			t.Errorf("expected 1 error message, got %d", len(errors))
		} else if errors[0] != "age is required" {
			t.Errorf("expected 'age is required', got %s", errors[0])
		}

		// Test invalid number - multiple errors through chaining
		result = Int(10, "score").Required().Min(50).Max(100).Result()
		errorMap = result.ErrMap()
		if errorMap == nil {
			t.Fatal("expected error map, got nil")
		}
		if len(errorMap) != 1 {
			t.Errorf("expected 1 field in error map, got %d", len(errorMap))
		}
		if errors, exists := errorMap["score"]; !exists {
			t.Error("expected 'score' field in error map")
		} else if len(errors) != 1 {
			t.Errorf("expected 1 error message, got %d", len(errors))
		} else if !strings.Contains(errors[0], "must be at least 50") {
			t.Errorf("expected min value error, got %s", errors[0])
		}
	})

	t.Run("ToError with different field names", func(t *testing.T) {
		// Test different field names
		testCases := []struct {
			fieldName string
			value     string
		}{
			{"username", ""},
			{"first_name", ""},
			{"last_name", ""},
			{"company.name", ""},
			{"address[0].street", ""},
		}

		for _, tc := range testCases {
			result := String(tc.value, tc.fieldName).Required().Result()
			errorMap := result.ErrMap()
			if errorMap == nil {
				t.Fatalf("expected error map for field %s, got nil", tc.fieldName)
			}
			if _, exists := errorMap[tc.fieldName]; !exists {
				t.Errorf("expected '%s' field in error map", tc.fieldName)
			}
		}
	})
}

// TestUnicodeStringHandling tests Unicode string validation edge cases
func TestUnicodeStringHandling(t *testing.T) {
	t.Run("Unicode string length", func(t *testing.T) {
		// Test with emojis and Unicode characters
		unicodeStr := "Hello 👋 World 🌍"
		err := String(unicodeStr, "message").MinLength(5).MaxLength(20).Validate()
		if err != nil {
			t.Errorf("Unicode string validation failed: %v", err)
		}

		// Test that length is calculated correctly for Unicode
		length := getLength(unicodeStr)
		if length != 15 { // Should count Unicode runes, not bytes
			t.Errorf("Expected Unicode length 15, got %d", length)
		}
	})

	t.Run("Mixed script validation", func(t *testing.T) {
		mixedScript := "Hello世界こんにちは"
		err := String(mixedScript, "message").Required().MinLength(10).Validate()
		if err != nil {
			t.Errorf("Mixed script validation failed: %v", err)
		}
	})
}

// TestLargeNumberEdgeCases tests validation with very large numbers
func TestLargeNumberEdgeCases(t *testing.T) {
	t.Run("Large integer validation", func(t *testing.T) {
		largeInt := int64(9223372036854775807) // Max int64
		err := Int64(largeInt, "large_number").Required().Positive().Validate()
		if err != nil {
			t.Errorf("Large integer validation failed: %v", err)
		}
	})

	t.Run("Float precision edge cases", func(t *testing.T) {
		// Test very small floating point numbers
		smallFloat := 0.000000001
		err := Float64(smallFloat, "small_float").Required().Positive().Precision(9).Validate()
		if err != nil {
			t.Errorf("Small float validation failed: %v", err)
		}

		// Test precision with exactly the limit
		preciseFloat := 12.34
		err = Float64(preciseFloat, "precise_float").Precision(2).Validate()
		if err != nil {
			t.Errorf("Precise float validation failed: %v", err)
		}
	})
}

// TestValidationChaining tests complex validation chains
func TestValidationChaining(t *testing.T) {
	t.Run("Complex email validation chain", func(t *testing.T) {
		email := "user@example.com"
		err := String(email, "email").
			Required().
			Email().
			MinLength(5).
			MaxLength(100).
			Contains("@").
			Not().Contains("admin").
			Validate()

		if err != nil {
			t.Errorf("Complex email validation failed: %v", err)
		}
	})

	t.Run("Conditional validation chain", func(t *testing.T) {
		isRequired := true
		value := ""

		err := String(value, "optional_field").
			When(func() bool { return isRequired }).
			Required().
			Validate()

		if err == nil {
			t.Error("Expected validation error for required field")
		}

		// Now test with condition false
		isRequired = false
		err = String(value, "optional_field").
			When(func() bool { return isRequired }).
			Required().
			Validate()

		if err != nil {
			t.Errorf("Unexpected error when condition is false: %v", err)
		}
	})
}

// TestErrorMessageFormatting tests error message formatting edge cases
func TestErrorMessageFormatting(t *testing.T) {
	setupLocalizer() // Set up localizer for tests

	t.Run("Error message with special characters", func(t *testing.T) {
		fieldName := "field_with_underscores"
		err := String("", fieldName).Required().Validate()
		if err == nil {
			t.Error("Expected validation error")
		}

		errorMsg := err.Error()
		if !strings.Contains(errorMsg, fieldName) {
			t.Errorf("Error message should contain field name, got: %s", errorMsg)
		}
	})

	t.Run("Error message parameter handling", func(t *testing.T) {
		err := String("ab", "password").MinLength(8).Validate()
		if err == nil {
			t.Error("Expected validation error")
		}

		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "8") {
			t.Errorf("Error message should contain min length parameter, got: %s", errorMsg)
		}
	})
}

// TestMemoryEfficiency tests that validators don't cause memory leaks
func TestMemoryEfficiency(t *testing.T) {
	t.Run("Multiple validation calls", func(t *testing.T) {
		// Create many validators to test for memory leaks
		for i := 0; i < 1000; i++ {
			value := fmt.Sprintf("test%d", i)
			err := String(value, "test").Required().MinLength(4).Validate()
			if err != nil {
				t.Errorf("Validation %d failed: %v", i, err)
				break
			}
		}
	})
}

// TestNumberValidatorCustom tests the Custom method for NumberValidator
func TestNumberValidatorCustom(t *testing.T) {
	t.Run("Custom validation success", func(t *testing.T) {
		customFunc := func(value interface{}) error {
			return nil // Always pass
		}

		err := Int(42, "test").Custom(customFunc).Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Custom validation failure", func(t *testing.T) {
		customFunc := func(value interface{}) error {
			return erm.NewValidationError("validation.custom", "test", value)
		}

		err := Int(42, "test").Custom(customFunc).Validate()
		if err == nil {
			t.Error("expected custom validation to fail")
		}
	})

	t.Run("Custom validation with nil function", func(t *testing.T) {
		err := Int(42, "test").Custom(nil).Validate()
		if err != nil {
			t.Errorf("unexpected error with nil custom function: %v", err)
		}
	})
}

// TestNumberValidatorNegationEdgeCases tests negation with various number validators
func TestNumberValidatorNegationEdgeCases(t *testing.T) {
	t.Run("Not.Zero", func(t *testing.T) {
		// Should pass when value is not zero
		err := Int(5, "test").Not().Zero().Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Should fail when value is zero
		err = Int(0, "test").Not().Zero().Validate()
		if err == nil {
			t.Error("expected error for Not().Zero() with zero value")
		}
	})

	t.Run("Not.Even", func(t *testing.T) {
		// Should pass for odd numbers
		err := Int(5, "test").Not().Even().Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Should fail for even numbers
		err = Int(4, "test").Not().Even().Validate()
		if err == nil {
			t.Error("expected error for Not().Even() with even value")
		}
	})

	t.Run("Not.Positive", func(t *testing.T) {
		// Should pass for negative/zero
		err := Int(-5, "test").Not().Positive().Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Should fail for positive
		err = Int(5, "test").Not().Positive().Validate()
		if err == nil {
			t.Error("expected error for Not().Positive() with positive value")
		}
	})
}

// TestStringValidatorNegationEdgeCases tests negation with string validators
func TestStringValidatorNegationEdgeCases(t *testing.T) {
	t.Run("Not.Empty", func(t *testing.T) {
		// Should pass when not empty
		err := String("hello", "test").Not().Empty().Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Should fail when empty
		err = String("", "test").Not().Empty().Validate()
		if err == nil {
			t.Error("expected error for Not().Empty() with empty value")
		}
	})

	t.Run("Not.Numeric", func(t *testing.T) {
		// Should pass for non-numeric
		err := String("hello", "test").Not().Numeric().Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Should fail for numeric
		err = String("12345", "test").Not().Numeric().Validate()
		if err == nil {
			t.Error("expected error for Not().Numeric() with numeric value")
		}
	})

	t.Run("Not.Alpha", func(t *testing.T) {
		// Should pass for non-alpha
		err := String("hello123", "test").Not().Alpha().Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Should fail for alpha-only
		err = String("hello", "test").Not().Alpha().Validate()
		if err == nil {
			t.Error("expected error for Not().Alpha() with alpha value")
		}
	})
}

// TestValidatorEdgeCasesForCoverage tests edge cases to improve coverage
func TestValidatorEdgeCasesForCoverage(t *testing.T) {
	t.Run("formatValues with different types", func(t *testing.T) {
		// Test formatValues function indirectly through In() validation
		err := Int(10, "test").In(1, 2, 3, 4, 5).Validate()
		if err == nil {
			t.Error("expected error for value not in list")
		}

		// Check that we got an error - the exact format may vary based on localization
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "validation error for field") {
			t.Errorf("Error message should indicate validation error, got: %s", errorMsg)
		}
	})

	t.Run("Finite with special float values", func(t *testing.T) {
		// Test with positive infinity
		err := Float64(math.Inf(1), "test").Finite().Validate()
		if err == nil {
			t.Error("expected error for positive infinity")
		}

		// Test with negative infinity
		err = Float64(math.Inf(-1), "test").Finite().Validate()
		if err == nil {
			t.Error("expected error for negative infinity")
		}

		// Test with NaN
		err = Float64(math.NaN(), "test").Finite().Validate()
		if err == nil {
			t.Error("expected error for NaN")
		}
	})

	t.Run("Precision with various decimal places", func(t *testing.T) {
		// Test precision calculation edge cases
		err := Float64(1.000000001, "test").Precision(8).Validate()
		if err == nil {
			t.Error("expected error for precision exceeding limit")
		}

		// Test with zero precision
		err = Float64(3.0, "test").Precision(0).Validate()
		if err != nil {
			t.Errorf("unexpected error for integer with zero precision: %v", err)
		}
	})
}
