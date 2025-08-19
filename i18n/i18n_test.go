package i18n

import (
	"fmt"
	"testing"

	"golang.org/x/text/language"
)

func TestSingleton(t *testing.T) {
	instance1 := GetInstance()
	instance2 := GetInstance()

	if instance1 != instance2 {
		t.Error("GetInstance() should return the same instance")
	}
}

func TestSetDefaultLanguage(t *testing.T) {
	manager := GetInstance()

	// Test setting and getting default language
	manager.SetDefaultLanguage(language.Spanish)
	if manager.GetDefaultLanguage() != language.Spanish {
		t.Error("Default language should be Spanish")
	}

	// Test global function
	SetDefaultLanguage(language.French)
	if manager.GetDefaultLanguage() != language.French {
		t.Error("Default language should be French")
	}
}

func TestAddTranslation(t *testing.T) {
	manager := GetInstance()

	// Test successful addition
	err := manager.AddTranslation(language.English, "hello", "Hello", "Hello")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test empty key
	err = manager.AddTranslation(language.English, "", "value", "")
	if err == nil {
		t.Error("Should return error for empty key")
	}

	// Test empty value
	err = manager.AddTranslation(language.English, "key", "", "")
	if err == nil {
		t.Error("Should return error for empty value")
	}

	// Test global function
	err = AddTranslation(language.Spanish, "hello", "Hola", "Hola")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestHasTranslation(t *testing.T) {
	manager := GetInstance()

	// Add a translation
	manager.AddTranslation(language.English, "test", "Test", "")

	// Test existing translation
	if !manager.HasTranslation(language.English, "test") {
		t.Error("Should find existing translation")
	}

	// Test non-existing translation
	if manager.HasTranslation(language.English, "nonexistent") {
		t.Error("Should not find non-existing translation")
	}

	// Test non-existing language
	if manager.HasTranslation(language.German, "test") {
		t.Error("Should not find translation in non-existing language")
	}
}

func TestTranslateSimple(t *testing.T) {
	manager := GetInstance()

	// Add translations
	manager.AddTranslation(language.English, "greeting", "Hello", "")
	manager.AddTranslation(language.Spanish, "greeting", "Hola", "")

	// Test existing translation
	result := manager.TranslateSimple(language.English, "greeting")
	if result != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", result)
	}

	// Test different language
	result = manager.TranslateSimple(language.Spanish, "greeting")
	if result != "Hola" {
		t.Errorf("Expected 'Hola', got '%s'", result)
	}

	// Test non-existing translation (should return key)
	result = manager.TranslateSimple(language.English, "nonexistent")
	if result != "nonexistent" {
		t.Errorf("Expected 'nonexistent', got '%s'", result)
	}

	// Test global function
	result = TranslateSimple(language.English, "greeting")
	if result != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", result)
	}
}

func TestTranslatePlural(t *testing.T) {
	manager := GetInstance()

	// Add translation with plural
	manager.AddTranslation(language.English, "item", "{{.Count}} item", "{{.Count}} items")

	// Test singular (count = 1)
	result := manager.TranslatePlural(language.English, "item", 1)
	if result != "{{.Count}} item" {
		t.Errorf("Expected '{{.Count}} item', got '%s'", result)
	}

	// Test plural (count != 1)
	result = manager.TranslatePlural(language.English, "item", 0)
	if result != "{{.Count}} items" {
		t.Errorf("Expected '{{.Count}} items', got '%s'", result)
	}

	result = manager.TranslatePlural(language.English, "item", 5)
	if result != "{{.Count}} items" {
		t.Errorf("Expected '{{.Count}} items', got '%s'", result)
	}

	// Test global function
	result = TranslatePlural(language.English, "item", 1)
	if result != "{{.Count}} item" {
		t.Errorf("Expected '{{.Count}} item', got '%s'", result)
	}
}

func TestTranslateWithTemplate(t *testing.T) {
	manager := GetInstance()

	// Add template translation
	manager.AddTranslation(language.English, "welcome", "Welcome, {{.Name}}!", "")
	manager.AddTranslation(language.English, "items", "You have {{.Count}} item", "You have {{.Count}} items")

	// Test template execution
	data := map[string]interface{}{"Name": "John"}
	result := manager.Translate(language.English, "welcome", 1, data)
	if result != "Welcome, John!" {
		t.Errorf("Expected 'Welcome, John!', got '%s'", result)
	}

	// Test template with count data
	countData := map[string]interface{}{"Count": 1}
	result = manager.Translate(language.English, "items", 1, countData)
	if result != "You have 1 item" {
		t.Errorf("Expected 'You have 1 item', got '%s'", result)
	}

	countData = map[string]interface{}{"Count": 5}
	result = manager.Translate(language.English, "items", 5, countData)
	if result != "You have 5 items" {
		t.Errorf("Expected 'You have 5 items', got '%s'", result)
	}

	// Test global function
	result = Translate(language.English, "welcome", 1, data)
	if result != "Welcome, John!" {
		t.Errorf("Expected 'Welcome, John!', got '%s'", result)
	}
}

func TestFallbackToDefaultLanguage(t *testing.T) {
	manager := GetInstance()

	// Set default language and add translation
	manager.SetDefaultLanguage(language.English)
	manager.AddTranslation(language.English, "fallback", "Default text", "")

	// Try to translate in non-existing language, should fall back to default
	result := manager.TranslateSimple(language.German, "fallback")
	if result != "Default text" {
		t.Errorf("Expected 'Default text', got '%s'", result)
	}
}

func TestEmptyPluralUseSingular(t *testing.T) {
	manager := GetInstance()

	// Add translation without plural (should use singular for both)
	manager.AddTranslation(language.English, "message", "Same for both", "")

	// Test both singular and plural return same value
	singular := manager.TranslatePlural(language.English, "message", 1)
	plural := manager.TranslatePlural(language.English, "message", 5)

	if singular != "Same for both" || plural != "Same for both" {
		t.Errorf("Both should return 'Same for both', got singular: '%s', plural: '%s'", singular, plural)
	}
}

func TestGetAvailableLanguages(t *testing.T) {
	manager := GetInstance()

	// Clear any existing translations by creating new instance for this test
	manager.translations = make(map[language.Tag]map[string]*Translation)

	// Add translations for different languages
	manager.AddTranslation(language.English, "test", "Test", "")
	manager.AddTranslation(language.Spanish, "test", "Prueba", "")
	manager.AddTranslation(language.French, "test", "Test", "")

	languages := manager.GetAvailableLanguages()

	if len(languages) != 3 {
		t.Errorf("Expected 3 languages, got %d", len(languages))
	}

	// Check if all languages are present
	langMap := make(map[language.Tag]bool)
	for _, lang := range languages {
		langMap[lang] = true
	}

	if !langMap[language.English] || !langMap[language.Spanish] || !langMap[language.French] {
		t.Error("Missing expected languages")
	}
}

func TestGetTranslationKeys(t *testing.T) {
	manager := GetInstance()

	// Clear any existing translations
	manager.translations = make(map[language.Tag]map[string]*Translation)

	// Add multiple translations for English
	manager.AddTranslation(language.English, "hello", "Hello", "")
	manager.AddTranslation(language.English, "goodbye", "Goodbye", "")
	manager.AddTranslation(language.English, "thanks", "Thank you", "")

	keys := manager.GetTranslationKeys(language.English)

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check if all keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	if !keyMap["hello"] || !keyMap["goodbye"] || !keyMap["thanks"] {
		t.Error("Missing expected keys")
	}

	// Test non-existing language
	keys = manager.GetTranslationKeys(language.German)
	if keys != nil {
		t.Error("Should return nil for non-existing language")
	}
}

func TestTemplateError(t *testing.T) {
	manager := GetInstance()

	// Add translation with invalid template
	manager.AddTranslation(language.English, "invalid", "Hello {{.Name", "")

	// Should return original template string on error
	result := manager.Translate(language.English, "invalid", 1, map[string]interface{}{"Name": "John"})
	if result != "Hello {{.Name" {
		t.Errorf("Expected original template string on error, got '%s'", result)
	}

	// Test template execution error (different from parse error)
	manager.AddTranslation(language.English, "exec_error", "Hello {{.Name.Missing}}", "")
	result = manager.Translate(language.English, "exec_error", 1, map[string]interface{}{"Name": "John"})
	if result != "Hello {{.Name.Missing}}" {
		t.Errorf("Expected original template string on execution error, got '%s'", result)
	}
}

func TestAddTranslations(t *testing.T) {
	manager := GetInstance()

	// Test successful addition of multiple translations
	translations := map[string]*Translation{
		"hello":   {Singular: "Hello", Plural: ""},
		"goodbye": {Singular: "Goodbye", Plural: ""},
		"items": {
			Singular: "{{.Count}} item",
			Plural:   "{{.Count}} items",
		},
	}

	err := manager.AddTranslations(language.French, translations)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify translations were added
	if !manager.HasTranslation(language.French, "hello") {
		t.Error("Should find 'hello' translation")
	}
	if !manager.HasTranslation(language.French, "goodbye") {
		t.Error("Should find 'goodbye' translation")
	}
	if !manager.HasTranslation(language.French, "items") {
		t.Error("Should find 'items' translation")
	}

	// Test that plurals default to singular when empty
	result := manager.TranslatePlural(language.French, "hello", 5)
	if result != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", result)
	}

	// Test global function
	err = AddTranslations(language.German, translations)
	if err != nil {
		t.Errorf("Unexpected error with global function: %v", err)
	}
}

func TestAddTranslationsErrors(t *testing.T) {
	manager := GetInstance()

	// Test empty translations map
	err := manager.AddTranslations(language.English, map[string]*Translation{})
	if err == nil {
		t.Error("Should return error for empty translations map")
	}

	// Test empty key
	translations := map[string]*Translation{
		"": {Singular: "value", Plural: ""},
	}
	err = manager.AddTranslations(language.English, translations)
	if err == nil {
		t.Error("Should return error for empty key")
	}

	// Test nil translation
	translations = map[string]*Translation{
		"key": nil,
	}
	err = manager.AddTranslations(language.English, translations)
	if err == nil {
		t.Error("Should return error for nil translation")
	}

	// Test empty singular value
	translations = map[string]*Translation{
		"key": {Singular: "", Plural: "plural"},
	}
	err = manager.AddTranslations(language.English, translations)
	if err == nil {
		t.Error("Should return error for empty singular value")
	}
}

func TestAddTranslationsWithExisting(t *testing.T) {
	manager := GetInstance()

	// Add initial translation
	err := manager.AddTranslation(language.Italian, "existing", "Existing", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Add more translations to the same language
	translations := map[string]*Translation{
		"new1": {Singular: "New1", Plural: ""},
		"new2": {Singular: "New2", Plural: ""},
	}
	err = manager.AddTranslations(language.Italian, translations)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify both existing and new translations exist
	if !manager.HasTranslation(language.Italian, "existing") {
		t.Error("Should still have existing translation")
	}
	if !manager.HasTranslation(language.Italian, "new1") {
		t.Error("Should have new1 translation")
	}
	if !manager.HasTranslation(language.Italian, "new2") {
		t.Error("Should have new2 translation")
	}
}

func TestAddTranslationsWithTemplates(t *testing.T) {
	manager := GetInstance()

	// Add translations with templates
	translations := map[string]*Translation{
		"welcome": {
			Singular: "Welcome {{.Name}}!",
			Plural:   "",
		},
		"files": {
			Singular: "You have {{.Count}} file",
			Plural:   "You have {{.Count}} files",
		},
	}

	err := manager.AddTranslations(language.Portuguese, translations)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test template execution
	data := map[string]interface{}{"Name": "João"}
	result := manager.Translate(language.Portuguese, "welcome", 1, data)
	if result != "Welcome João!" {
		t.Errorf("Expected 'Welcome João!', got '%s'", result)
	}

	// Test plural template
	data = map[string]interface{}{"Count": 5}
	result = manager.Translate(language.Portuguese, "files", 5, data)
	if result != "You have 5 files" {
		t.Errorf("Expected 'You have 5 files', got '%s'", result)
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := GetInstance()

	// Test concurrent read/write operations
	done := make(chan bool, 100)

	// Start multiple goroutines adding translations
	for i := 0; i < 50; i++ {
		go func(i int) {
			manager.AddTranslation(language.English, fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), "")
			done <- true
		}(i)
	}

	// Start multiple goroutines reading translations
	for i := 0; i < 50; i++ {
		go func(i int) {
			manager.TranslateSimple(language.English, fmt.Sprintf("key%d", i))
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}
