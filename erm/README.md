# ERM - Error Management Package

A lightweight, reusable error management package for Go applications following onion/clean architecture patterns. This package enriches errors with stack traces, HTTP status codes, and safe user-facing messages while maintaining full compatibility with Go's standard error handling.

## Features

- **Interface-based Design**: Clean interface (`Error`) with concrete implementation (`StackError`) for flexibility
- **Automatic Operation Detection**: Automatically detects operation names from the call stack
- **Stack Traces**: Automatically captures stack traces when errors are created
- **HTTP Status Codes**: Direct mapping to HTTP status codes (no intermediate types)
- **Safe Messages**: Provides user-friendly messages separate from internal error details
- **Error Wrapping**: Full compatibility with Go's `errors.Is/As` functionality
- **Helper Functions**: Convenient functions to extract information from any error type
- **Zero Dependencies**: Uses only Go standard library
- **Onion Architecture**: Designed specifically for clean/onion architecture patterns
- **Thread Safety**: All functions are safe for concurrent use

## Installation

```bash
go get github.com/c3p0-box/utils/erm
```

## Usage

### Basic Error Creation

```go
import "github.com/c3p0-box/utils/erm"

// Create a new error with automatic operation detection
err := erm.New(http.StatusBadRequest, "Invalid email format", originalErr)

// Extract information using helper functions
status := erm.Status(err)    // 400
message := erm.Message(err)  // "Invalid email format"
stack := erm.Stack(err)      // []uintptr (program counters)

// Or use interface methods directly
status = err.Code()          // 400
operation := err.Op()        // "UserService.ValidateEmail" (auto-detected)
```

### Convenience Constructors

```go
// Common HTTP errors with automatic operation detection
err := erm.BadRequest("Invalid email format", originalErr)
err := erm.NotFound("User not found", originalErr)
err := erm.Internal("Database connection failed", originalErr)

// All return erm.Error interface
fmt.Printf("Status: %d, Message: %s", err.Code(), erm.Message(err))
```

### Error Wrapping

```go
// Wrap existing errors while preserving metadata
repoErr := erm.NotFound("User not found", sql.ErrNoRows)
serviceErr := erm.Wrap(repoErr)

// Status and message are preserved through wrapping
erm.Status(serviceErr)  // 404
erm.Message(serviceErr) // "User not found"

// Operation is updated to the wrapping context
serviceErr.Op()         // "UserService.GetUser" (auto-detected)
```

### Stack Trace Formatting

```go
err := erm.New(http.StatusInternalServerError, "Something went wrong", originalErr)
formatted := erm.FormatStack(err)
fmt.Println(formatted)
// Output:
// main.someFunction
//     /path/to/file.go:123
// main.main
//     /path/to/file.go:456
```

### Working with Interface

```go
func handleError(err error) {
    // Helper functions work with any error type
    status := erm.Status(err)    // Works with both erm.Error and standard errors
    message := erm.Message(err)  // Returns appropriate message for any error
    
    // Type assertion for erm-specific functionality
    if ermErr, ok := err.(erm.Error); ok {
        operation := ermErr.Op()
        stack := ermErr.Stack()
        fmt.Printf("Operation: %s, Stack frames: %d", operation, len(stack))
    }
}
```

## Onion Architecture Usage

### Repository Layer (Adapters)
```go
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := r.db.QueryRowContext(ctx, "SELECT * FROM users WHERE email = $1", email).Scan(&user)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, erm.NotFound("User not found", err)
    }
    if err != nil {
        return nil, erm.Internal("Database error", err)
    }
    
    return &user, nil
}
```

### Service Layer (Use Cases)
```go
func (s *userService) GetProfile(ctx context.Context, email string) (*UserProfile, error) {
    if email == "" {
        return nil, erm.BadRequest("Email is required", nil)
    }
    
    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return nil, erm.Wrap(err) // Preserves original status and message
    }
    
    return &UserProfile{Name: user.Name, Email: user.Email}, nil
}
```

### Handler Layer (Controllers)
```go
func (h *userHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
    email := r.URL.Query().Get("email")
    
    profile, err := h.userService.GetProfile(r.Context(), email)
    if err != nil {
        status := erm.Status(err)    // Works with any error type
        message := erm.Message(err)  // Safe for client consumption
        
        // Log full error with stack trace for debugging
        if ermErr, ok := err.(erm.Error); ok {
            log.Printf("Error: %v\nStack:\n%s", err, erm.FormatStack(ermErr))
        } else {
            log.Printf("Error: %v", err)
        }
        
        // Return safe message to client
        http.Error(w, message, status)
        return
    }
    
    json.NewEncoder(w).Encode(profile)
}
```

## API Reference

### Types

- **`Error` interface**: Main error interface with methods for accessing error information
- **`StackError` struct**: Concrete implementation of Error interface with stack traces

### Core Functions

- **`New(code int, msg string, err error) Error`**: Create new error with automatic operation detection and stack trace capture
- **`Wrap(err error) Error`**: Wrap existing error with new operation context while preserving metadata

### Interface Methods

- **`Error() string`**: Standard error message (satisfies error interface)
- **`Op() string`**: Get operation name where error occurred
- **`Code() int`**: Get HTTP status code associated with error
- **`Unwrap() error`**: Get wrapped error (for errors.Is/As compatibility)
- **`Stack() []uintptr`**: Get stack trace program counters

### Helper Functions

- **`Status(err error) int`**: Extract HTTP status code from any error
- **`Message(err error) string`**: Extract safe user message from any error  
- **`Stack(err error) []uintptr`**: Extract stack trace from any error
- **`FormatStack(err Error) string`**: Format stack trace for logging

### Convenience Constructors

- **`BadRequest(msg string, err error) Error`**: Create 400 Bad Request error
- **`Unauthorized(msg string, err error) Error`**: Create 401 Unauthorized error
- **`Forbidden(msg string, err error) Error`**: Create 403 Forbidden error
- **`NotFound(msg string, err error) Error`**: Create 404 Not Found error
- **`Conflict(msg string, err error) Error`**: Create 409 Conflict error
- **`Internal(msg string, err error) Error`**: Create 500 Internal Server Error

## Best Practices

1. **Interface Usage**: Use the `Error` interface in function signatures for flexibility
2. **Helper Functions**: Use helper functions (`Status`, `Message`, `Stack`) to safely extract information from any error
3. **Automatic Operation Detection**: The package automatically detects operation names from the call stack
4. **Safe Messages**: Keep user-facing messages generic and safe - don't leak internal details
5. **Error Wrapping**: Use `Wrap()` to add context while preserving original error metadata
6. **Stack Traces**: Use `FormatStack()` for logging, not for user responses
7. **Nil Handling**: All functions and methods handle nil errors gracefully
8. **Type Assertions**: Use type assertions only when you need erm-specific functionality

## Error Handling Patterns

### Propagating Errors
```go
// Good: Wrap errors to add context while preserving metadata
func (s *service) Process() error {
    data, err := s.repo.GetData()
    if err != nil {
        return erm.Wrap(err) // Preserves status/message, updates operation
    }
    return nil
}
```

### Converting Standard Errors
```go
// Convert standard errors to erm errors with appropriate status
func (s *service) ValidateInput(input string) error {
    if input == "" {
        return erm.BadRequest("Input cannot be empty", nil)
    }
    
    if err := validateFormat(input); err != nil {
        return erm.BadRequest("Invalid format", err)
    }
    
    return nil
}
```

### Handling Mixed Error Types
```go
func handleAnyError(err error) {
    // Helper functions work with any error type
    status := erm.Status(err)    // 500 for standard errors, actual code for erm errors
    message := erm.Message(err)  // HTTP status text for standard errors
    
    // Access erm-specific features only when needed
    if ermErr, ok := err.(erm.Error); ok {
        log.Printf("Operation: %s", ermErr.Op())
        if len(ermErr.Stack()) > 0 {
            log.Printf("Stack trace:\n%s", erm.FormatStack(ermErr))
        }
    }
}
```

## Testing

```bash
cd erm
go test -v -cover
```

## Migration from Previous Version

If upgrading from a previous version where `Error` was a struct:

### Before (struct-based)
```go
err := erm.New(400, "Bad request", nil)
operation := err.Op    // Direct field access
status := err.Code     // Direct field access
```

### After (interface-based)
```go
err := erm.New(400, "Bad request", nil)
operation := err.Op()  // Method call
status := err.Code()   // Method call

// Or use helper functions for mixed error types
status := erm.Status(err)    // Works with any error
message := erm.Message(err)  // Works with any error
```

## Performance Considerations

- **Stack trace capture**: Minimal overhead, captured only once at error creation
- **Operation detection**: Fast call stack inspection with automatic caching
- **Memory usage**: Efficient storage of stack traces as program counters
- **Thread safety**: All operations are lock-free and safe for concurrent use

## License

This package does not permit usage for anyone unless permission is granted explicitly or it's being used by the author of the package.
