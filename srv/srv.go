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
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Middleware represents an HTTP middleware function that takes an http.Handler
// and returns a new http.Handler, typically wrapping the original handler
// with additional functionality.
type Middleware func(next http.Handler) http.Handler

// MiddlewareChain combines multiple middleware functions into a single middleware.
// The middleware are applied in reverse order, so the first middleware in the list
// will be the outermost wrapper.
//
// Example:
//
//	chain := MiddlewareChain(Logging, Recover)
//	handler := chain(mux) // Results in: Logging(Recover(mux))
func MiddlewareChain(m ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}
		return next
	}
}

// MiddlewareWriter is a wrapper around http.ResponseWriter that captures
// the HTTP status code for logging purposes. It implements http.ResponseWriter
// and stores the status code when WriteHeader is called.
type MiddlewareWriter struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader captures the status code and calls the underlying ResponseWriter's WriteHeader.
// If WriteHeader is not called explicitly, the status code defaults to http.StatusOK.
func (w *MiddlewareWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Logging is a middleware that logs HTTP requests with structured logging using slog.
// It captures the HTTP method, path, status code, user agent, and remote address.
// The status code is captured using MiddlewareWriter.
//
// The logged information includes:
//   - name: "views.Logging" (logger identifier)
//   - status: HTTP status code
//   - method: HTTP method (GET, POST, etc.)
//   - path: Request URL path
//   - user-agent: Client user agent string
//   - remote-addr: Client remote address
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mw := &MiddlewareWriter{w, http.StatusOK}
		next.ServeHTTP(mw, r)
		slog.With(
			slog.String("name", "views.Logging"),
			slog.Int("status", mw.StatusCode),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("user-agent", r.UserAgent()),
			slog.String("remote-addr", r.RemoteAddr),
		).Info("request completed")
	})
}

// Recover is a middleware that recovers from panics during HTTP request processing.
// When a panic occurs, it logs the error using structured logging and allows the
// request to complete gracefully instead of crashing the server.
//
// The panic is logged with:
//   - name: "views.Recover" (logger identifier)
//   - error: The recovered panic value
//
// After logging, the middleware allows the panic to continue, which will typically
// result in a 500 Internal Server Error response.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.With(
					slog.String("name", "views.Recover"),
					slog.Any("error", err),
				).Error("recovered from panic")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

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
