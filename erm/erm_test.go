package erm

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/text/language"
)

// Tests use the automatic GetLocalizer system

// =============================================================================
// Core Constructor Tests
// =============================================================================

// TestNew tests the New function with various inputs
func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		code      int
		msg       string
		err       error
		wantCode  int
		wantStack bool // true if stack trace should be present
	}{
		{"with 400 code", http.StatusBadRequest, "bad request", errors.New("root"), http.StatusBadRequest, false},
		{"with zero code (becomes 500)", 0, "test message", errors.New("test"), http.StatusInternalServerError, true},
		{"with 500 code", http.StatusInternalServerError, "server error", errors.New("db error"), http.StatusInternalServerError, true},
		{"with nil error", http.StatusBadRequest, "test message", nil, http.StatusBadRequest, false},
		{"with 404 code", http.StatusNotFound, "not found", errors.New("resource missing"), http.StatusNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(tt.code, tt.msg, tt.err)

			// Verify interface implementation
			if _, ok := e.(Error); !ok {
				t.Fatal("New() should return Error interface")
			}

			// Verify fields
			if e.Code() != tt.wantCode {
				t.Fatalf("Code() = %d, want %d", e.Code(), tt.wantCode)
			}

			if Message(e) != tt.msg {
				t.Fatalf("Message() = %q, want %q", Message(e), tt.msg)
			}

			// Test stack trace presence based on status code
			hasStack := len(e.Stack()) > 0
			if hasStack != tt.wantStack {
				if tt.wantStack {
					t.Fatal("Stack() should not be empty for 500 errors")
				} else {
					t.Fatal("Stack() should be empty for non-500 errors")
				}
			}

			// Test error chain
			if tt.err != nil {
				if !errors.Is(e, tt.err) {
					t.Fatal("errors.Is should find wrapped error")
				}

				// Test errors.As
				var target *StackError
				if !errors.As(e, &target) {
					t.Fatal("errors.As should extract *StackError")
				}
			}
		})
	}
}

// =============================================================================
// Basic StackError Method Tests
// =============================================================================

// TestStackErrorError tests the Error() method formatting with various configurations
func TestStackErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *StackError
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "<nil>",
		},
		{
			name:     "root present",
			err:      &StackError{root: errors.New("underlying")},
			expected: "underlying",
		},
		{
			name:     "only root present",
			err:      &StackError{root: errors.New("underlying")},
			expected: "underlying",
		},
		{
			name:     "only message present",
			err:      &StackError{msg: "test message"},
			expected: "test message",
		},
		{
			name:     "nothing present",
			err:      &StackError{},
			expected: "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestStackErrorMethods tests individual StackError methods
func TestStackErrorMethods(t *testing.T) {
	// Test nil receiver handling
	var nilErr *StackError

	if nilErr.Error() != "<nil>" {
		t.Fatal("nil Error() should return '<nil>'")
	}

	if nilErr.Code() != 0 {
		t.Fatal("nil Code() should return 0")
	}

	if nilErr.Stack() != nil {
		t.Fatal("nil Stack() should return nil")
	}

	if nilErr.Unwrap() != nil {
		t.Fatal("nil Unwrap() should return nil")
	}

	// Test with actual error
	err := &StackError{
		code:  http.StatusBadRequest,
		msg:   "test message",
		root:  errors.New("root error"),
		stack: []uintptr{1, 2, 3},
	}

	if err.Code() != http.StatusBadRequest {
		t.Fatalf("Code() = %d, want %d", err.Code(), http.StatusBadRequest)
	}

	if len(err.Stack()) != 3 {
		t.Fatalf("Stack() length = %d, want 3", len(err.Stack()))
	}

	// Test that we get a copy of the stack
	stack1 := err.Stack()
	stack2 := err.Stack()
	if &stack1[0] == &stack2[0] {
		t.Fatal("Stack() should return a copy, not the same slice")
	}

	if err.Unwrap() != err.root {
		t.Fatal("Unwrap() should return root error")
	}

	// Test error without root
	errNoRoot := &StackError{code: 200, msg: "test"}
	if errNoRoot.Error() != "test" {
		t.Fatalf("Error() = %q, want %q", errNoRoot.Error(), "test")
	}
}

// =============================================================================
// Validation-Related Method Tests
// =============================================================================

// TestValidationErrorCapabilities tests the new validation error capabilities
func TestValidationErrorCapabilities(t *testing.T) {
	t.Run("NewValidationError", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")

		if err.MessageKey() != "validation.required" {
			t.Fatalf("MessageKey() = %q, want %q", err.MessageKey(), "validation.required")
		}

		if err.FieldName() != "email" {
			t.Fatalf("FieldName() = %q, want %q", err.FieldName(), "email")
		}

		if err.Value() != "" {
			t.Fatalf("Value() = %v, want empty string", err.Value())
		}

		// Test localized error formatting - will use fallback since no localizer set
		if err.Error() == "" {
			t.Fatalf("Error() should return a non-empty message")
		}
	})

	t.Run("WithParam", func(t *testing.T) {
		err := NewValidationError("validation.min_length", "password", "123").
			WithParam("min", 8)

		params := err.Params()
		if params["min"] != 8 {
			t.Fatalf("Expected min parameter to be 8, got %v", params["min"])
		}
	})

	t.Run("WithValue", func(t *testing.T) {
		err := NewValidationError("validation.required", "age", 25)

		if err.Value() != 25 {
			t.Fatalf("Value() = %v, want 25", err.Value())
		}
	})

	t.Run("WithMessageKey", func(t *testing.T) {
		err := New(http.StatusBadRequest, "Test error", nil).
			WithMessageKey("validation.custom")

		if err.MessageKey() != "validation.custom" {
			t.Fatalf("MessageKey() = %q, want %q", err.MessageKey(), "validation.custom")
		}
	})

	t.Run("Validation error builders return new instances", func(t *testing.T) {
		original := NewValidationError("validation.required", "field", "value")

		withParam := original.WithParam("test", "value")
		withField := original.WithFieldName("newField")
		withValue := original.WithValue("newValue")
		withKey := original.WithMessageKey("validation.custom")

		// All should be different instances
		if original == withParam || original == withField ||
			original == withValue || original == withKey {
			t.Error("Builder methods should return new instances")
		}

		// Original should be unchanged
		if original.FieldName() != "field" {
			t.Error("Original error should be unchanged after builder calls")
		}
	})
}

// =============================================================================
// Error Collection Tests
// =============================================================================

// TestErrorCollection tests the new error collection functionality
func TestErrorCollection(t *testing.T) {
	t.Run("AddError with single error", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)
		childErr := NewValidationError("{{field}} is required", "email", "")

		container.AddError(childErr)

		// Should have child errors
		if !container.HasErrors() {
			t.Error("Container should have child errors")
		}

		// Should contain the added error
		errors := container.AllErrors()
		if len(errors) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(errors))
		}

		if errors[0] != childErr {
			t.Error("Child error should match the added error")
		}
	})

	t.Run("AddError with multiple errors", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)

		err1 := NewValidationError("{{field}} is required", "email", "")
		err2 := NewValidationError("{{field}} is invalid", "phone", "123")

		container.AddError(err1)
		container.AddError(err2)

		errors := container.AllErrors()
		if len(errors) != 2 {
			t.Fatalf("Expected 2 errors, got %d", len(errors))
		}

		// Check that both errors are present
		found := make(map[Error]bool)
		for _, err := range errors {
			found[err] = true
		}

		if !found[err1] || !found[err2] {
			t.Error("Both errors should be present in the collection")
		}
	})

	t.Run("AddError flattens nested errors", func(t *testing.T) {
		// Create a container with child errors
		parent := New(http.StatusBadRequest, "Parent error", nil)
		child1 := NewValidationError("validation.required", "email", "")
		child2 := NewValidationError("validation.invalid", "phone", "123")

		parent.AddError(child1)
		parent.AddError(child2)

		// Add the container to another error (should flatten)
		newParent := New(http.StatusBadRequest, "New parent", nil)
		newParent.AddError(parent)

		// Should have flattened the errors
		errors := newParent.AllErrors()
		if len(errors) != 2 {
			t.Fatalf("Expected 2 flattened errors, got %d", len(errors))
		}

		// Should contain the original child errors, not the container
		found1, found2 := false, false
		for _, err := range errors {
			if err == child1 {
				found1 = true
			}
			if err == child2 {
				found2 = true
			}
		}

		if !found1 || !found2 {
			t.Error("Both original child errors should be present after flattening")
		}
	})

	t.Run("ToError with child errors", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)

		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123").WithParam("min", 8)

		container.AddError(err1)
		container.AddError(err2)
		errorMap := container.ErrMap()

		if errorMap == nil {
			t.Fatal("ErrMap should return error map")
		}

		// Should have entries for both fields
		if emailErrors, ok := errorMap["email"]; !ok || len(emailErrors) != 1 {
			t.Error("Should have one error for email field")
		}

		if passwordErrors, ok := errorMap["password"]; !ok || len(passwordErrors) != 1 {
			t.Error("Should have one error for password field")
		}
	})

	t.Run("Error message formatting with multiple errors", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)

		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123").WithParam("min", 8)

		container.AddError(err1)
		container.AddError(err2)
		errorMsg := container.Error()

		// Should format multiple errors
		if !strings.Contains(errorMsg, "multiple errors:") {
			t.Errorf("Error message should indicate multiple errors, got: %s", errorMsg)
		}

		if !strings.Contains(errorMsg, "email is required") {
			t.Errorf("Error message should contain email error, got: %s", errorMsg)
		}

		if !strings.Contains(errorMsg, "password must be at least") {
			t.Errorf("Error message should contain password error, got: %s", errorMsg)
		}
	})

	t.Run("Error message with single child error", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)
		childErr := NewValidationError("validation.required", "email", "")

		container.AddError(childErr)
		errorMsg := container.Error()

		// Should return the single error message directly
		if errorMsg != "email is required" {
			t.Errorf("Expected 'email is required', got: %s", errorMsg)
		}
	})

	t.Run("HasErrors returns correct values", func(t *testing.T) {
		emptyContainer := New(http.StatusBadRequest, "Empty container", nil)
		if emptyContainer.HasErrors() {
			t.Error("Empty container should not have errors")
		}

		emptyContainer.AddError(NewValidationError("test", "field", "value"))
		containerWithErrors := emptyContainer
		if !containerWithErrors.HasErrors() {
			t.Error("Container with errors should return true for HasErrors")
		}
	})

	t.Run("Nil handling", func(t *testing.T) {
		var nilError *StackError

		// AddError with nil receiver should do nothing (can't modify nil)
		err := NewValidationError("test", "field", "value")
		nilError.AddError(err) // This should not panic and do nothing

		// AddError with nil error should do nothing
		container := New(http.StatusBadRequest, "Container", nil)
		container.AddError(nil) // Should not change container
		if container.HasErrors() {
			t.Error("AddError with nil error should not add anything")
		}

		// AllErrors with nil should return nil
		if nilError.AllErrors() != nil {
			t.Error("AllErrors with nil receiver should return nil")
		}

		// HasErrors with nil should return false
		if nilError.HasErrors() {
			t.Error("HasErrors with nil receiver should return false")
		}
	})
}

// TestAddErrors tests the AddErrors batch method functionality
func TestAddErrors(t *testing.T) {
	t.Run("AddErrors with multiple errors", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)

		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123")
		err3 := NewValidationError("validation.email", "email", "invalid")

		errors := []Error{err1, err2, err3}
		container.AddErrors(errors)

		// Should have child errors
		if !container.HasErrors() {
			t.Error("Container should have child errors")
		}

		// Should contain all added errors
		allErrors := container.AllErrors()
		if len(allErrors) != 3 {
			t.Fatalf("Expected 3 errors, got %d", len(allErrors))
		}

		// Check that all errors are present
		found := make(map[Error]bool)
		for _, err := range allErrors {
			found[err] = true
		}

		if !found[err1] || !found[err2] || !found[err3] {
			t.Error("All errors should be present in the collection")
		}
	})

	t.Run("AddErrors with empty slice", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)

		container.AddErrors([]Error{})

		// Should not have child errors
		if container.HasErrors() {
			t.Error("Container should not have child errors when adding empty slice")
		}

		// Should have no errors
		allErrors := container.AllErrors()
		if len(allErrors) != 0 {
			t.Fatalf("Expected 0 errors, got %d", len(allErrors))
		}
	})

	t.Run("AddErrors with nil slice", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)

		container.AddErrors(nil)

		// Should not have child errors
		if container.HasErrors() {
			t.Error("Container should not have child errors when adding nil slice")
		}

		// Should have no errors
		allErrors := container.AllErrors()
		if len(allErrors) != 0 {
			t.Fatalf("Expected 0 errors, got %d", len(allErrors))
		}
	})

	t.Run("AddErrors with nil receiver", func(t *testing.T) {
		var nilError *StackError

		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123")

		// Should not panic
		nilError.AddErrors([]Error{err1, err2})
	})

	t.Run("AddErrors with slice containing nil errors", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)

		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123")

		errors := []Error{err1, nil, err2, nil}
		container.AddErrors(errors)

		// Should have only non-nil child errors
		allErrors := container.AllErrors()
		if len(allErrors) != 2 {
			t.Fatalf("Expected 2 errors, got %d", len(allErrors))
		}

		// Check that only non-nil errors are present
		found := make(map[Error]bool)
		for _, err := range allErrors {
			found[err] = true
		}

		if !found[err1] || !found[err2] {
			t.Error("Non-nil errors should be present in the collection")
		}
	})

	t.Run("AddErrors flattens nested errors", func(t *testing.T) {
		// Create container with nested errors
		parent := New(http.StatusBadRequest, "Parent error", nil)
		child1 := NewValidationError("validation.required", "email", "")
		child2 := NewValidationError("validation.invalid", "phone", "123")
		parent.AddError(child1)
		parent.AddError(child2)

		// Create another container with a nested error
		parent2 := New(http.StatusBadRequest, "Parent error 2", nil)
		child3 := NewValidationError("validation.min_length", "password", "123")
		parent2.AddError(child3)

		// Add both containers to a new parent using AddErrors
		newParent := New(http.StatusBadRequest, "New parent", nil)
		newParent.AddErrors([]Error{parent, parent2})

		// Should have flattened all errors
		errors := newParent.AllErrors()
		if len(errors) != 3 {
			t.Fatalf("Expected 3 flattened errors, got %d", len(errors))
		}

		// Should contain the original child errors, not the containers
		found := make(map[Error]bool)
		for _, err := range errors {
			found[err] = true
		}

		if !found[child1] || !found[child2] || !found[child3] {
			t.Error("All original child errors should be present after flattening")
		}
	})

	t.Run("AddErrors equivalent to multiple AddError calls", func(t *testing.T) {
		// Test using AddError multiple times
		container1 := New(http.StatusBadRequest, "Container 1", nil)
		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123")
		err3 := NewValidationError("validation.email", "email", "invalid")

		container1.AddError(err1)
		container1.AddError(err2)
		container1.AddError(err3)

		// Test using AddErrors
		container2 := New(http.StatusBadRequest, "Container 2", nil)
		container2.AddErrors([]Error{err1, err2, err3})

		// Both should have the same errors
		errors1 := container1.AllErrors()
		errors2 := container2.AllErrors()

		if len(errors1) != len(errors2) {
			t.Fatalf("Expected same number of errors, got %d vs %d", len(errors1), len(errors2))
		}

		// Check that all errors match
		found1 := make(map[Error]bool)
		for _, err := range errors1 {
			found1[err] = true
		}

		for _, err := range errors2 {
			if !found1[err] {
				t.Error("AddErrors should produce same result as multiple AddError calls")
			}
		}
	})

	t.Run("AddErrors preserves error order", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container error", nil)

		err1 := NewValidationError("validation.required", "field1", "")
		err2 := NewValidationError("validation.required", "field2", "")
		err3 := NewValidationError("validation.required", "field3", "")

		errors := []Error{err1, err2, err3}
		container.AddErrors(errors)

		// Check order is preserved (note: order depends on flattening behavior)
		allErrors := container.AllErrors()
		if len(allErrors) != 3 {
			t.Fatalf("Expected 3 errors, got %d", len(allErrors))
		}

		// Since these are validation errors without nested children,
		// they should be added in the same order
		if allErrors[0] != err1 || allErrors[1] != err2 || allErrors[2] != err3 {
			t.Error("Error order should be preserved when adding simple errors")
		}
	})
}

// =============================================================================
// Localization Tests
// =============================================================================

// TestLocalization tests the localization functionality
func TestLocalization(t *testing.T) {
	t.Run("Basic localization", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")
		msg := err.Error()

		if msg != "email is required" {
			t.Errorf("Expected 'email is required', got: %s", msg)
		}
	})

	t.Run("Localization with parameters", func(t *testing.T) {
		err := NewValidationError("validation.min_length", "password", "123").WithParam("min", 8)
		msg := err.Error()

		if !strings.Contains(msg, "8") {
			t.Errorf("Expected message to contain '8', got: %s", msg)
		}
	})

	t.Run("Custom localizer", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")

		// Test localization with English language tag
		msg := err.LocalizedError(language.English)

		if msg != "email is required" {
			t.Errorf("Expected localized message, got: %s", msg)
		}
	})

	t.Run("Fallback for missing message", func(t *testing.T) {
		// Test with a non-existent message key
		err := NewValidationError("validation.nonexistent", "field", "value")
		msg := err.Error()

		// Should fall back to some reasonable default
		if msg == "" {
			t.Error("Should have fallback message for missing keys")
		}
	})
}

// TestLocalizedErrMap tests the LocalizedErrMap functionality
func TestLocalizedErrMap(t *testing.T) {
	t.Run("Single error localized", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")
		errorMap := err.LocalizedErrMap(language.English)

		if errorMap == nil {
			t.Fatal("Expected error map, got nil")
		}

		if len(errorMap) != 1 {
			t.Fatalf("Expected 1 field in error map, got %d", len(errorMap))
		}

		if errors, ok := errorMap["email"]; !ok || len(errors) != 1 {
			t.Error("Expected one error for email field")
		}
	})

	t.Run("Multiple errors localized", func(t *testing.T) {
		container := New(http.StatusBadRequest, "Container", nil)
		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123").WithParam("min", 8)

		container.AddError(err1)
		container.AddError(err2)
		errorMap := container.LocalizedErrMap(language.English)

		if len(errorMap) != 2 {
			t.Fatalf("Expected 2 fields in error map, got %d", len(errorMap))
		}

		if _, ok := errorMap["email"]; !ok {
			t.Error("Expected email field in error map")
		}

		if _, ok := errorMap["password"]; !ok {
			t.Error("Expected password field in error map")
		}
	})

	t.Run("Custom language", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")

		errorMap := err.LocalizedErrMap(language.English)
		if errorMap == nil {
			t.Fatal("Expected error map, got nil")
		}
	})
}

// =============================================================================
// Helper Function Tests
// =============================================================================

// TestHelperFunctions tests Status, Message, Stack, and FormatStack helpers
func TestHelperFunctions(t *testing.T) {
	t.Run("Status helper", func(t *testing.T) {
		// Test with nil
		if Status(nil) != http.StatusOK {
			t.Fatalf("Status(nil) = %d, want %d", Status(nil), http.StatusOK)
		}

		// Test with erm error
		ermErr := New(http.StatusBadRequest, "test", nil)
		if Status(ermErr) != http.StatusBadRequest {
			t.Fatalf("Status(ermErr) = %d, want %d", Status(ermErr), http.StatusBadRequest)
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		if Status(stdErr) != http.StatusInternalServerError {
			t.Fatalf("Status(stdErr) = %d, want %d", Status(stdErr), http.StatusInternalServerError)
		}

		// Test with Error interface
		var e Error = &StackError{code: http.StatusNotFound}
		if Status(e) != http.StatusNotFound {
			t.Fatalf("Status(interface) = %d, want %d", Status(e), http.StatusNotFound)
		}
	})

	t.Run("Message helper", func(t *testing.T) {
		// Test with nil
		if Message(nil) != "" {
			t.Fatalf("Message(nil) = %q, want empty", Message(nil))
		}

		// Test with erm error with message
		ermErr := New(http.StatusBadRequest, "custom message", nil)
		if Message(ermErr) != "custom message" {
			t.Fatalf("Message(ermErr) = %q, want 'custom message'", Message(ermErr))
		}

		// Test with erm error with empty message
		emptyMsgErr := &StackError{code: http.StatusNotFound, msg: ""}
		if Message(emptyMsgErr) != "Not Found" {
			t.Fatalf("Message(empty) = %q, want 'Not Found'", Message(emptyMsgErr))
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		if Message(stdErr) != "Internal Server Error" {
			t.Fatalf("Message(stdErr) = %q, want 'Internal Server Error'", Message(stdErr))
		}
	})

	t.Run("Stack helper", func(t *testing.T) {
		// Test with nil
		if Stack(nil) != nil {
			t.Fatal("Stack(nil) should return nil")
		}

		// Test with erm error (500 has stack trace)
		ermErr := New(http.StatusInternalServerError, "test", nil)
		if len(Stack(ermErr)) == 0 {
			t.Fatal("Stack(ermErr) should not be empty for 500 errors")
		}

		// Test with erm error (400 has no stack trace)
		clientErr := New(http.StatusBadRequest, "test", nil)
		if len(Stack(clientErr)) != 0 {
			t.Fatal("Stack(clientErr) should be empty for client errors")
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		if Stack(stdErr) != nil {
			t.Fatal("Stack(stdErr) should return nil")
		}

		// Test with Error interface
		var e Error = &StackError{stack: []uintptr{1, 2, 3}}
		if len(Stack(e)) != 3 {
			t.Fatalf("Stack(interface) length = %d, want 3", len(Stack(e)))
		}
	})
}

// TestFormatStack tests FormatStack function for complete coverage
func TestFormatStack(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if FormatStack(nil) != "" {
			t.Fatal("FormatStack(nil) should return empty string")
		}
	})

	t.Run("error with empty stack", func(t *testing.T) {
		err := &StackError{stack: []uintptr{}}
		if FormatStack(err) != "" {
			t.Fatal("FormatStack with empty stack should return empty string")
		}
	})

	t.Run("error with stack", func(t *testing.T) {
		err := New(http.StatusInternalServerError, "test", errors.New("test"))
		formatted := FormatStack(err)

		if formatted == "" {
			t.Fatal("FormatStack should return non-empty string for 500 errors")
		}

		if !strings.Contains(formatted, ".go:") {
			t.Fatal("FormatStack should contain file and line info")
		}

		lines := strings.Split(formatted, "\n")
		if len(lines) < 2 {
			t.Fatal("FormatStack should contain multiple lines")
		}
	})

	t.Run("error with nil stack", func(t *testing.T) {
		err := &StackError{stack: nil}
		if FormatStack(err) != "" {
			t.Fatal("FormatStack with nil stack should return empty string")
		}
	})
}

// TestWrap tests the Wrap function comprehensively
func TestWrap(t *testing.T) {
	t.Run("wrap nil", func(t *testing.T) {
		if Wrap(nil) != nil {
			t.Fatal("Wrap(nil) should return nil")
		}
	})

	t.Run("wrap erm error", func(t *testing.T) {
		original := New(http.StatusNotFound, "not found", errors.New("root"))
		wrapped := Wrap(original)

		// Properties should be preserved
		if Status(wrapped) != Status(original) {
			t.Fatal("Wrap should preserve status")
		}
		if Message(wrapped) != Message(original) {
			t.Fatal("Wrap should preserve message")
		}

		// Error chain should be maintained
		if !errors.Is(wrapped, original) {
			t.Fatal("Wrap should maintain error chain")
		}
	})

	t.Run("wrap standard error", func(t *testing.T) {
		stdErr := errors.New("standard error")
		wrapped := Wrap(stdErr)

		if wrapped == stdErr {
			t.Fatal("Wrap should create new erm error for standard errors")
		}

		if _, ok := wrapped.(Error); !ok {
			t.Fatal("Wrap should return Error interface")
		}

		if Status(wrapped) != http.StatusInternalServerError {
			t.Fatalf("Wrapped standard error should have 500 status, got %d", Status(wrapped))
		}

		if !errors.Is(wrapped, stdErr) {
			t.Fatal("Wrapped error should maintain error chain")
		}
	})
}

// =============================================================================
// Convenience Constructor Tests
// =============================================================================

// TestConvenienceConstructors tests all convenience constructor functions
func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name      string
		fn        func(msg string, err error) Error
		expected  int
		wantStack bool // true if stack trace should be present
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest, false},
		{"Unauthorized", Unauthorized, http.StatusUnauthorized, false},
		{"Forbidden", Forbidden, http.StatusForbidden, false},
		{"Conflict", Conflict, http.StatusConflict, false},
		{"Internal", Internal, http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with error
			err := tt.fn("test message", errors.New("test"))

			if Status(err) != tt.expected {
				t.Fatalf("Status() = %d, want %d", Status(err), tt.expected)
			}

			if Message(err) != "test message" {
				t.Fatalf("Message() = %q, want %q", Message(err), "test message")
			}

			// Test stack trace presence based on status code
			hasStack := len(Stack(err)) > 0
			if hasStack != tt.wantStack {
				if tt.wantStack {
					t.Fatal("stack trace should be captured for Internal (500) errors")
				} else {
					t.Fatal("stack trace should not be captured for client (4xx) errors")
				}
			}

			// Test with nil error
			errNil := tt.fn("test message", nil)
			if se, ok := errNil.(*StackError); ok {
				// With current implementation, root can be nil when nil error is passed
				// but the message should still be preserved
				if se.msg != "test message" {
					t.Fatalf("Message should be preserved, got: %q", se.msg)
				}
			} else {
				t.Fatal("Constructor should return *StackError")
			}
		})
	}
}

// TestNotFound tests the NotFound convenience constructor
func TestNotFound(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		underlyingErr := errors.New("resource not found in database")
		err := NotFound("user", underlyingErr)

		// Test status code
		if Status(err) != http.StatusNotFound {
			t.Fatalf("Status() = %d, want %d", Status(err), http.StatusNotFound)
		}

		// Test that it has message key
		if se, ok := err.(*StackError); ok {
			if se.messageKey != "error.not_found" {
				t.Fatalf("MessageKey() = %q, want %q", se.messageKey, "error.not_found")
			}
			if se.fieldName != "user" {
				t.Fatalf("FieldName() = %q, want %q", se.fieldName, "user")
			}
		} else {
			t.Fatal("NotFound should return *StackError")
		}

		// Test that it doesn't capture stack trace (4xx error)
		if len(Stack(err)) > 0 {
			t.Fatal("NotFound should not capture stack trace for 4xx errors")
		}

		// Test underlying error is preserved
		if errors.Unwrap(err) != underlyingErr {
			t.Fatal("Underlying error should be preserved")
		}
	})

	t.Run("with nil error", func(t *testing.T) {
		err := NotFound("resource", nil)

		if Status(err) != http.StatusNotFound {
			t.Fatalf("Status() = %d, want %d", Status(err), http.StatusNotFound)
		}

		if errors.Unwrap(err) != nil {
			t.Fatal("Unwrap should return nil when created with nil error")
		}
	})
}

// =============================================================================
// Validation Error Constructor Tests
// =============================================================================

// TestValidationErrorConstructors tests the convenience constructors for validation errors
func TestValidationErrorConstructors(t *testing.T) {
	t.Run("RequiredError", func(t *testing.T) {
		err := RequiredError("email", "")
		if err.Error() != "email is required" {
			t.Fatalf("Error() = %q, want %q", err.Error(), "email is required")
		}
	})

	t.Run("MinLengthError", func(t *testing.T) {
		err := MinLengthError("password", "abc", 8)
		if err.Error() != "password must be at least 8 characters long" {
			t.Fatalf("Error() = %q, want expected message", err.Error())
		}
	})

	t.Run("MaxLengthError", func(t *testing.T) {
		err := MaxLengthError("description", "very long text", 100)
		if !strings.Contains(err.Error(), "at most 100") {
			t.Fatalf("Error() = %q, should contain 'at most 100'", err.Error())
		}
	})

	t.Run("EmailError", func(t *testing.T) {
		err := EmailError("email", "invalid-email")
		if err.Error() != "email must be a valid email address" {
			t.Fatalf("Error() = %q, want expected message", err.Error())
		}
	})

	t.Run("MinValueError", func(t *testing.T) {
		err := MinValueError("age", 16, 18)
		if !strings.Contains(err.Error(), "at least 18") {
			t.Fatalf("Error() = %q, should contain 'at least 18'", err.Error())
		}
	})

	t.Run("MaxValueError", func(t *testing.T) {
		err := MaxValueError("age", 150, 100)
		if !strings.Contains(err.Error(), "at most 100") {
			t.Fatalf("Error() = %q, should contain 'at most 100'", err.Error())
		}
	})

	t.Run("DuplicateError", func(t *testing.T) {
		err := DuplicateError("email", "user@example.com")
		if err.Error() != "email already exists, another record has the same value" {
			t.Fatalf("Error() = %q, want expected message", err.Error())
		}
		if err.MessageKey() != "validation.duplicate" {
			t.Fatalf("MessageKey() = %q, want 'validation.duplicate'", err.MessageKey())
		}
		if err.FieldName() != "email" {
			t.Fatalf("FieldName() = %q, want 'email'", err.FieldName())
		}
		if err.Value() != "user@example.com" {
			t.Fatalf("Value() = %v, want 'user@example.com'", err.Value())
		}
	})

	t.Run("InvalidError", func(t *testing.T) {
		err := InvalidError("format", "invalid-data")
		if err.Error() != "format value is invalid" {
			t.Fatalf("Error() = %q, want expected message", err.Error())
		}
		if err.MessageKey() != "validation.invalid" {
			t.Fatalf("MessageKey() = %q, want 'validation.invalid'", err.MessageKey())
		}
		if err.FieldName() != "format" {
			t.Fatalf("FieldName() = %q, want 'format'", err.FieldName())
		}
		if err.Value() != "invalid-data" {
			t.Fatalf("Value() = %v, want 'invalid-data'", err.Value())
		}
	})
}

// =============================================================================
// Edge Cases & Integration Tests
// =============================================================================

// TestNilHandling tests behavior with nil values
func TestNilHandling(t *testing.T) {
	var nilErr *StackError

	// All interface methods should handle nil gracefully
	if nilErr.Error() != "<nil>" {
		t.Errorf("Error() = %q, want %q", nilErr.Error(), "<nil>")
	}

	if nilErr.Code() != 0 {
		t.Errorf("Code() = %d, want %d", nilErr.Code(), 0)
	}

	if nilErr.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", nilErr.Unwrap())
	}

	if nilErr.Stack() != nil {
		t.Errorf("Stack() = %v, want nil", nilErr.Stack())
	}

	if nilErr.MessageKey() != "" {
		t.Errorf("MessageKey() = %q, want empty string", nilErr.MessageKey())
	}

	if nilErr.FieldName() != "" {
		t.Errorf("FieldName() = %q, want empty string", nilErr.FieldName())
	}

	if nilErr.Value() != nil {
		t.Errorf("Value() = %v, want nil", nilErr.Value())
	}

	if nilErr.Params() != nil {
		t.Errorf("Params() = %v, want nil", nilErr.Params())
	}

	// Builder methods should handle nil gracefully
	if nilErr.WithMessageKey("test") != nil {
		t.Errorf("WithMessageKey() should return nil for nil receiver")
	}

	if nilErr.WithFieldName("test") != nil {
		t.Errorf("WithFieldName() should return nil for nil receiver")
	}

	if nilErr.WithValue("test") != nil {
		t.Errorf("WithValue() should return nil for nil receiver")
	}

	if nilErr.WithParam("key", "value") != nil {
		t.Errorf("WithParam() should return nil for nil receiver")
	}

	// Error collection methods should handle nil gracefully
	if nilErr.HasErrors() {
		t.Error("HasErrors() should return false for nil receiver")
	}

	if nilErr.AllErrors() != nil {
		t.Error("AllErrors() should return nil for nil receiver")
	}

	if nilErr.ErrMap() != nil {
		t.Error("ErrMap() should return nil for nil receiver")
	}

	if nilErr.LocalizedError(language.English) != "<nil>" {
		t.Error("LocalizedError() should return '<nil>' for nil receiver")
	}

	if nilErr.LocalizedErrMap(language.English) != nil {
		t.Error("LocalizedErrMap() should return nil for nil receiver")
	}
}

// TestErrorChaining tests complex error scenarios
func TestErrorChaining(t *testing.T) {
	root := errors.New("root cause")
	layer1 := New(http.StatusNotFound, "not found", root)
	layer2 := Wrap(layer1)
	layer3 := Wrap(layer2)

	// Should preserve original metadata
	if Status(layer3) != http.StatusNotFound {
		t.Fatalf("Status() = %d, want %d", Status(layer3), http.StatusNotFound)
	}

	if Message(layer3) != "not found" {
		t.Fatalf("Message() = %q, want %q", Message(layer3), "not found")
	}

	// Should maintain error chain
	if !errors.Is(layer3, root) {
		t.Fatal("errors.Is should find root error")
	}

	if !errors.Is(layer3, layer1) {
		t.Fatal("errors.Is should find layer1 error")
	}
}

// TestEdgeCases tests remaining edge cases for comprehensive coverage
func TestEdgeCases(t *testing.T) {
	t.Run("error with empty message fallback", func(t *testing.T) {
		e := New(http.StatusBadRequest, "", errors.New("test"))
		msg := Message(e)
		if msg != "Bad Request" {
			t.Fatalf("Message() should return status text when msg is empty, got: %q", msg)
		}
	})

	t.Run("mixed error types", func(t *testing.T) {
		standardErr := errors.New("standard error")

		if Status(standardErr) != http.StatusInternalServerError {
			t.Fatalf("Status() = %d, want %d", Status(standardErr), http.StatusInternalServerError)
		}

		if Message(standardErr) != "Internal Server Error" {
			t.Fatalf("Message() = %q, want %q", Message(standardErr), "Internal Server Error")
		}

		if Stack(standardErr) != nil {
			t.Fatal("Stack() should return nil for non-erm errors")
		}
	})

	t.Run("goroutine context", func(t *testing.T) {
		// Test creating errors in goroutines
		done := make(chan Error, 1)
		go func() {
			done <- New(http.StatusInternalServerError, "goroutine error", nil)
		}()
		err := <-done
		if err == nil {
			t.Fatal("Goroutine should create valid error")
		}
		if len(err.Stack()) == 0 {
			t.Fatal("Goroutine error should have stack trace for 500 error")
		}
	})
}

// =============================================================================
// Test Helper Types and Functions
// =============================================================================

// Test helper types and functions
type testService struct{}

func (s testService) testMethod() Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}

func (s *testService) testPointerMethod() Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}

func simpleFunc() Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}

// MockLocale for testing
type MockLocale struct {
	messages map[string]string
}

func (m *MockLocale) GetMessage(code string) string {
	return m.messages[code]
}

func (m *MockLocale) Messages() map[string]string {
	return m.messages
}

// TestValidationErrorFunctions tests validation error creation methods
func TestValidationErrorFunctions(t *testing.T) {

	t.Run("NewValidationError_function", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")

		if err.Code() != 400 {
			t.Fatalf("Code() = %d, want 400", err.Code())
		}
		if err.FieldName() != "email" {
			t.Fatalf("FieldName() = %q, want 'email'", err.FieldName())
		}
		if err.Value() != "" {
			t.Fatalf("Value() = %v, want empty string", err.Value())
		}
		if err.MessageKey() != "validation.required" {
			t.Fatalf("MessageKey() = %q, want 'validation.required'", err.MessageKey())
		}
		if !strings.Contains(err.Error(), "email is required") {
			t.Fatalf("Error() = %q, should contain 'email is required'", err.Error())
		}
	})

	t.Run("ValidationError_with_custom_status", func(t *testing.T) {
		err := New(422, "", nil).
			WithMessageKey("validation.custom").
			WithFieldName("field").
			WithValue("value")

		if err.Code() != 422 {
			t.Fatalf("Code() = %d, want 422", err.Code())
		}
		if err.FieldName() != "field" {
			t.Fatalf("FieldName() = %q, want 'field'", err.FieldName())
		}
		if err.Value() != "value" {
			t.Fatalf("Value() = %v, want 'value'", err.Value())
		}
		if err.MessageKey() != "validation.custom" {
			t.Fatalf("MessageKey() = %q, want 'validation.custom'", err.MessageKey())
		}
		// Should show validation error for field when message not found
		if !strings.Contains(err.Error(), "validation error for field") {
			t.Fatalf("Error() = %q, should contain 'validation error for field'", err.Error())
		}
	})
}

// TestInternalFunctionEdgeCases improves coverage for internal functions
func TestInternalFunctionEdgeCases(t *testing.T) {

	t.Run("WithParam_edge_cases", func(t *testing.T) {
		err := NewValidationError("validation.min_length", "password", "123")

		// Test with nil value
		err2 := err.WithParam("min", nil)
		if err2 == nil {
			t.Fatal("WithParam should return non-nil error")
		}

		// Test with multiple params
		err3 := err.WithParam("min", 8).WithParam("max", 100).WithParam("min", 5) // Override existing
		if err3 == nil {
			t.Fatal("WithParam chaining should return non-nil error")
		}
		params := err3.Params()
		if params["min"] != 5 {
			t.Fatalf("Expected min param to be overridden to 5, got %v", params["min"])
		}
		if params["max"] != 100 {
			t.Fatalf("Expected max param to be 100, got %v", params["max"])
		}
	})

	t.Run("getFallbackMessage_edge_cases", func(t *testing.T) {
		// Test with empty message key
		err := NewValidationError("", "field", "value")
		msg := err.Error()
		if msg == "" {
			t.Fatal("Error() should return non-empty message even for empty message key")
		}

		// Test with very long message key - should show validation error format
		longKey := strings.Repeat("validation.very.long.key.", 10)
		err2 := NewValidationError(longKey, "field", "value")
		msg2 := err2.Error()
		if !strings.Contains(msg2, "validation error for field") {
			t.Fatalf("Error() should show validation error format, got %q", msg2)
		}
	})

	t.Run("LocalizedErrMap_edge_cases", func(t *testing.T) {
		// Test with empty errors
		emptyErr := New(400, "", nil)
		errMap := emptyErr.ErrMap()
		if len(errMap) != 0 {
			t.Fatalf("ErrMap() should be empty for error with no children, got %v", errMap)
		}

		// Test with errors that have empty field names
		err1 := NewValidationError("validation.required", "", "value")
		err2 := NewValidationError("validation.email", "", "invalid@")

		container := New(400, "", nil)
		container.AddError(err1)
		container.AddError(err2)

		errMap = container.ErrMap()
		if len(errMap) == 0 {
			t.Fatal("ErrMap() should not be empty when container has errors")
		}

		// Errors with empty field names should be grouped under "error" key
		if errorMsgs, exists := errMap["error"]; exists {
			if len(errorMsgs) != 2 {
				t.Fatalf("Expected 2 error messages under 'error' key, got %d", len(errorMsgs))
			}
		}
	})

	t.Run("formatChildErrors_edge_cases", func(t *testing.T) {
		// Create error with many child errors to test different formatting paths
		container := New(400, "", nil)

		// Add multiple errors to test formatMultipleErrors path
		for i := 0; i < 5; i++ {
			fieldName := fmt.Sprintf("field%d", i)
			container.AddError(RequiredError(fieldName, ""))
		}

		msg := container.Error()
		if !strings.Contains(msg, "multiple errors:") {
			t.Fatalf("Error() with multiple children should contain 'multiple errors:', got %q", msg)
		}

		// Test single child error
		singleContainer := New(400, "", nil)
		singleContainer.AddError(RequiredError("email", ""))
		singleMsg := singleContainer.Error()
		if !strings.Contains(singleMsg, "email is required") {
			t.Fatalf("Error() with single child should contain child message, got %q", singleMsg)
		}
	})
}
