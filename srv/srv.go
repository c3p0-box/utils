// Package srv provides HTTP server utilities with middleware support and graceful shutdown.
//
// This package offers a collection of HTTP middleware components and a server runner
// that supports graceful shutdown with signal handling. It includes logging and recovery
// middleware, along with utilities for chaining multiple middleware together.
//
// Example usage:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte("OK"))
//	})
//
//	// Chain middleware
//	handler := MiddlewareChain(Logging, Recover)(mux)
//
//	// Run server with graceful shutdown
//	err := RunServer(handler, "localhost", "8080", func() error {
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
	"syscall"
	"time"
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

// HandlerFunc defines a handler function that receives an HttpContext and returns an error.
// This allows for more elegant error handling compared to traditional http.HandlerFunc.
// If the handler returns an error, it will be passed to the configured error handler.
//
// Example:
//
//	func getUserHandler(ctx *HttpContext) error {
//	    id := ctx.Param("id")
//	    user, err := getUserByID(id)
//	    if err != nil {
//	        return err  // Error will be handled by error handler
//	    }
//	    return ctx.JSON(200, user)
//	}
type HandlerFunc func(ctx *HttpContext) error

// Mux provides a convenient wrapper around Go's standard http.ServeMux
// with helper methods for common HTTP operations, RESTful routing, and centralized error handling.
//
// The Mux supports two types of handlers:
//   - Traditional http.Handler and http.HandlerFunc via Handle() and HandleFunc()
//   - Enhanced HandlerFunc via HTTP method helpers (Get, Post, etc.) with automatic error handling
type Mux struct {
	mux        *http.ServeMux
	errHandler func(ctx *HttpContext, err error)
}

// NewMux creates a new Mux instance with an underlying http.ServeMux and a default error handler.
// The default error handler responds with a 500 Internal Server Error and logs the error.
func NewMux() *Mux {
	return &Mux{
		mux: http.NewServeMux(),
		errHandler: func(ctx *HttpContext, err error) {
			ctx.JSON(500, map[string]string{"error": "Internal Server Error"})
		},
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
//	mux.ErrorHandler(func(ctx *HttpContext, err error) {
//	    log.Printf("Handler error: %v", err)
//	    ctx.JSON(500, map[string]string{"error": err.Error()})
//	})
func (m *Mux) ErrorHandler(handler func(c *HttpContext, err error)) {
	m.errHandler = handler
}

// execHandler is an internal method that wraps HandlerFunc with error handling.
// It creates an HttpContext and passes it to the handler. If the handler returns
// an error, it calls the configured error handler with the same context.
func (m *Mux) execHandler(pattern string, handler HandlerFunc) {
	m.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := NewHttpContext(w, r)
		if err := handler(ctx); err != nil {
			m.errHandler(ctx, err)
		}
	})
}

// ============================
// HTTP Method Helpers
// ============================

// Get registers a handler for GET requests to the specified pattern.
func (m *Mux) Get(pattern string, handler HandlerFunc) {
	m.execHandler("GET "+pattern, handler)
}

// Post registers a handler for POST requests to the specified pattern.
func (m *Mux) Post(pattern string, handler HandlerFunc) {
	m.execHandler("POST "+pattern, handler)
}

// Put registers a handler for PUT requests to the specified pattern.
func (m *Mux) Put(pattern string, handler HandlerFunc) {
	m.execHandler("PUT "+pattern, handler)
}

// Delete registers a handler for DELETE requests to the specified pattern.
func (m *Mux) Delete(pattern string, handler HandlerFunc) {
	m.execHandler("DELETE "+pattern, handler)
}

// Patch registers a handler for PATCH requests to the specified pattern.
func (m *Mux) Patch(pattern string, handler HandlerFunc) {
	m.execHandler("PATCH "+pattern, handler)
}

// Head registers a handler for HEAD requests to the specified pattern.
func (m *Mux) Head(pattern string, handler HandlerFunc) {
	m.execHandler("HEAD "+pattern, handler)
}

// Options registers a handler for OPTIONS requests to the specified pattern.
func (m *Mux) Options(pattern string, handler HandlerFunc) {
	m.execHandler("OPTIONS "+pattern, handler)
}
