// Package erm provides localization and internationalization utilities
// using the custom github.com/c3p0-box/utils/i18n package.
//
// This file contains the localization infrastructure for the ERM error
// management system, providing on-demand message resolution and
// per-language translation management.
package erm

import (
	"sync"

	"github.com/c3p0-box/utils/i18n"
	"golang.org/x/text/language"
)

// Global internationalization state
var (
	// initOnce ensures messages are only initialized once
	initOnce sync.Once
)

func init() {
	// Initialize all translation messages on package load
	initOnce.Do(initializeMessages)
}

// GetLocalizer returns a localizer for the specified language.
// This function maintains backward compatibility with the old API but now
// uses our custom i18n package internally.
// For unsupported languages, it will fall back to English messages.
// It's safe to call this method concurrently.
func GetLocalizer(tag language.Tag) *Localizer {
	return &Localizer{language: tag}
}

// Localizer is a compatibility wrapper that provides the same API as go-i18n's Localizer
// but uses our custom i18n package internally.
type Localizer struct {
	language language.Tag
}

// Localize translates a message using our custom i18n package.
// It maintains the same API as go-i18n's Localizer.Localize method.
func (l *Localizer) Localize(config *LocalizeConfig) (string, error) {
	if config == nil {
		return "", nil
	}

	result := i18n.Translate(l.language, config.MessageID, 1, config.TemplateData)
	if result == config.MessageID {
		// Translation not found - return empty string to match go-i18n behavior
		return "", nil
	}
	return result, nil
}

// MustLocalize translates a message and returns the result or the message ID if translation fails.
// It maintains the same API as go-i18n's Localizer.MustLocalize method.
func (l *Localizer) MustLocalize(config *LocalizeConfig) string {
	if config == nil {
		return ""
	}

	return i18n.Translate(l.language, config.MessageID, 1, config.TemplateData)
}

// LocalizeConfig provides the configuration for message localization.
// It maintains the same structure as go-i18n's LocalizeConfig.
type LocalizeConfig struct {
	MessageID    string
	TemplateData interface{}
}

// initializeMessages adds all standard validation messages to our custom i18n package
func initializeMessages() {
	// Set English as the default language
	i18n.SetDefaultLanguage(language.English)

	// Add all English validation messages
	messages := map[string]*i18n.Translation{
		"validation.required": {
			Singular: "{{.field}} is required",
			Plural:   "",
		},
		"validation.min_length": {
			Singular: "{{.field}} must be at least {{.min}} characters long",
			Plural:   "",
		},
		"validation.max_length": {
			Singular: "{{.field}} must be at most {{.max}} characters long",
			Plural:   "",
		},
		"validation.email": {
			Singular: "{{.field}} must be a valid email address",
			Plural:   "",
		},
		"validation.min_value": {
			Singular: "{{.field}} must be at least {{.min}}",
			Plural:   "",
		},
		"validation.max_value": {
			Singular: "{{.field}} must be at most {{.max}}",
			Plural:   "",
		},
		"validation.invalid": {
			Singular: "{{.field}} value is invalid",
			Plural:   "",
		},
		"validation.empty": {
			Singular: "{{.field}} must be empty",
			Plural:   "",
		},
		"validation.not_empty": {
			Singular: "{{.field}} must not be empty",
			Plural:   "",
		},
		"validation.duplicate": {
			Singular: "{{.field}} already exists, another record has the same value",
			Plural:   "",
		},
		"error.multiple": {
			Singular: "multiple errors: {{.errors}}",
			Plural:   "",
		},
		"error.not_found": {
			Singular: "{{.field}} is not found",
			Plural:   "",
		},
		"error.unknown": {
			Singular: "unknown error",
			Plural:   "",
		},
	}

	// Add all messages to the i18n package for English
	err := i18n.AddTranslations(language.English, messages)
	if err != nil {
		// This should never happen with valid messages, but handle gracefully
		panic("Failed to initialize ERM validation messages: " + err.Error())
	}
}
