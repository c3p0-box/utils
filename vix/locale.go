package vix

import (
	"fmt"
	"strings"
)

// Locale represents a localization configuration for validation messages.
// It provides a flexible template-based system for error messages with
// support for parameter substitution and pluralization.
type Locale struct {
	Code     string
	Name     string
	Messages map[string]string
}

// NewLocale creates a new Locale with the given code and name.
func NewLocale(code, name string) *Locale {
	return &Locale{
		Code:     code,
		Name:     name,
		Messages: make(map[string]string),
	}
}

// SetMessage sets a localized message for the given validation code.
func (l *Locale) SetMessage(code, message string) *Locale {
	l.Messages[code] = message
	return l
}

// GetMessage retrieves a localized message for the given validation code.
// Returns the message if found, otherwise returns an empty string.
func (l *Locale) GetMessage(code string) string {
	if message, exists := l.Messages[code]; exists {
		return message
	}
	return ""
}

// HasMessage checks if a message exists for the given validation code.
func (l *Locale) HasMessage(code string) bool {
	_, exists := l.Messages[code]
	return exists
}

// Merge merges another locale into this one, overriding existing messages.
func (l *Locale) Merge(other *Locale) *Locale {
	for code, message := range other.Messages {
		l.Messages[code] = message
	}
	return l
}

// Clone creates a copy of this locale.
func (l *Locale) Clone() *Locale {
	clone := &Locale{
		Code:     l.Code,
		Name:     l.Name,
		Messages: make(map[string]string),
	}
	for code, message := range l.Messages {
		clone.Messages[code] = message
	}
	return clone
}

// FormatMessage formats a message template with the given parameters.
func (l *Locale) FormatMessage(template string, params map[string]interface{}) string {
	result := template

	// Replace parameters
	for key, value := range params {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result
}

// Default English locale
var DefaultLocale = &Locale{
	Code: "en",
	Name: "English",
	Messages: map[string]string{
		// Basic validation messages
		CodeRequired:     "{{field}} is required",
		CodeMinLength:    "{{field}} must be at least {{min}} characters long",
		CodeMaxLength:    "{{field}} must be at most {{max}} characters long",
		CodeExactLength:  "{{field}} must be exactly {{length}} characters long",
		CodeEmail:        "{{field}} must be a valid email address",
		CodeURL:          "{{field}} must be a valid URL",
		CodeNumeric:      "{{field}} must be numeric",
		CodeAlpha:        "{{field}} must contain only letters",
		CodeAlphaNumeric: "{{field}} must contain only letters and numbers",
		CodeRegex:        "{{field}} format is invalid",
		CodeIn:           "{{field}} must be one of: {{values}}",
		CodeNotIn:        "{{field}} must not be one of: {{values}}",
		CodeMin:          "{{field}} must be at least {{min}}",
		CodeMax:          "{{field}} must be at most {{max}}",
		CodeBetween:      "{{field}} must be between {{min}} and {{max}}",
		CodeAfter:        "{{field}} must be after {{date}}",
		CodeBefore:       "{{field}} must be before {{date}}",
		CodeDateFormat:   "{{field}} must be a valid date in format {{format}}",

		// Negated messages
		"not_" + CodeRequired:     "{{field}} must be empty",
		"not_" + CodeMinLength:    "{{field}} must be less than {{min}} characters long",
		"not_" + CodeMaxLength:    "{{field}} must be more than {{max}} characters long",
		"not_" + CodeExactLength:  "{{field}} must not be exactly {{length}} characters long",
		"not_" + CodeEmail:        "{{field}} must not be a valid email address",
		"not_" + CodeURL:          "{{field}} must not be a valid URL",
		"not_" + CodeNumeric:      "{{field}} must not be numeric",
		"not_" + CodeAlpha:        "{{field}} must not contain only letters",
		"not_" + CodeAlphaNumeric: "{{field}} must not contain only letters and numbers",
		"not_" + CodeRegex:        "{{field}} must not match the required format",
		"not_" + CodeNotIn:        "{{field}} may be one of: {{values}}",
		"not_" + CodeMin:          "{{field}} must be less than {{min}}",
		"not_" + CodeMax:          "{{field}} must be more than {{max}}",
		"not_" + CodeBetween:      "{{field}} must not be between {{min}} and {{max}}",
		"not_" + CodeAfter:        "{{field}} must not be after {{date}}",
		"not_" + CodeBefore:       "{{field}} must not be before {{date}}",
		"not_" + CodeDateFormat:   "{{field}} must not be a valid date in format {{format}}",
	},
}

// Spanish locale
var SpanishLocale = &Locale{
	Code: "es",
	Name: "Español",
	Messages: map[string]string{
		CodeRequired:     "{{field}} es requerido",
		CodeMinLength:    "{{field}} debe tener al menos {{min}} caracteres",
		CodeMaxLength:    "{{field}} debe tener como máximo {{max}} caracteres",
		CodeExactLength:  "{{field}} debe tener exactamente {{length}} caracteres",
		CodeEmail:        "{{field}} debe ser una dirección de correo válida",
		CodeURL:          "{{field}} debe ser una URL válida",
		CodeNumeric:      "{{field}} debe ser numérico",
		CodeAlpha:        "{{field}} debe contener solo letras",
		CodeAlphaNumeric: "{{field}} debe contener solo letras y números",
		CodeRegex:        "{{field}} tiene un formato inválido",
		CodeIn:           "{{field}} debe ser uno de: {{values}}",
		CodeNotIn:        "{{field}} no debe ser uno de: {{values}}",
		CodeMin:          "{{field}} debe ser al menos {{min}}",
		CodeMax:          "{{field}} debe ser como máximo {{max}}",
		CodeBetween:      "{{field}} debe estar entre {{min}} y {{max}}",
		CodeAfter:        "{{field}} debe ser después de {{date}}",
		CodeBefore:       "{{field}} debe ser antes de {{date}}",
		CodeDateFormat:   "{{field}} debe ser una fecha válida en formato {{format}}",
	},
}

// French locale
var FrenchLocale = &Locale{
	Code: "fr",
	Name: "Français",
	Messages: map[string]string{
		CodeRequired:     "{{field}} est requis",
		CodeMinLength:    "{{field}} doit avoir au moins {{min}} caractères",
		CodeMaxLength:    "{{field}} doit avoir au maximum {{max}} caractères",
		CodeExactLength:  "{{field}} doit avoir exactement {{length}} caractères",
		CodeEmail:        "{{field}} doit être une adresse email valide",
		CodeURL:          "{{field}} doit être une URL valide",
		CodeNumeric:      "{{field}} doit être numérique",
		CodeAlpha:        "{{field}} doit contenir uniquement des lettres",
		CodeAlphaNumeric: "{{field}} doit contenir uniquement des lettres et des chiffres",
		CodeRegex:        "{{field}} a un format invalide",
		CodeIn:           "{{field}} doit être l'un de: {{values}}",
		CodeNotIn:        "{{field}} ne doit pas être l'un de: {{values}}",
		CodeMin:          "{{field}} doit être au moins {{min}}",
		CodeMax:          "{{field}} doit être au maximum {{max}}",
		CodeBetween:      "{{field}} doit être entre {{min}} et {{max}}",
		CodeAfter:        "{{field}} doit être après {{date}}",
		CodeBefore:       "{{field}} doit être avant {{date}}",
		CodeDateFormat:   "{{field}} doit être une date valide au format {{format}}",
	},
}

// German locale
var GermanLocale = &Locale{
	Code: "de",
	Name: "Deutsch",
	Messages: map[string]string{
		CodeRequired:     "{{field}} ist erforderlich",
		CodeMinLength:    "{{field}} muss mindestens {{min}} Zeichen lang sein",
		CodeMaxLength:    "{{field}} darf höchstens {{max}} Zeichen lang sein",
		CodeExactLength:  "{{field}} muss genau {{length}} Zeichen lang sein",
		CodeEmail:        "{{field}} muss eine gültige E-Mail-Adresse sein",
		CodeURL:          "{{field}} muss eine gültige URL sein",
		CodeNumeric:      "{{field}} muss numerisch sein",
		CodeAlpha:        "{{field}} darf nur Buchstaben enthalten",
		CodeAlphaNumeric: "{{field}} darf nur Buchstaben und Zahlen enthalten",
		CodeRegex:        "{{field}} hat ein ungültiges Format",
		CodeIn:           "{{field}} muss einer von: {{values}} sein",
		CodeNotIn:        "{{field}} darf nicht einer von: {{values}} sein",
		CodeMin:          "{{field}} muss mindestens {{min}} sein",
		CodeMax:          "{{field}} darf höchstens {{max}} sein",
		CodeBetween:      "{{field}} muss zwischen {{min}} und {{max}} liegen",
		CodeAfter:        "{{field}} muss nach {{date}} liegen",
		CodeBefore:       "{{field}} muss vor {{date}} liegen",
		CodeDateFormat:   "{{field}} muss ein gültiges Datum im Format {{format}} sein",
	},
}

// LocaleRegistry manages available locales
type LocaleRegistry struct {
	locales  map[string]*Locale
	default_ *Locale
}

// NewLocaleRegistry creates a new locale registry.
func NewLocaleRegistry() *LocaleRegistry {
	registry := &LocaleRegistry{
		locales:  make(map[string]*Locale),
		default_: DefaultLocale,
	}

	// Register default locales
	registry.Register(DefaultLocale)
	registry.Register(SpanishLocale)
	registry.Register(FrenchLocale)
	registry.Register(GermanLocale)

	return registry
}

// Register registers a locale in the registry.
func (lr *LocaleRegistry) Register(locale *Locale) {
	lr.locales[locale.Code] = locale
}

// Get retrieves a locale by code, returns default if not found.
func (lr *LocaleRegistry) Get(code string) *Locale {
	if locale, exists := lr.locales[code]; exists {
		return locale
	}
	return lr.default_
}

// SetDefault sets the default locale.
func (lr *LocaleRegistry) SetDefault(locale *Locale) {
	lr.default_ = locale
}

// GetDefault returns the default locale.
func (lr *LocaleRegistry) GetDefault() *Locale {
	return lr.default_
}

// List returns all registered locales.
func (lr *LocaleRegistry) List() map[string]*Locale {
	result := make(map[string]*Locale)
	for code, locale := range lr.locales {
		result[code] = locale
	}
	return result
}

// Global locale registry instance
var GlobalLocaleRegistry = NewLocaleRegistry()
