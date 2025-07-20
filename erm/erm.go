// Package erm provides comprehensive error management utilities for Go applications
// following onion/clean architecture patterns. It enriches errors with stack traces,
// HTTP status codes, safe user-facing messages, validation error capabilities,
// and error collection support for handling multiple related errors, serving as a unified
// error management system for both application and validation errors.
//
// The package follows KISS and SOLID principles, uses standard go-i18n for internationalization,
// and provides a clean API for error propagation across application layers. It unifies
// general application errors and validation errors under a single, consistent interface.
//
// The package uses an interface-based design where Error is the main interface
// implemented by StackError. This provides flexibility while maintaining
// type safety and compatibility with Go's standard error handling.
//
// # Basic Usage
//
//	// Create errors with automatic operation detection
//	err := erm.New(http.StatusBadRequest, "Invalid email", originalErr)
//
//	// Use convenience constructors
//	err := erm.BadRequest("Invalid input", originalErr)
//
//	// Extract information safely from any error
//	status := erm.Status(err)   // Works with any error type
//	message := erm.Message(err) // Safe for user consumption
//
// # Validation Error Usage
//
//	// Create validation errors with message keys
//	err := erm.NewValidationError("validation.required", "email", "")
//	err = err.WithParam("min", 5)
//
//	// Use convenience constructors for common validations
//	err := erm.RequiredError("email", "")
//	err := erm.MinLengthError("password", "123", 8)
//
//	// Localized formatting with standard go-i18n
//	fmt.Println(err.Error()) // "email is required" or localized message
//
// # Error Collection Usage
//
//	// Create error containers and collect multiple errors
//	container := erm.New(http.StatusBadRequest, "Validation errors", nil)
//	result := container.AddError(err1).AddError(err2)
//	errorMap := result.ErrMap() // Localized error map
//
// # Internationalization
//
// The package uses the standard github.com/nicksnyder/go-i18n/v2/i18n package
// for internationalization. Messages are resolved on-demand when Error() or
// ToError() methods are called, using the configured default localizer.
//
//	// Set global localizer for default language
//	erm.SetLocalizer(localizer)
//
//	// Get localized error messages
//	localizedMsg := err.LocalizedError(customLocalizer)
//	localizedMap := err.LocalizedErrMap(customLocalizer)
//
// All functions handle nil errors gracefully and are safe for concurrent use.
// The package serves as a unified error management system that eliminates the need
// for separate validation error types while maintaining full compatibility with
// Go's standard error handling patterns.
package erm

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const NonFieldErrors = "non_field_errors"

// =============================================================================
// Core Types & Interfaces
// =============================================================================

// Error represents an enriched application error that extends the standard Go error
// interface with additional metadata and capabilities.
//
// Beyond the standard Error() method, it provides:
//   - Code: HTTP status codes for API responses
//   - Stack: Stack traces for debugging (captured only for 500 errors)
//   - Localization: Message key-based internationalization support
//   - Validation: Field-level validation error capabilities
//   - Error Collection: Ability to collect multiple related errors
//   - Safe Messages: User-facing messages separate from internal errors
//
// Error values are immutable after creation and safe for concurrent access.
// They integrate with Go's standard error handling patterns including
// errors.Is, errors.As, and error wrapping/unwrapping.
//
// All methods return appropriate zero values when called on nil receivers,
// making Error instances safe to use without explicit nil checks in most cases.
type Error interface {
	error

	// Code returns the HTTP status code associated with this error
	Code() int

	// Unwrap returns the wrapped error for errors.Is/As compatibility
	Unwrap() error

	// Stack returns the stack trace as an array of program counters.
	// Returns nil for client errors (4xx) and non-nil for server errors (500)
	Stack() []uintptr

	// MessageKey returns the i18n message key for localization
	MessageKey() string

	// FieldName returns the field name for validation errors
	FieldName() string

	// Value returns the value being validated for validation errors
	Value() interface{}

	// Params returns template parameters for i18n message substitution
	Params() map[string]interface{}

	// AddError adds another error to this error's collection.
	// This is used for collecting multiple validation errors.
	AddError(Error)

	// AddErrors adds multiple errors to this error's collection.
	// This is a convenience method for adding multiple errors at once.
	AddErrors([]Error)

	// AllErrors returns all child errors. Returns empty slice if no child errors.
	AllErrors() []Error

	// HasErrors returns true if this error contains child errors.
	HasErrors() bool

	// LocalizedError returns the localized error message using the provided localizer
	LocalizedError(*i18n.Localizer) string

	// LocalizedErrMap returns a map of field names to localized error messages
	LocalizedErrMap(*i18n.Localizer) map[string][]string

	// ErrMap returns a map of field names to error messages using the default localizer
	ErrMap() map[string][]string

	// WithMessageKey sets the i18n message key and returns a new Error
	WithMessageKey(messageKey string) Error

	// WithFieldName sets the field name being validated
	WithFieldName(fieldName string) Error

	// WithValue sets the value being validated
	WithValue(value interface{}) Error

	// WithParam adds a template parameter for i18n substitution
	WithParam(key string, value interface{}) Error
}

// StackError represents an application error enriched with stack trace,
// HTTP status code, safe user-facing message, validation error capabilities,
// and support for collecting multiple related errors with i18n support.
//
// StackError captures the following information when created:
//   - code: HTTP status code (e.g., http.StatusBadRequest)
//   - msg: Safe user-facing message for client responses
//   - root: Wrapped underlying error maintaining error chain
//   - stack: Stack trace as program counters for debugging (only for 500 errors)
//   - messageKey: i18n message key for localization (e.g., "validation.required")
//   - fieldName: Field name being validated
//   - value: Value being validated
//   - params: Template parameters for i18n substitution
//   - errors: Child errors for batch validation scenarios (single-level only)
//
// StackError values are immutable after creation and are safe for
// concurrent access. They satisfy Go's standard error wrapping
// expectations and work with errors.Is/As functions.
type StackError struct {
	code       int
	msg        string
	root       error
	stack      []uintptr
	messageKey string
	fieldName  string
	value      interface{}
	params     map[string]interface{}
	errors     []Error
}

// =============================================================================
// Core Constructors
// =============================================================================

// New creates a new Error with stack trace capture and HTTP status code.
// Stack traces are only captured for Internal Server Errors (HTTP 500) to optimize
// performance for client errors which don't need debugging information.
//
// Parameters:
//   - code: HTTP status code (if 0, defaults to http.StatusInternalServerError)
//   - msg: User-safe message for client responses
//   - err: Underlying error to wrap (can be nil; if nil, no root error is stored)
//
// When err is nil, the returned Error will have a nil root error but will still
// contain the provided message and status code. The message can be accessed via
// the Error interface methods.
//
// Stack traces are only captured for server errors (500) where debugging is needed.
//
// Example:
//
//	err := erm.New(http.StatusBadRequest, "Invalid email format", validationErr)
//	// err.Code() returns 400
//	// err.Stack() returns nil (no stack trace for client errors)
//	// err.Unwrap() returns validationErr
//
//	errWithoutRoot := erm.New(http.StatusBadRequest, "Custom message", nil)
//	// errWithoutRoot.Unwrap() returns nil
//	// errWithoutRoot.Error() returns "Custom message"
//
//	serverErr := erm.New(http.StatusInternalServerError, "Database error", dbErr)
//	// serverErr.Stack() returns captured stack trace for debugging
func New(code int, msg string, err error) Error {
	skip := 2 // Skip New and the function that called New
	if code == 0 {
		code = http.StatusInternalServerError
	}

	// Only capture stack traces for Internal Server Errors (500)
	// Client errors (4xx) don't need stack traces for debugging
	var stack []uintptr
	if code == http.StatusInternalServerError {
		// Capture up to 32 PCs, skipping appropriate frames
		const depth = 32
		var pcs [depth]uintptr
		n := runtime.Callers(skip, pcs[:])
		stack = pcs[:n]
	}

	return &StackError{
		code:  code,
		msg:   msg,
		root:  err,
		stack: stack, // Will be nil for non-500 errors
	}
}

// =============================================================================
// Basic StackError Methods
// =============================================================================

// Error returns the error message as a string, satisfying Go's error interface.
// Uses the default localizer for internationalization when a message key is available.
// If child errors are present, it formats them as a collection.
func (e *StackError) Error() string {
	if e == nil {
		return "<nil>"
	}

	// If we have a message key or child errors, use localized error formatting
	if e.messageKey != "" || len(e.errors) > 0 {
		return e.LocalizedError(GetLocalizer())
	}

	// Otherwise use the existing logic for non-localized errors
	if e.root != nil {
		return e.root.Error()
	}
	if e.msg != "" {
		return e.msg
	}
	return "unknown error"
}

// Code returns the HTTP status code associated with this error.
func (e *StackError) Code() int {
	if e == nil {
		return 0
	}
	return e.code
}

// Unwrap returns the underlying error for Go 1.13+ error wrapping support.
func (e *StackError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.root
}

// Stack returns the captured stack trace as program counters.
// Returns nil for client errors (4xx) and stack trace for server errors (500).
func (e *StackError) Stack() []uintptr {
	if e == nil {
		return nil
	}
	// Return copy to prevent modification
	if e.stack == nil {
		return nil
	}
	stack := make([]uintptr, len(e.stack))
	copy(stack, e.stack)
	return stack
}

// =============================================================================
// Validation-Related Methods
// =============================================================================

// MessageKey returns the i18n message key for localization.
// Returns empty string for nil receivers or if no message key was set.
func (e *StackError) MessageKey() string {
	if e == nil {
		return ""
	}
	return e.messageKey
}

// FieldName returns the field name being validated.
// Returns empty string for nil receivers or if no field name was set.
func (e *StackError) FieldName() string {
	if e == nil {
		return ""
	}
	return e.fieldName
}

// Value returns the value being validated.
// Returns nil for nil receivers or if no value was set.
func (e *StackError) Value() interface{} {
	if e == nil {
		return nil
	}
	return e.value
}

// Params returns the template parameters.
// Returns nil for nil receivers or if no parameters were set.
func (e *StackError) Params() map[string]interface{} {
	if e == nil {
		return nil
	}
	return e.params
}

// WithMessageKey sets the i18n message key.
func (e *StackError) WithMessageKey(messageKey string) Error {
	if e == nil {
		return nil
	}
	new := *e
	new.messageKey = messageKey
	return &new
}

// WithFieldName sets the field name being validated.
func (e *StackError) WithFieldName(fieldName string) Error {
	if e == nil {
		return nil
	}
	new := *e
	new.fieldName = fieldName
	return &new
}

// WithValue sets the value being validated.
func (e *StackError) WithValue(value interface{}) Error {
	if e == nil {
		return nil
	}
	new := *e
	new.value = value
	return &new
}

// WithParam adds a template parameter.
func (e *StackError) WithParam(key string, value interface{}) Error {
	if e == nil {
		return nil
	}
	new := *e
	if new.params == nil {
		new.params = make(map[string]interface{})
	} else {
		// Copy the params map to avoid modifying the original
		newParams := make(map[string]interface{})
		for k, v := range new.params {
			newParams[k] = v
		}
		new.params = newParams
	}
	new.params[key] = value
	return &new
}

// =============================================================================
// Error Collection Methods
// =============================================================================

// AddError adds a child error to this error's collection.
// If the added error already contains child errors, they are flattened
// to prevent deep nesting (only one level of error collection is allowed).
func (e *StackError) AddError(err Error) {
	if e == nil || err == nil {
		return
	}

	// Initialize errors slice if needed
	if e.errors == nil {
		e.errors = make([]Error, 0)
	}

	// If the added error has child errors, flatten them to prevent deep nesting
	if err.HasErrors() {
		childErrors := err.AllErrors()
		for _, childErr := range childErrors {
			if childErr != nil {
				e.errors = append(e.errors, childErr)
			}
		}
	} else {
		// Add the error directly
		e.errors = append(e.errors, err)
	}
}

// AddErrors adds multiple errors to this error's collection.
// This is a convenience method for adding multiple errors at once.
func (e *StackError) AddErrors(errs []Error) {
	if e == nil {
		return
	}

	for _, err := range errs {
		e.AddError(err)
	}
}

// AllErrors returns all child errors. Returns empty slice if no child errors.
func (e *StackError) AllErrors() []Error {
	if e == nil {
		return nil
	}
	return e.errors
}

// HasErrors returns true if this error contains child errors.
func (e *StackError) HasErrors() bool {
	if e == nil {
		return false
	}
	return len(e.errors) > 0
}

// =============================================================================
// Localization Methods
// =============================================================================

// ErrMap returns a map of field names to error messages using the default localizer.
// Returns nil if no errors exist. Convenience method for LocalizedErrMap(GetLocalizer()).
func (e *StackError) ErrMap() map[string][]string {
	return e.LocalizedErrMap(GetLocalizer())
}

// LocalizedError returns the error message using the provided localizer.
// Falls back to default localizer if provided localizer is nil.
func (e *StackError) LocalizedError(localizer *i18n.Localizer) string {
	if e == nil {
		return "<nil>"
	}

	// Handle child errors
	if len(e.errors) > 0 {
		return e.formatChildErrors(localizer)
	}

	// Handle message key localization
	if e.messageKey != "" {
		if msg := e.localizeMessage(localizer); msg != "" {
			return msg
		}
	}

	// Fallback to existing error message logic
	return e.getFallbackMessage()
}

// formatChildErrors handles formatting multiple child errors.
func (e *StackError) formatChildErrors(localizer *i18n.Localizer) string {
	var messages []string
	for _, err := range e.errors {
		if err != nil {
			messages = append(messages, err.LocalizedError(localizer))
		}
	}

	switch len(messages) {
	case 0:
		return ""
	case 1:
		return messages[0]
	default:
		return e.formatMultipleErrors(messages, localizer)
	}
}

// formatMultipleErrors formats multiple error messages.
func (e *StackError) formatMultipleErrors(messages []string, localizer *i18n.Localizer) string {
	usedLocalizer := e.getEffectiveLocalizer(localizer)
	if usedLocalizer != nil {
		return usedLocalizer.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "error.multiple",
			TemplateData: map[string]interface{}{
				"errors": strings.Join(messages, "; "),
			},
		})
	}
	return fmt.Sprintf("multiple errors: %s", strings.Join(messages, "; "))
}

// localizeMessage attempts to localize a message using the message key.
func (e *StackError) localizeMessage(localizer *i18n.Localizer) string {
	usedLocalizer := e.getEffectiveLocalizer(localizer)
	if usedLocalizer == nil {
		return ""
	}

	templateData := e.buildTemplateData()
	msg, err := usedLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    e.messageKey,
		TemplateData: templateData,
	})
	if err == nil && msg != "" {
		return msg
	}
	return ""
}

// buildTemplateData creates template data for localization.
func (e *StackError) buildTemplateData() map[string]interface{} {
	templateData := make(map[string]interface{})

	if e.fieldName != "" {
		templateData["field"] = e.fieldName
	}
	if e.value != nil {
		templateData["value"] = e.value
	}
	if e.params != nil {
		for key, value := range e.params {
			templateData[key] = value
		}
	}
	return templateData
}

// getEffectiveLocalizer returns the provided localizer or falls back to default.
func (e *StackError) getEffectiveLocalizer(localizer *i18n.Localizer) *i18n.Localizer {
	if localizer != nil {
		return localizer
	}
	return GetLocalizer()
}

// getFallbackMessage provides fallback error messages when localization fails.
func (e *StackError) getFallbackMessage() string {
	if e.root != nil {
		return e.root.Error()
	}
	if e.msg != "" {
		return e.msg
	}
	if e.messageKey != "" && e.fieldName != "" {
		return fmt.Sprintf("validation error for field '%s'", e.fieldName)
	}
	if e.messageKey != "" {
		return fmt.Sprintf("validation error (key: %s)", e.messageKey)
	}
	return "unknown error"
}

// LocalizedErrMap returns a map of field names to localized error messages
// using the provided localizer. Falls back to default localizer if nil.
func (e *StackError) LocalizedErrMap(localizer *i18n.Localizer) map[string][]string {
	if e == nil {
		return nil
	}

	result := make(map[string][]string)

	// If we have child errors, process them
	if len(e.errors) > 0 {
		for _, err := range e.errors {
			if err == nil {
				continue
			}

			fieldName := err.FieldName()
			if fieldName == "" {
				fieldName = "error" // Fallback for errors without field names
			}

			result[fieldName] = append(result[fieldName], err.LocalizedError(localizer))
		}
	} else if e.messageKey != "" {
		// If no child errors, treat this error as the single error
		fieldName := e.fieldName
		if fieldName == "" {
			fieldName = "error" // Fallback for errors without field names
		}

		result[fieldName] = append(result[fieldName], e.LocalizedError(localizer))
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// =============================================================================
// Helper Functions
// =============================================================================

// Status extracts the HTTP status code from any error, providing
// a safe way to get status codes from mixed error types.
//
// Returns:
//   - For erm.Error: the actual status code
//   - For standard errors: http.StatusInternalServerError (500)
//   - For nil errors: http.StatusOK (200)
func Status(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if e, ok := err.(Error); ok {
		return e.Code()
	}

	return http.StatusInternalServerError
}

// Message extracts a safe user-facing message from any error.
// The returned message is safe to send to clients without leaking
// internal implementation details.
//
// Returns:
//   - For erm errors with custom message: the custom message
//   - For erm errors without message: HTTP status text (e.g., "Bad Request")
//   - For standard errors: "Internal Server Error"
//   - For nil errors: empty string
func Message(err error) string {
	if err == nil {
		return ""
	}

	// Use Error interface for consistency with Status function
	if e, ok := err.(Error); ok {
		// Try to get message from StackError implementation
		if se, ok := e.(*StackError); ok && se.msg != "" {
			return se.msg
		}
		// Fallback to HTTP status text
		return http.StatusText(e.Code())
	}

	// For non-erm errors, return status text
	return http.StatusText(http.StatusInternalServerError)
}

// Stack extracts the stack trace from any error that supports it.
// Use this with FormatStack to get human-readable stack traces
// for logging and debugging.
//
// Returns:
//   - For erm errors: captured stack trace as program counters
//   - For other errors: nil
//   - For nil errors: nil
func Stack(err error) []uintptr {
	if err == nil {
		return nil
	}

	if e, ok := err.(Error); ok {
		return e.Stack()
	}

	return nil
}

// Wrap wraps an error with erm error capabilities while preserving
// the original error's metadata when possible.
//
// Behavior:
//   - If err is nil: returns nil
//   - If err is already an erm error: returns it unchanged
//   - If err is a standard error: wraps it with http.StatusInternalServerError
//
// The operation context is updated to reflect the current call site,
// making it useful for adding operation context as errors bubble up
// through application layers.
func Wrap(err error) Error {
	if err == nil {
		return nil
	}

	// If it's already an erm error, don't wrap it
	if e, ok := err.(Error); ok {
		return e
	}

	// For standard errors, default to 500
	return New(http.StatusInternalServerError, "Internal Server Error", err)
}

// FormatStack formats a stack trace into a human-readable string
// suitable for logging and debugging. Each frame shows the function
// name, file path, and line number.
//
// Returns empty string if the error is nil or has no stack trace.
//
// Example output:
//
//	main.processUser
//		/app/user.go:42
//	main.handleRequest
//		/app/handler.go:28
func FormatStack(err Error) string {
	if err == nil {
		return ""
	}

	pcs := err.Stack()
	if len(pcs) == 0 {
		return ""
	}

	var buf strings.Builder
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		_, _ = fmt.Fprintf(&buf, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return buf.String()
}

// =============================================================================
// Convenience Constructors
// =============================================================================
//
// HTTP status code convenience constructors. Each creates an error with the
// corresponding HTTP status code, user-safe message, and optional underlying error.
// All constructors follow the same pattern: StatusName(msg string, err error) Error.

// BadRequest creates a 400 Bad Request error.
func BadRequest(msg string, err error) Error {
	return New(http.StatusBadRequest, msg, err)
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(msg string, err error) Error {
	return New(http.StatusUnauthorized, msg, err)
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(msg string, err error) Error {
	return New(http.StatusForbidden, msg, err)
}

// NotFound creates a 404 Not Found error.
func NotFound(msg string, err error) Error {
	return New(http.StatusNotFound, msg, err)
}

// Conflict creates a 409 Conflict error.
func Conflict(msg string, err error) Error {
	return New(http.StatusConflict, msg, err)
}

// Internal creates a 500 Internal Server Error.
func Internal(msg string, err error) Error {
	return New(http.StatusInternalServerError, msg, err)
}

// =============================================================================
// Validation Error Constructors
// =============================================================================

// NewValidationError creates a new validation error with the specified message key, field name, and value.
// This is the primary constructor for validation errors that will be used by the validation package.
// All validation errors are created with HTTP 400 Bad Request status.
//
// Example:
//
//	err := erm.NewValidationError("validation.required", "email", "")
//	err = err.WithParam("min", 5)
func NewValidationError(messageKey, fieldName string, value interface{}) Error {
	return New(http.StatusBadRequest, "", nil).
		WithMessageKey(messageKey).
		WithFieldName(fieldName).
		WithValue(value)
}

// Common validation error constructors using standard message keys.
// These provide convenient creation of typical validation errors with
// proper internationalization support.

// RequiredError creates a "required" validation error.
func RequiredError(fieldName string, value interface{}) Error {
	return NewValidationError("validation.required", fieldName, value)
}

// MinLengthError creates a "min_length" validation error with minimum length parameter.
func MinLengthError(fieldName string, value interface{}, min int) Error {
	return NewValidationError("validation.min_length", fieldName, value).
		WithParam("min", min)
}

// MaxLengthError creates a "max_length" validation error with maximum length parameter.
func MaxLengthError(fieldName string, value interface{}, max int) Error {
	return NewValidationError("validation.max_length", fieldName, value).
		WithParam("max", max)
}

// EmailError creates an "email" validation error.
func EmailError(fieldName string, value interface{}) Error {
	return NewValidationError("validation.email", fieldName, value)
}

// MinValueError creates a "min_value" validation error with minimum value parameter.
func MinValueError(fieldName string, value interface{}, min interface{}) Error {
	return NewValidationError("validation.min_value", fieldName, value).
		WithParam("min", min)
}

// MaxValueError creates a "max_value" validation error with maximum value parameter.
func MaxValueError(fieldName string, value interface{}, max interface{}) Error {
	return NewValidationError("validation.max_value", fieldName, value).
		WithParam("max", max)
}

// DuplicateError creates a "duplicate" validation error for unique constraint violations.
func DuplicateError(fieldName string, value interface{}) Error {
	return NewValidationError("validation.duplicate", fieldName, value)
}
