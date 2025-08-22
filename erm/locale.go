// Package erm provides localization and internationalization utilities
// using the custom github.com/c3p0-box/utils/i18n package.
//
// This file contains the localization infrastructure for the ERM error
// management system, providing on-demand message resolution and
// per-language translation management.
//
// # Message Constants
//
// This package exports validation message key constants that are used
// throughout the validation system. These constants provide type safety
// and avoid hardcoded strings in validation logic.
//
// The constants are organized into three categories:
//   - Standard validation messages (e.g., MsgRequired, MsgEmail)
//   - Negated validation messages (e.g., MsgNotRequired, MsgNotEmail)
//   - Special validation messages (e.g., MsgMustBeZero, MsgDivisorZero)
//
// All constants map to actual translation keys defined in initializeMessages()
// and can be extended to support additional languages by calling
// i18n.AddTranslations() with the appropriate translations.
package erm

import (
	"sync"

	"github.com/c3p0-box/utils/i18n"
	"golang.org/x/text/language"
)

// Validation message key constants for localized error messages.
// These constants map to translation keys defined in initializeMessages().
// They provide type safety and avoid hardcoded strings throughout the validation system.
const (
	MsgRequired    = "validation.required"
	MsgEmpty       = "validation.empty"
	MsgMinLength   = "validation.min_length"
	MsgMaxLength   = "validation.max_length"
	MsgExactLength = "validation.exact_length"

	MsgEmail        = "validation.email"
	MsgURL          = "validation.url"
	MsgNumeric      = "validation.numeric"
	MsgAlpha        = "validation.alpha"
	MsgAlphaNumeric = "validation.alpha_numeric"
	MsgRegex        = "validation.regex"
	MsgIn           = "validation.in"
	MsgNotIn        = "validation.not_in"
	MsgContains     = "validation.contains"
	MsgStartsWith   = "validation.starts_with"
	MsgEndsWith     = "validation.ends_with"
	MsgLowercase    = "validation.lowercase"
	MsgUppercase    = "validation.uppercase"
	MsgInteger      = "validation.integer"
	MsgFloat        = "validation.float"
	MsgJSON         = "validation.json"
	MsgBase64       = "validation.base64"
	MsgUUID         = "validation.uuid"
	MsgSlug         = "validation.slug"
	MsgMin          = "validation.min_value"
	MsgMax          = "validation.max_value"
	MsgBetween      = "validation.between"
	MsgZero         = "validation.zero"
	MsgEqual        = "validation.equal"
	MsgEqualTo      = "validation.equal_to"
	MsgGreaterThan  = "validation.greater_than"
	MsgLessThan     = "validation.less_than"
	MsgPositive     = "validation.positive"
	MsgNegative     = "validation.negative"
	MsgEven         = "validation.even"
	MsgOdd          = "validation.odd"
	MsgMultipleOf   = "validation.multiple_of"
	MsgFinite       = "validation.finite"
	MsgPrecision    = "validation.precision"
	MsgInvalid      = "validation.invalid"
	MsgDuplicate    = "validation.duplicate"

	// Negated validation message constants
	MsgNotEmpty        = "validation.not_empty"
	MsgNotEqualTo      = "validation.not_equal_to"
	MsgNotMinLength    = "validation.not_min_length"
	MsgNotMaxLength    = "validation.not_max_length"
	MsgNotExactLength  = "validation.not_exact_length"
	MsgNotBetween      = "validation.not_between"
	MsgNotEmail        = "validation.not_email"
	MsgNotURL          = "validation.not_url"
	MsgNotNumeric      = "validation.not_numeric"
	MsgNotAlpha        = "validation.not_alpha"
	MsgNotAlphaNumeric = "validation.not_alpha_numeric"
	MsgNotRegex        = "validation.not_regex"
	MsgNotContains     = "validation.not_contains"
	MsgNotStartsWith   = "validation.not_starts_with"
	MsgNotEndsWith     = "validation.not_ends_with"
	MsgNotLowercase    = "validation.not_lowercase"
	MsgNotUppercase    = "validation.not_uppercase"
	MsgNotInteger      = "validation.not_integer"
	MsgNotFloat        = "validation.not_float"
	MsgNotJSON         = "validation.not_json"
	MsgNotBase64       = "validation.not_base64"
	MsgNotUUID         = "validation.not_uuid"
	MsgNotSlug         = "validation.not_slug"
	MsgNotZero         = "validation.not_zero"
	MsgNotMinValue     = "validation.not_min_value"
	MsgNotMaxValue     = "validation.not_max_value"
	MsgNotGreaterThan  = "validation.not_greater_than"
	MsgNotLessThan     = "validation.not_less_than"
	MsgNotPositive     = "validation.not_positive"
	MsgNotNegative     = "validation.not_negative"
	MsgNotEven         = "validation.not_even"
	MsgNotOdd          = "validation.not_odd"
	MsgNotMultipleOf   = "validation.not_multiple_of"
	MsgNotFinite       = "validation.not_finite"
	MsgNotPrecision    = "validation.not_precision"

	// Special validation message constants
	MsgMustBeZero  = "validation.must_be_zero"
	MsgDivisorZero = "validation.divisor_zero"

	// Error message constants
	MsgErrorMultiple       = "error.multiple"
	MsgErrorNotFound       = "error.not_found"
	MsgErrorInvalidRequest = "error.invalid_request"
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
		MsgRequired: {
			Singular: "{{.field}} is required",
			Plural:   "",
		},
		MsgMinLength: {
			Singular: "{{.field}} must be at least {{.min}} characters long",
			Plural:   "",
		},
		MsgMaxLength: {
			Singular: "{{.field}} must be at most {{.max}} characters long",
			Plural:   "",
		},
		MsgEmail: {
			Singular: "{{.field}} must be a valid email address",
			Plural:   "",
		},
		MsgMin: {
			Singular: "{{.field}} must be at least {{.min}}",
			Plural:   "",
		},
		MsgMax: {
			Singular: "{{.field}} must be at most {{.max}}",
			Plural:   "",
		},
		MsgInvalid: {
			Singular: "{{.field}} value is invalid",
			Plural:   "",
		},

		MsgEmpty: {
			Singular: "{{.field}} must be empty",
			Plural:   "",
		},
		MsgNotEmpty: {
			Singular: "{{.field}} must not be empty",
			Plural:   "",
		},
		MsgDuplicate: {
			Singular: "{{.field}} already exists, another record has the same value",
			Plural:   "",
		},

		// Additional validation messages for VIX integration
		MsgExactLength: {
			Singular: "{{.field}} must be exactly {{.length}} characters long",
			Plural:   "",
		},

		MsgURL: {
			Singular: "{{.field}} must be a valid URL",
			Plural:   "",
		},
		MsgNumeric: {
			Singular: "{{.field}} must contain only numeric characters",
			Plural:   "",
		},
		MsgAlpha: {
			Singular: "{{.field}} must contain only alphabetic characters",
			Plural:   "",
		},
		MsgAlphaNumeric: {
			Singular: "{{.field}} must contain only alphanumeric characters",
			Plural:   "",
		},
		MsgRegex: {
			Singular: "{{.field}} does not match the required pattern",
			Plural:   "",
		},
		MsgIn: {
			Singular: "{{.field}} must be one of: {{.values}}",
			Plural:   "",
		},
		MsgNotIn: {
			Singular: "{{.field}} must not be one of: {{.values}}",
			Plural:   "",
		},
		MsgContains: {
			Singular: "{{.field}} must contain '{{.substring}}'",
			Plural:   "",
		},
		MsgStartsWith: {
			Singular: "{{.field}} must start with '{{.prefix}}'",
			Plural:   "",
		},
		MsgEndsWith: {
			Singular: "{{.field}} must end with '{{.suffix}}'",
			Plural:   "",
		},
		MsgLowercase: {
			Singular: "{{.field}} must be in lowercase",
			Plural:   "",
		},
		MsgUppercase: {
			Singular: "{{.field}} must be in uppercase",
			Plural:   "",
		},
		MsgInteger: {
			Singular: "{{.field}} must be a valid integer",
			Plural:   "",
		},
		MsgFloat: {
			Singular: "{{.field}} must be a valid number",
			Plural:   "",
		},
		MsgJSON: {
			Singular: "{{.field}} must be valid JSON",
			Plural:   "",
		},
		MsgBase64: {
			Singular: "{{.field}} must be valid base64",
			Plural:   "",
		},
		MsgUUID: {
			Singular: "{{.field}} must be a valid UUID",
			Plural:   "",
		},
		MsgSlug: {
			Singular: "{{.field}} must be a valid slug",
			Plural:   "",
		},
		MsgBetween: {
			Singular: "{{.field}} must be between {{.min}} and {{.max}}",
			Plural:   "",
		},
		MsgZero: {
			Singular: "{{.field}} must be zero",
			Plural:   "",
		},
		MsgEqual: {
			Singular: "{{.field}} must equal {{.expected}}",
			Plural:   "",
		},
		MsgEqualTo: {
			Singular: "{{.field}} must equal {{.expected}}",
			Plural:   "",
		},
		MsgGreaterThan: {
			Singular: "{{.field}} must be greater than {{.value}}",
			Plural:   "",
		},
		MsgLessThan: {
			Singular: "{{.field}} must be less than {{.value}}",
			Plural:   "",
		},
		MsgPositive: {
			Singular: "{{.field}} must be positive",
			Plural:   "",
		},
		MsgNegative: {
			Singular: "{{.field}} must be negative",
			Plural:   "",
		},
		MsgEven: {
			Singular: "{{.field}} must be even",
			Plural:   "",
		},
		MsgOdd: {
			Singular: "{{.field}} must be odd",
			Plural:   "",
		},
		MsgMultipleOf: {
			Singular: "{{.field}} must be a multiple of {{.divisor}}",
			Plural:   "",
		},
		MsgFinite: {
			Singular: "{{.field}} must be finite",
			Plural:   "",
		},
		MsgPrecision: {
			Singular: "{{.field}} must have at most {{.places}} decimal places",
			Plural:   "",
		},
		MsgDivisorZero: {
			Singular: "{{.field}} divisor cannot be zero",
			Plural:   "",
		},
		// Negated validation messages
		MsgNotEqualTo: {
			Singular: "{{.field}} must not equal {{.expected}}",
			Plural:   "",
		},
		MsgNotMinLength: {
			Singular: "{{.field}} must not be at least {{.min}} characters long",
			Plural:   "",
		},
		MsgNotMaxLength: {
			Singular: "{{.field}} must not be at most {{.max}} characters long",
			Plural:   "",
		},
		MsgNotExactLength: {
			Singular: "{{.field}} must not be exactly {{.length}} characters long",
			Plural:   "",
		},
		MsgNotBetween: {
			Singular: "{{.field}} must not be between {{.min}} and {{.max}}",
			Plural:   "",
		},
		MsgNotEmail: {
			Singular: "{{.field}} must not be a valid email address",
			Plural:   "",
		},
		MsgNotURL: {
			Singular: "{{.field}} must not be a valid URL",
			Plural:   "",
		},
		MsgNotNumeric: {
			Singular: "{{.field}} must not contain only numeric characters",
			Plural:   "",
		},
		MsgNotAlpha: {
			Singular: "{{.field}} must not contain only alphabetic characters",
			Plural:   "",
		},
		MsgNotAlphaNumeric: {
			Singular: "{{.field}} must not contain only alphanumeric characters",
			Plural:   "",
		},
		MsgNotRegex: {
			Singular: "{{.field}} must not match the pattern",
			Plural:   "",
		},
		MsgNotContains: {
			Singular: "{{.field}} must not contain '{{.substring}}'",
			Plural:   "",
		},
		MsgNotStartsWith: {
			Singular: "{{.field}} must not start with '{{.prefix}}'",
			Plural:   "",
		},
		MsgNotEndsWith: {
			Singular: "{{.field}} must not end with '{{.suffix}}'",
			Plural:   "",
		},
		MsgNotLowercase: {
			Singular: "{{.field}} must not be in lowercase",
			Plural:   "",
		},
		MsgNotUppercase: {
			Singular: "{{.field}} must not be in uppercase",
			Plural:   "",
		},
		MsgNotInteger: {
			Singular: "{{.field}} must not be a valid integer",
			Plural:   "",
		},
		MsgNotFloat: {
			Singular: "{{.field}} must not be a valid number",
			Plural:   "",
		},
		MsgNotJSON: {
			Singular: "{{.field}} must not be valid JSON",
			Plural:   "",
		},
		MsgNotBase64: {
			Singular: "{{.field}} must not be valid base64",
			Plural:   "",
		},
		MsgNotUUID: {
			Singular: "{{.field}} must not be a valid UUID",
			Plural:   "",
		},
		MsgNotSlug: {
			Singular: "{{.field}} must not be a valid slug",
			Plural:   "",
		},
		MsgNotZero: {
			Singular: "{{.field}} must not be zero",
			Plural:   "",
		},
		MsgNotPositive: {
			Singular: "{{.field}} must not be positive",
			Plural:   "",
		},
		MsgNotNegative: {
			Singular: "{{.field}} must not be negative",
			Plural:   "",
		},
		MsgNotEven: {
			Singular: "{{.field}} must not be even",
			Plural:   "",
		},
		MsgNotOdd: {
			Singular: "{{.field}} must not be odd",
			Plural:   "",
		},
		MsgNotMultipleOf: {
			Singular: "{{.field}} must not be a multiple of {{.divisor}}",
			Plural:   "",
		},
		MsgNotFinite: {
			Singular: "{{.field}} must not be finite",
			Plural:   "",
		},
		MsgNotPrecision: {
			Singular: "{{.field}} must not have {{.places}} decimal places",
			Plural:   "",
		},
		MsgNotMinValue: {
			Singular: "{{.field}} must not be at least {{.min}}",
			Plural:   "",
		},
		MsgNotMaxValue: {
			Singular: "{{.field}} must not be at most {{.max}}",
			Plural:   "",
		},
		MsgMustBeZero: {
			Singular: "{{.field}} must be zero",
			Plural:   "",
		},
		MsgNotGreaterThan: {
			Singular: "{{.field}} must not be greater than {{.value}}",
			Plural:   "",
		},
		MsgNotLessThan: {
			Singular: "{{.field}} must not be less than {{.value}}",
			Plural:   "",
		},
		MsgErrorMultiple: {
			Singular: "multiple errors: {{.errors}}",
			Plural:   "",
		},
		MsgErrorNotFound: {
			Singular: "{{.field}} is not found",
			Plural:   "",
		},
		MsgErrorInvalidRequest: {
			Singular: "invalid request",
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
