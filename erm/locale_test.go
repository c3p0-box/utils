package erm

import (
	"testing"

	"golang.org/x/text/language"
)

// TestGetLocalizer tests the new GetLocalizer functionality
func TestGetLocalizer(t *testing.T) {
	t.Run("GetLocalizer with English", func(t *testing.T) {
		localizer := GetLocalizer(language.English)
		if localizer == nil {
			t.Fatal("GetLocalizer(language.English) should return a non-nil localizer")
		}

		// Test that we can use it for localization
		err := NewValidationError("validation.required", "email", "")
		localizedMsg := err.LocalizedError(language.English)
		expected := "email is required"
		if localizedMsg != expected {
			t.Errorf("Expected %q, got %q", expected, localizedMsg)
		}
	})

	t.Run("GetLocalizer with same language returns same localizer", func(t *testing.T) {
		localizer1 := GetLocalizer(language.English)
		localizer2 := GetLocalizer(language.English)

		if localizer1 != localizer2 {
			t.Error("GetLocalizer should return the same localizer for the same language")
		}
	})

	t.Run("GetLocalizer with different languages", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")

		// Both should work since we include English messages in all bundles
		englishMsg := err.LocalizedError(language.English)
		spanishMsg := err.LocalizedError(language.Spanish)

		// Both should return English messages since we only have English messages loaded
		if englishMsg != "email is required" {
			t.Errorf("English localization failed: %q", englishMsg)
		}
		if spanishMsg != "email is required" {
			t.Errorf("Spanish localization should fallback to English: %q", spanishMsg)
		}
	})

	t.Run("Concurrent access", func(t *testing.T) {
		done := make(chan bool, 10)

		// Test concurrent access to GetLocalizer
		for i := 0; i < 10; i++ {
			go func() {
				localizer := GetLocalizer(language.English)
				if localizer == nil {
					t.Error("GetLocalizer should not return nil")
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestLocalizationWithNewAPI tests localization using the new language.Tag API
func TestLocalizationWithNewAPI(t *testing.T) {
	t.Run("Basic validation error localization", func(t *testing.T) {
		err := NewValidationError("validation.required", "username", "")
		result := err.LocalizedError(language.English)
		expected := "username is required"

		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("Multiple validation errors", func(t *testing.T) {
		container := New(400, "Validation failed", nil)

		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123").WithParam("min", 8)

		container.AddError(err1)
		container.AddError(err2)

		localizedMsg := container.LocalizedError(language.Spanish) // Should fallback to English
		if localizedMsg == "" {
			t.Error("Localized message should not be empty")
		}
	})

	t.Run("Different language tags", func(t *testing.T) {
		err := NewValidationError("validation.email", "email", "invalid-email")
		result := err.LocalizedError(language.French) // Should fallback to English
		expected := "email must be a valid email address"

		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
}

// TestLocalizationErrorMap tests the LocalizedErrMap functionality with language tags
func TestLocalizationErrorMap(t *testing.T) {
	t.Run("Single validation error", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")
		errorMap := err.LocalizedErrMap(language.English)

		if errorMap == nil {
			t.Fatal("Expected error map, got nil")
		}

		if len(errorMap) != 1 {
			t.Fatalf("Expected 1 field in error map, got %d", len(errorMap))
		}

		if emailErrors, exists := errorMap["email"]; !exists || len(emailErrors) != 1 {
			t.Error("Expected email field with 1 error")
		} else if emailErrors[0] != "email is required" {
			t.Errorf("Expected 'email is required', got %q", emailErrors[0])
		}
	})

	t.Run("Multiple validation errors", func(t *testing.T) {
		container := New(400, "Validation failed", nil)

		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123").WithParam("min", 8)

		container.AddError(err1)
		container.AddError(err2)

		errorMap := container.LocalizedErrMap(language.English)

		if len(errorMap) != 2 {
			t.Fatalf("Expected 2 fields in error map, got %d", len(errorMap))
		}

		if emailErrors, exists := errorMap["email"]; !exists || len(emailErrors) != 1 {
			t.Error("Expected email field with 1 error")
		}

		if passwordErrors, exists := errorMap["password"]; !exists || len(passwordErrors) != 1 {
			t.Error("Expected password field with 1 error")
		} else if passwordErrors[0] != "password must be at least 8 characters long" {
			t.Errorf("Expected password length error, got %q", passwordErrors[0])
		}
	})

	t.Run("Error without field name", func(t *testing.T) {
		err := NewValidationError("validation.invalid", "", "somevalue")
		errorMap := err.LocalizedErrMap(language.English)

		if errorMap == nil {
			t.Fatal("Expected error map, got nil")
		}

		if errorErrors, exists := errorMap["error"]; !exists || len(errorErrors) != 1 {
			t.Error("Expected error field with 1 error for validation without field name")
		}
	})
}

// TestConcurrentLocalization tests concurrent access to localization methods
func TestConcurrentLocalization(t *testing.T) {
	err := NewValidationError("validation.required", "email", "")
	done := make(chan bool, 20)

	// Test concurrent LocalizedError calls
	for i := 0; i < 10; i++ {
		go func() {
			msg := err.LocalizedError(language.English)
			if msg != "email is required" {
				t.Error("Concurrent LocalizedError failed")
			}
			done <- true
		}()
	}

	// Test concurrent LocalizedErrMap calls
	for i := 0; i < 10; i++ {
		go func() {
			errorMap := err.LocalizedErrMap(language.English)
			if errorMap == nil || len(errorMap) != 1 {
				t.Error("Concurrent LocalizedErrMap failed")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}
