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
		localizedMsg := err.LocalizedError(localizer)
		expected := "email is required"
		if localizedMsg != expected {
			t.Errorf("Expected %q, got %q", expected, localizedMsg)
		}
	})

	t.Run("GetLocalizer with same language returns same instance", func(t *testing.T) {
		localizer1 := GetLocalizer(language.English)
		localizer2 := GetLocalizer(language.English)

		if localizer1 != localizer2 {
			t.Error("GetLocalizer should return the same instance for the same language")
		}
	})

	t.Run("GetLocalizer with different languages", func(t *testing.T) {
		englishLocalizer := GetLocalizer(language.English)
		spanishLocalizer := GetLocalizer(language.Spanish)

		if englishLocalizer == spanishLocalizer {
			t.Error("GetLocalizer should return different instances for different languages")
		}

		// Both should work for validation (Spanish falls back to English messages)
		err := NewValidationError("validation.required", "email", "")

		englishMsg := err.LocalizedError(englishLocalizer)
		spanishMsg := err.LocalizedError(spanishLocalizer)

		// For now, both should return English since we only have English messages
		expected := "email is required"
		if englishMsg != expected {
			t.Errorf("English localizer: expected %q, got %q", expected, englishMsg)
		}
		if spanishMsg != expected {
			t.Errorf("Spanish localizer: expected %q, got %q", expected, spanishMsg)
		}
	})

	t.Run("GetLocalizer concurrent access", func(t *testing.T) {
		// Test concurrent access to ensure thread safety
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				localizer := GetLocalizer(language.English)
				if localizer == nil {
					t.Error("GetLocalizer should not return nil")
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestLocalizationWithNewAPI tests the new localization API
func TestLocalizationWithNewAPI(t *testing.T) {
	t.Run("Basic localization with default English", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")
		expected := "email is required"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Localization with parameters", func(t *testing.T) {
		err := NewValidationError("validation.min_length", "password", "123")
		err = err.WithParam("min", 8)

		localizer := GetLocalizer(language.English)
		result := err.LocalizedError(localizer)
		expected := "password must be at least 8 characters long"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("Custom localizer with different language", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")

		// Get Spanish localizer (will fallback to English for now)
		spanishLocalizer := GetLocalizer(language.Spanish)
		localizedMsg := err.LocalizedError(spanishLocalizer)

		// Since we only have English messages, it should return English
		expected := "email is required"
		if localizedMsg != expected {
			t.Errorf("Expected %q, got %q", expected, localizedMsg)
		}
	})

	t.Run("Fallback for missing message", func(t *testing.T) {
		err := NewValidationError("nonexistent.message", "field", "value")
		localizer := GetLocalizer(language.English)
		result := err.LocalizedError(localizer)

		// Should fallback to a default message
		if result == "" {
			t.Error("Should not return empty string for missing message")
		}
	})
}

// TestErrorMapLocalization tests error map localization with new API
func TestErrorMapLocalization(t *testing.T) {
	t.Run("Single error localized", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")
		errorMap := err.LocalizedErrMap(GetLocalizer(language.English))

		if errorMap == nil {
			t.Fatal("Error map should not be nil")
		}

		if len(errorMap) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(errorMap))
		}

		emailErrors, exists := errorMap["email"]
		if !exists {
			t.Fatal("Expected 'email' key in error map")
		}

		if len(emailErrors) != 1 {
			t.Fatalf("Expected 1 error for email, got %d", len(emailErrors))
		}

		expected := "email is required"
		if emailErrors[0] != expected {
			t.Errorf("Expected %q, got %q", expected, emailErrors[0])
		}
	})

	t.Run("Multiple errors localized", func(t *testing.T) {
		container := New(400, "Validation errors", nil)
		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.required", "password", "")

		container.AddError(err1)
		container.AddError(err2)
		errorMap := container.LocalizedErrMap(GetLocalizer(language.English))

		if len(errorMap) != 2 {
			t.Fatalf("Expected 2 errors, got %d", len(errorMap))
		}

		// Check both errors are present
		if _, exists := errorMap["email"]; !exists {
			t.Error("Expected 'email' key in error map")
		}
		if _, exists := errorMap["password"]; !exists {
			t.Error("Expected 'password' key in error map")
		}
	})

	t.Run("ErrMap convenience method", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")
		errorMap := err.ErrMap() // Uses default English localizer

		if len(errorMap) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(errorMap))
		}

		emailErrors := errorMap["email"]
		expected := "email is required"
		if emailErrors[0] != expected {
			t.Errorf("Expected %q, got %q", expected, emailErrors[0])
		}
	})
}
