# Vix Validator Package

A type-safe, expressive, and extensible validation library for Go that follows clean architecture principles and integrates seamlessly with the ERM error management package.

## Features

- **Type-safe validation** with Go generics
- **Function chaining** for readable validation rules
- **Multi-field validation** with comprehensive error reporting
- **Namespace support** for nested structures
- **Indexed namespace support** for arrays and slices
- **Internationalization** with template-based messages
- **Integration with ERM** error management
- **Custom validators** and extensibility
- **Conditional validation** with When/Unless
- **Logical operators** (And, Or, Not)
- **JSON error output** for API responses
- **No external dependencies** - uses only Go standard library

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/c3p0-box/utils/vix"
)

func main() {
    // String validation
    err := vix.String("john@example.com", "email").
        Required().
        Email().
        MaxLength(100).
        Validate()
    
    if err != nil {
        fmt.Printf("Email validation failed: %v\n", err)
    }
    
    // Number validation
    err = vix.Int(25, "age").
        Required().
        Min(18).
        Max(100).
        Validate()
    
    if err != nil {
        fmt.Printf("Age validation failed: %v\n", err)
    }
}
```

### Multi-Field Validation

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/c3p0-box/utils/vix"
)

func main() {
    // Validate multiple fields at once
    val := vix.Is(
        vix.String("Bob", "full_name").Required().LengthBetween(4, 20),
        vix.Int(17, "age").GreaterThan(18),
        vix.String("singl", "status").In("married", "single"),
    )

    if !val.Valid() {
        // Get structured error output
        errorMap := val.ToError()
        out, _ := json.MarshalIndent(errorMap, "", "  ")
        fmt.Println(string(out))
        
        // Output:
        // {
        //   "age": [
        //     "age must be greater than 18"
        //   ],
        //   "full_name": [
        //     "full_name must have a length between 4 and 20"
        //   ],
        //   "status": [
        //     "status must be one of: married, single"
        //   ]
        // }
    }
}
```

### Field-Specific Validation Check

```go
val := vix.Is(
    vix.Int(16, "age").GreaterThan(18),
    vix.String("single", "status").In("married", "single"),
)

if !val.IsValid("age") {
    fmt.Println("Warning: someone underage is trying to sign up")
}
```

### Advanced Usage

```go
func validateUserRegistration(user User) error {
    // Email validation
    if err := vix.String(user.Email, "email").
        Required().
        Email().
        MaxLength(255).
        Validate(); err != nil {
        return err
    }
    
    // Password validation
    if err := vix.String(user.Password, "password").
        Required().
        MinLength(8).
        MaxLength(128).
        Regex(passwordRegex).
        Validate(); err != nil {
        return err
    }
    
    // Age validation with conditions
    if err := vix.Int(user.Age, "age").
        When(func() bool { return user.Type == "adult" }).
        Required().
        Min(18).
        Max(100).
        Validate(); err != nil {
        return err
    }
    
    return nil
}
```

## Multi-Field Validation

The vix package provides comprehensive multi-field validation capabilities, allowing you to validate multiple fields at once and get structured error output. You can use the simplified package-level functions `vix.Is()`, `vix.In()`, and `vix.InRow()` for easy multi-field validation, or use the explicit `vix.V().Is()` syntax for more complex scenarios.

### Basic Multi-Field Validation

```go
// Create a validation orchestrator
val := vix.Is(
    vix.String("test@example.com", "email").Required().Email(),
    vix.Int(25, "age").Required().Min(18),
    vix.String("John", "name").Required().MinLength(2),
)

// Check overall validity
if val.Valid() {
    fmt.Println("All validations passed")
} else {
    fmt.Println("Some validations failed")
}

// Check specific field validity
if !val.IsValid("email") {
    fmt.Println("Email validation failed")
}
```

### Structured Error Output

```go
val := vix.Is(
    vix.String("", "email").Required(),
    vix.Int(16, "age").Required().Min(18),
)

if !val.Valid() {
    // Get error map
    errorMap := val.ToError()
    
    // Convert to JSON
    jsonBytes, _ := val.ToJSON()
    fmt.Println(string(jsonBytes))
    
    // Output:
    // {
    //   "email": [
    //     "email is required"
    //   ],
    //   "age": [
    //     "age must be at least 18"
    //   ]
    // }
}
```

### Namespace Validation (Nested Structures)

The `In()` method allows you to validate nested structures with namespaced field names:

```go
type Address struct {
    Name   string
    Street string
}

type Person struct {
    Name    string
    Address Address
}

p := Person{"Bob", Address{"", "1600 Amphitheatre Pkwy"}}

val := vix.Is(vix.String(p.Name, "name").LengthBetween(4, 20)).
    In("address", vix.Is(
        vix.String(p.Address.Name, "name").Required(),
        vix.String(p.Address.Street, "street").Required(),
    ))

if !val.Valid() {
    errorMap := val.ToError()
    out, _ := json.MarshalIndent(errorMap, "", "  ")
    fmt.Println(string(out))
    
    // Output:
    // {
    //   "address.name": [
    //     "name is required"
    //   ],
    //   "name": [
    //     "name must have a length between 4 and 20"
    //   ]
    // }
}
```

### Indexed Namespace Validation (Arrays and Slices)

The `InRow()` method allows you to validate arrays and slices with indexed namespaces:

```go
type Address struct {
    Name   string
    Street string
}

type Person struct {
    Name      string
    Addresses []Address
}

p := Person{
    "Bob",
    []Address{
        {"", "1600 Amphitheatre Pkwy"},
        {"Home", ""},
    },
}

val := vix.Is(vix.String(p.Name, "name").LengthBetween(4, 20))

for i, a := range p.Addresses {
    val.InRow("addresses", i, vix.Is(
        vix.String(a.Name, "name").Required(),
        vix.String(a.Street, "street").Required(),
    ))
}

if !val.Valid() {
    errorMap := val.ToError()
    out, _ := json.MarshalIndent(errorMap, "", "  ")
    fmt.Println(string(out))
    
    // Output:
    // {
    //   "addresses[0].name": [
    //     "name is required"
    //   ],
    //   "addresses[1].street": [
    //     "street is required"
    //   ],
    //   "name": [
    //     "name must have a length between 4 and 20"
    //   ]
    // }
}
```

### Working with Validation Results

```go
val := vix.Is(
    vix.String("test@example.com", "email").Required().Email(),
    vix.String("", "name").Required(),
    vix.Int(25, "age").Required().Min(18),
)

// Get all field names
fieldNames := val.FieldNames()
fmt.Printf("Validated fields: %v\n", fieldNames)

// Get specific field result
emailResult := val.GetFieldResult("email")
if emailResult != nil && emailResult.Valid() {
    fmt.Println("Email is valid")
}

// Get first error
if err := val.Error(); err != nil {
    fmt.Printf("First error: %v\n", err)
}

// Get all errors
errors := val.AllErrors()
for _, err := range errors {
    fmt.Printf("Error: %v\n", err)
}

// String representation
fmt.Println(val.String())
```

### Multi-Field Validation with Localization

```go
val := vix.Is(
    vix.String("", "email").Required(),
    vix.Int(16, "age").Required().Min(18),
).WithLocale(vix.SpanishLocale)

if !val.Valid() {
    errorMap := val.ToError()
    out, _ := json.MarshalIndent(errorMap, "", "  ")
    fmt.Println(string(out))
    
    // Output:
    // {
    //   "email": [
    //     "email es requerido"
    //   ],
    //   "age": [
    //     "age debe ser al menos 18"
    //   ]
    // }
}
```

## String Validation

### Basic String Rules

```go
vix.String(value, "field_name").
    Required().              // Must not be empty
    Empty().                 // Must be empty
    MinLength(5).            // Minimum length
    MaxLength(100).          // Maximum length
    ExactLength(10).         // Exact length
    LengthBetween(5, 50).    // Length within range
    Validate()
```

### Format Validation

```go
vix.String(value, "field_name").
    Email().                 // Valid email format
    URL().                   // Valid URL format
    Numeric().               // Contains only numbers
    Alpha().                 // Contains only letters
    AlphaNumeric().          // Contains only letters and numbers
    UUID().                  // Valid UUID format
    Slug().                  // Valid URL slug
    JSON().                  // Valid JSON format
    Base64().                // Valid base64 encoding
    Validate()
```

### String Content Rules

```go
vix.String(value, "field_name").
    Contains("substring").   // Must contain substring
    StartsWith("prefix").    // Must start with prefix
    EndsWith("suffix").      // Must end with suffix
    Lowercase().             // Must be lowercase
    Uppercase().             // Must be uppercase
    Regex(pattern).          // Must match regex pattern
    In("option1", "option2"). // Must be one of options
    NotIn("bad1", "bad2").   // Must not be one of options
    Validate()
```

### Advanced String Validation

```go
vix.String(value, "field_name").
    Integer().               // Must be a valid integer string
    Float().                 // Must be a valid float string
    Validate()
```

## Number Validation

### Generic Number Validation

```go
// Works with any numeric type
vix.Numeric(42, "count").
    Required().              // Must not be zero
    Min(1).                  // Minimum value
    Max(100).                // Maximum value
    Between(1, 100).         // Value within range
    Positive().              // Must be positive
    Negative().              // Must be negative
    Even().                  // Must be even
    Odd().                   // Must be odd
    MultipleOf(5).           // Must be multiple of value
    Validate()
```

### Type-Specific Number Validation

```go
// Type-specific validators
vix.Int(42, "count").Min(0).Validate()
validator.Float64(3.14, "pi").Between(0.0, 10.0).Validate()
validator.Uint8(255, "byte_value").Max(255).Validate()
```

### Advanced Number Rules

```go
vix.Float64(value, "field_name").
    Finite().                // Must be finite (not NaN or Inf)
    Precision(2).            // Max decimal places
    In(1, 2, 3, 5, 8).       // Must be one of values
    NotIn(0, -1).            // Must not be one of values
    Validate()
```

## Conditional Validation

### When/Unless Conditions

```go
// Validate only when condition is true
vix.String(value, "field_name").
    When(func() bool { return someCondition }).
    Required().
    Validate()

// Validate only when condition is false
vix.String(value, "field_name").
    Unless(func() bool { return someCondition }).
    Required().
    Validate()
```

### Complex Conditional Logic

```go
func validateUser(user User) error {
    // Email required only for online users
    if err := vix.String(user.Email, "email").
        When(func() bool { return user.Type == "online" }).
        Required().
        Email().
        Validate(); err != nil {
        return err
    }
    
    // Phone required only when email is empty
    if err := vix.String(user.Phone, "phone").
        When(func() bool { return user.Email == "" }).
        Required().
        MinLength(10).
        Validate(); err != nil {
        return err
    }
    
    return nil
}
```

## Negation with Not()

```go
// Reverse the validation logic
vix.String(value, "field_name").
    Not().Required().        // Must be empty
    Validate()

vix.String(value, "field_name").
    Not().Email().           // Must not be a valid email
    Validate()

vix.Int(value, "field_name").
    Not().Between(10, 20).   // Must not be between 10 and 20
    Validate()
```

## Custom Validation

### Custom Validation Functions

```go
func validatePassword(password string) error {
    return vix.String(password, "password").
        Required().
        MinLength(8).
        Custom(func(value interface{}) error {
            pwd := value.(string)
            if !hasUppercase(pwd) {
                return vix.NewValidationError(
                    "no_uppercase",
                    "{{field}} must contain at least one uppercase letter",
                    "password",
                    pwd,
                )
            }
            return nil
        }).
        Validate()
}
```

### Creating Custom Validators

```go
// Custom validator for credit card numbers
type CreditCardValidator struct {
    *vix.BaseValidator
}

func CreditCard(value string, fieldName string) *CreditCardValidator {
    return &CreditCardValidator{
        BaseValidator: vix.NewBaseValidator(value, fieldName),
    }
}

func (ccv *CreditCardValidator) Valid() *CreditCardValidator {
    // Implement Luhn algorithm validation
    if !ccv.shouldValidate() {
        return ccv
    }
    
    // Custom validation logic here
    if !isValidCreditCard(ccv.value.(string)) {
        ccv.addValidationError("invalid_credit_card", "{{field}} is not a valid credit card number", nil)
    }
    
    return ccv
}
```

## Internationalization

### Using Different Locales

```go
// Default English
err := vix.String("", "name").Required().Validate()
// Output: name is required

// Spanish
result := vix.String("", "nombre").Required().Result().WithLocale(vix.SpanishLocale)
err = result.Error()
// Output: nombre es requerido

// French
result = vix.String("", "nom").Required().Result().WithLocale(vix.FrenchLocale)
err = result.Error()
// Output: nom est requis

// German
result = vix.String("", "name").Required().Result().WithLocale(vix.GermanLocale)
err = result.Error()
// Output: name ist erforderlich
```

### Creating Custom Locales

```go
// Create a custom locale
customLocale := vix.NewLocale("es-MX", "Mexican Spanish")
customLocale.SetMessage(vix.CodeRequired, "{{field}} es obligatorio")
customLocale.SetMessage(vix.CodeEmail, "{{field}} debe ser un email válido")

// Use custom locale
result := vix.String("", "email").Required().Result().WithLocale(customLocale)
err = result.Error()
// Output: email es obligatorio
```

### Custom Error Messages

```go
// Custom message for specific validation
err := vix.String("", "username").
    Required().
    WithCustomMessage("Please provide a username").
    Validate()

// Template-based custom messages
err := vix.String("", "email").
    Required().
    WithTemplate("{{field}} is mandatory for registration").
    Validate()
```

## Integration with ERM Package

The validator package integrates seamlessly with the ERM error management package:

```go
func CreateUser(user User) (*User, error) {
    // Validate user data
    if err := vix.String(user.Email, "email").
        Required().
        Email().
        MaxLength(255).
        Validate(); err != nil {
        // ERM integration - wrap validation error
        return nil, erm.BadRequest("Invalid user data", err)
    }
    
    // Business logic here
    createdUser, err := userRepository.Create(user)
    if err != nil {
        return nil, erm.Internal("Failed to create user", err)
    }
    
    return createdUser, nil
}
```

### Multi-Field Validation with ERM

```go
func CreateUserWithMultiValidation(user User) (*User, error) {
    // Validate all fields at once
    val := vix.Is(
        vix.String(user.Email, "email").Required().Email().MaxLength(255),
        vix.String(user.Password, "password").Required().MinLength(8),
        vix.Int(user.Age, "age").Required().Min(18),
    )
    
    if !val.Valid() {
        // Create structured error response
        errorMap := val.ToError()
        jsonBytes, _ := val.ToJSON()
        
        return nil, erm.BadRequest("Validation failed", 
            fmt.Errorf("validation errors: %s", string(jsonBytes)))
    }
    
    // Business logic here
    createdUser, err := userRepository.Create(user)
    if err != nil {
        return nil, erm.Internal("Failed to create user", err)
    }
    
    return createdUser, nil
}
```

### HTTP Status Codes

```go
// Set custom HTTP status for validation errors
result := vix.String("", "admin_key").
    Required().
    Result().
    WithHTTPStatus(http.StatusUnauthorized)

if err := result.Error(); err != nil {
    // This error will have HTTP status 401
    return erm.Unauthorized("Admin key required", err)
}
```

## Validation Results

### Working with Results

```go
// Get full validation result
result := vix.String("invalid-email", "email").
    Required().
    Email().
    Result()

// Check if validation passed
if result.Valid() {
    fmt.Println("Validation passed")
} else {
    fmt.Println("Validation failed")
    
    // Get first error
    if err := result.Error(); err != nil {
        fmt.Printf("First error: %v\n", err)
    }
    
    // Get all errors
    for _, err := range result.AllErrors() {
        fmt.Printf("Error: %v\n", err)
    }
}
```

### Individual Field Error Mapping

The `ToError()` method is available for individual field validations and returns a `map[string][]string` structure, making it easy to integrate with API responses:

```go
// Single field validation with structured error output
result := vix.String("", "email").Required().Email().Result()
errorMap := result.ToError()

if errorMap != nil {
    // Convert to JSON for API response
    jsonBytes, _ := json.Marshal(errorMap)
    fmt.Println(string(jsonBytes))
    
    // Output:
    // {
    //   "email": [
    //     "email is required"
    //   ]
    // }
}

// Multiple errors on a single field
result = vix.String("a", "password").
    Required().
    MinLength(8).
    MaxLength(128).
    Result()

errorMap = result.ToError()
if errorMap != nil {
    jsonBytes, _ := json.Marshal(errorMap)
    fmt.Println(string(jsonBytes))
    
    // Output:
    // {
    //   "password": [
    //     "password must be at least 8 characters"
    //   ]
    // }
}

// Number validation with error mapping
result = vix.Int(0, "age").Required().Min(18).Result()
errorMap = result.ToError()

if errorMap != nil {
    jsonBytes, _ := json.Marshal(errorMap)
    fmt.Println(string(jsonBytes))
    
    // Output:
    // {
    //   "age": [
    //     "age is required"
    //   ]
    // }
}
```

### Localized Error Mapping

The `ToError()` method works seamlessly with localization:

```go
// Spanish localized errors
result := vix.String("", "email").Required().Result().WithLocale(vix.SpanishLocale)
errorMap := result.ToError()

if errorMap != nil {
    jsonBytes, _ := json.Marshal(errorMap)
    fmt.Println(string(jsonBytes))
    
    // Output:
    // {
    //   "email": [
    //     "email es requerido"
    //   ]
    // }
}
```

### Comparison: Individual vs Multi-Field Validation

```go
// Individual field validation
func validateEmailIndividually(email string) map[string][]string {
    result := vix.String(email, "email").Required().Email().Result()
    return result.ToError() // Returns nil if valid
}

// Multi-field validation
func validateUserFields(user User) map[string][]string {
    val := vix.Is(
        vix.String(user.Email, "email").Required().Email(),
        vix.String(user.Password, "password").Required().MinLength(8),
        vix.Int(user.Age, "age").Required().Min(18),
    )
    return val.ToError() // Returns nil if all fields valid
}

// Usage
emailErrors := validateEmailIndividually("invalid-email")
userErrors := validateUserFields(user)

// Both return the same map[string][]string structure
```

### Collecting Multiple Validation Errors

```go
func validateUserData(user User) []error {
    var errors []error
    
    // Validate email
    if err := vix.String(user.Email, "email").Required().Email().Validate(); err != nil {
        errors = append(errors, err)
    }
    
    // Validate password
    if err := vix.String(user.Password, "password").Required().MinLength(8).Validate(); err != nil {
        errors = append(errors, err)
    }
    
    // Validate age
    if err := vix.Int(user.Age, "age").Required().Min(18).Validate(); err != nil {
        errors = append(errors, err)
    }
    
    return errors
}
```

### Using Multi-Field Validation (Recommended)

```go
func validateUserDataWithMultiValidator(user User) map[string][]string {
    val := vix.Is(
        vix.String(user.Email, "email").Required().Email(),
        vix.String(user.Password, "password").Required().MinLength(8),
        vix.Int(user.Age, "age").Required().Min(18),
    )
    
    return val.ToError() // Returns nil if valid, error map if invalid
}
```

## Performance Considerations

### Efficient Validation

```go
// Use early returns for performance
func validateUser(user User) error {
    // Quick checks first
    if user.Email == "" {
        return vix.String("", "email").Required().Validate()
    }
    
    // More expensive validations later
    if !isValidEmailDomain(user.Email) {
        return validator.NewValidationError("invalid_domain", "{{field}} domain is not allowed", "email", user.Email)
    }
    
    return nil
}
```

### Reusable Validation Rules

```go
// Create reusable validation rules
var (
    EmailRules = []vix.Rule{
        validator.Required(),
        validator.Email(),
        validator.MaxLength(255),
    }
    
    PasswordRules = []vix.Rule{
        vix.Required(),
        vix.MinLength(8),
        vix.MaxLength(128),
        vix.Regex(passwordRegex),
    }
)

func validateUser(user User) error {
    if err := vix.String(user.Email, "email").Apply(EmailRules...).Validate(); err != nil {
        return err
    }
    
    if err := vix.String(user.Password, "password").Apply(PasswordRules...).Validate(); err != nil {
        return err
    }
    
    return nil
}
```

## Best Practices

### 1. Use Descriptive Field Names

```go
// Good
vix.String(user.Email, "email").Required().Email().Validate()

// Bad
vix.String(user.Email, "field1").Required().Email().Validate()
```

### 2. Chain Validations Logically

```go
// Good - most basic checks first
vix.String(password, "password").
    Required().           // Check existence first
    MinLength(8).         // Then check length
    MaxLength(128).       // Then max length
    Regex(complexRegex).  // Complex checks last
    Validate()
```

### 3. Use Multi-Field Validation for Complex Forms

```go
// Good - validate all fields at once
func validateRegistrationForm(form RegistrationForm) map[string][]string {
    val := vix.Is(
        vix.String(form.Email, "email").Required().Email(),
        vix.String(form.Password, "password").Required().MinLength(8),
        vix.String(form.ConfirmPassword, "confirm_password").Required().Equal(form.Password),
        vix.Int(form.Age, "age").Required().Min(18),
    )
    
    return val.ToError()
}

// Bad - individual field validation
func validateRegistrationFormBad(form RegistrationForm) []error {
    var errors []error
    
    if err := vix.String(form.Email, "email").Required().Email().Validate(); err != nil {
        errors = append(errors, err)
    }
    
    if err := vix.String(form.Password, "password").Required().MinLength(8).Validate(); err != nil {
        errors = append(errors, err)
    }
    
    // ... more individual validations
    
    return errors
}
```

### 4. Use Conditional Validation

```go
// Good - conditional validation
vix.String(phone, "phone").
    When(func() bool { return user.Email == "" }).
    Required().
    Validate()

// Bad - manual conditional logic
if user.Email == "" {
    if err := validator.String(phone, "phone").Required().Validate(); err != nil {
        return err
    }
}
```

### 5. Handle Errors Appropriately

```go
// Good - wrap with ERM for proper error handling
if err := vix.String(user.Email, "email").Required().Email().Validate(); err != nil {
    return erm.BadRequest("Invalid email address", err)
}

// Bad - return validation error directly
if err := vix.String(user.Email, "email").Required().Email().Validate(); err != nil {
    return err
}
```

### 6. Use Localization for User-Facing Errors

```go
// Good - localized error messages
result := vix.String("", "email").
    Required().
    Result().
    WithLocale(getLocaleFromRequest(req))

if err := result.Error(); err != nil {
    return erm.BadRequest("Validation failed", err)
}
```

### 7. Use Namespaces for Nested Structures

```go
// Good - use namespaces for nested validation
func validateUserWithAddress(user UserWithAddress) map[string][]string {
    val := vix.Is(vix.String(user.Name, "name").Required()).
        In("address", vix.Is(
            vix.String(user.Address.Street, "street").Required(),
            vix.String(user.Address.City, "city").Required(),
            vix.String(user.Address.ZipCode, "zip_code").Required(),
        ))
    
    return val.ToError()
}
```

### 8. Use Indexed Namespaces for Arrays

```go
// Good - use indexed namespaces for array validation
func validateUserWithAddresses(user UserWithAddresses) map[string][]string {
    val := vix.Is(vix.String(user.Name, "name").Required())
    
    for i, addr := range user.Addresses {
        val.InRow("addresses", i, vix.Is(
            vix.String(addr.Street, "street").Required(),
            vix.String(addr.City, "city").Required(),
        ))
    }
    
    return val.ToError()
}
```

## Testing

### Testing Validation Logic

```go
func TestUserValidation(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "test@example.com", false},
        {"invalid email", "invalid-email", true},
        {"empty email", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := vix.String(tt.email, "email").Required().Email().Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("validation error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Testing Multi-Field Validation

```go
func TestMultiFieldValidation(t *testing.T) {
    val := vix.Is(
        vix.String("", "email").Required(),
        vix.Int(16, "age").Required().Min(18),
    )
    
    if val.Valid() {
        t.Error("validation should have failed")
    }
    
    if val.IsValid("email") {
        t.Error("email should be invalid")
    }
    
    if val.IsValid("age") {
        t.Error("age should be invalid")
    }
    
    errorMap := val.ToError()
    if len(errorMap) != 2 {
        t.Errorf("expected 2 errors, got %d", len(errorMap))
    }
}
```

### Testing Custom Validators

```go
func TestCustomValidator(t *testing.T) {
    validator := CreditCard("4111111111111111", "credit_card")
    err := vix.Valid().Validate()
    
    if err != nil {
        t.Errorf("expected valid credit card, got error: %v", err)
    }
}
```

## Migration Guide

### From Individual Field Validation to Multi-Field

```go
// Old approach - individual field validation
func validateUserOld(user User) []error {
    var errors []error
    
    if err := vix.String(user.Email, "email").Required().Email().Validate(); err != nil {
        errors = append(errors, err)
    }
    
    if err := vix.String(user.Password, "password").Required().MinLength(8).Validate(); err != nil {
        errors = append(errors, err)
    }
    
    if err := vix.Int(user.Age, "age").Required().Min(18).Validate(); err != nil {
        errors = append(errors, err)
    }
    
    return errors
}

// New approach - multi-field validation
func validateUserNew(user User) map[string][]string {
    val := vix.Is(
        vix.String(user.Email, "email").Required().Email(),
        vix.String(user.Password, "password").Required().MinLength(8),
        vix.Int(user.Age, "age").Required().Min(18),
    )
    
    return val.ToError()
}
```

### From Struct Tags to Function Chaining

```go
// Old approach with struct tags
type User struct {
    Email    string `validate:"required,email,max=255"`
    Password string `validate:"required,min=8,max=128"`
    Age      int    `validate:"required,min=18,max=100"`
}

// New approach with function chaining
func (u User) Validate() map[string][]string {
    val := vix.Is(
        vix.String(u.Email, "email").Required().Email().MaxLength(255),
        vix.String(u.Password, "password").Required().MinLength(8).MaxLength(128),
        vix.Int(u.Age, "age").Required().Min(18).Max(100),
    )
    
    return val.ToError()
}
```

## API Reference

### Multi-Field Validation Methods

| Method | Description | Example |
|--------|-------------|---------|
| `V()` | Create validation orchestrator | `vix.V()` |
| `NewValidationOrchestrator()` | Create validation orchestrator | `vix.NewValidationOrchestrator()` |
| `Is(validators...)` | Create orchestrator and add field validators | `vix.Is(vix.String("test", "field").Required())` |
| `In(namespace, orchestrator)` | Create orchestrator with namespaced validators | `vix.In("address", subValidator)` |
| `InRow(namespace, index, orchestrator)` | Create orchestrator with indexed validators | `vix.InRow("items", 0, subValidator)` |
| `Is(validators...)` (method) | Add field validators to existing orchestrator | `val.Is(vix.String("test", "field").Required())` |
| `In(namespace, orchestrator)` (method) | Add namespaced validators to existing orchestrator | `val.In("address", subValidator)` |
| `InRow(namespace, index, orchestrator)` (method) | Add indexed namespaced validators to existing orchestrator | `val.InRow("items", 0, subValidator)` |
| `Valid()` | Check if all validations passed | `val.Valid()` |
| `IsValid(field)` | Check if specific field is valid | `val.IsValid("email")` |
| `ToError()` | Get structured error map | `val.ToError()` |
| `ToJSON()` | Get JSON error output | `val.ToJSON()` |
| `Error()` | Get first error | `val.Error()` |
| `AllErrors()` | Get all errors | `val.AllErrors()` |
| `String()` | Get string representation | `val.String()` |
| `FieldNames()` | Get validated field names | `val.FieldNames()` |
| `GetFieldResult(field)` | Get result for specific field | `val.GetFieldResult("email")` |
| `WithLocale(locale)` | Set locale for all fields | `val.WithLocale(spanishLocale)` |

### String Validator Methods

| Method | Description | Example |
|--------|-------------|---------|
| `Required()` | Must not be empty | `Required()` |
| `Empty()` | Must be empty | `Empty()` |
| `MinLength(n)` | Minimum length | `MinLength(5)` |
| `MaxLength(n)` | Maximum length | `MaxLength(100)` |
| `ExactLength(n)` | Exact length | `ExactLength(10)` |
| `LengthBetween(min, max)` | Length within range | `LengthBetween(5, 50)` |
| `Email()` | Valid email format | `Email()` |
| `URL()` | Valid URL format | `URL()` |
| `Numeric()` | Contains only numbers | `Numeric()` |
| `Alpha()` | Contains only letters | `Alpha()` |
| `AlphaNumeric()` | Contains only letters and numbers | `AlphaNumeric()` |
| `Regex(pattern)` | Matches regex pattern | `Regex(regexp.MustCompile("^[A-Z]+$"))` |
| `In(values...)` | Must be one of values | `In("option1", "option2")` |
| `NotIn(values...)` | Must not be one of values | `NotIn("bad1", "bad2")` |
| `Contains(substring)` | Must contain substring | `Contains("test")` |
| `StartsWith(prefix)` | Must start with prefix | `StartsWith("prefix")` |
| `EndsWith(suffix)` | Must end with suffix | `EndsWith("suffix")` |
| `Lowercase()` | Must be lowercase | `Lowercase()` |
| `Uppercase()` | Must be uppercase | `Uppercase()` |
| `UUID()` | Valid UUID format | `UUID()` |
| `Slug()` | Valid URL slug | `Slug()` |
| `JSON()` | Valid JSON format | `JSON()` |
| `Base64()` | Valid base64 encoding | `Base64()` |
| `Integer()` | Valid integer string | `Integer()` |
| `Float()` | Valid float string | `Float()` |

### Number Validator Methods

| Method | Description | Example |
|--------|-------------|---------|
| `Required()` | Must not be zero | `Required()` |
| `Zero()` | Must be zero | `Zero()` |
| `Min(n)` | Minimum value | `Min(18)` |
| `Max(n)` | Maximum value | `Max(100)` |
| `Between(min, max)` | Value within range | `Between(1, 100)` |
| `Equal(n)` | Must equal value | `Equal(42)` |
| `GreaterThan(n)` | Must be greater than | `GreaterThan(0)` |
| `LessThan(n)` | Must be less than | `LessThan(100)` |
| `Positive()` | Must be positive | `Positive()` |
| `Negative()` | Must be negative | `Negative()` |
| `Even()` | Must be even | `Even()` |
| `Odd()` | Must be odd | `Odd()` |
| `MultipleOf(n)` | Must be multiple of | `MultipleOf(5)` |
| `In(values...)` | Must be one of values | `In(1, 2, 3, 5, 8)` |
| `NotIn(values...)` | Must not be one of values | `NotIn(0, -1)` |
| `Finite()` | Must be finite (floats) | `Finite()` |
| `Precision(n)` | Max decimal places | `Precision(2)` |

### Common Methods

| Method | Description | Example |
|--------|-------------|---------|
| `Not()` | Negate next validation | `Not().Required()` |
| `When(condition)` | Conditional validation | `When(func() bool { return true })` |
| `Unless(condition)` | Inverse conditional | `Unless(func() bool { return false })` |
| `Custom(fn)` | Custom validation function | `Custom(func(v interface{}) error { return nil })` |
| `Validate()` | Execute validation | `Validate()` |
| `Result()` | Get validation result | `Result()` |

### ValidationResult Methods

| Method | Description | Example |
|--------|-------------|---------|
| `Valid()` | Check if validation passed | `result.Valid()` |
| `Error()` | Get first error | `result.Error()` |
| `AllErrors()` | Get all errors | `result.AllErrors()` |
| `ToError()` | Get structured error map | `result.ToError()` |
| `WithLocale(locale)` | Set locale for error messages | `result.WithLocale(spanishLocale)` |
| `WithHTTPStatus(status)` | Set HTTP status code | `result.WithHTTPStatus(422)` |

This comprehensive validation package provides a powerful, flexible, and type-safe way to validate data in Go applications while maintaining clean architecture principles and seamless integration with the ERM error management system. The multi-field validation capabilities make it easy to validate complex forms and nested structures with detailed error reporting suitable for modern web APIs.
