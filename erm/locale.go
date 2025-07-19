// Package erm provides localization and internationalization utilities
// using the standard github.com/nicksnyder/go-i18n/v2/i18n package.
//
// This file contains the localization infrastructure for the ERM error
// management system, providing on-demand message resolution and
// global localizer management.
package erm

import (
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// Global internationalization state
var (
	// defaultLocalizer is used by Error() and ToError() methods when no specific localizer is provided
	defaultLocalizer *i18n.Localizer
	// localizerMutex protects concurrent access to defaultLocalizer
	localizerMutex sync.RWMutex
)

// SetLocalizer sets the global default localizer used by Error() and ToError() methods.
// This localizer will be used when no specific localizer is provided to localization methods.
// It's safe to call this method concurrently.
func SetLocalizer(localizer *i18n.Localizer) {
	localizerMutex.Lock()
	defer localizerMutex.Unlock()
	defaultLocalizer = localizer
}

// GetLocalizer returns the current global default localizer.
// Returns nil if no localizer has been set.
// It's safe to call this method concurrently.
func GetLocalizer() *i18n.Localizer {
	localizerMutex.RLock()
	defer localizerMutex.RUnlock()
	return defaultLocalizer
}

// CreateDefaultBundle creates a default i18n bundle with English as the default language.
// This is a convenience function for basic setup. For advanced usage, create your own bundle
// and load message files as needed.
func CreateDefaultBundle() *i18n.Bundle {
	bundle := i18n.NewBundle(language.English)

	// Add default English messages for common validation errors
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.required",
		Other: "{{.field}} is required",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.min_length",
		Other: "{{.field}} must be at least {{.min}} characters long",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.max_length",
		Other: "{{.field}} must be at most {{.max}} characters long",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.email",
		Other: "{{.field}} must be a valid email address",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.min_value",
		Other: "{{.field}} must be at least {{.min}}",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.max_value",
		Other: "{{.field}} must be at most {{.max}}",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.invalid",
		Other: "{{.field}} is invalid",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.empty",
		Other: "{{.field}} must be empty",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "validation.not_empty",
		Other: "{{.field}} must not be empty",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "error.multiple",
		Other: "multiple errors: {{.errors}}",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "error.unknown",
		Other: "unknown error",
	})

	return bundle
}

// InitializeDefaultLocalizer creates and sets a default English localizer.
// This is a convenience function for quick setup. Call this during application initialization
// if you want basic English error messages without setting up your own bundle.
func InitializeDefaultLocalizer() {
	bundle := CreateDefaultBundle()
	localizer := i18n.NewLocalizer(bundle, language.English.String())
	SetLocalizer(localizer)
}
