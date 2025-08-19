# Lightweight Internationalization (i18n) Library

A simple, thread-safe internationalization library for Go that provides translation management with template support and pluralization.

## Features

- **Singleton Pattern**: Global instance for easy access across your application
- **Language Support**: Uses `golang.org/x/text/language.Tag` for robust language handling
- **Template Support**: Go templates for dynamic content injection
- **Pluralization**: Automatic singular/plural form selection based on count
- **Thread-Safe**: Safe for concurrent access
- **Minimal Dependencies**: Only Go standard library + `golang.org/x/text`
- **Fallback Support**: Automatic fallback to default language
- **Template Caching**: Efficient template parsing and caching

## Installation

```bash
go get github.com/c3p0-box/utils/i18n
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/c3p0-box/utils/i18n"
    "golang.org/x/text/language"
)

func main() {
    // Set default language
    i18n.SetDefaultLanguage(language.English)

    // Add translations
    i18n.AddTranslation(language.English, "hello", "Hello", "")
    i18n.AddTranslation(language.Spanish, "hello", "Hola", "")

    // Translate
    fmt.Println(i18n.TranslateSimple(language.English, "hello")) // Output: Hello
    fmt.Println(i18n.TranslateSimple(language.Spanish, "hello")) // Output: Hola
}
```

## API Reference

### Core Functions

#### `SetDefaultLanguage(lang language.Tag)`
Sets the default fallback language.

#### `AddTranslation(lang language.Tag, key, value, plural string) error`
Adds a translation for the specified language and key.
- `lang`: Language tag
- `key`: Translation key
- `value`: Singular form (can be a Go template)
- `plural`: Plural form (optional, uses singular if empty)

#### `AddTranslations(lang language.Tag, translations map[string]*Translation) error`
Adds multiple translations for the specified language at once. More efficient than calling `AddTranslation` multiple times.
- `lang`: Language tag
- `translations`: Map where keys are translation keys and values are Translation structs

#### `Translate(lang language.Tag, key string, count int, data interface{}) string`
Retrieves and processes a translation with template data.
- `lang`: Target language
- `key`: Translation key
- `count`: Determines singular (1) vs plural (≠1) form
- `data`: Template data (can be nil)

#### `TranslateSimple(lang language.Tag, key string) string`
Simple translation without templates or count.

#### `TranslatePlural(lang language.Tag, key string, count int) string`
Translation with plural support but without template data.

### Manager Instance Methods

#### `GetInstance() *Manager`
Returns the singleton manager instance.

#### `(*Manager) GetAvailableLanguages() []language.Tag`
Returns all languages that have translations.

#### `(*Manager) GetTranslationKeys(lang language.Tag) []string`
Returns all translation keys for a given language.

#### `(*Manager) HasTranslation(lang language.Tag, key string) bool`
Checks if a translation exists.

## Usage Examples

### Basic Translation

```go
// Add simple translations
i18n.AddTranslation(language.English, "welcome", "Welcome!", "")
i18n.AddTranslation(language.French, "welcome", "Bienvenue!", "")

// Translate
msg := i18n.TranslateSimple(language.French, "welcome")
fmt.Println(msg) // Output: Bienvenue!
```

### Template Usage

```go
// Add translation with template
i18n.AddTranslation(language.English, "user_info", 
    "Hello {{.Name}}, you have {{.Messages}} message", 
    "Hello {{.Name}}, you have {{.Messages}} messages")

// Use with data
data := map[string]interface{}{
    "Name": "Alice",
    "Messages": 3,
}
msg := i18n.Translate(language.English, "user_info", 3, data)
fmt.Println(msg) // Output: Hello Alice, you have 3 messages
```

### Pluralization

```go
// Add translation with plural forms
i18n.AddTranslation(language.English, "file_count",
    "{{.Count}} file", "{{.Count}} files")

// Singular
data := map[string]interface{}{"Count": 1}
msg := i18n.Translate(language.English, "file_count", 1, data)
fmt.Println(msg) // Output: 1 file

// Plural
data = map[string]interface{}{"Count": 5}
msg = i18n.Translate(language.English, "file_count", 5, data)
fmt.Println(msg) // Output: 5 files
```

### Bulk Translation Addition

```go
// Create multiple translations at once
translations := map[string]*i18n.Translation{
    "hello": {Singular: "Hello", Plural: ""},
    "goodbye": {Singular: "Goodbye", Plural: ""},
    "item_count": {
        Singular: "{{.Count}} item", 
        Plural: "{{.Count}} items",
    },
}

// Add all translations for English at once
err := i18n.AddTranslations(language.English, translations)
if err != nil {
    log.Fatal(err)
}

// Use the translations
fmt.Println(i18n.TranslateSimple(language.English, "hello")) // Output: Hello
data := map[string]interface{}{"Count": 5}
fmt.Println(i18n.Translate(language.English, "item_count", 5, data)) // Output: 5 items
```

### Fallback Behavior

```go
// Set default language
i18n.SetDefaultLanguage(language.English)

// Add only English translation
i18n.AddTranslation(language.English, "error", "An error occurred", "")

// Request German translation - falls back to English
msg := i18n.TranslateSimple(language.German, "error")
fmt.Println(msg) // Output: An error occurred

// Request non-existing key - returns the key itself
msg = i18n.TranslateSimple(language.English, "non_existing")
fmt.Println(msg) // Output: non_existing
```

### Working with Manager Instance

```go
manager := i18n.GetInstance()

// Add translation using manager
manager.AddTranslation(language.Japanese, "thanks", "ありがとう", "")

// Check what languages are available
languages := manager.GetAvailableLanguages()
fmt.Printf("Available: %v\n", languages)

// Get all keys for a language
keys := manager.GetTranslationKeys(language.Japanese)
fmt.Printf("Japanese keys: %v\n", keys)

// Check if translation exists
exists := manager.HasTranslation(language.Japanese, "thanks")
fmt.Printf("Exists: %v\n", exists)
```

## Migration from go-i18n/v2

If you're migrating from `github.com/nicksnyder/go-i18n/v2`, here are the key differences:

### Old (go-i18n/v2):
```go
bundle := i18n.NewBundle(language.English)
localizer := i18n.NewLocalizer(bundle, "en")
msg := localizer.Localize(&i18n.LocalizeConfig{
    MessageID: "welcome",
    TemplateData: map[string]interface{}{"Name": "Alice"},
})
```

### New (this library):
```go
i18n.SetDefaultLanguage(language.English)
i18n.AddTranslation(language.English, "welcome", "Welcome {{.Name}}", "")
msg := i18n.Translate(language.English, "welcome", 1, map[string]interface{}{"Name": "Alice"})
```

## Thread Safety

This library is fully thread-safe. All operations use appropriate mutexes to ensure safe concurrent access.

## Performance

- **Template Caching**: Templates are parsed once and cached for reuse
- **Efficient Lookups**: Fast map-based lookups for translations
- **Minimal Allocations**: Optimized for low memory overhead

## Limitations

- Simple pluralization (singular/plural only, no complex plural rules)
- No message loading from files (programmatic API only)
- No ICU MessageFormat support (Go templates only)

## License

This library is part of the c3p0-box/utils package and follows the same license terms.
