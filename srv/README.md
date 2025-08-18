# srv - HTTP Server Utilities

The `srv` package provides comprehensive HTTP server utilities for Go applications, including middleware support, graceful shutdown, HTTP context abstraction, and enhanced routing capabilities.

## Features

- **🛠️ HTTP Context**: Convenient request/response wrapper with value storage
- **🔀 Enhanced Router**: Extended ServeMux with RESTful HTTP method helpers
- **🔗 Middleware System**: Composable HTTP middleware with easy chaining
- **📊 Built-in Middleware**: Logging and panic recovery middleware included
- **🛑 Graceful Shutdown**: HTTP server with signal-based graceful shutdown
- **📝 Structured Logging**: Integration with Go's structured logging (`log/slog`)
- **⚡ Zero Dependencies**: Only uses Go standard library

## Installation

```bash
go get github.com/c3p0-box/utils/srv
```

## Quick Start

```go
package main

import (
    "errors"
    "log"
    "github.com/c3p0-box/utils/srv"
)

func main() {
    // Create enhanced router with error handling
    mux := srv.NewMux()
    
    // Add routes with new HandlerFunc signature (returns error)
    mux.Get("/users", func(ctx *srv.HttpContext) error {
        return ctx.JSON(200, map[string]string{"message": "Users list"})
    })
    
    mux.Post("/users", func(ctx *srv.HttpContext) error {
        name := ctx.FormValue("name")
        if name == "" {
            return errors.New("name is required")  // Error handled automatically
        }
        return ctx.JSON(201, map[string]string{"created": name})
    })
    
    // Set custom error handler (optional)
    mux.ErrorHandler(func(ctx *srv.HttpContext, err error) {
        log.Printf("Handler error: %v", err)
        ctx.JSON(400, map[string]string{"error": err.Error()})
    })
    
    // Chain middleware (logging and recovery)
    handler := srv.MiddlewareChain(srv.Logging, srv.Recover)(mux)
    
    // Run server with graceful shutdown
    err := srv.RunServer(handler, "localhost", "8080", func() error {
        log.Println("Cleaning up...")
        return nil
    })
    
    if err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### 🛠️ HttpContext - Request/Response Wrapper

#### Overview
`HttpContext` provides a convenient wrapper around `http.Request` and `http.ResponseWriter` with additional functionality for value storage, parameter handling, and response generation.

#### Constructor
```go
func NewHttpContext(w http.ResponseWriter, r *http.Request) *HttpContext
```

#### Value Store (Thread-Safe)
```go
ctx.Set("user", userObj)           // Store value
user := ctx.Get("user")            // Retrieve value
```

#### Request Information
```go
method := ctx.Method()             // HTTP method: "GET", "POST", etc.
path := ctx.Path()                 // URL path: "/api/users"
isTLS := ctx.IsTLS()              // true for HTTPS requests
isWS := ctx.IsWebSocket()         // true for WebSocket upgrades
```

#### Parameters & Headers
```go
// Query parameters
name := ctx.QueryParam("name")     // Single query parameter
query := ctx.Query()               // All query parameters

// Path parameters (Go 1.22+ ServeMux)
id := ctx.Param("id")              // Path parameter: /users/{id}

// Form data
username := ctx.FormValue("username")

// Headers
auth := ctx.GetHeader("Authorization")
ctx.SetHeader("Content-Type", "application/json")
ctx.AddHeader("X-Custom", "value")
```

#### Cookies
```go
// Read cookies
cookie, err := ctx.Cookie("session")
allCookies := ctx.Cookies()

// Set cookies
ctx.SetCookie(&http.Cookie{
    Name:  "session",
    Value: "abc123",
    Path:  "/",
})
```

#### Response Methods
```go
// JSON response (with error handling)
err := ctx.JSON(200, data)

// Text response
err := ctx.String(200, "Hello, World!")

// HTML response
err := ctx.HTML(200, "<h1>Welcome</h1>")

// Redirects
ctx.Redirect(302, "/login")

// Custom status code
ctx.WriteHeader(204)
```

### 🔀 Mux - Enhanced HTTP Router

#### Overview
`Mux` wraps Go's standard `http.ServeMux` with convenient HTTP method helpers, RESTful routing support, and centralized error handling.

#### HandlerFunc Type
```go
type HandlerFunc func(ctx *HttpContext) error
```

The new `HandlerFunc` signature provides several advantages:
- **Automatic HttpContext**: No need to manually create HttpContext
- **Error Handling**: Return errors instead of writing error responses manually
- **Cleaner Code**: Less boilerplate, more focused business logic

#### Constructor
```go
mux := srv.NewMux()  // Includes default error handler
```

#### HTTP Method Helpers (New HandlerFunc Signature)
```go
mux.Get("/users", func(ctx *HttpContext) error {
    return ctx.JSON(200, users)
})

mux.Post("/users", func(ctx *HttpContext) error {
    user := parseUser(ctx)
    if err := validateUser(user); err != nil {
        return err  // Automatically handled by error handler
    }
    return ctx.JSON(201, user)
})

mux.Put("/users/{id}", func(ctx *HttpContext) error {
    id := ctx.Param("id")
    return updateUser(id, ctx)
})

mux.Delete("/users/{id}", func(ctx *HttpContext) error {
    return deleteUser(ctx.Param("id"))
})

// All HTTP methods supported
mux.Patch("/users/{id}", handler)  // PATCH requests
mux.Head("/ping", handler)         // HEAD requests
mux.Options("/api/*", handler)     // OPTIONS requests
```

#### Error Handling
```go
// Custom error handler (optional)
mux.ErrorHandler(func(ctx *HttpContext, err error) {
    log.Printf("Handler error: %v", err)
    
    // Handle different error types
    switch e := err.(type) {
    case *ValidationError:
        ctx.JSON(400, map[string]string{"error": e.Error()})
    case *NotFoundError:
        ctx.JSON(404, map[string]string{"error": "Not found"})
    default:
        ctx.JSON(500, map[string]string{"error": "Internal server error"})
    }
})

// Default error handler returns 500 with generic message
```

#### Standard ServeMux Methods (Traditional Handlers)
```go
mux.Handle("/api/", apiHandler)         // Register http.Handler
mux.HandleFunc("/health", healthFunc)   // Register http.HandlerFunc
```

#### Access Underlying ServeMux
```go
stdMux := mux.Mux()  // Get *http.ServeMux for advanced usage
```

### 🔗 Middleware System

#### Types
```go
type Middleware func(next http.Handler) http.Handler
```

#### Built-in Middleware

**Logging Middleware**
```go
handler := srv.Logging(mux)  // Logs all HTTP requests
```
Captures: method, path, status code, user agent, remote address

**Recovery Middleware**
```go
handler := srv.Recover(mux)  // Recovers from panics
```
Prevents server crashes by catching and logging panics

#### Middleware Chaining
```go
// Chain multiple middleware (applied in reverse order)
handler := srv.MiddlewareChain(
    srv.Logging,    // Outermost: logs all requests
    srv.Recover,    // Recovers from panics
    CustomCORS,     // Custom CORS middleware
    RateLimiting,   // Innermost: rate limiting
)(mux)
```

### 🛑 RunServer - Graceful Server

#### Function Signature
```go
func RunServer(handler http.Handler, host string, port string, cleanup func() error) error
```

#### Parameters
- `handler`: HTTP handler (can be Mux, middleware chain, etc.)
- `host`: Host to bind to (defaults to "0.0.0.0" if empty)
- `port`: Port to listen on (defaults to "8000" if empty)
- `cleanup`: Function called during shutdown for resource cleanup

#### Graceful Shutdown Process
1. Listens for SIGINT/SIGTERM signals
2. Stops accepting new requests
3. Waits up to 10 seconds for existing requests to complete
4. Calls the cleanup function
5. Returns any errors that occurred

## Usage Examples

### Basic RESTful API

```go
mux := srv.NewMux()

// Set custom error handler for better error responses
mux.ErrorHandler(func(ctx *srv.HttpContext, err error) {
    log.Printf("API Error: %v", err)
    ctx.JSON(500, map[string]string{"error": err.Error()})
})

// Users API with new HandlerFunc signature
mux.Get("/users", listUsers)
mux.Post("/users", createUser)
mux.Get("/users/{id}", getUser)
mux.Put("/users/{id}", updateUser)
mux.Delete("/users/{id}", deleteUser)

func getUser(ctx *srv.HttpContext) error {
    // Get path parameter
    id := ctx.Param("id")
    
    // Store in context for other middleware
    ctx.Set("userID", id)
    
    // Get user with error handling
    user, err := getUserByID(id)
    if err != nil {
        return err  // Automatically handled by error handler
    }
    
    // Return JSON response
    return ctx.JSON(200, user)
}

func createUser(ctx *srv.HttpContext) error {
    var user User
    if err := json.NewDecoder(ctx.Request().Body).Decode(&user); err != nil {
        return fmt.Errorf("invalid JSON: %w", err)
    }
    
    if err := validateUser(user); err != nil {
        return err
    }
    
    createdUser, err := saveUser(user)
    if err != nil {
        return fmt.Errorf("failed to save user: %w", err)
    }
    
    return ctx.JSON(201, createdUser)
}

// Start server
srv.RunServer(mux, "localhost", "8080", func() error {
    return db.Close()
})
```

### Advanced Middleware Integration

```go
// Custom middleware
func Authentication(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := srv.NewHttpContext(w, r)
        
        token := ctx.GetHeader("Authorization")
        if token == "" {
            ctx.JSON(401, map[string]string{"error": "Unauthorized"})
            return
        }
        
        // Validate token and store user
        user := validateToken(token)
        ctx.Set("user", user)
        
        next.ServeHTTP(w, r)
    })
}

func CORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := srv.NewHttpContext(w, r)
        ctx.SetHeader("Access-Control-Allow-Origin", "*")
        ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if ctx.Method() == "OPTIONS" {
            ctx.WriteHeader(204)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

// Chain all middleware
mux := srv.NewMux()
handler := srv.MiddlewareChain(
    srv.Logging,        // Log all requests
    srv.Recover,        // Panic recovery
    CORS,               // CORS headers
    Authentication,     // JWT authentication
)(mux)
```

### Enhanced HandlerFunc with Error Handling

```go
// Register handler with automatic HttpContext and error handling
mux.Post("/users", createUserHandler)

func createUserHandler(ctx *srv.HttpContext) error {
    // Parse form data (HttpContext provided automatically)
    name := ctx.FormValue("name")
    email := ctx.FormValue("email")
    
    // Validate input - return error instead of manual response
    if name == "" || email == "" {
        return errors.New("name and email are required")
    }
    
    // Get authenticated user from context
    authUser, ok := ctx.Get("user").(*User)
    if !ok {
        return errors.New("authentication required")
    }
    
    // Create user
    user := &User{
        Name:      name,
        Email:     email,
        CreatedBy: authUser.ID,
    }
    
    // Database operation with error handling
    if err := saveUser(user); err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }
    
    // Return created user (errors handled automatically)
    return ctx.JSON(201, user)
}

// Error handling patterns
func advancedErrorHandling(ctx *srv.HttpContext) error {
    // Custom error types for different HTTP status codes
    user, err := validateAndGetUser(ctx)
    if err != nil {
        switch {
        case errors.Is(err, ErrUserNotFound):
            return &HTTPError{Code: 404, Message: "User not found"}
        case errors.Is(err, ErrValidation):
            return &HTTPError{Code: 400, Message: err.Error()}
        default:
            return err  // 500 from default error handler
        }
    }
    
    return ctx.JSON(200, user)
}

// Custom error type for specific HTTP responses
type HTTPError struct {
    Code    int
    Message string
}

func (e *HTTPError) Error() string {
    return e.Message
}

// Custom error handler that understands HTTPError
mux.ErrorHandler(func(ctx *srv.HttpContext, err error) {
    if httpErr, ok := err.(*HTTPError); ok {
        ctx.JSON(httpErr.Code, map[string]string{"error": httpErr.Message})
        return
    }
    
    // Default error handling
    log.Printf("Unexpected error: %v", err)
    ctx.JSON(500, map[string]string{"error": "Internal server error"})
})
```

### File Upload Handling

```go
mux.Post("/upload", func(ctx *srv.HttpContext) error {
    // Parse multipart form (32MB max)
    if err := ctx.Request().ParseMultipartForm(32 << 20); err != nil {
        return fmt.Errorf("invalid form: %w", err)
    }
    
    // Get uploaded file
    file, header, err := ctx.Request().FormFile("file")
    if err != nil {
        return errors.New("no file uploaded")
    }
    defer file.Close()
    
    // Validate file type
    if !isValidFileType(header.Filename) {
        return errors.New("invalid file type")
    }
    
    // Process file with error handling
    savedPath, err := saveUploadedFile(file, header.Filename)
    if err != nil {
        return fmt.Errorf("failed to save file: %w", err)
    }
    
    // Return success response
    return ctx.JSON(200, map[string]string{
        "message":  "File uploaded successfully",
        "filename": header.Filename,
        "path":     savedPath,
    })
})

// Helper functions with proper error handling
func isValidFileType(filename string) bool {
    validTypes := []string{".jpg", ".jpeg", ".png", ".pdf", ".txt"}
    ext := strings.ToLower(filepath.Ext(filename))
    for _, validType := range validTypes {
        if ext == validType {
            return true
        }
    }
    return false
}

func saveUploadedFile(file multipart.File, filename string) (string, error) {
    // Create unique filename
    uniqueName := fmt.Sprintf("%d_%s", time.Now().Unix(), filename)
    destPath := filepath.Join("/uploads", uniqueName)
    
    // Create destination file
    dest, err := os.Create(destPath)
    if err != nil {
        return "", err
    }
    defer dest.Close()
    
    // Copy file content
    if _, err := io.Copy(dest, file); err != nil {
        return "", err
    }
    
    return destPath, nil
}
```

## Testing

The package includes comprehensive tests covering all functionality:

```bash
# Run tests
go test ./srv

# Run tests with coverage
go test -cover ./srv

# Run tests with verbose output
go test -v ./srv

# Run benchmarks
go test -bench=. ./srv
```

### Test Coverage

- ✅ **HttpContext**: Value store, request/response methods, thread safety
- ✅ **Mux**: HTTP method helpers, routing, integration tests
- ✅ **Middleware**: Type safety, chaining, logging, recovery
- ✅ **RunServer**: Parameter validation, error handling, cleanup
- ✅ **Integration**: Cross-component functionality tests
- ✅ **Benchmarks**: Performance testing for all components

**Current Coverage**: 87.0% of statements

## Performance

Benchmark results on Apple M1 Max:

```
BenchmarkHttpContext_ValueStore     99.00 ns/op    21 B/op    2 allocs/op
BenchmarkHttpContext_JSON          938.8 ns/op  1280 B/op   17 allocs/op  
BenchmarkMux_RouteMatching         271.8 ns/op   224 B/op    5 allocs/op
```

## Best Practices

### 1. Middleware Order
Place middleware in logical order - logging first, authentication/authorization before business logic:

```go
handler := srv.MiddlewareChain(
    srv.Logging,        // First: log everything
    srv.Recover,        // Second: catch panics
    CORS,               // Third: CORS headers
    Authentication,     // Fourth: auth before business logic
    RateLimiting,       // Last: rate limiting
)(mux)
```

### 2. Context Usage
Use HttpContext consistently for request/response operations:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := srv.NewHttpContext(w, r)
    // Use ctx for all operations instead of w/r directly
}
```

### 3. Error Handling
Always handle errors from response methods:

```go
if err := ctx.JSON(200, data); err != nil {
    log.Printf("Failed to write JSON response: %v", err)
}
```

### 4. Resource Cleanup
Always provide cleanup functions for graceful shutdown:

```go
srv.RunServer(handler, host, port, func() error {
    db.Close()
    cache.Close()
    return nil
})
```

## Advanced Configuration

For custom server configurations, use the individual components:

```go
mux := srv.NewMux()
handler := srv.MiddlewareChain(srv.Logging, srv.Recover)(mux)

server := &http.Server{
    Addr:         ":8080",
    Handler:      handler,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}

log.Fatal(server.ListenAndServe())
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This package is part of the c3p0-box/utils collection and follows the same license terms.