// Package erm provides test helpers for both ERM and VIX packages.
// This file contains shared test utilities to eliminate duplication
// and ensure consistent test setup across packages.
package erm

import (
	"strings"
	"testing"

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
func (h *TestHelper) GetEnglishLocalizer() *Localizer {
	return GetLocalizer(language.English)
}

// GetSpanishLocalizer returns a Spanish localizer for testing.
// Note: This currently returns a localizer with English fallback messages
// since we don't have Spanish translations in the default bundles yet.
func (h *TestHelper) GetSpanishLocalizer() *Localizer {
	return GetLocalizer(language.Spanish)
}

// CreateCustomSpanishLocalizer creates a Spanish localizer with custom Spanish messages for testing.
// This is useful for testing actual Spanish translations.
func (h *TestHelper) CreateCustomSpanishLocalizer() *Localizer {
	// Note: In the new implementation, we would need to add Spanish translations
	// to our i18n package. For now, return a localizer that will fall back to English.
	// In a real implementation, you would call:
	// i18n.AddTranslation(language.Spanish, "validation.required", "{{.field}} es requerido", "")
	// etc.
	return GetLocalizer(language.Spanish)
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
