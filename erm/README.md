# ERM - Error Management Package

A lightweight, reusable error management package for Go applications following onion/clean architecture patterns. This package enriches errors with stack traces, HTTP status codes, and safe user-facing messages while maintaining full compatibility with Go's standard error handling.

## Features

- **Automatic Operation Detection**: Automatically detects operation names from the call stack
- **Stack Traces**: Automatically captures stack traces when errors are created
- **HTTP Status Codes**: Direct mapping to HTTP status codes (no intermediate types)
- **Safe Messages**: Provides user-friendly messages separate from internal error details
- **Error Wrapping**: Full compatibility with Go's `errors.Is/As` functionality
- **Zero Dependencies**: Uses only Go standard library
- **Onion Architecture**: Designed specifically for clean/onion architecture patterns

## Installation

```bash
go get github.com/c3p0-box/c3p1/backend/erm
```

## Usage

### Basic Error Creation

```go
import "github.com/c3p0-box/c3p1/backend/erm"

// Create a new error with automatic operation detection
err := erm.New(http.StatusBadRequest, "Invalid email format", originalErr)

// Extract information
status := erm.Status(err)    // 400
message := erm.Message(err)  // "Invalid email format"
stack := erm.Stack(err)      // []uintptr (program counters)
```

### Convenience Constructors

```go
// Common HTTP errors with automatic operation detection
err := erm.BadRequest("Invalid email format", originalErr)
err := erm.NotFound("User not found", originalErr)
err := erm.Internal("Database connection failed", originalErr)
```

### Error Wrapping

```go
// Wrap existing errors while preserving metadata
repoErr := erm.NotFound("User not found", sql.ErrNoRows)
serviceErr := erm.Wrap(repoErr)

// Status and message are preserved through wrapping
erm.Status(serviceErr)  // 404
erm.Message(serviceErr) // "User not found"
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
        status := erm.Status(err)
        message := erm.Message(err)
        
        // Log full error with stack trace for debugging
        log.Printf("Error: %v\nStack:\n%s", err, erm.FormatStack(err))
        
        // Return safe message to client
        http.Error(w, message, status)
        return
    }
    
    json.NewEncoder(w).Encode(profile)
}
```

## API Reference

### Types

- `Error`: Main error type with stack trace, HTTP status, and message
- `New(code, msg, err)`: Create new error with automatic operation detection
- `Wrap(err)`: Wrap existing error with new operation context

### Helper Functions

- `Status(err) int`: Extract HTTP status code from any error
- `Message(err) string`: Extract safe user message from any error  
- `Stack(err) []uintptr`: Extract stack trace from any error
- `FormatStack(err) string`: Format stack trace for logging

### Convenience Constructors

- `BadRequest(msg, err)`: Create 400 Bad Request error
- `Unauthorized(msg, err)`: Create 401 Unauthorized error
- `Forbidden(msg, err)`: Create 403 Forbidden error
- `NotFound(msg, err)`: Create 404 Not Found error
- `Conflict(msg, err)`: Create 409 Conflict error
- `Internal(msg, err)`: Create 500 Internal Server Error

## Best Practices

1. **Automatic Operation Detection**: The package automatically detects operation names from the call stack, eliminating the need to manually specify them
2. **Safe Messages**: Keep user-facing messages generic and safe - don't leak internal details
3. **Error Wrapping**: Use `Wrap()` to add context while preserving original error metadata
4. **Stack Traces**: Use `FormatStack()` for logging, not for user responses
5. **Nil Handling**: All functions handle nil errors gracefully
6. **Direct Status Codes**: Use HTTP status codes directly instead of intermediate types

## Testing

```bash
cd backend/erm
go test -v
```

## License

This package is part of the c3p1 project and follows the same license terms.
