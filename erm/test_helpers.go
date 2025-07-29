// Package erm provides test helpers for both ERM and VIX packages.
// This file contains shared test utilities to eliminate duplication
// and ensure consistent test setup across packages.
package erm

import (
	"strings"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// TestHelper provides common test utilities for ERM and VIX packages.
type TestHelper struct {
	t *testing.T
}

// NewTestHelper creates a new test helper.
// The new GetLocalizer system handles initialization automatically,
// so no explicit localizer setup is required.
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{
		t: t,
	}
}

// GetEnglishLocalizer returns the English localizer for testing.
func (h *TestHelper) GetEnglishLocalizer() *i18n.Localizer {
	return GetLocalizer(language.English)
}

// GetSpanishLocalizer returns a Spanish localizer for testing.
// Note: This currently returns a localizer with English fallback messages
// since we don't have Spanish translations in the default bundles yet.
func (h *TestHelper) GetSpanishLocalizer() *i18n.Localizer {
	return GetLocalizer(language.Spanish)
}

// CreateCustomSpanishLocalizer creates a Spanish localizer with custom Spanish messages for testing.
// This is useful for testing actual Spanish translations.
func (h *TestHelper) CreateCustomSpanishLocalizer() *i18n.Localizer {
	spanishBundle := i18n.NewBundle(language.Spanish)
	spanishBundle.AddMessages(language.Spanish, &i18n.Message{
		ID:    "validation.required",
		Other: "{{.field}} es requerido",
	})
	spanishBundle.AddMessages(language.Spanish, &i18n.Message{
		ID:    "validation.min_length",
		Other: "{{.field}} debe tener al menos {{.min}} caracteres",
	})
	spanishBundle.AddMessages(language.Spanish, &i18n.Message{
		ID:    "validation.email",
		Other: "{{.field}} debe ser una dirección de email válida",
	})

	return i18n.NewLocalizer(spanishBundle, "es")
}

// AssertErrorContains checks that an error contains the expected text.
// This eliminates duplicate error checking patterns in tests.
func (h *TestHelper) AssertErrorContains(err error, expectedText string) {
	h.t.Helper()
	if err == nil {
		h.t.Fatalf("expected error containing %q, but got nil", expectedText)
	}
	if !strings.Contains(err.Error(), expectedText) {
		h.t.Fatalf("expected error containing %q, got %q", expectedText, err.Error())
	}
}

// AssertNoError checks that no error occurred.
func (h *TestHelper) AssertNoError(err error) {
	h.t.Helper()
	if err != nil {
		h.t.Fatalf("expected no error, got: %v", err)
	}
}

// AssertErrorEquals checks that the error message exactly matches expected.
func (h *TestHelper) AssertErrorEquals(err error, expected string) {
	h.t.Helper()
	if err == nil {
		h.t.Fatalf("expected error %q, but got nil", expected)
	}
	if err.Error() != expected {
		h.t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

// Package-level convenience functions

// SetupTestLocalizer is a deprecated function maintained for backward compatibility.
// The new GetLocalizer system handles initialization automatically.
// Deprecated: No setup is required with the new GetLocalizer system.
func SetupTestLocalizer() {
	// No-op: GetLocalizer system handles initialization automatically
}
