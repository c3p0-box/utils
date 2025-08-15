// Package vix provides a type-safe, expressive, and extensible validation
// library for Go. It follows clean architecture principles and integrates
// seamlessly with the ERM centralized error management package for unified error handling
// and error collection with standard i18n support.
//
// Unlike tag-based validation libraries, this package uses function chaining
// to create readable and maintainable validation rules. The package supports
// type-safe validation with Go generics, internationalization using the standard
// go-i18n package through ERM integration, conditional validation, and comprehensive
// error reporting suitable for modern APIs.
//
// All validation errors are now unified under the erm.Error interface, providing
// consistent error handling across application and validation layers with
// on-demand localization support.
//
// # Quick Start
//
// Basic validation example:
//
//	import "github.com/c3p0-box/utils/vix"
//
//	// Single field validation
//	err := vix.String("john@example.com", "email").
//		Required().
//		Email().
//		MaxLength(100).
//		Validate()
//
//	if err != nil {
//		log.Printf("Email validation failed: %v", err)
//	}
//
// # Multi-Field Validation
//
// The package excels at validating multiple fields with structured error output:
//
//	val := vix.Is(
//		vix.String("", "email").Required().Email(),
//		vix.Int(16, "age").Required().Min(18),
//	)
//
//	if !val.Valid() {
//		errorMap := val.ErrMap()
//		jsonBytes, _ := val.ToJSON()
//		// Returns structured error map suitable for API responses
//	}
//
// # Conditional Validation
//
// Validators support conditional logic:
//
//	err := vix.String(phone, "phone").
//		When(func() bool { return email == "" }).
//		Required().
//		Validate()
//
// # Integration with Clean Architecture
//
// The package integrates seamlessly with onion/clean architecture patterns
// using the unified erm error management system:
//
//	func (s *UserService) CreateUser(user User) error {
//		if err := vix.String(user.Email, "email").Required().Email().Validate(); err != nil {
//			return erm.BadRequest("Invalid email", err)
//		}
//		// Business logic continues...
//		return nil
//	}
//
// # Internationalization
//
// Error messages are localized using the standard go-i18n package through ERM integration.
// Set up a localizer in your application initialization:
//
//	// Get localizers for different languages
//	englishLocalizer := erm.GetLocalizer(language.English)
//	spanishLocalizer := erm.GetLocalizer(language.Spanish)
//
// All validation errors will then be automatically localized when converted to strings.
package vix

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/c3p0-box/utils/erm"
)

// =============================================================================
// Interfaces
// =============================================================================

// Validator defines the interface that all validation rules must implement.
// It provides a single Validate method that returns an error if validation fails.
type Validator interface {
	Validate() error
	Result() *ValidationResult
}

// ValidatorChain defines the interface for validator chaining operations.
// This interface eliminates code duplication by providing common methods
// that can be shared across different validator types.
type ValidatorChain interface {
	// Not negates the next validation rule
	Not() ValidatorChain

	// When adds a condition that must be true for validation to run
	When(condition func() bool) ValidatorChain

	// Unless adds a condition that must be false for validation to run
	Unless(condition func() bool) ValidatorChain

	// Custom validates using a custom validation function
	Custom(fn func(value interface{}, fieldName string) error) ValidatorChain
}

// =============================================================================
// Core Types
// =============================================================================

// ValidationResult holds the result of a validation operation.
// It contains the original value, field name, and collects validation errors
// as a slice of erm.Error instances for unified error handling.
// Internationalization is handled automatically through the erm package.
// All validation errors use HTTP 400 Bad Request status code.
type ValidationResult struct {
	Value     interface{}
	FieldName string
	errors    []erm.Error // Slice of validation errors
	IsValid   bool
}

// NewValidationResult creates a new ValidationResult with the given value and field name.
func NewValidationResult(value interface{}, fieldName string) *ValidationResult {
	if fieldName == "" {
		fieldName = "value"
	}

	return &ValidationResult{
		Value:     value,
		FieldName: fieldName,
		errors:    []erm.Error{},
		IsValid:   true,
	}
}

// AddError adds a validation error to the result.
func (vr *ValidationResult) AddError(err error) *ValidationResult {
	if err != nil {
		if ermErr, ok := err.(erm.Error); ok {
			vr.errors = append(vr.errors, ermErr)
		} else {
			// Convert regular error to erm.Error
			ermErr := erm.New(http.StatusBadRequest, err.Error(), err)
			vr.errors = append(vr.errors, ermErr)
		}
		vr.IsValid = false
	}
	return vr
}

// Valid returns true if no validation errors occurred.
func (vr *ValidationResult) Valid() bool {
	return vr.IsValid && len(vr.errors) == 0
}

// Error returns the validation error container, or nil if validation passed.
func (vr *ValidationResult) Error() error {
	if vr.Valid() {
		return nil
	}

	// Create a container error and add all errors as children
	container := erm.New(http.StatusBadRequest, "", nil)
	container.AddErrors(vr.errors)
	return container
}

// AllErrors returns all validation errors.
func (vr *ValidationResult) AllErrors() []erm.Error {
	if vr.errors == nil {
		return []erm.Error{}
	}

	return vr.errors
}

// ErrMap returns a map of field names to error messages.
// Returns nil if validation passed, otherwise returns the structured error map.
func (vr *ValidationResult) ErrMap() map[string][]string {
	if vr.Valid() {
		return nil
	}

	// Create a container error with all errors and use its ErrMap method
	container := erm.New(http.StatusBadRequest, "", nil)
	container.AddErrors(vr.errors)
	return container.ErrMap()
}

// =============================================================================
// Base Functionality
// =============================================================================

// BaseValidator provides common functionality for all validators.
type BaseValidator struct {
	value      interface{}
	fieldName  string
	result     *ValidationResult
	negated    bool
	conditions []func() bool
}

// NewBaseValidator creates a new BaseValidator.
func NewBaseValidator(value interface{}, fieldName string) *BaseValidator {
	return &BaseValidator{
		value:      value,
		fieldName:  fieldName,
		result:     NewValidationResult(value, fieldName),
		negated:    false,
		conditions: []func() bool{},
	}
}

// Not negates the next validation rule.
func (bv *BaseValidator) Not() *BaseValidator {
	bv.negated = true
	return bv
}

// When adds a condition that must be true for validation to run.
func (bv *BaseValidator) When(condition func() bool) *BaseValidator {
	bv.conditions = append(bv.conditions, condition)
	return bv
}

// Unless adds a condition that must be false for validation to run.
func (bv *BaseValidator) Unless(condition func() bool) *BaseValidator {
	bv.conditions = append(bv.conditions, func() bool { return !condition() })
	return bv
}

// shouldValidate checks if validation should run based on conditions.
func (bv *BaseValidator) shouldValidate() bool {
	for _, condition := range bv.conditions {
		if !condition() {
			return false
		}
	}
	return true
}

// addValidationError adds a validation error, handling negation.
// Uses message keys for internationalization instead of templates.
func (bv *BaseValidator) addValidationError(code, messageKey string, params map[string]interface{}) {
	if !bv.shouldValidate() {
		return
	}

	err := erm.NewValidationError(messageKey, bv.fieldName, bv.value)

	for key, value := range params {
		err = err.WithParam(key, value)
	}

	if bv.negated {
		// For negated validations, we need to flip the logic
		negatedMessageKey := "validation.not_" + strings.TrimPrefix(messageKey, "validation.")
		err = erm.NewValidationError(negatedMessageKey, bv.fieldName, bv.value)
		for key, value := range params {
			err = err.WithParam(key, value)
		}
	}

	bv.result.AddError(err)
	bv.negated = false // Reset negation after use
}

// Validate returns the validation result.
func (bv *BaseValidator) Validate() error {
	if !bv.result.Valid() {
		return bv.result.Error()
	}
	return nil
}

// Result returns the full validation result.
func (bv *BaseValidator) Result() *ValidationResult {
	return bv.result
}

// Custom validates using a custom validation function.
// The function receives both the value being validated and the field name,
// allowing for more contextual error messages.
//
// Example:
//
//	err := vix.String("test", "username").
//		Custom(func(value interface{}, fieldName string) error {
//			str := value.(string)
//			if strings.Contains(str, "admin") {
//				return erm.NewValidationError("{{field}} cannot contain 'admin'", fieldName, value)
//			}
//			return nil
//		}).
//		Validate()
func (bv *BaseValidator) Custom(fn func(value interface{}, fieldName string) error) *BaseValidator {
	if !bv.shouldValidate() {
		return bv
	}

	// Handle nil function gracefully
	if fn == nil {
		return bv
	}

	err := fn(bv.value, bv.fieldName)
	if err != nil {
		if bv.negated {
			// If negated and custom validation failed, it's actually valid
			bv.negated = false
			return bv
		}
		bv.result.AddError(err)
	} else if bv.negated {
		// If negated and custom validation passed, it's invalid
		bv.result.AddError(erm.NewValidationError("validation.custom_negated", bv.fieldName, bv.value))
	}

	bv.negated = false
	return bv
}

// =============================================================================
// Constants and Patterns
// =============================================================================

// Common validation message keys for i18n
const (
	MsgRequired      = "validation.required"
	MsgEmpty         = "validation.empty"
	MsgMinLength     = "validation.min_length"
	MsgMaxLength     = "validation.max_length"
	MsgExactLength   = "validation.exact_length"
	MsgLengthBetween = "validation.length_between"
	MsgEmail         = "validation.email"
	MsgURL           = "validation.url"
	MsgNumeric       = "validation.numeric"
	MsgAlpha         = "validation.alpha"
	MsgAlphaNumeric  = "validation.alpha_numeric"
	MsgRegex         = "validation.regex"
	MsgIn            = "validation.in"
	MsgNotIn         = "validation.not_in"
	MsgContains      = "validation.contains"
	MsgStartsWith    = "validation.starts_with"
	MsgEndsWith      = "validation.ends_with"
	MsgLowercase     = "validation.lowercase"
	MsgUppercase     = "validation.uppercase"
	MsgInteger       = "validation.integer"
	MsgFloat         = "validation.float"
	MsgJSON          = "validation.json"
	MsgBase64        = "validation.base64"
	MsgUUID          = "validation.uuid"
	MsgSlug          = "validation.slug"
	MsgMin           = "validation.min_value"
	MsgMax           = "validation.max_value"
	MsgBetween       = "validation.between"
	MsgZero          = "validation.zero"
	MsgEqual         = "validation.equal"
	MsgEqualTo       = "validation.equal_to"
	MsgGreaterThan   = "validation.greater_than"
	MsgLessThan      = "validation.less_than"
	MsgPositive      = "validation.positive"
	MsgNegative      = "validation.negative"
	MsgEven          = "validation.even"
	MsgOdd           = "validation.odd"
	MsgMultipleOf    = "validation.multiple_of"
	MsgFinite        = "validation.finite"
	MsgPrecision     = "validation.precision"
	MsgAfter         = "validation.after"
	MsgBefore        = "validation.before"
	MsgDateFormat    = "validation.date_format"
)

// Common validation patterns
var (
	EmailRegex        = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	URLRegex          = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	NumericRegex      = regexp.MustCompile(`^[0-9]+$`)
	AlphaRegex        = regexp.MustCompile(`^[a-zA-Z]+$`)
	AlphaNumericRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

// =============================================================================
// Utility Functions
// =============================================================================

// isEmpty checks if a value is considered empty based on its type
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []byte:
		return len(v) == 0
	case []string:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() == 0
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() == 0
	case float32, float64:
		return reflect.ValueOf(v).Float() == 0
	case bool:
		return !v
	default:
		// Use reflection for other types
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
			return val.Len() == 0
		case reflect.Ptr, reflect.Interface:
			return val.IsNil()
		default:
			return false
		}
	}
}

// getLength returns the length of a value based on its type
func getLength(value interface{}) int {
	switch v := value.(type) {
	case string:
		return len([]rune(v))
	case []byte:
		return len(v)
	case []string:
		return len(v)
	case map[string]interface{}:
		return len(v)
	default:
		// Use reflection for other slice types
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan, reflect.String:
			return val.Len()
		default:
			return 0
		}
	}
}

// toString converts a value to string representation
func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

// toTime converts a value to time.Time if possible
func toTime(value interface{}) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case *time.Time:
		if v != nil {
			return *v, true
		}
		return time.Time{}, false
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

// =============================================================================
// Package-Level Functions
// =============================================================================

// Is creates a new ValidationOrchestrator and adds the given validators to it.
// This is a convenience function that allows you to write vix.Is(...) instead of vix.V().Is(...).
//
// Example:
//
//	val := vix.Is(
//		vix.String("test@example.com", "email").Required().Email(),
//		vix.Int(25, "age").Required().Min(18),
//	)
//
//	if !val.Valid() {
//		errorMap := val.ErrMap()
//		// handle errors
//	}
func Is(validators ...Validator) *ValidationOrchestrator {
	return V().Is(validators...)
}

// In creates a new ValidationOrchestrator with a single namespaced validation orchestrator.
// This is a convenience function that allows you to write vix.In(...) instead of vix.V().In(...).
//
// Example:
//
//	val := vix.In("address", vix.Is(
//		vix.String(address.Street, "street").Required(),
//		vix.String(address.City, "city").Required(),
//	))
func In(namespace string, orchestrator *ValidationOrchestrator) *ValidationOrchestrator {
	return V().In(namespace, orchestrator)
}

// InRow creates a new ValidationOrchestrator with a single indexed namespaced validation orchestrator.
// This is a convenience function that allows you to write vix.InRow(...) instead of vix.V().InRow(...).
//
// Example:
//
//	val := vix.InRow("addresses", 0, vix.Is(
//		vix.String(address.Street, "street").Required(),
//		vix.String(address.City, "city").Required(),
//	))
func InRow(namespace string, index int, orchestrator *ValidationOrchestrator) *ValidationOrchestrator {
	return V().InRow(namespace, index, orchestrator)
}
