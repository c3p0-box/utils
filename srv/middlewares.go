package srv

import (
	"log/slog"
	"net/http"
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
