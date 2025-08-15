# VIX - Type-Safe Validation Library for Go

A modern, type-safe, and expressive validation library for Go that follows clean architecture principles. VIX integrates seamlessly with the ERM centralized error management package for unified error handling and **standard internationalization using go-i18n**.

## Features

- **Type-Safe Validation**: Leverage Go generics for compile-time type safety
- **Expressive API**: Fluent, chainable syntax for readable validation rules
- **Standard i18n**: Uses `github.com/nicksnyder/go-i18n/v2/i18n` through ERM integration
- **Function Chaining**: Readable and maintainable validation rules without struct tags
- **Conditional Validation**: Built-in support for conditional validation logic
- **Clean Architecture**: Integrates perfectly with onion/clean architecture patterns  
- **Unified Error Management**: All validation errors are ERM Error instances with automatic localization
- **Comprehensive Types**: Support for strings, numbers, dates, and custom validations
- **Performance Optimized**: Lazy evaluation and efficient validation chains

## Installation

```bash
go get github.com/c3p0-box/utils/vix
```

## Quick Start

### Basic Validation

```go
import "github.com/c3p0-box/utils/vix"

// Single field validation
err := vix.String("john@example.com", "email").
    Required().
    Email().
    MaxLength(100).
    Validate()

if err != nil {
    log.Printf("Email validation failed: %v", err)
}
```

### Multi-Field Validation

```go
// Multiple field validation with structured error output
validator := vix.Is(
    vix.String("", "email").Required().Email(),
    vix.String("123", "password").Required().MinLength(8),
    vix.Int(16, "age").Required().Min(18),
)

if !validator.Valid() {
    // Get structured error map suitable for API responses
    errorMap := validator.ErrMap()
    // Returns: {
    //   "email": ["email is required"],
    //   "password": ["password must be at least 8 characters long"], 
    //   "age": ["age must be at least 18"]
    // }
    
    jsonBytes, _ := validator.ToJSON()
    fmt.Println(string(jsonBytes))
}
```

## Internationalization

VIX uses ERM's i18n integration. See the [ERM README](../erm/README.md#internationalization-setup) for complete setup instructions.

**Quick Setup:**
```go
// No setup required - localization is automatic!

// All VIX validation errors are automatically localized to English
result := vix.String("", "email").Required().Result()
fmt.Println(result.Error().Error()) // "email is required"
```

## String Validation

```go
validator := vix.String("example@email.com", "email")

// Chain validation rules
err := validator.
    Required().              // Field must not be empty
    Email().                // Must be valid email format
    MinLength(5).           // Minimum 5 characters
    MaxLength(100).         // Maximum 100 characters
    Validate()              // Execute validation

// Or get the validation result
result := validator.
    Required().
    Email().
    Result()

if !result.Valid() {
    fmt.Println(result.Error()) // Localized error message
}
```

### Available String Validations

```go
vix.String(value, "fieldName").
    Required().                    // Must not be empty
    Empty().                      // Must be empty
    MinLength(5).                 // Minimum length
    MaxLength(100).               // Maximum length
    ExactLength(10).              // Exact length
    LengthBetween(5, 100).        // Length range
    Email().                      // Valid email format
    URL().                        // Valid URL format
    Numeric().                    // Contains only numbers
    Alpha().                      // Contains only letters
    AlphaNumeric().               // Contains only letters and numbers
    Regex(pattern).               // Matches regex pattern
    In("val1", "val2").          // Value must be in list
    NotIn("val1", "val2").       // Value must not be in list
    EqualTo("expected").         // Value must equal expected string (with optional custom message)
    Contains("substring").        // Must contain substring
    StartsWith("prefix").         // Must start with prefix
    EndsWith("suffix").           // Must end with suffix
    Lowercase().                  // Must be lowercase
    Uppercase().                  // Must be uppercase
    Integer().                    // Must be valid integer
    Float().                      // Must be valid float
    JSON().                       // Must be valid JSON
    Base64().                     // Must be valid base64
    UUID().                       // Must be valid UUID
    Slug()                        // Must be valid slug
```

## Number Validation

```go
// Integer validation
err := vix.Int(25, "age").
    Required().                   // Must not be zero
    Min(18).                     // Minimum value
    Max(100).                    // Maximum value
    Between(18, 65).             // Value range
    Positive().                  // Must be positive
    Even().                      // Must be even number
    Validate()

// Float validation
err = vix.Float64(3.14159, "pi").
    Required().
    Min(0.0).
    Max(10.0).
    Precision(2).                // Maximum 2 decimal places
    Finite().                    // Must be finite (not NaN/Inf)
    Validate()
```

### Available Number Validations

```go
vix.Int(value, "fieldName").       // For int validation
vix.Float64(value, "fieldName").   // For float64 validation

// Common validations for both
    Required().                    // Must not be zero
    Zero().                       // Must be zero
    Min(value).                   // Minimum value
    Max(value).                   // Maximum value
    Between(min, max).            // Value range
    Equal(expected).              // Must equal expected value
    GreaterThan(value).           // Must be greater than
    LessThan(value).              // Must be less than
    Positive().                   // Must be positive
    Negative().                   // Must be negative
    In(val1, val2).              // Must be in list
    NotIn(val1, val2).           // Must not be in list
    EqualTo(expected).           // Must equal expected value (with optional custom message)
    MultipleOf(divisor).          // Must be multiple of divisor

// Integer-specific
    Even().                       // Must be even
    Odd().                        // Must be odd

// Float-specific
    Finite().                     // Must be finite (not NaN/Inf)
    Precision(places).            // Maximum decimal places
```

## Conditional Validation

```go
// Validate phone only if email is empty
phoneValidator := vix.String(phone, "phone").
    When(func() bool { return email == "" }).
    Required().
    Regex(`^\d{10}$`)

// Skip validation based on condition
ageValidator := vix.Int(age, "age").
    Unless(func() bool { return isGuest }).
    Required().
    Min(18)
```

## Advanced Usage

### Custom Validation

```go
validator := vix.String("customValue", "field").
    Custom(func(value interface{}, fieldName string) error {
        str := value.(string)
        if !isValidCustomFormat(str) {
            return erm.NewValidationError("validation.custom_format", fieldName, str)
        }
        return nil
    })
```

### EqualTo with Custom Messages

```go
// String validation with default message
err := vix.String("john", "username").
    EqualTo("admin").  // Uses default error message
    Validate()

// String validation with custom message
err = vix.String("john", "username").
    EqualTo("admin", "{{field}} must be exactly '{{expected}}'").
    Validate()

// Number validation with custom message
err = vix.Int(25, "age").
    EqualTo(18, "{{field}} must be exactly {{expected}} years old").
    Validate()

// Works with negation too
err = vix.String("admin", "username").
    Not().EqualTo("root", "{{field}} cannot be '{{expected}}'").
    Validate()
```

### Negation

```go
// Use Not() to negate any validation
validator := vix.String("admin", "username").
    Not().In("admin", "root", "system") // Username must NOT be in this list
```

### Nested Validation

```go
// Validate nested structures
userValidator := vix.Is(
    vix.String(user.Name, "name").Required().MinLength(2),
    vix.String(user.Email, "email").Required().Email(),
)

addressValidator := vix.Is(
    vix.String(user.Address.Street, "address.street").Required(),
    vix.String(user.Address.City, "address.city").Required(),
)

// Combine validators
finalValidator := vix.Is(userValidator, addressValidator)
```

### Array/Slice Validation

```go
// Validate array elements
emails := []string{"user1@example.com", "invalid-email", "user2@example.com"}

validator := vix.V()
for i, email := range emails {
    fieldName := fmt.Sprintf("emails[%d]", i)
    validator = validator.Is(
        vix.String(email, fieldName).Required().Email(),
    )
}

if !validator.Valid() {
    errorMap := validator.ErrMap()
    // Returns: {"emails[1]": ["emails[1] must be a valid email address"]}
}
```

## Error Handling

### Single Field Errors

```go
result := vix.String("", "email").Required().Result()

if !result.Valid() {
    err := result.Error()
    fmt.Println(err.Error())     // Localized error message
    
    // Access ERM error details
    if ermErr, ok := err.(erm.Error); ok {
        fmt.Println(ermErr.FieldName())  // "email"
        fmt.Println(ermErr.MessageKey()) // "validation.required"
        fmt.Println(ermErr.Value())      // ""
    }
}
```

### Multi-Field Errors

```go
validator := vix.Is(
    vix.String("", "email").Required().Email(),
    vix.String("123", "password").Required().MinLength(8),
)

if !validator.Valid() {
    // Structured error map
    errorMap := validator.ErrMap()
    // Returns: {
    //   "email": ["email is required"],
    //   "password": ["password must be at least 8 characters long"]
    // }
    
    // JSON output for APIs
    jsonBytes, _ := validator.ToJSON()
    
    // Access all errors through the container error
    if err := validator.Error(); err != nil {
        if ermErr, ok := err.(erm.Error); ok {
            allErrors := ermErr.AllErrors()
            for _, err := range allErrors {
                fmt.Printf("Field: %s, Error: %s\n", 
                    err.(erm.Error).FieldName(), 
                    err.Error())
            }
        }
    }
}
```

### Custom Localization

```go
// Use specific language per request
result := vix.String("", "email").Required().Result()
if !result.Valid() {
    spanishMsg := result.Error().(erm.Error).LocalizedError(language.Spanish)
    fmt.Println(spanishMsg) // "email is required" (English fallback)
}
```

## Clean Architecture Integration

VIX integrates seamlessly with clean architecture patterns through unified ERM error handling:

```go
// Service Layer
func (s *userService) CreateUser(req CreateUserRequest) (*User, error) {
    validator := vix.Is(
        vix.String(req.Email, "email").Required().Email(),
        vix.String(req.Password, "password").Required().MinLength(8),
        vix.Int(req.Age, "age").Required().Min(18),
    )
    
    if !validator.Valid() {
        return nil, erm.BadRequest("Validation failed", validator.Error())
    }
    
    // Business logic...
    return s.userRepo.Create(user)
}

// Handler Layer - ERM provides structured error responses
func (h *userHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    user, err := h.userService.CreateUser(req)
    if err != nil {
        // ERM automatically formats validation errors for APIs
        erm.WriteJSONError(w, err) // See ERM documentation
        return
    }
    
    json.NewEncoder(w).Encode(user)
}
```

## API Reference

### Core Functions

- `String(value, fieldName string) *StringValidator` - Create string validator
- `Int(value int, fieldName string) *NumberValidator[int]` - Create int validator  
- `Float64(value float64, fieldName string) *NumberValidator[float64]` - Create float validator
- `Is(validators ...Validator) *ValidationOrchestrator` - Multi-field validation
- `V() *ValidationOrchestrator` - Create validation orchestrator

### Validator Interface

```go
type Validator interface {
    Valid() bool
    Error() error
    AllErrors() []error
    Result() *ValidationResult
}
```

### ValidationResult

```go
type ValidationResult struct {
    Value      interface{}
    FieldName  string
    IsValid    bool
}

func (vr *ValidationResult) Valid() bool
func (vr *ValidationResult) Error() error
func (vr *ValidationResult) AllErrors() []error
func (vr *ValidationResult) ErrMap() map[string][]string
func (vr *ValidationResult) ToJSON() ([]byte, error)
```

### ValidationOrchestrator

```go
// ValidationOrchestrator manages multiple field validations
func (vo *ValidationOrchestrator) Valid() bool
func (vo *ValidationOrchestrator) Error() error  // Returns single erm.Error containing all validation errors
func (vo *ValidationOrchestrator) ErrMap() map[string][]string
func (vo *ValidationOrchestrator) ToJSON() ([]byte, error)
```

**Note:** `ValidationOrchestrator.Error()` returns a single `erm.Error` that contains all validation errors as child errors. Use `err.(erm.Error).AllErrors()` to access individual errors if needed.

## Migration from Custom Locale System

The `WithLocale()` method has been removed. Use `erm.GetLocalizer(language.Tag)` to get localizers for different languages. See [ERM documentation](../erm/README.md#internationalization-setup) for details.

## Best Practices

1. **Use localized validation**: Get localizers using `erm.GetLocalizer(language.Tag)` for different languages
2. **Use structured validation**: Prefer `vix.Is()` for multi-field validation
3. **Validate at boundaries**: Validate input at service layer boundaries
4. **Consistent field names**: Use consistent field naming across your application
5. **Custom validators**: Create reusable custom validators for domain-specific rules
6. **Error collection**: Use error collection for batch validation scenarios
7. **Conditional validation**: Use `When()` and `Unless()` for complex business rules

## Performance Considerations

- **Lazy evaluation**: Validation chains are only executed when needed
- **Memory efficient**: Validators reuse internal structures where possible
- **Minimal allocations**: Careful design to minimize memory allocations
- **Early termination**: Validation stops at first failure for single-field validation
- **Batch processing**: Multi-field validation collects all errors efficiently

## Testing

```bash
go test -v ./vix
go test -bench=. ./vix
```

## License

This package is part of the c3p0-box/utils collection.