package i18n

import (
	"fmt"
	"strings"
	"sync"
	"text/template"

	"golang.org/x/text/language"
)

// Translation represents a single translation entry with optional plural form
type Translation struct {
	Singular string // Template string for singular form
	Plural   string // Template string for plural form (optional)
}

// Manager is the singleton manager for all translations
type Manager struct {
	mu              sync.RWMutex
	translations    map[language.Tag]map[string]*Translation
	defaultLanguage language.Tag
	templateCache   map[string]*template.Template
}

var (
	instance *Manager
	once     sync.Once
)

// GetInstance returns the singleton instance of the translation manager
func GetInstance() *Manager {
	once.Do(func() {
		instance = &Manager{
			translations:    make(map[language.Tag]map[string]*Translation),
			defaultLanguage: language.English,
			templateCache:   make(map[string]*template.Template),
		}
	})
	return instance
}

// SetDefaultLanguage sets the default fallback language
func (m *Manager) SetDefaultLanguage(lang language.Tag) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultLanguage = lang
}

// GetDefaultLanguage returns the current default language
func (m *Manager) GetDefaultLanguage() language.Tag {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultLanguage
}

// AddTranslation adds a new translation for the specified language and key
// If plural is empty, the singular form will be used for plural as well
func (m *Manager) AddTranslation(lang language.Tag, key, value, plural string) error {
	if key == "" {
		return fmt.Errorf("translation key cannot be empty")
	}
	if value == "" {
		return fmt.Errorf("translation value cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize language map if it doesn't exist
	if m.translations[lang] == nil {
		m.translations[lang] = make(map[string]*Translation)
	}

	// Use singular form for plural if not provided
	if plural == "" {
		plural = value
	}

	m.translations[lang][key] = &Translation{
		Singular: value,
		Plural:   plural,
	}

	return nil
}

// AddTranslations adds multiple translations for the specified language at once
// This is more efficient than calling AddTranslation multiple times as it only acquires the lock once
// translations is a map where keys are translation keys and values are Translation structs
// If a Translation's Plural field is empty, the Singular form will be used for plural as well
func (m *Manager) AddTranslations(lang language.Tag, translations map[string]*Translation) error {
	if len(translations) == 0 {
		return fmt.Errorf("translations map cannot be empty")
	}

	// Validate all translations first before acquiring lock
	for key, translation := range translations {
		if key == "" {
			return fmt.Errorf("translation key cannot be empty")
		}
		if translation == nil {
			return fmt.Errorf("translation cannot be nil for key '%s'", key)
		}
		if translation.Singular == "" {
			return fmt.Errorf("translation value cannot be empty for key '%s'", key)
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize language map if it doesn't exist
	if m.translations[lang] == nil {
		m.translations[lang] = make(map[string]*Translation)
	}

	// Add all translations
	for key, translation := range translations {
		// Use singular form for plural if not provided
		plural := translation.Plural
		if plural == "" {
			plural = translation.Singular
		}

		m.translations[lang][key] = &Translation{
			Singular: translation.Singular,
			Plural:   plural,
		}
	}

	return nil
}

// HasTranslation checks if a translation exists for the given language and key
func (m *Manager) HasTranslation(lang language.Tag, key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if langMap, exists := m.translations[lang]; exists {
		_, exists := langMap[key]
		return exists
	}
	return false
}

// getTemplate returns a cached template or creates a new one
func (m *Manager) getTemplate(templateStr string) (*template.Template, error) {
	// Check cache first (without lock for performance)
	if tmpl, exists := m.templateCache[templateStr]; exists {
		return tmpl, nil
	}

	// Create new template
	tmpl, err := template.New("").Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Cache the template
	m.templateCache[templateStr] = tmpl
	return tmpl, nil
}

// executeTemplate executes a template with the given data
func (m *Manager) executeTemplate(templateStr string, data interface{}) (string, error) {
	tmpl, err := m.getTemplate(templateStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// Translate retrieves and processes a translation for the given language and key
// count determines whether to use singular (count == 1) or plural (count != 1) form
// data is passed to the template for processing
func (m *Manager) Translate(lang language.Tag, key string, count int, data interface{}) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try to find translation in requested language
	if langMap, exists := m.translations[lang]; exists {
		if translation, exists := langMap[key]; exists {
			return m.processTranslation(translation, count, data)
		}
	}

	// Fallback to default language if available
	if lang != m.defaultLanguage {
		if langMap, exists := m.translations[m.defaultLanguage]; exists {
			if translation, exists := langMap[key]; exists {
				return m.processTranslation(translation, count, data)
			}
		}
	}

	// Return the key itself if no translation found
	return key
}

// processTranslation processes a translation entry with the given count and data
func (m *Manager) processTranslation(translation *Translation, count int, data interface{}) string {
	var templateStr string

	// Choose singular or plural form based on count
	if count == 1 {
		templateStr = translation.Singular
	} else {
		templateStr = translation.Plural
	}

	// If no data provided, return template string as-is
	if data == nil {
		return templateStr
	}

	// Execute template with data
	result, err := m.executeTemplate(templateStr, data)
	if err != nil {
		// Return original template string if template execution fails
		return templateStr
	}

	return result
}

// TranslateSimple is a convenience method for simple translations without templates
func (m *Manager) TranslateSimple(lang language.Tag, key string) string {
	return m.Translate(lang, key, 1, nil)
}

// TranslatePlural is a convenience method for plural translations without templates
func (m *Manager) TranslatePlural(lang language.Tag, key string, count int) string {
	return m.Translate(lang, key, count, nil)
}

// GetAvailableLanguages returns a list of all languages that have translations
func (m *Manager) GetAvailableLanguages() []language.Tag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	languages := make([]language.Tag, 0, len(m.translations))
	for lang := range m.translations {
		languages = append(languages, lang)
	}
	return languages
}

// GetTranslationKeys returns all translation keys for a given language
func (m *Manager) GetTranslationKeys(lang language.Tag) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if langMap, exists := m.translations[lang]; exists {
		keys := make([]string, 0, len(langMap))
		for key := range langMap {
			keys = append(keys, key)
		}
		return keys
	}
	return nil
}

// Convenience functions for global instance

// SetDefaultLanguage sets the default language for the global instance
func SetDefaultLanguage(lang language.Tag) {
	GetInstance().SetDefaultLanguage(lang)
}

// AddTranslation adds a translation to the global instance
func AddTranslation(lang language.Tag, key, value, plural string) error {
	return GetInstance().AddTranslation(lang, key, value, plural)
}

// AddTranslations adds multiple translations to the global instance
func AddTranslations(lang language.Tag, translations map[string]*Translation) error {
	return GetInstance().AddTranslations(lang, translations)
}

// Translate translates using the global instance
func Translate(lang language.Tag, key string, count int, data interface{}) string {
	return GetInstance().Translate(lang, key, count, data)
}

// TranslateSimple translates using the global instance without templates
func TranslateSimple(lang language.Tag, key string) string {
	return GetInstance().TranslateSimple(lang, key)
}

// TranslatePlural translates with plural support using the global instance
func TranslatePlural(lang language.Tag, key string, count int) string {
	return GetInstance().TranslatePlural(lang, key, count)
}
