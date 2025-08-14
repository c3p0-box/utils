# ERM - Enhanced Error Management for Go

A comprehensive error management package for Go applications following clean architecture patterns. ERM enriches errors with stack traces, HTTP status codes, safe user-facing messages, validation error capabilities, and error collection support, while providing **standard internationalization through go-i18n**.

## Features

- **Enhanced Errors**: Stack traces (for 500 errors), HTTP status codes
- **Performance Optimized**: Stack traces only captured for Internal Server Errors (500) where debugging is needed
- **Validation Support**: Unified validation errors with field-level granularity  
- **Error Collection**: Collect multiple related errors with automatic flattening
- **Standard i18n**: Uses `github.com/nicksnyder/go-i18n/v2/i18n` for internationalization
- **On-Demand Localization**: Messages resolved when `Error()` or `ErrMap()` called
- **Per-Language Localizers**: Automatic creation and caching of localizers for different languages
- **Clean Architecture**: Interface-based design for testability and flexibility
- **Memory Efficient**: Immutable error objects safe for concurrent access

## Installation

```bash
go get github.com/c3p0-box/utils/erm
```

## Quick Start

### Basic Error Creation

```go
import "github.com/c3p0-box/utils/erm"

// Create client errors (no stack trace for performance)
err := erm.New(http.StatusBadRequest, "Invalid email", originalErr)
err := erm.BadRequest("Invalid input", originalErr)

// Create server errors (with stack trace for debugging)
err := erm.New(http.StatusInternalServerError, "Database connection failed", dbErr)
err := erm.Internal("Database connection failed", dbErr)

// Create errors without underlying error (err parameter can be nil)
err := erm.New(http.StatusBadRequest, "Custom validation message", nil)
// err.Unwrap() returns nil, but err.Error() returns "Custom validation message"
```

### Stack Trace Behavior

```go
// Client errors (4xx) - no stack trace for performance
clientErr := erm.BadRequest("Invalid input", nil)
fmt.Println(len(clientErr.Stack()) == 0) // true - no stack trace

// Server errors (500) - stack trace captured for debugging
serverErr := erm.Internal("Database error", dbErr)
fmt.Println(len(serverErr.Stack()) > 0) // true - stack trace available

// Zero code defaults to 500 and captures stack trace
defaultErr := erm.New(0, "Something went wrong", nil)
fmt.Println(len(defaultErr.Stack()) > 0) // true - becomes 500 error
```

### Validation Errors with i18n

```go
// Create validation errors with message keys for i18n
err := erm.NewValidationError("validation.required", "email", "")
err = err.WithParam("min", 5)

// Use convenience constructors
err := erm.RequiredError("email", "")
err := erm.MinLengthError("password", "123", 8)
err := erm.EmailError("email", "invalid-email")
err := erm.DuplicateError("email", "user@example.com")
err := erm.InvalidError("format", "invalid-data")
err := erm.NotFound("user", dbErr) // 404 error with message key "error.not_found"
```

### Internationalization Setup

```go
import (
    "github.com/nicksnyder/go-i18n/v2/i18n"
    "golang.org/x/text/language"
)

// Get localizers for different languages (created automatically)
englishLocalizer := erm.GetLocalizer(language.English)
spanishLocalizer := erm.GetLocalizer(language.Spanish)

// For advanced usage with custom message files:
bundle := i18n.NewBundle(language.English)
bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
bundle.LoadMessageFile("locales/en.json")
bundle.LoadMessageFile("locales/es.json")

// Create custom localizer
customLocalizer := i18n.NewLocalizer(bundle, "es")
```

### Message File Format

Create message files in standard go-i18n format:

**locales/en.json**
```json
{
  "validation.required": {
    "other": "{{.field}} is required"
  },
  "validation.min_length": {
    "other": "{{.field}} must be at least {{.min}} characters long"
  },
  "validation.email": {
    "other": "{{.field}} must be a valid email address"
  },
  "error.not_found": {
    "other": "{{.field}} not found"
  }
}
```

**locales/es.json**
```json
{
  "validation.required": {
    "other": "{{.field}} es requerido"
  },
  "validation.min_length": {
    "other": "{{.field}} debe tener al menos {{.min}} caracteres"
  },
  "validation.email": {
    "other": "{{.field}} debe ser una dirección de email válida"
  },
  "error.not_found": {
    "other": "{{.field}} no encontrado"
  }
}
```

### On-Demand Localization

```go
// Default localization (uses English by default)
err := erm.RequiredError("email", "")
fmt.Println(err.Error()) // "email is required"

// Custom localization with different languages
localizedMsg := err.LocalizedError(language.Spanish)
fmt.Println(localizedMsg) // Currently returns English since Spanish bundle uses English fallback

// For custom languages, extend the GetLocalizer system with your own bundles
// The system will automatically use English fallback for any language
spanishMsg := err.LocalizedError(language.Spanish)
fmt.Println(spanishMsg) // "email is required" (English fallback)

// Structured errors for APIs
errorMap := err.LocalizedErrMap(language.Spanish)
// Returns: {"email": ["email is required"]}

// Convenience method using default English localizer
errorMap = err.ErrMap()
// Returns: {"email": ["email is required"]}
```

### Error Collection

```go
// Collect multiple validation errors one by one
container := erm.New(http.StatusBadRequest, "Validation errors", nil)

err1 := erm.RequiredError("email", "")
err2 := erm.MinLengthError("password", "123", 8)

container.AddError(err1)
container.AddError(err2)

// Or collect multiple errors at once using AddErrors
container2 := erm.New(http.StatusBadRequest, "Validation errors", nil)
errors := []erm.Error{
    erm.RequiredError("email", ""),
    erm.MinLengthError("password", "123", 8),
    erm.EmailError("email", "invalid-email"),
}
container2.AddErrors(errors)

// Get localized error map for API responses
errorMap := container.ErrMap() // Uses English
// Or with specific language
errorMap = container.LocalizedErrMap(language.Spanish)

// Format: {"email": ["email is required"], "password": ["password must be at least 8 characters long"]}
```

## Migration from SetLocalizer System

The old `SetLocalizer`/`GetDefaultLocalizer` API has been replaced with `GetLocalizer(language.Tag)`:

### Before:
```go
// Old API (deprecated)
bundle := i18n.NewBundle(language.English)
// ... add messages to bundle ...
localizer := i18n.NewLocalizer(bundle, "en")
erm.SetLocalizer(localizer)
```

### After:
```go
// New API
englishLocalizer := erm.GetLocalizer(language.English)
spanishLocalizer := erm.GetLocalizer(language.Spanish)
```

## API Reference

### Core Functions

- `New(code int, msg string, err error) Error` - Create enriched error (stack traces only for 500 errors)  
- `GetLocalizer(tag language.Tag) *i18n.Localizer` - Get or create localizer for language

### Validation Constructors

- `NewValidationError(messageKey, fieldName string, value interface{}) Error`
- `RequiredError(fieldName string, value interface{}) Error`
- `MinLengthError(fieldName string, value interface{}, min int) Error`
- `EmailError(fieldName string, value interface{}) Error`
- `DuplicateError(fieldName string, value interface{}) Error`
- `InvalidError(fieldName string, value interface{}) Error`
- `NotFound(fieldName string, err error) Error` - Creates 404 error with "error.not_found" message key

### Error Interface

```go
type Error interface {
    Error() string                              // Localized with English
    LocalizedError(language.Tag) string        // Localized with specific language
    LocalizedErrMap(language.Tag) map[string][]string
    
    MessageKey() string                         // i18n message key
    FieldName() string
    Value() interface{}
    Params() map[string]interface{}
    
    AddError(Error)                             // Error collection (mutable)
    AddErrors([]Error)                          // Batch error collection (mutable)
    ErrMap() map[string][]string                // Structured errors (uses English localizer)
    // ... other methods
}
```

## VIX Integration

ERM powers VIX validation. See [VIX documentation](../vix/README.md) for validation usage.

## Best Practices

1. **Use per-language localizers**: Get localizers using `erm.GetLocalizer(language.Tag)` for different languages
2. **Use message keys**: Prefer message keys over hardcoded templates for better maintainability
3. **Organize message files**: Group related validations and use consistent key naming
4. **Error collection**: Use error collection for batch validation scenarios
5. **Structured output**: Use `ErrMap()` for API error responses
6. **Performance-aware error codes**: Use appropriate HTTP status codes - stack traces are only captured for 500 errors to optimize performance for client errors

## License

This package is part of the c3p0-box/utils collection.