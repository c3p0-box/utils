// Package srv provides HTTP server utilities with middleware support, graceful shutdown,
// enhanced routing with URL reversing, and standardized error handling via the erm package.
//
// This package offers a collection of HTTP middleware components, a server runner
// that supports graceful shutdown with signal handling, and an enhanced router with
// URL reversing capabilities. It includes logging and recovery middleware, along with
// utilities for chaining multiple middleware together. All errors use the erm package
// for consistent HTTP status codes and internationalized error messages.
//
// Example usage:
//
//	mux := srv.NewMux()
//	mux.Get("users", "/users", func(ctx srv.Context) error {
//		return ctx.JSON(200, users)
//	})
//	mux.Get("user", "/users/{id}", func(ctx srv.Context) error {
//		id := ctx.Param("id")
//		editURL, _ := mux.Reverse("user", map[string]string{"id": id})
//		return ctx.JSON(200, map[string]interface{}{
//			"user": getUser(id),
//			"edit_url": editURL,
//		})
//	})
//
//	// Chain middleware
//	handler := srv.MiddlewareChain(srv.Logging, srv.Recover)(mux)
//
//	// Run server with graceful shutdown
//	err := srv.RunServer(handler, "localhost", "8080", func() error {
//		// cleanup logic here
//		return nil
//	})
package srv

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/c3p0-box/utils/erm"
)

const (
	charsetUTF8 = "charset=UTF-8"
	// MIMEApplicationJSON JavaScript Object Notation (JSON) https://www.rfc-editor.org/rfc/rfc8259
	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = MIMEApplicationJavaScript + "; " + charsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + charsetUTF8
	MIMETextXML                          = "text/xml"
	MIMETextXMLCharsetUTF8               = MIMETextXML + "; " + charsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
)

// RunServer starts an HTTP server with graceful shutdown capabilities.
// It listens on the specified host and port, and shuts down gracefully when
// receiving SIGINT or SIGTERM signals.
//
// Parameters:
//   - handler: The HTTP handler to serve requests
//   - host: The host to bind to (defaults to "0.0.0.0" if empty)
//   - port: The port to listen on (defaults to "8000" if empty)
//   - cleanup: A function called during shutdown for resource cleanup
//
// The server performs the following shutdown sequence:
//  1. Receives shutdown signal (SIGINT/SIGTERM)
//  2. Stops accepting new requests
//  3. Waits up to 10 seconds for existing requests to complete
//  4. Calls the cleanup function
//  5. Returns any errors that occurred
//
// Returns an error if the server fails to start or if shutdown encounters an error.
//
// Example:
//
//	err := RunServer(handler, "localhost", "8080", func() error {
//		// Close database connections, cleanup resources, etc.
//		return db.Close()
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
func RunServer(handler http.Handler, host string, port string, cleanup func() error) error {
	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = "8000"
	}

	server := http.Server{
		Addr:    host + ":" + port,
		Handler: handler,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		close(serverErrCh)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case <-stop:
		// proceed to shut down
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
		// server closed without error
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	if err := cleanup(); err != nil {
		return err
	}
	return nil
}

// ============================
// Mux - HTTP Router Wrapper
// ============================

// Route represents a named route with its URL pattern.
// Multiple HTTP methods can share the same route if they have the same pattern.
// This is used internally for URL reversing functionality.
type Route struct {
	Name    string // The route name for URL reversing
	Pattern string // URL pattern (e.g., "/users/{id}")
}

// HandlerFunc defines a handler function that receives an Context and returns an error.
// This allows for more elegant error handling compared to traditional http.HandlerFunc.
// If the handler returns an error, it will be passed to the configured error handler.
//
// Example:
//
//	func getUserHandler(ctx Context) error {
//	    id := ctx.Param("id")
//	    user, err := getUserByID(id)
//	    if err != nil {
//	        return err  // Error will be handled by error handler
//	    }
//	    return ctx.JSON(200, user)
//	}
type HandlerFunc func(ctx Context) error

// HandlerFuncMiddleware represents middleware that works with HandlerFunc.
// It takes a HandlerFunc and returns a new HandlerFunc, allowing middleware
// to be chained while maintaining the srv package's error handling pattern.
// This enables middleware to work directly with the Context interface and
// maintain the elegant error handling approach.
//
// Example:
//
//	func AuthMiddleware(next HandlerFunc) HandlerFunc {
//	    return func(ctx Context) error {
//	        token := ctx.GetHeader("Authorization")
//	        if token == "" {
//	            return erm.Unauthorized("missing authorization header", nil)
//	        }
//	        // Store authenticated user in context
//	        ctx.Set("user", getUserFromToken(token))
//	        return next(ctx)  // Continue to next handler
//	    }
//	}
type HandlerFuncMiddleware func(next HandlerFunc) HandlerFunc

// Mux provides a convenient wrapper around Go's standard http.ServeMux
// with helper methods for common HTTP operations, RESTful routing, URL reversing, and centralized error handling.
//
// The Mux supports two types of handlers:
//   - Traditional http.Handler and http.HandlerFunc via Handle() and HandleFunc()
//   - Enhanced HandlerFunc via HTTP method helpers (Get, Post, etc.) with automatic error handling
//   - HandlerFunc middleware via the Middleware() method for chaining Context-aware middleware
type Mux struct {
	mux         *http.ServeMux
	errHandler  func(ctx Context, err error)
	routes      map[string]Route        // Named routes for URL reversing, key format: "name"
	routesMu    sync.RWMutex            // Protects routes map from concurrent access
	middlewares []HandlerFuncMiddleware // HandlerFunc middleware stack
}

// NewMux creates a new Mux instance with an underlying http.ServeMux and a default error handler.
// The default error handler responds with a 500 Internal Server Error and logs the error.
// The returned Mux is safe for concurrent use by multiple goroutines.
func NewMux() *Mux {
	return &Mux{
		mux: http.NewServeMux(),
		errHandler: func(ctx Context, err error) {
			_ = ctx.String(http.StatusInternalServerError, "Something went wrong")
		},
		routes:      make(map[string]Route),
		routesMu:    sync.RWMutex{},
		middlewares: make([]HandlerFuncMiddleware, 0),
	}
}

// Mux returns the underlying http.ServeMux for advanced usage or
// integration with other HTTP libraries.
func (m *Mux) Mux() *http.ServeMux {
	return m.mux
}

// ServeHTTP implements http.Handler interface, allowing Mux to be used
// directly as an HTTP handler.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mux.ServeHTTP(w, r)
}

// Handle registers a handler for the given pattern.
func (m *Mux) Handle(pattern string, handler http.Handler) {
	m.mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for the given pattern.
func (m *Mux) HandleFunc(pattern string, handler http.HandlerFunc) {
	m.mux.HandleFunc(pattern, handler)
}

// ErrorHandler sets a custom error handler for all HandlerFunc-based routes.
// The error handler will be called whenever a HandlerFunc returns a non-nil error.
// If no custom error handler is set, the default handler returns a 500 Internal Server Error.
//
// Example:
//
//	mux.ErrorHandler(func(ctx Context, err error) {
//	    log.Printf("Handler error: %v", err)
//	    ctx.JSON(500, map[string]string{"error": err.Error()})
//	})
func (m *Mux) ErrorHandler(handler func(c Context, err error)) {
	m.errHandler = handler
}

// Middleware adds HandlerFunc-based middleware to the Mux.
// Middleware will be applied to all routes registered after this method is called.
// Middleware are applied in the order they are added (first added = outermost wrapper).
//
// The middleware function receives the next HandlerFunc in the chain and returns
// a new HandlerFunc that typically calls the next function with additional logic
// before, after, or around the call. This allows middleware to work directly with
// the Context interface and maintain the elegant error handling pattern.
//
// Example:
//
//	// Add authentication middleware
//	mux.Middleware(func(next HandlerFunc) HandlerFunc {
//	    return func(ctx Context) error {
//	        token := ctx.GetHeader("Authorization")
//	        if token == "" {
//	            return erm.Unauthorized("missing authorization header", nil)
//	        }
//	        // Continue to next handler
//	        return next(ctx)
//	    }
//	})
//
//	// Add timing middleware
//	mux.Middleware(func(next HandlerFunc) HandlerFunc {
//	    return func(ctx Context) error {
//	        start := time.Now()
//	        err := next(ctx)
//	        duration := time.Since(start)
//	        log.Printf("Request took %v", duration)
//	        return err
//	    }
//	})
func (m *Mux) Middleware(middleware HandlerFuncMiddleware) {
	m.middlewares = append(m.middlewares, middleware)
}

// applyMiddleware applies all registered HandlerFunc middleware to a handler.
// Middleware are applied in reverse order so that the first added middleware
// becomes the outermost wrapper, which is the expected behavior.
func (m *Mux) applyMiddleware(handler HandlerFunc) HandlerFunc {
	// Apply middleware in reverse order (last added = innermost)
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		handler = m.middlewares[i](handler)
	}
	return handler
}

// execHandler is an internal method that wraps HandlerFunc with error handling,
// applies registered middleware, and registers named routes for URL reversing when a name is provided.
// It creates a Context and passes it to the middleware chain and handler. If the handler returns
// an error, it calls the configured error handler with the same context.
// This method is safe for concurrent use.
func (m *Mux) execHandler(name, method, pattern string, handler HandlerFunc) {
	// Register named route if name is provided
	if name != "" {
		// Protect routes map access with write lock
		m.routesMu.Lock()
		m.routes[name] = Route{Name: name, Pattern: pattern}
		m.routesMu.Unlock()
	}

	// Apply all registered middleware to the handler
	finalHandler := m.applyMiddleware(handler)

	// Register the handler with the HTTP mux
	fullPattern := method + " " + pattern
	m.mux.HandleFunc(fullPattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := NewHttpContext(w, r)
		if err := finalHandler(ctx); err != nil {
			m.errHandler(ctx, err)
		}
	})
}

// ============================
// HTTP Method Helpers
// ============================

// Get registers a named handler for GET requests to the specified pattern.
// If name is provided, the route can be used for URL generation via the Reverse method.
// If name is empty string, the route is registered without naming.
//
// Example:
//
//	m.Get("user-list", "/users", handler)                    // Named route
//	m.Get("", "/health", handler)                            // Unnamed route
//	url, err := m.Reverse("user-list", nil)                  // Generate: "/users"
func (m *Mux) Get(name, pattern string, handler HandlerFunc) {
	m.execHandler(name, "GET", pattern, handler)
}

// Post registers a named handler for POST requests to the specified pattern.
// If name is provided, the route can be used for URL generation via the Reverse method.
// If name is empty string, the route is registered without naming.
func (m *Mux) Post(name, pattern string, handler HandlerFunc) {
	m.execHandler(name, "POST", pattern, handler)
}

// Put registers a named handler for PUT requests to the specified pattern.
// If name is provided, the route can be used for URL generation via the Reverse method.
// If name is empty string, the route is registered without naming.
func (m *Mux) Put(name, pattern string, handler HandlerFunc) {
	m.execHandler(name, "PUT", pattern, handler)
}

// Delete registers a named handler for DELETE requests to the specified pattern.
// If name is provided, the route can be used for URL generation via the Reverse method.
// If name is empty string, the route is registered without naming.
func (m *Mux) Delete(name, pattern string, handler HandlerFunc) {
	m.execHandler(name, "DELETE", pattern, handler)
}

// Patch registers a named handler for PATCH requests to the specified pattern.
// If name is provided, the route can be used for URL generation via the Reverse method.
// If name is empty string, the route is registered without naming.
func (m *Mux) Patch(name, pattern string, handler HandlerFunc) {
	m.execHandler(name, "PATCH", pattern, handler)
}

// Head registers a named handler for HEAD requests to the specified pattern.
// If name is provided, the route can be used for URL generation via the Reverse method.
// If name is empty string, the route is registered without naming.
func (m *Mux) Head(name, pattern string, handler HandlerFunc) {
	m.execHandler(name, "HEAD", pattern, handler)
}

// Options registers a named handler for OPTIONS requests to the specified pattern.
// If name is provided, the route can be used for URL generation via the Reverse method.
// If name is empty string, the route is registered without naming.
func (m *Mux) Options(name, pattern string, handler HandlerFunc) {
	m.execHandler(name, "OPTIONS", pattern, handler)
}

// ============================
// URL Reversing
// ============================

// Reverse generates a URL for the named route with the provided parameters.
// Parameters should be provided as a map where keys match the path parameter names in the
// route pattern (e.g., "id" for "/users/{id}").
//
// Multiple HTTP methods can share the same route name if they have the same pattern,
// enabling RESTful route naming (e.g., "users" for both GET /users and POST /users).
//
// Parameters:
//   - name: The route name specified when registering the route
//   - params: Map of parameter names to values for substitution
//
// Returns the generated URL path and any error encountered. Errors are returned as
// erm.Error instances: erm.NotFound for unknown routes and erm.RequiredError for
// missing required parameters.
//
// Example:
//
//	m.Get("user-profile", "/users/{id}", handler)
//	m.Post("users", "/users", handler)
//
//	// Generate URLs
//	profileURL, err := m.Reverse("user-profile", map[string]string{"id": "123"})
//	// Returns: "/users/123"
//
//	usersURL, err := m.Reverse("users", nil)
//	// Returns: "/users"
//
//	// Error handling
//	_, err := m.Reverse("non-existent", nil)
//	// Returns: erm.NotFound error with 404 status
//
//	_, err := m.Reverse("user-profile", nil)
//	// Returns: erm.RequiredError for missing {id} parameter
func (m *Mux) Reverse(name string, params map[string]string) (string, error) {
	// Protect routes map access with read lock
	m.routesMu.RLock()
	route, exists := m.routes[name]
	m.routesMu.RUnlock()

	if !exists {
		return "", erm.NotFound(name, nil)
	}

	url := route.Pattern

	// Replace path parameters if provided
	for paramName, paramValue := range params {
		placeholder := "{" + paramName + "}"
		if strings.Contains(url, placeholder) {
			url = strings.ReplaceAll(url, placeholder, paramValue)
		}
		// Ignore parameters that don't exist in the pattern
	}

	// Check if there are any unreplaced parameters
	if strings.Contains(url, "{") && strings.Contains(url, "}") {
		return "", erm.RequiredError(name, route.Pattern)
	}

	return url, nil
}
