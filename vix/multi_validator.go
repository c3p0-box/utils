package vix

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c3p0-box/utils/erm"
	"golang.org/x/text/language"
)

// ValidationOrchestrator manages multiple validators and provides a unified interface
// for validating multiple fields and creating comprehensive error reports.
// It supports namespace organization for nested structures and arrays.
//
// The orchestrator collects validation results from multiple validators and
// provides methods to check validity, retrieve errors, and generate structured
// output suitable for API responses. All error messages are automatically
// localized through the ERM package's i18n system.
//
// The orchestrator uses an erm.Error container internally to collect and manage
// all validation errors, leveraging ERM's error collection capabilities for
// efficient error handling and reporting.
type ValidationOrchestrator struct {
	fieldResults map[string]*ValidationResult // Field name to validation result mapping
	fieldOrder   []string                     // Preserve field order
	err          erm.Error                    // Container for all validation errors
}

// NewValidationOrchestrator creates a new ValidationOrchestrator.
func NewValidationOrchestrator() *ValidationOrchestrator {
	return &ValidationOrchestrator{
		fieldResults: make(map[string]*ValidationResult),
		fieldOrder:   make([]string, 0),
		err:          erm.New(400, "", nil),
	}
}

// V creates a new ValidationOrchestrator. This is a shorthand function for convenience.
func V() *ValidationOrchestrator {
	return NewValidationOrchestrator()
}

// Is adds multiple validators to the orchestrator.
func (vo *ValidationOrchestrator) Is(validators ...Validator) *ValidationOrchestrator {
	for _, validator := range validators {
		result := validator.Result()
		vo.addResult(result.FieldName, result)
	}
	return vo
}

// In adds validations within a namespace. The namespace is prefixed to field names.
func (vo *ValidationOrchestrator) In(namespace string, orchestrator *ValidationOrchestrator) *ValidationOrchestrator {
	for fieldName, result := range orchestrator.fieldResults {
		namespacedField := namespace + "." + fieldName
		vo.addResult(namespacedField, result)
	}
	return vo
}

// InRow adds validations within an indexed namespace. The namespace and index are prefixed to field names.
func (vo *ValidationOrchestrator) InRow(namespace string, index int, orchestrator *ValidationOrchestrator) *ValidationOrchestrator {
	for fieldName, result := range orchestrator.fieldResults {
		namespacedField := fmt.Sprintf("%s[%d].%s", namespace, index, fieldName)
		vo.addResult(namespacedField, result)
	}
	return vo
}

// addResult adds a validation result to the orchestrator.
func (vo *ValidationOrchestrator) addResult(fieldName string, result *ValidationResult) {
	if _, exists := vo.fieldResults[fieldName]; !exists {
		vo.fieldOrder = append(vo.fieldOrder, fieldName)
	}
	vo.fieldResults[fieldName] = result
}

// Valid returns true if all validations passed.
func (vo *ValidationOrchestrator) Valid() bool {
	// Clear old errors by creating a new container
	vo.err = erm.New(400, "", nil)

	// Collect errors from all field results, preserving namespaced field names
	for _, fieldName := range vo.fieldOrder {
		result := vo.fieldResults[fieldName]
		if !result.Valid() {
			// Update field names to match the namespaced field name managed by orchestrator
			errors := result.AllErrors()
			namespacedErrors := make([]erm.Error, 0, len(errors))
			for _, err := range errors {
				// Create a new error with the namespaced field name
				namespacedErr := err.WithFieldName(fieldName)
				namespacedErrors = append(namespacedErrors, namespacedErr)
			}
			vo.err.AddErrors(namespacedErrors)
		}
	}

	return !vo.err.HasErrors()
}

// IsValid returns true if the specific field validation passed.
func (vo *ValidationOrchestrator) IsValid(fieldName string) bool {
	if result, exists := vo.fieldResults[fieldName]; exists {
		return result.Valid()
	}
	return true // If field doesn't exist, consider it valid
}

// ErrMap returns a map of field names to error messages using the default localizer.
// Now leverages ERM's localized error map functionality while preserving
// the namespaced field names managed by the orchestrator.
func (vo *ValidationOrchestrator) ErrMap() map[string][]string {
	if vo.Valid() {
		return nil
	}

	// Use ERM's localized error map with namespaced field names
	return vo.err.ErrMap()
}

// LocalizedErrMap returns a map of field names to localized error messages
// for the specified language. This provides full internationalization support
// while preserving the orchestrator's namespaced field structure.
func (vo *ValidationOrchestrator) LocalizedErrMap(tag language.Tag) map[string][]string {
	if vo.Valid() {
		return nil
	}

	// Use ERM's localized error map with the specified language
	return vo.err.LocalizedErrMap(tag)
}

// Error returns a single erm.Error containing all validation errors as children.
func (vo *ValidationOrchestrator) Error() error {
	// Call Valid() to ensure errors are collected from field results
	if vo.Valid() {
		return nil
	}

	// Return the error container directly
	return vo.err
}

// FieldNames returns all field names that have been validated.
func (vo *ValidationOrchestrator) FieldNames() []string {
	return append([]string(nil), vo.fieldOrder...)
}

// GetFieldResult returns the validation result for a specific field.
func (vo *ValidationOrchestrator) GetFieldResult(fieldName string) *ValidationResult {
	return vo.fieldResults[fieldName]
}

// ToJSON returns the error map as JSON bytes.
func (vo *ValidationOrchestrator) ToJSON() ([]byte, error) {
	if vo.Valid() {
		return []byte("{}"), nil
	}

	errorMap := vo.ErrMap()
	return json.Marshal(errorMap)
}

// String returns a string representation of the validation state.
func (vo *ValidationOrchestrator) String() string {
	if vo.Valid() {
		return "ValidationOrchestrator: Valid"
	}

	var messages []string
	for _, fieldName := range vo.fieldOrder {
		result := vo.fieldResults[fieldName]
		if !result.Valid() {
			for _, err := range result.AllErrors() {
				messages = append(messages, fmt.Sprintf("%s: %s", fieldName, err.Error()))
			}
		}
	}

	return fmt.Sprintf("ValidationOrchestrator: Invalid (%s)", strings.Join(messages, ", "))
}
