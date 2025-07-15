// Package vix provides a type-safe, expressive, and extensible validation
// library for Go. It follows clean architecture principles and integrates
// seamlessly with the ERM error management package.
//
// Unlike tag-based validation libraries, this package uses function chaining
// to create readable and maintainable validation rules. The package supports
// type-safe validation with Go generics, internationalization, conditional
// validation, and comprehensive error reporting suitable for modern APIs.
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
//		errorMap := val.ToError()
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
// The package integrates seamlessly with onion/clean architecture patterns:
//
//	func (s *UserService) CreateUser(user User) error {
//		if err := vix.String(user.Email, "email").Required().Email().Validate(); err != nil {
//			return erm.BadRequest("Invalid email", err)
//		}
//		// Business logic continues...
//		return nil
//	}
package vix

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"
)

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
	Custom(fn func(value interface{}) error) ValidatorChain
}

// ValidationResult holds the result of a validation operation.
// It contains the original value, field name, and any validation errors.
type ValidationResult struct {
	Value      interface{}
	FieldName  string
	Errors     []error
	IsValid    bool
	locale     *Locale
	httpStatus int
}

// NewValidationResult creates a new ValidationResult with the given value and field name.
func NewValidationResult(value interface{}, fieldName string) *ValidationResult {
	if fieldName == "" {
		fieldName = "value"
	}
	return &ValidationResult{
		Value:      value,
		FieldName:  fieldName,
		Errors:     []error{},
		IsValid:    true,
		locale:     DefaultLocale,
		httpStatus: http.StatusBadRequest,
	}
}

// WithLocale sets the locale for error messages.
func (vr *ValidationResult) WithLocale(locale *Locale) *ValidationResult {
	vr.locale = locale

	// Update locale for existing errors
	for _, err := range vr.Errors {
		if ve, ok := err.(*ValidationError); ok {
			ve.locale = locale
		}
	}

	return vr
}

// WithHTTPStatus sets the HTTP status code for validation errors.
func (vr *ValidationResult) WithHTTPStatus(status int) *ValidationResult {
	vr.httpStatus = status
	return vr
}

// AddError adds a validation error to the result.
func (vr *ValidationResult) AddError(err error) *ValidationResult {
	if err != nil {
		vr.Errors = append(vr.Errors, err)
		vr.IsValid = false
	}
	return vr
}

// Valid returns true if no validation errors occurred.
func (vr *ValidationResult) Valid() bool {
	return vr.IsValid
}

// Error returns the first validation error, or nil if validation passed.
func (vr *ValidationResult) Error() error {
	if len(vr.Errors) == 0 {
		return nil
	}
	return vr.Errors[0]
}

// AllErrors returns all validation errors.
func (vr *ValidationResult) AllErrors() []error {
	return vr.Errors
}

// ToError returns a map of field names to error messages.
// Returns nil if validation passed, otherwise returns a map with the field name
// as key and array of error messages as value.
func (vr *ValidationResult) ToError() map[string][]string {
	if vr.Valid() {
		return nil
	}

	errorMap := make(map[string][]string)
	var messages []string

	for _, err := range vr.Errors {
		messages = append(messages, err.Error())
	}

	if len(messages) > 0 {
		errorMap[vr.FieldName] = messages
	}

	return errorMap
}

// ValidationError represents a validation error with template support.
type ValidationError struct {
	Code       string
	Template   string
	FieldName  string
	Value      interface{}
	Params     map[string]interface{}
	locale     *Locale
	httpStatus int
}

// NewValidationError creates a new ValidationError.
func NewValidationError(code, template, fieldName string, value interface{}) *ValidationError {
	return &ValidationError{
		Code:       code,
		Template:   template,
		FieldName:  fieldName,
		Value:      value,
		Params:     make(map[string]interface{}),
		locale:     DefaultLocale,
		httpStatus: http.StatusBadRequest,
	}
}

// WithParam adds a parameter for template substitution.
func (ve *ValidationError) WithParam(key string, value interface{}) *ValidationError {
	ve.Params[key] = value
	return ve
}

// WithLocale sets the locale for error message formatting.
func (ve *ValidationError) WithLocale(locale *Locale) *ValidationError {
	ve.locale = locale
	return ve
}

// WithHTTPStatus sets the HTTP status code for the error.
func (ve *ValidationError) WithHTTPStatus(status int) *ValidationError {
	ve.httpStatus = status
	return ve
}

// Error implements the error interface.
func (ve *ValidationError) Error() string {
	template := ve.Template
	if ve.locale != nil {
		if localized, exists := ve.locale.Messages[ve.Code]; exists {
			template = localized
		}
	}

	return ve.formatTemplate(template)
}

// formatTemplate formats the error message template with parameters.
func (ve *ValidationError) formatTemplate(template string) string {
	result := template

	// Replace field name and value
	result = strings.ReplaceAll(result, "{{field}}", ve.FieldName)
	result = strings.ReplaceAll(result, "{{value}}", fmt.Sprintf("%v", ve.Value))

	// Replace custom parameters
	for key, value := range ve.Params {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result
}

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
func (bv *BaseValidator) addValidationError(code, template string, params map[string]interface{}) {
	if !bv.shouldValidate() {
		return
	}

	err := NewValidationError(code, template, bv.fieldName, bv.value).
		WithLocale(bv.result.locale).
		WithHTTPStatus(bv.result.httpStatus)

	for key, value := range params {
		err.WithParam(key, value)
	}

	if bv.negated {
		// For negated validations, we need to flip the logic
		err.Code = "not_" + err.Code
		err.Template = "{{field}} must not " + strings.ToLower(template[strings.Index(template, " ")+1:])
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
func (bv *BaseValidator) Custom(fn func(value interface{}) error) *BaseValidator {
	if !bv.shouldValidate() {
		return bv
	}

	// Handle nil function gracefully
	if fn == nil {
		return bv
	}

	err := fn(bv.value)
	if err != nil {
		if bv.negated {
			// If negated and custom validation failed, it's actually valid
			bv.negated = false
			return bv
		}
		bv.result.AddError(err)
	} else if bv.negated {
		// If negated and custom validation passed, it's invalid
		bv.result.AddError(NewValidationError("custom_negated", "{{field}} must not satisfy custom validation", bv.fieldName, bv.value))
	}

	bv.negated = false
	return bv
}

// Common validation codes
const (
	CodeRequired     = "required"
	CodeMinLength    = "min_length"
	CodeMaxLength    = "max_length"
	CodeExactLength  = "exact_length"
	CodeEmail        = "email"
	CodeURL          = "url"
	CodeNumeric      = "numeric"
	CodeAlpha        = "alpha"
	CodeAlphaNumeric = "alpha_numeric"
	CodeRegex        = "regex"
	CodeIn           = "in"
	CodeNotIn        = "not_in"
	CodeMin          = "min"
	CodeMax          = "max"
	CodeBetween      = "between"
	CodeAfter        = "after"
	CodeBefore       = "before"
	CodeDateFormat   = "date_format"
)

// Common validation patterns
var (
	EmailRegex        = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	URLRegex          = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	NumericRegex      = regexp.MustCompile(`^[0-9]+$`)
	AlphaRegex        = regexp.MustCompile(`^[a-zA-Z]+$`)
	AlphaNumericRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

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

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

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

// Package-level convenience functions for multi-field validation

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
//		errorMap := val.ToError()
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
