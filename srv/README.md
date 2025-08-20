# srv - HTTP Server Utilities

The `srv` package provides comprehensive HTTP server utilities for Go applications, including middleware support, graceful shutdown, HTTP context abstraction, and enhanced routing capabilities.

## Features

- **🛠️ HTTP Context**: Convenient request/response wrapper with value storage
- **🔀 Enhanced Router**: Extended ServeMux with RESTful HTTP method helpers
- **🔄 URL Reversing**: Named routes with automatic URL generation and parameter substitution
- **🔗 Middleware System**: Composable HTTP middleware with easy chaining
- **📊 Built-in Middleware**: Logging, panic recovery, CORS, and trailing slash middleware included
- **🛑 Graceful Shutdown**: HTTP server with signal-based graceful shutdown
- **📝 Structured Logging**: Integration with Go's structured logging (`log/slog`)
- **🔗 ERM Integration**: Uses the erm package for standardized error handling and HTTP status codes
- **⚡ Minimal Dependencies**: Uses Go standard library plus c3p0-box/utils/erm for errors

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
    "github.com/c3p0-box/utils/erm"
)

func main() {
    // Create enhanced router with error handling
    mux := srv.NewMux()
    
    // Add named routes for URL generation
    mux.Get("users", "/users", func(ctx *srv.HttpContext) error {
        // Generate URL to create user endpoint
        createURL, _ := mux.Reverse("users", nil)
        return ctx.JSON(200, map[string]interface{}{
            "message": "Users list",
            "create_url": createURL,
        })
    })
    
    mux.Post("users", "/users", func(ctx *srv.HttpContext) error {
        name := ctx.FormValue("name")
        if name == "" {
            return errors.New("name is required")  // Error handled automatically
        }
        // Generate URL to view the created user
        userURL, _ := mux.Reverse("user", map[string]string{"id": "123"})
        return ctx.JSON(201, map[string]interface{}{
            "created": name,
            "user_url": userURL,
        })
    })
    
    mux.Get("user", "/users/{id}", func(ctx *srv.HttpContext) error {
        return ctx.JSON(200, map[string]string{"id": ctx.Param("id")})
    })
    
    // Set custom error handler with erm integration (optional)
    mux.ErrorHandler(func(ctx *srv.HttpContext, err error) {
        log.Printf("Handler error: %v", err)
        // erm errors provide proper HTTP status codes
        status := erm.Status(err)  // Extract HTTP status from erm error
        message := erm.Message(err) // Get user-safe message
        ctx.JSON(status, map[string]string{"error": message})
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

#### HTTP Method Helpers with Optional Naming
```go
// Named routes for URL generation
mux.Get("user-list", "/users", func(ctx *HttpContext) error {
    return ctx.JSON(200, users)
})

mux.Post("user-create", "/users", func(ctx *HttpContext) error {
    user := parseUser(ctx)
    if err := validateUser(user); err != nil {
        return err  // Automatically handled by error handler
    }
    return ctx.JSON(201, user)
})

// Unnamed routes (empty string name)
mux.Get("", "/health", func(ctx *HttpContext) error {
    return ctx.String(200, "OK")
})

// Same name for different methods (RESTful pattern)
mux.Get("users", "/users", listUsersHandler)      // GET /users
mux.Post("users", "/users", createUserHandler)    // POST /users
mux.Get("user", "/users/{id}", getUserHandler)    // GET /users/{id}
mux.Put("user", "/users/{id}", updateUserHandler) // PUT /users/{id}
mux.Delete("user", "/users/{id}", deleteUserHandler) // DELETE /users/{id}

// All HTTP methods supported
mux.Patch("user-patch", "/users/{id}", handler)  // PATCH requests
mux.Head("health-check", "/ping", handler)       // HEAD requests
mux.Options("cors", "/api/*", handler)           // OPTIONS requests
```

#### URL Reversing (Simplified)
```go
// Generate URLs from route names (no method parameter needed)
usersURL, err := mux.Reverse("users", nil)
// Returns: "/users" (works for both GET and POST since same pattern)

userURL, err := mux.Reverse("user", map[string]string{"id": "123"})
// Returns: "/users/123" (works for GET, PUT, DELETE since same pattern)

// Error handling with erm package integration
userURL, err := mux.Reverse("non-existent", nil)
// Returns: "", erm.NotFound error with 404 status

userURL, err := mux.Reverse("user-profile", nil)  // Missing {id} parameter
// Returns: "", erm.RequiredError for missing parameters
```

#### URL Generation in Handlers
```go
mux.Get("users", "/users", func(ctx *HttpContext) error {
    // Generate URLs to related endpoints
    createURL, _ := mux.Reverse("users", nil)  // Simplified: no method needed
    
    return ctx.JSON(200, map[string]interface{}{
        "users": getUsersList(),
        "links": map[string]string{
            "create": createURL,
        },
    })
})

mux.Get("user", "/users/{id}", func(ctx *HttpContext) error {
    id := ctx.Param("id")
    
    // Generate URLs with dynamic parameters (no method needed)
    editURL, _ := mux.Reverse("user", map[string]string{"id": id})
    
    return ctx.JSON(200, map[string]interface{}{
        "user": getUser(id),
        "links": map[string]string{
            "edit": editURL,  // Same URL for PUT, DELETE since same pattern
        },
    })
})
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

**CORS Middleware**
```go
// Default CORS configuration (allows all origins)
handler := srv.CORS(srv.DefaultCORSConfig)(mux)

// Custom CORS configuration
corsConfig := srv.CORSConfig{
    AllowOrigins:     []string{"https://example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
}
handler := srv.CORS(corsConfig)(mux)
```
Handles Cross-Origin Resource Sharing with configurable origins, methods, headers, and security options

**Trailing Slash Middleware**
```go
// Default configuration (internal forward)
handler := srv.AddTrailingSlash(srv.DefaultTrailingSlashConfig)(mux)

// Redirect configuration
redirectConfig := srv.TrailingSlashConfig{RedirectCode: 301}
handler := srv.AddTrailingSlash(redirectConfig)(mux)
```
Adds trailing slashes to URLs for consistency and SEO. Can either redirect or forward internally

#### Middleware Chaining
```go
// Chain multiple middleware (applied in reverse order)
handler := srv.MiddlewareChain(
    srv.Logging,                                // Outermost: logs all requests
    srv.Recover,                                // Recovers from panics
    srv.AddTrailingSlash(srv.DefaultTrailingSlashConfig), // URL normalization
    srv.CORS(srv.DefaultCORSConfig),           // Built-in CORS middleware
    RateLimiting,                               // Innermost: rate limiting
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

// Users API with enhanced naming and URL generation
mux.Get("users", "/users", listUsers)
mux.Post("users", "/users", createUser)
mux.Get("user", "/users/{id}", getUser)
mux.Put("user", "/users/{id}", updateUser)
mux.Delete("user", "/users/{id}", deleteUser)

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
    
    // Generate URLs to related endpoints
    editURL, _ := mux.Reverse("user", map[string]string{"id": id})
    
    // Return JSON response with links
    return ctx.JSON(200, map[string]interface{}{
        "user": user,
        "links": map[string]string{
            "edit": editURL,
        },
    })
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

// Custom CORS configuration example
corsConfig := srv.CORSConfig{
    AllowOrigins:     []string{"https://example.com", "https://app.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
}
corsMiddleware := srv.CORS(corsConfig)

// Chain all middleware
mux := srv.NewMux()
handler := srv.MiddlewareChain(
    srv.Logging,                                // Log all requests
    srv.Recover,                                // Panic recovery
    srv.AddTrailingSlash(srv.DefaultTrailingSlashConfig), // URL normalization
    corsMiddleware,                             // CORS headers with custom config
    Authentication,                             // JWT authentication
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
- ✅ **URL Reversing**: Named routes, parameter substitution, edge cases, integration
- ✅ **Middleware**: Type safety, chaining, logging, recovery, CORS, trailing slash
- ✅ **RunServer**: Parameter validation, error handling, cleanup
- ✅ **Integration**: Cross-component functionality tests
- ✅ **Benchmarks**: Performance testing for all components

**Current Coverage**: 94.2% of statements

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
    srv.Logging,                                // First: log everything
    srv.Recover,                                // Second: catch panics
    srv.AddTrailingSlash(srv.DefaultTrailingSlashConfig), // Third: URL normalization
    srv.CORS(srv.DefaultCORSConfig),           // Fourth: CORS headers
    Authentication,                             // Fifth: auth before business logic
    RateLimiting,                               // Last: rate limiting
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

### 4. CORS Configuration
Configure CORS appropriately for your security requirements:

```go
// Development - permissive CORS (or use DefaultCORSConfig)
devCorsConfig := srv.CORSConfig{
    AllowOrigins: []string{"*"},
}
corsMiddleware := srv.CORS(devCorsConfig)
// Or simply: corsMiddleware := srv.CORS(srv.DefaultCORSConfig)

// Production - restrictive CORS
prodCorsConfig := srv.CORSConfig{
    AllowOrigins:     []string{"https://yourdomain.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
}
corsMiddleware := srv.CORS(prodCorsConfig)
```

### 5. Trailing Slash Configuration
Configure trailing slash behavior based on your needs:

```go
// Default - internal forward (no redirect)
trailingSlashMiddleware := srv.AddTrailingSlash(srv.DefaultTrailingSlashConfig)

// SEO-friendly permanent redirect
seoConfig := srv.TrailingSlashConfig{RedirectCode: 301}
trailingSlashMiddleware := srv.AddTrailingSlash(seoConfig)

// Temporary redirect for testing
testConfig := srv.TrailingSlashConfig{RedirectCode: 302}
trailingSlashMiddleware := srv.AddTrailingSlash(testConfig)
```

### 6. Resource Cleanup
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