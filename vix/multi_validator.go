package vix

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationOrchestrator manages multiple field validations and provides
// comprehensive error reporting with support for namespacing and indexing.
type ValidationOrchestrator struct {
	fieldResults map[string]*ValidationResult
	fieldOrder   []string
	locale       *Locale
}

// NewValidationOrchestrator creates a new ValidationOrchestrator.
func NewValidationOrchestrator() *ValidationOrchestrator {
	return &ValidationOrchestrator{
		fieldResults: make(map[string]*ValidationResult),
		fieldOrder:   make([]string, 0),
		locale:       DefaultLocale,
	}
}

// V creates a new ValidationOrchestrator. This is a shorthand function for convenience.
func V() *ValidationOrchestrator {
	return NewValidationOrchestrator()
}

// Is adds a validation result to the orchestrator.
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

// WithLocale sets the locale for all validation results.
func (vo *ValidationOrchestrator) WithLocale(locale *Locale) *ValidationOrchestrator {
	vo.locale = locale
	for _, result := range vo.fieldResults {
		result.WithLocale(locale)
	}
	return vo
}

// addResult adds a validation result to the orchestrator.
func (vo *ValidationOrchestrator) addResult(fieldName string, result *ValidationResult) {
	if result.locale == nil {
		result.WithLocale(vo.locale)
	}

	if _, exists := vo.fieldResults[fieldName]; !exists {
		vo.fieldOrder = append(vo.fieldOrder, fieldName)
	}
	vo.fieldResults[fieldName] = result
}

// Valid returns true if all validations passed.
func (vo *ValidationOrchestrator) Valid() bool {
	for _, result := range vo.fieldResults {
		if !result.Valid() {
			return false
		}
	}
	return true
}

// IsValid returns true if the specific field validation passed.
func (vo *ValidationOrchestrator) IsValid(fieldName string) bool {
	if result, exists := vo.fieldResults[fieldName]; exists {
		return result.Valid()
	}
	return true // Field not found means no validation was performed, so it's valid
}

// ToError returns a structured error map suitable for JSON serialization.
// Returns nil if all validations passed.
func (vo *ValidationOrchestrator) ToError() map[string][]string {
	if vo.Valid() {
		return nil
	}

	errorMap := make(map[string][]string)

	for _, fieldName := range vo.fieldOrder {
		result := vo.fieldResults[fieldName]
		if !result.Valid() {
			var messages []string
			for _, err := range result.AllErrors() {
				messages = append(messages, err.Error())
			}
			if len(messages) > 0 {
				errorMap[fieldName] = messages
			}
		}
	}

	return errorMap
}

// ToJSON returns the error map as JSON bytes.
func (vo *ValidationOrchestrator) ToJSON() ([]byte, error) {
	errorMap := vo.ToError()
	if errorMap == nil {
		return []byte("{}"), nil
	}
	return json.MarshalIndent(errorMap, "", "  ")
}

// Error returns the first error found, or nil if all validations passed.
func (vo *ValidationOrchestrator) Error() error {
	for _, fieldName := range vo.fieldOrder {
		result := vo.fieldResults[fieldName]
		if !result.Valid() {
			return result.Error()
		}
	}
	return nil
}

// AllErrors returns all errors from all fields.
func (vo *ValidationOrchestrator) AllErrors() []error {
	var allErrors []error
	for _, fieldName := range vo.fieldOrder {
		result := vo.fieldResults[fieldName]
		allErrors = append(allErrors, result.AllErrors()...)
	}
	return allErrors
}

// FieldNames returns all field names that have been validated.
func (vo *ValidationOrchestrator) FieldNames() []string {
	return append([]string(nil), vo.fieldOrder...)
}

// GetFieldResult returns the validation result for a specific field.
func (vo *ValidationOrchestrator) GetFieldResult(fieldName string) *ValidationResult {
	return vo.fieldResults[fieldName]
}

// String returns a human-readable string representation of all errors.
func (vo *ValidationOrchestrator) String() string {
	if vo.Valid() {
		return "validation passed"
	}

	var messages []string
	for _, fieldName := range vo.fieldOrder {
		result := vo.fieldResults[fieldName]
		if !result.Valid() {
			for _, err := range result.AllErrors() {
				messages = append(messages, err.Error())
			}
		}
	}

	return strings.Join(messages, "; ")
}
