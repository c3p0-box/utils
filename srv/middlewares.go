package srv

import (
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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

// =============================================================================
// CORS Middleware
// =============================================================================

// CORSConfig defines the configuration for CORS middleware.
type CORSConfig struct {
	// AllowOrigins determines the value of the Access-Control-Allow-Origin
	// response header. This header defines a list of origins that may access the
	// resource. The wildcard characters '*' and '?' are supported and are
	// converted to regex fragments '.*' and '.' accordingly.
	//
	// Security: use extreme caution when handling the origin, and carefully
	// validate any logic. Remember that attackers may register hostile domain names.
	//
	// Optional. Default value []string{"*"}.
	AllowOrigins []string

	// AllowMethods determines the value of the Access-Control-Allow-Methods
	// response header. This header specifies the list of methods allowed when
	// accessing the resource. This is used in response to a preflight request.
	//
	// Optional. Default value []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE"}.
	AllowMethods []string

	// AllowHeaders determines the value of the Access-Control-Allow-Headers
	// response header. This header is used in response to a preflight request to
	// indicate which HTTP headers can be used when making the actual request.
	//
	// Optional. Default value []string{}.
	AllowHeaders []string

	// AllowCredentials determines the value of the Access-Control-Allow-Credentials
	// response header. This header indicates whether or not the response to the
	// request can be exposed when the credentials mode is true.
	//
	// Optional. Default value false.
	// Security: avoid using AllowCredentials = true with AllowOrigins = "*".
	AllowCredentials bool

	// ExposeHeaders determines the value of Access-Control-Expose-Headers, which
	// defines a list of headers that clients are allowed to access.
	//
	// Optional. Default value []string{}.
	ExposeHeaders []string

	// MaxAge determines the value of the Access-Control-Max-Age response header.
	// This header indicates how long (in seconds) the results of a preflight
	// request can be cached.
	//
	// Optional. Default value 0 - meaning header is not sent.
	MaxAge int
}

// DefaultCORSConfig is the default CORS middleware config.
var DefaultCORSConfig = CORSConfig{
	AllowOrigins: []string{"*"},
	AllowMethods: []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPut,
		http.MethodPatch,
		http.MethodPost,
		http.MethodDelete,
	},
	AllowHeaders:     []string{},
	AllowCredentials: false,
	ExposeHeaders:    []string{},
	MaxAge:           0,
}

// CORS returns a Cross-Origin Resource Sharing (CORS) middleware with the provided configuration.
//
// Security: Poorly configured CORS can compromise security because it allows
// relaxation of the browser's Same-Origin policy. Use caution when configuring
// origins and credentials.
//
// Example usage:
//
//	// Use default configuration (allows all origins)
//	corsMiddleware := srv.CORS(srv.DefaultCORSConfig)
//
//	// Use custom configuration
//	corsConfig := srv.CORSConfig{
//		AllowOrigins:     []string{"https://example.com", "https://app.example.com"},
//		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
//		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
//		AllowCredentials: true,
//		MaxAge:           3600,
//	}
//	corsMiddleware := srv.CORS(corsConfig)
//	handler := srv.MiddlewareChain(corsMiddleware)(mux)
func CORS(config CORSConfig) Middleware {
	// Apply defaults if not set
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = DefaultCORSConfig.AllowMethods
	}

	// Compile origin patterns for wildcard support
	allowOriginPatterns := make([]*regexp.Regexp, 0, len(config.AllowOrigins))
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			continue // "*" is handled separately
		}
		// Convert wildcard patterns to regex
		pattern := regexp.QuoteMeta(origin)
		pattern = strings.ReplaceAll(pattern, "\\*", ".*")
		pattern = strings.ReplaceAll(pattern, "\\?", ".")
		pattern = "^" + pattern + "$"
		if re, err := regexp.Compile(pattern); err == nil {
			allowOriginPatterns = append(allowOriginPatterns, re)
		}
	}

	// Pre-compute header values
	allowMethods := strings.Join(config.AllowMethods, ", ")
	allowHeaders := strings.Join(config.AllowHeaders, ", ")
	exposeHeaders := strings.Join(config.ExposeHeaders, ", ")
	maxAge := ""
	if config.MaxAge > 0 {
		maxAge = strconv.Itoa(config.MaxAge)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Always add Vary header for Origin
			w.Header().Add("Vary", "Origin")

			// Handle preflight requests (OPTIONS method with CORS headers)
			isPreflight := r.Method == http.MethodOptions &&
				r.Header.Get("Access-Control-Request-Method") != ""

			// If no origin, this is likely not a CORS request
			if origin == "" {
				if !isPreflight {
					next.ServeHTTP(w, r)
					return
				}
				// For preflight without origin, return 204
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Check if origin is allowed
			allowOrigin := ""

			// Check exact matches and wildcard
			for _, o := range config.AllowOrigins {
				if o == "*" {
					if !config.AllowCredentials {
						allowOrigin = "*"
						break
					}
					// If credentials are allowed, we need to echo the specific origin
					allowOrigin = origin
					break
				}
				if o == origin {
					allowOrigin = origin
					break
				}
			}

			// Check pattern matches if no exact match found
			if allowOrigin == "" {
				for _, pattern := range allowOriginPatterns {
					if pattern.MatchString(origin) {
						allowOrigin = origin
						break
					}
				}
			}

			// Origin not allowed
			if allowOrigin == "" {
				if !isPreflight {
					next.ServeHTTP(w, r)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle simple requests
			if !isPreflight {
				if exposeHeaders != "" {
					w.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
				}
				next.ServeHTTP(w, r)
				return
			}

			// Handle preflight requests
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")

			w.Header().Set("Access-Control-Allow-Methods", allowMethods)

			if allowHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
			} else {
				// Echo the requested headers if no specific headers configured
				if requestedHeaders := r.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
					w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
				}
			}

			if maxAge != "" {
				w.Header().Set("Access-Control-Max-Age", maxAge)
			}

			w.WriteHeader(http.StatusNoContent)
		})
	}
}
