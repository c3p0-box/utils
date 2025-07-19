package erm

import (
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// TestLocaleInternationalization tests the i18n localization capabilities
func TestLocaleInternationalization(t *testing.T) {
	// Set up a test localizer
	bundle := CreateDefaultBundle()
	localizer := i18n.NewLocalizer(bundle, "en")
	SetLocalizer(localizer)

	t.Run("Basic localization", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")
		expected := "email is required"
		if err.Error() != expected {
			t.Fatalf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Localization with parameters", func(t *testing.T) {
		err := NewValidationError("validation.min_length", "password", "123").
			WithParam("min", 8)

		expected := "password must be at least 8 characters long"
		if err.Error() != expected {
			t.Fatalf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Custom localizer", func(t *testing.T) {
		// Create a Spanish bundle for testing
		spanishBundle := i18n.NewBundle(language.Spanish)
		spanishBundle.AddMessages(language.Spanish, &i18n.Message{
			ID:    "validation.required",
			Other: "{{.field}} es requerido",
		})

		spanishLocalizer := i18n.NewLocalizer(spanishBundle, "es")

		err := NewValidationError("validation.required", "email", "")
		spanishMsg := err.LocalizedError(spanishLocalizer)

		expected := "email es requerido"
		if spanishMsg != expected {
			t.Fatalf("LocalizedError() = %q, want %q", spanishMsg, expected)
		}
	})

	t.Run("Fallback for missing message", func(t *testing.T) {
		err := NewValidationError("validation.nonexistent", "field", "value")

		// Should fall back to default error message formatting
		errorMsg := err.Error()
		if errorMsg == "" {
			t.Fatal("Error() should return some fallback message for missing keys")
		}
	})
}

// TestLocalizationHelpers tests the locale helper functions
func TestLocalizationHelpers(t *testing.T) {
	t.Run("SetLocalizer and GetLocalizer", func(t *testing.T) {
		// Create a test localizer
		bundle := CreateDefaultBundle()
		localizer := i18n.NewLocalizer(bundle, "en")

		// Set it
		SetLocalizer(localizer)

		// Get it back
		retrieved := GetLocalizer()
		if retrieved != localizer {
			t.Error("GetLocalizer should return the same localizer that was set")
		}
	})

	t.Run("CreateDefaultBundle", func(t *testing.T) {
		bundle := CreateDefaultBundle()
		if bundle == nil {
			t.Fatal("CreateDefaultBundle should return a non-nil bundle")
		}

		// Test that it includes required messages
		localizer := i18n.NewLocalizer(bundle, "en")
		msg, err := localizer.Localize(&i18n.LocalizeConfig{
			MessageID: "validation.required",
			TemplateData: map[string]interface{}{
				"field": "test",
			},
		})
		if err != nil {
			t.Fatalf("Default bundle should contain validation.required message: %v", err)
		}
		if msg != "test is required" {
			t.Fatalf("Expected 'test is required', got: %q", msg)
		}
	})

	t.Run("InitializeDefaultLocalizer", func(t *testing.T) {
		// Clear current localizer
		SetLocalizer(nil)

		// Initialize default
		InitializeDefaultLocalizer()

		// Check that it was set
		localizer := GetLocalizer()
		if localizer == nil {
			t.Fatal("InitializeDefaultLocalizer should set a non-nil localizer")
		}

		// Test that it works
		err := NewValidationError("validation.required", "email", "")
		if err.Error() != "email is required" {
			t.Fatalf("Expected localized message, got: %q", err.Error())
		}
	})
}

// TestLocalizationConcurrency tests concurrent access to localizer
func TestLocalizationConcurrency(t *testing.T) {
	bundle := CreateDefaultBundle()
	localizer := i18n.NewLocalizer(bundle, "en")

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			SetLocalizer(localizer)
			retrieved := GetLocalizer()
			if retrieved == nil {
				t.Error("GetLocalizer should not return nil")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestLocalizedErrMap tests the LocalizedErrMap method
func TestLocaleLocalizedErrMap(t *testing.T) {
	// Set up a test localizer
	bundle := CreateDefaultBundle()
	localizer := i18n.NewLocalizer(bundle, "en")
	SetLocalizer(localizer)

	t.Run("Single error localized", func(t *testing.T) {
		err := NewValidationError("validation.required", "email", "")
		errorMap := err.LocalizedErrMap(localizer)

		if errorMap == nil {
			t.Fatal("LocalizedErrMap should return error map")
		}

		if emailErrors, ok := errorMap["email"]; !ok || len(emailErrors) != 1 {
			t.Error("Should have one error for email field")
		} else if emailErrors[0] != "email is required" {
			t.Errorf("Expected 'email is required', got: %q", emailErrors[0])
		}
	})

	t.Run("Multiple errors localized", func(t *testing.T) {
		container := New(400, "Container error", nil)
		err1 := NewValidationError("validation.required", "email", "")
		err2 := NewValidationError("validation.min_length", "password", "123").WithParam("min", 8)

		container.AddError(err1)
		container.AddError(err2)
		errorMap := container.LocalizedErrMap(localizer)

		if errorMap == nil {
			t.Fatal("LocalizedErrMap should return error map")
		}

		if emailErrors, ok := errorMap["email"]; !ok || len(emailErrors) != 1 {
			t.Error("Should have one error for email field")
		}

		if passwordErrors, ok := errorMap["password"]; !ok || len(passwordErrors) != 1 {
			t.Error("Should have one error for password field")
		}
	})

	t.Run("Custom localizer", func(t *testing.T) {
		// Create Spanish localizer
		spanishBundle := i18n.NewBundle(language.Spanish)
		spanishBundle.AddMessages(language.Spanish, &i18n.Message{
			ID:    "validation.required",
			Other: "{{.field}} es requerido",
		})
		spanishLocalizer := i18n.NewLocalizer(spanishBundle, "es")

		err := NewValidationError("validation.required", "email", "")
		errorMap := err.LocalizedErrMap(spanishLocalizer)

		if errorMap == nil {
			t.Fatal("LocalizedErrMap should return error map")
		}

		if emailErrors, ok := errorMap["email"]; !ok || len(emailErrors) != 1 {
			t.Error("Should have one error for email field")
		} else if emailErrors[0] != "email es requerido" {
			t.Errorf("Expected 'email es requerido', got: %q", emailErrors[0])
		}
	})
}
