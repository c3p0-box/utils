// Package erm provides localization and internationalization utilities
// using the standard github.com/nicksnyder/go-i18n/v2/i18n package.
//
// This file contains the localization infrastructure for the ERM error
// management system, providing on-demand message resolution and
// per-language localizer management.
package erm

import (
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// Global internationalization state
var (
	// localizers stores per-language localizers, created on demand
	localizers map[language.Tag]*i18n.Localizer
	// localizerMutex protects concurrent access to localizers map
	localizerMutex sync.RWMutex
)

func init() {
	localizers = make(map[language.Tag]*i18n.Localizer)
}

// GetLocalizer returns a localizer for the specified language.
// If no localizer exists for the language, it creates one with English messages as fallback.
// For unsupported languages, it creates a bundle with English messages.
// It's safe to call this method concurrently.
func GetLocalizer(tag language.Tag) *i18n.Localizer {
	// First try to get existing localizer with read lock
	localizerMutex.RLock()
	if localizer, exists := localizers[tag]; exists {
		localizerMutex.RUnlock()
		return localizer
	}
	localizerMutex.RUnlock()

	// Need to create new localizer, acquire write lock
	localizerMutex.Lock()
	defer localizerMutex.Unlock()

	// Double-check in case another goroutine created it while we were waiting
	if localizer, exists := localizers[tag]; exists {
		return localizer
	}

	// Create bundle for the requested language with English messages as fallback
	bundle := createBundleForLanguage(tag)
	localizer := i18n.NewLocalizer(bundle, tag.String(), language.English.String())
	localizers[tag] = localizer
	return localizer
}

// createBundleForLanguage creates a bundle for the specified language.
// Currently, all bundles contain English messages. In the future, this can be
// extended to load language-specific message files.
func createBundleForLanguage(tag language.Tag) *i18n.Bundle {
	// Always use English as the bundle's default language to ensure proper fallback
	// The localizer will handle language preference and fallback
	bundle := i18n.NewBundle(language.English)

	// Add English messages to all bundles
	addEnglishMessages(bundle)

	// TODO: In the future, we can load language-specific messages here
	// For example:
	// if tag == language.Spanish {
	//     addSpanishMessages(bundle)
	// }

	return bundle
}

// addEnglishMessages adds standard English validation messages to a bundle
func addEnglishMessages(bundle *i18n.Bundle) {
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
		Other: "{{.field}} value is invalid",
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
		ID:    "validation.duplicate",
		Other: "{{.field}} already exists, another record has the same value",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "error.multiple",
		Other: "multiple errors: {{.errors}}",
	})
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "error.unknown",
		Other: "unknown error",
	})
}
