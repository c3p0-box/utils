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
	t         *testing.T
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
}

// NewTestHelper creates a new test helper with default English localizer setup.
// This eliminates duplicate localizer setup code across test files.
func NewTestHelper(t *testing.T) *TestHelper {
	bundle := CreateDefaultBundle()
	localizer := i18n.NewLocalizer(bundle, "en")
	SetLocalizer(localizer)

	return &TestHelper{
		t:         t,
		bundle:    bundle,
		localizer: localizer,
	}
}

// SetupLocalizer ensures the default English localizer is configured.
// This is a convenience method that can be called from any test.
func (h *TestHelper) SetupLocalizer() {
	SetLocalizer(h.localizer)
}

// CreateSpanishLocalizer creates a Spanish localizer for testing localization.
func (h *TestHelper) CreateSpanishLocalizer() *i18n.Localizer {
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

// SetupTestLocalizer is a package-level convenience function for quick test setup.
// This can be called from any test file to ensure proper localizer configuration.
func SetupTestLocalizer() {
	bundle := CreateDefaultBundle()
	localizer := i18n.NewLocalizer(bundle, "en")
	SetLocalizer(localizer)
}
