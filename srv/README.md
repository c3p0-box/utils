# srv - HTTP Server Utilities

The `srv` package provides HTTP server utilities with middleware support and graceful shutdown capabilities for Go applications.

## Features

- **Middleware System**: Composable HTTP middleware with easy chaining
- **Built-in Middleware**: Logging and panic recovery middleware included
- **Graceful Shutdown**: HTTP server with signal-based graceful shutdown
- **Structured Logging**: Integration with Go's structured logging (`log/slog`)
- **Zero Dependencies**: Only uses Go standard library

## Installation

```bash
go get github.com/c3p0-box/utils/srv
```

## Quick Start

```go
package main

import (
    "net/http"
    "log"
    "github.com/c3p0-box/utils/srv"
)

func main() {
    // Create your HTTP handler
    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    // Chain middleware (logging and recovery)
    handler := srv.MiddlewareChain(srv.Logging, srv.Recover)(mux)
    
    // Run server with graceful shutdown
    err := srv.RunServer(handler, "localhost", "8080", func() error {
        // Cleanup logic here (close DB connections, etc.)
        log.Println("Cleaning up...")
        return nil
    })
    
    if err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### Types

#### `Middleware`

```go
type Middleware func(next http.Handler) http.Handler
```

A middleware function that wraps an HTTP handler with additional functionality.

#### `MiddlewareWriter`

```go
type MiddlewareWriter struct {
    http.ResponseWriter
    StatusCode int
}
```

A response writer wrapper that captures the HTTP status code for logging purposes.

### Functions

#### `MiddlewareChain(m ...Middleware) Middleware`

Combines multiple middleware functions into a single middleware. Middleware are applied in reverse order (first middleware in the list becomes the outermost wrapper).

**Example:**
```go
// This creates: Logging(Recover(Cors(handler)))
chain := srv.MiddlewareChain(srv.Logging, srv.Recover, Cors)
wrappedHandler := chain(handler)
```

#### `Logging(next http.Handler) http.Handler`

Middleware that logs HTTP requests using structured logging. Captures:
- HTTP method and path
- Response status code
- User agent and remote address
- Request completion message

**Log Output Example:**
```
INFO request completed name=views.Logging status=200 method=GET path=/api/users user-agent="Mozilla/5.0..." remote-addr="192.168.1.1:12345"
```

#### `Recover(next http.Handler) http.Handler`

Middleware that recovers from panics during request processing and logs them. Prevents the server from crashing when handlers panic.

**Log Output Example:**
```
ERROR recovered from panic name=views.Recover error="runtime error: index out of range"
```

#### `RunServer(handler http.Handler, host string, port string, cleanup func() error) error`

Starts an HTTP server with graceful shutdown capabilities.

**Parameters:**
- `handler`: The HTTP handler to serve requests
- `host`: Host to bind to (defaults to "0.0.0.0" if empty)
- `port`: Port to listen on (defaults to "8000" if empty)  
- `cleanup`: Function called during shutdown for resource cleanup

**Graceful Shutdown Process:**
1. Listens for SIGINT/SIGTERM signals
2. Stops accepting new requests
3. Waits up to 10 seconds for existing requests to complete
4. Calls the cleanup function
5. Returns any errors that occurred

## Usage Examples

### Basic Server

```go
mux := http.NewServeMux()
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello, World!"))
})

err := srv.RunServer(mux, "", "", func() error {
    return nil // No cleanup needed
})
```

### Server with Middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
    // This might panic sometimes
    data := getData()
    json.NewEncoder(w).Encode(data)
})

// Add logging and panic recovery
handler := srv.MiddlewareChain(srv.Logging, srv.Recover)(mux)

err := srv.RunServer(handler, "0.0.0.0", "8080", func() error {
    return db.Close() // Close database connection
})
```

### Custom Middleware

```go
// Create custom middleware
func CORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        next.ServeHTTP(w, r)
    })
}

func RateLimiting(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Rate limiting logic here
        next.ServeHTTP(w, r)
    })
}

// Chain custom middleware with built-in ones
handler := srv.MiddlewareChain(
    srv.Logging,     // Outermost: logs everything
    srv.Recover,     // Recovers from panics
    CORS,            // Adds CORS headers
    RateLimiting,    // Innermost: rate limiting
)(mux)
```

### Server with Database Cleanup

```go
db, err := sql.Open("postgres", connectionString)
if err != nil {
    log.Fatal(err)
}

handler := srv.MiddlewareChain(srv.Logging, srv.Recover)(mux)

err = srv.RunServer(handler, "localhost", "8080", func() error {
    log.Println("Closing database connection...")
    return db.Close()
})
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
```

### Test Coverage

- ✅ Middleware type and basic functionality
- ✅ MiddlewareChain with single and multiple middleware
- ✅ MiddlewareWriter status code capture
- ✅ Logging middleware request capture
- ✅ Recover middleware panic handling
- ✅ RunServer parameter validation and error handling
- ✅ Integration tests with middleware chains

**Note**: `RunServer` function testing is limited to parameter validation and error conditions since the function is designed to run indefinitely waiting for OS signals. Full server lifecycle testing requires integration tests in production-like environments.

## Best Practices

1. **Middleware Order**: Place logging middleware first to capture all requests, even those that panic.

2. **Panic Recovery**: Always include recovery middleware in production to prevent server crashes.

3. **Graceful Shutdown**: Always provide a cleanup function to properly close resources.

4. **Structured Logging**: The logging middleware uses structured logging for better observability.

## Configuration

### Environment Variables

The package respects standard Go HTTP server practices. You can configure timeouts and other server settings by creating a custom `http.Server`:

```go
server := &http.Server{
    Addr:         "localhost:8080",
    Handler:      handler,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

For more complex server configurations, you may want to use the individual middleware components with your own server setup rather than `RunServer`.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This package is part of the c3p0-box/utils collection and follows the same license terms.
