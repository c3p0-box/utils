package srv

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Register common types for gob encoding/decoding in cookie store
func init() {
	gob.Register(map[string]interface{}{})
	gob.Register(map[string]int{})
	gob.Register(map[string]string{})
	gob.Register(map[int]interface{}{})
	gob.Register(map[int]int{})
	gob.Register(map[int]string{})
	gob.Register([]string{})
	gob.Register([]int{})
	gob.Register([]interface{}{})
}

// =============================================================================
// HandlerFunc-based Middleware for Context-Aware Operations
// =============================================================================

// LoggingMiddleware is a HandlerFunc-based middleware that logs HTTP requests
// with structured logging using slog. It works directly with the Context interface
// and maintains the elegant error handling pattern.
//
// The logged information includes:
//   - name: "srv.Logging" (logger identifier)
//   - method: HTTP method (GET, POST, etc.)
//   - path: Request URL path
//   - user-agent: Client user agent string
//   - remote-addr: Client remote address
//   - duration: Request processing time
//
// Note: This middleware cannot capture the exact status code since it works at the
// HandlerFunc level, but it provides comprehensive logging of request information.
//
// Example:
//
//	mux.Middleware(srv.LoggingMiddleware)
func LoggingMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx Context) error {
		start := time.Now()

		// Execute the next handler
		err := next(ctx)

		// Log the request
		duration := time.Since(start)
		req := ctx.Request()
		slog.With(
			slog.String("name", "srv.Logging"),
			slog.String("method", req.Method),
			slog.String("path", req.URL.Path),
			slog.String("user-agent", req.UserAgent()),
			slog.String("remote-addr", req.RemoteAddr),
			slog.Duration("duration", duration),
		).Info("request completed")

		return err
	}
}

// RecoverMiddleware is a HandlerFunc-based middleware that recovers from panics
// during HTTP request processing. It works directly with the Context interface
// and maintains the elegant error handling pattern.
//
// When a panic occurs, it logs the error using structured logging and converts
// the panic to an error that can be handled by the error handler.
//
// The panic is logged with:
//   - name: "srv.Recover" (logger identifier)
//   - error: The recovered panic value
//   - path: Request URL path
//   - method: HTTP method
//
// Example:
//
//	mux.Middleware(srv.RecoverMiddleware)
func RecoverMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.With(
					slog.String("name", "srv.Recover"),
					slog.Any("error", r),
					slog.String("path", ctx.Request().URL.Path),
					slog.String("method", ctx.Request().Method),
				).Error("recovered from panic")

				// Convert panic to error
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("panic: %v", r)
				}
			}
		}()

		return next(ctx)
	}
}

// CORSMiddleware returns a HandlerFunc-based CORS middleware that handles Cross-Origin Resource
// Sharing (CORS) according to the W3C specification. It supports both simple and preflight
// requests with comprehensive configuration options for security and compatibility.
//
// Features:
//   - Origin validation with exact matching or wildcard support
//   - Preflight request handling for complex CORS scenarios
//   - Credential support with proper security constraints
//   - Custom expose headers for client access to response headers
//   - Configurable cache control via MaxAge for preflight responses
//   - Automatic Vary header management for proper caching behavior
//   - Security hardening against CORS misconfigurations
//
// Security Considerations:
//   - When AllowCredentials is true and AllowOrigins contains "*", the middleware
//     will echo the specific origin instead of using "*" to prevent security issues
//   - Origins are validated with exact string matching to prevent subdomain attacks
//   - Always adds appropriate Vary headers for proper cache behavior
//
// Preflight Limitations:
// For proper CORS support with preflight requests, you need to register OPTIONS handlers
// for your routes, as HandlerFunc middleware can only process requests that match registered
// routes. For automatic preflight handling without explicit OPTIONS routes, consider using
// a different CORS solution.
//
// Example usage:
//
//	// Basic configuration (allows all origins)
//	mux.Middleware(srv.CORSMiddleware(srv.DefaultCORSConfig))
//
//	// Production configuration with specific origins and credentials
//	corsConfig := srv.CORSConfig{
//	    AllowOrigins:     []string{"https://app.example.com", "https://admin.example.com"},
//	    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
//	    AllowHeaders:     []string{"Content-Type", "Authorization", "X-API-Key"},
//	    AllowCredentials: true,
//	    ExposeHeaders:    []string{"X-Total-Count", "X-Rate-Limit"},
//	    MaxAge:           3600, // Cache preflight for 1 hour
//	}
//	mux.Middleware(srv.CORSMiddleware(corsConfig))
//
//	// Remember to register OPTIONS handlers for preflight support
//	mux.Options("", "/api/users", func(ctx Context) error { return nil })
//	mux.Post("", "/api/users", createUser)
func CORSMiddleware(config CORSConfig) HandlerFuncMiddleware {
	// Apply defaults if not set
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}

	// Pre-compute header values
	allowMethods := strings.Join(config.AllowMethods, ", ")
	allowHeaders := strings.Join(config.AllowHeaders, ", ")
	exposeHeaders := strings.Join(config.ExposeHeaders, ", ")
	maxAge := "0"
	if config.MaxAge > 0 {
		maxAge = strconv.Itoa(config.MaxAge)
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			req := ctx.Request()
			origin := req.Header.Get("Origin")
			preFlight := http.MethodOptions == req.Method

			// Always add Vary header for Origin
			ctx.AddHeader(HeaderVary, HeaderOrigin)

			// If no origin, this is likely not a CORS request
			if origin == "" {
				if preFlight {
					ctx.WriteHeader(http.StatusNoContent)
					return nil
				}
				return next(ctx)
			}

			// Simple origin check (for HandlerFunc version)
			allowOrigin := ""
			for _, o := range config.AllowOrigins {
				if o == "*" {
					if !config.AllowCredentials {
						allowOrigin = "*"
						break
					}
					// If credentials are allowed, echo specific origin
					allowOrigin = origin
					break
				}
				if o == origin {
					allowOrigin = origin
					break
				}
			}

			// Origin not allowed
			if allowOrigin == "" {
				if preFlight {
					ctx.WriteHeader(http.StatusNoContent)
					return nil
				}
				return next(ctx)
			}

			ctx.SetHeader(HeaderAccessControlAllowOrigin, allowOrigin)
			if config.AllowCredentials {
				ctx.SetHeader(HeaderAccessControlAllowCredentials, "true")
			}
			if !preFlight {
				if exposeHeaders != "" {
					ctx.SetHeader(HeaderAccessControlExposeHeaders, exposeHeaders)
				}
				return next(ctx)
			}

			// Preflight request
			ctx.AddHeader(HeaderVary, HeaderAccessControlRequestMethod)
			ctx.AddHeader(HeaderVary, HeaderAccessControlRequestHeaders)

			ctx.SetHeader("Access-Control-Allow-Methods", allowMethods)
			if allowHeaders != "" {
				ctx.AddHeader(HeaderAccessControlAllowHeaders, allowHeaders)
			} else {
				reqHeaders := req.Header.Get(HeaderAccessControlRequestHeaders)
				ctx.SetHeader(HeaderAccessControlAllowHeaders, reqHeaders)
			}
			if config.MaxAge != 0 {
				ctx.SetHeader(HeaderAccessControlMaxAge, maxAge)
			}
			ctx.WriteHeader(http.StatusNoContent)
			return nil
		}
	}
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
	// Optional. Default value []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE", "OPTIONS"}.
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
		http.MethodOptions,
	},
	AllowHeaders:     []string{},
	AllowCredentials: false,
	ExposeHeaders:    []string{},
	MaxAge:           0,
}

// =============================================================================
// Trailing Slash Middleware
// =============================================================================

// TrailingSlashConfig defines the configuration for AddTrailingSlash middleware.
type TrailingSlashConfig struct {
	// RedirectCode is the HTTP status code used when redirecting the request.
	// If set to 0, the request is forwarded internally without a redirect.
	// If set to a redirect code (e.g., 301, 302), an HTTP redirect is performed.
	//
	// Optional. Default value 0 (forward internally).
	RedirectCode int
}

// DefaultTrailingSlashConfig is the default AddTrailingSlash middleware config.
var DefaultTrailingSlashConfig = TrailingSlashConfig{
	RedirectCode: 0, // Forward internally by default
}

// AddTrailingSlashMiddleware returns a HandlerFunc-based middleware that adds a trailing
// slash to request URLs that don't already have one. It works directly with the Context
// interface and maintains the elegant error handling pattern.
//
// The middleware can either redirect the client to the URL with trailing slash
// (when RedirectCode is set) or forward the request internally (when RedirectCode is 0).
//
// Security: The middleware includes protection against open redirect vulnerabilities
// by sanitizing URLs that contain multiple slashes or backslashes.
//
// Example usage:
//
//	// Default behavior (internal forward)
//	mux.Middleware(srv.AddTrailingSlashMiddleware(srv.DefaultTrailingSlashConfig))
//
//	// Redirect with 301 status code
//	config := srv.TrailingSlashConfig{RedirectCode: 301}
//	mux.Middleware(srv.AddTrailingSlashMiddleware(config))
func AddTrailingSlashMiddleware(config TrailingSlashConfig) HandlerFuncMiddleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			req := ctx.Request()
			path := req.URL.Path

			// Skip if path already has trailing slash or is root
			if strings.HasSuffix(path, "/") {
				return next(ctx)
			}

			// Add trailing slash
			newPath := path + "/"

			// Build new URI with query string if present
			uri := newPath
			if req.URL.RawQuery != "" {
				uri += "?" + req.URL.RawQuery
			}

			// Sanitize URI to prevent open redirect attacks
			uri = sanitizeURI(uri)

			// Handle redirect vs forward
			if config.RedirectCode != 0 {
				// Perform HTTP redirect
				return ctx.Redirect(config.RedirectCode, uri)
			}

			// Forward internally by modifying the request
			req.URL.Path = newPath
			req.RequestURI = uri

			return next(ctx)
		}
	}
}

// =============================================================================
// Session Middleware
// =============================================================================

// SessionMiddleware returns a HandlerFunc-based session middleware that automatically
// manages sessions for each request. It loads existing sessions from the store or
// creates new ones as needed, makes the session available through the Context,
// and automatically saves the session after the request completes.
//
// The middleware integrates seamlessly with the srv package's Context interface,
// allowing easy session access via ctx.Get("session") or helper methods.
//
// Example usage:
//
//	// Create session store
//	store := srv.NewInMemoryStore("myapp-session", srv.NewOptions())
//
//	// Add session middleware
//	mux.Middleware(srv.SessionMiddleware(store, "myapp-session"))
//
//	// Use in handlers
//	mux.Get("profile", "/profile", func(ctx srv.Context) error {
//		session := ctx.Get("session").(*srv.Session)
//		userID := session.Get("userID")
//		if userID == nil {
//			return ctx.Redirect(302, "/login")
//		}
//		return ctx.JSON(200, map[string]interface{}{"userID": userID})
//	})
func SessionMiddleware(store Store, sessionName string) HandlerFuncMiddleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			req := ctx.Request()

			// Try to get existing session
			session, err := store.Get(req, sessionName)
			if err != nil {
				// Create new session if not found
				session, err = store.New(req, sessionName)
				if err != nil {
					return fmt.Errorf("failed to create session: %w", err)
				}
			}

			// Store session in context for handler access
			ctx.Set("session", session)

			// Execute the handler
			err = next(ctx)

			// Save session after request (regardless of handler error)
			if saveErr := session.Save(req, ctx.Response()); saveErr != nil {
				// Log save error but don't override handler error
				slog.With(
					slog.String("name", "srv.SessionMiddleware"),
					slog.String("error", saveErr.Error()),
				).Error("failed to save session")
			}

			return err
		}
	}
}

// sanitizeURI prevents open redirect attacks by sanitizing URIs that start with
// multiple slashes or backslashes. Double slashes at the beginning of a URI
// can be interpreted as absolute URIs by browsers, making applications vulnerable
// to open redirect attacks.
func sanitizeURI(uri string) string {
	// Replace multiple leading slashes/backslashes with a single slash
	if len(uri) > 1 && (uri[0] == '\\' || uri[0] == '/') && (uri[1] == '\\' || uri[1] == '/') {
		uri = "/" + strings.TrimLeft(uri, `/\`)
	}
	return uri
}

// =============================================================================
// Session Management
// =============================================================================

// Options holds configuration for session cookies.
type Options struct {
	// Path sets the cookie path. Defaults to "/".
	Path string
	// Domain sets the cookie domain.
	Domain string
	// MaxAge sets the maximum age for the session in seconds.
	// If <= 0, the session cookie will be deleted when the browser closes.
	MaxAge int
	// Secure indicates whether the cookie should only be sent over HTTPS.
	Secure bool
	// HttpOnly indicates whether the cookie should be accessible only through HTTP requests.
	// This prevents access via JavaScript, mitigating XSS attacks.
	HttpOnly bool
	// SameSite controls when cookies are sent with cross-site requests.
	SameSite http.SameSite
}

// NewOptions returns Options with secure defaults.
func NewOptions() *Options {
	return &Options{
		Path:     "/",
		MaxAge:   86400,                   // 24 hours
		Secure:   true,                    // Always secure in production
		HttpOnly: true,                    // Prevent XSS
		SameSite: http.SameSiteStrictMode, // CSRF protection
	}
}

// Session represents a user session with associated data and configuration.
// It provides a secure way to maintain state across HTTP requests.
type Session struct {
	// The ID of the session, generated by stores. It should not be used for
	// user data.
	ID string
	// Values contains the user-data for the session.
	Values  map[interface{}]interface{}
	Options *Options
	IsNew   bool
	store   Store
	name    string
}

// Get retrieves a value from the session by key.
func (s *Session) Get(key interface{}) interface{} {
	if s.Values == nil {
		return nil
	}
	return s.Values[key]
}

// Set stores a value in the session with the given key.
func (s *Session) Set(key, value interface{}) {
	if s.Values == nil {
		s.Values = make(map[interface{}]interface{})
	}
	s.Values[key] = value
}

// Delete removes a key from the session.
func (s *Session) Delete(key interface{}) {
	if s.Values == nil {
		return
	}
	delete(s.Values, key)
}

// Clear removes all values from the session.
func (s *Session) Clear() {
	s.Values = make(map[interface{}]interface{})
}

// Save persists the session to the underlying store.
func (s *Session) Save(r *http.Request, w http.ResponseWriter) error {
	if s.store == nil {
		return fmt.Errorf("no store configured for session")
	}
	return s.store.Save(r, w, s)
}

// Store defines the interface for session storage backends.
// Implementations must be thread-safe.
type Store interface {
	// Get should return a cached session.
	Get(r *http.Request, name string) (*Session, error)

	// New should create and return a new session.
	//
	// Note that New should never return a nil session, even in the case of
	// an error if using the Registry infrastructure to cache the session.
	New(r *http.Request, name string) (*Session, error)

	// Save should persist session to the underlying store implementation.
	Save(r *http.Request, w http.ResponseWriter, s *Session) error
}

// =============================================================================
// In-Memory Session Store Implementation
// =============================================================================

// sessionData holds session information with expiration.
type sessionData struct {
	Values    map[interface{}]interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
}

// InMemoryStore provides an in-memory session store implementation.
// It is thread-safe and suitable for development and single-instance deployments.
// For production with multiple instances, consider a distributed store.
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*sessionData
	options  *Options
	name     string
	cleanup  *time.Ticker
}

// NewInMemoryStore creates a new in-memory session store with the specified options.
// It automatically starts a cleanup routine to remove expired sessions.
func NewInMemoryStore(name string, options *Options) *InMemoryStore {
	if options == nil {
		options = NewOptions()
	}

	store := &InMemoryStore{
		sessions: make(map[string]*sessionData),
		options:  options,
		name:     name,
		cleanup:  time.NewTicker(time.Hour), // Cleanup expired sessions every hour
	}

	// Start cleanup routine
	go store.cleanupExpiredSessions()

	return store
}

// Get retrieves an existing session or returns nil if not found or expired.
func (s *InMemoryStore) Get(r *http.Request, name string) (*Session, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	data, exists := s.sessions[cookie.Value]
	s.mu.RUnlock()

	if !exists || time.Now().After(data.ExpiresAt) {
		return nil, http.ErrNoCookie
	}

	session := &Session{
		ID:      cookie.Value,
		Values:  data.Values,
		Options: s.options,
		IsNew:   false,
		store:   s,
		name:    name,
	}

	return session, nil
}

// New creates a new session with a unique ID.
func (s *InMemoryStore) New(_ *http.Request, name string) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	session := &Session{
		ID:      sessionID,
		Values:  make(map[interface{}]interface{}),
		Options: s.options,
		IsNew:   true,
		store:   s,
		name:    name,
	}

	return session, nil
}

// Save persists the session to the in-memory store and sets the session cookie.
func (s *InMemoryStore) Save(_ *http.Request, w http.ResponseWriter, session *Session) error {
	if session.ID == "" {
		return fmt.Errorf("session ID is empty")
	}

	// Calculate expiration time
	now := time.Now()
	var expiresAt time.Time
	if session.Options.MaxAge > 0 {
		expiresAt = now.Add(time.Duration(session.Options.MaxAge) * time.Second)
	} else {
		// Session cookie (expires when browser closes)
		expiresAt = now.Add(24 * time.Hour) // Default to 24 hours for cleanup
	}

	// Store session data
	s.mu.Lock()
	s.sessions[session.ID] = &sessionData{
		Values:    session.Values,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	s.mu.Unlock()

	// Set cookie
	cookie := &http.Cookie{
		Name:     session.name,
		Value:    session.ID,
		Path:     session.Options.Path,
		Domain:   session.Options.Domain,
		Secure:   session.Options.Secure,
		HttpOnly: session.Options.HttpOnly,
		SameSite: session.Options.SameSite,
	}

	if session.Options.MaxAge > 0 {
		cookie.MaxAge = session.Options.MaxAge
		cookie.Expires = expiresAt
	}

	http.SetCookie(w, cookie)
	return nil
}

// Close stops the cleanup routine and clears all sessions.
// This should be called when the store is no longer needed.
func (s *InMemoryStore) Close() {
	if s.cleanup != nil {
		s.cleanup.Stop()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[string]*sessionData)
}

// cleanupExpiredSessions removes expired sessions from the store.
func (s *InMemoryStore) cleanupExpiredSessions() {
	for range s.cleanup.C {
		now := time.Now()
		s.mu.Lock()
		for id, data := range s.sessions {
			if now.After(data.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// generateSessionID creates a cryptographically secure session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32) // 256 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// =============================================================================
// Cookie Store Implementation with Encryption
// =============================================================================

// CookieStore provides an encrypted cookie-based session store implementation.
// It stores all session data directly in encrypted cookies on the client side,
// eliminating the need for server-side session storage.
//
// The store uses AES-GCM encryption for authenticated encryption, ensuring both
// confidentiality and integrity of session data. Session values are serialized
// using gob encoding before encryption.
//
// This implementation is suitable for stateless applications or when you want
// to avoid server-side session storage. However, be aware of cookie size limits
// (typically ~4KB) and ensure your session data fits within these constraints.
type CookieStore struct {
	cipher  cipher.AEAD
	options *Options
	name    string
}

// NewCookieStore creates a new cookie-based session store with the specified
// encryption key and options. The encryption key must be 16, 24, or 32 bytes
// to select AES-128, AES-192, or AES-256 respectively.
//
// The store uses AES-GCM for authenticated encryption, providing both
// confidentiality and integrity protection for session data.
//
// Example usage:
//
//	// Generate a 32-byte key for AES-256
//	key := make([]byte, 32)
//	if _, err := rand.Read(key); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create cookie store
//	store, err := srv.NewCookieStore("app-session", key, srv.NewOptions())
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use with session middleware
//	mux.Middleware(srv.SessionMiddleware(store, "app-session"))
func NewCookieStore(name string, key []byte, options *Options) (*CookieStore, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256")
	}

	if options == nil {
		options = NewOptions()
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &CookieStore{
		cipher:  gcm,
		options: options,
		name:    name,
	}, nil
}

// Get retrieves an existing session from the encrypted cookie.
// If the cookie doesn't exist, is invalid, or cannot be decrypted,
// it returns an error.
func (c *CookieStore) Get(r *http.Request, name string) (*Session, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return nil, err
	}

	// Decode base64 cookie value
	encryptedData, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, http.ErrNoCookie
	}

	// Decrypt and deserialize session data
	values, err := c.decryptSessionData(encryptedData)
	if err != nil {
		return nil, http.ErrNoCookie
	}

	// Create session with decrypted data
	session := &Session{
		ID:      "", // Not used for cookie store
		Values:  values,
		Options: c.options,
		IsNew:   false,
		store:   c,
		name:    name,
	}

	return session, nil
}

// New creates a new session. Since this is a cookie store, no server-side
// storage is required. The session will be saved as an encrypted cookie
// when Save() is called.
func (c *CookieStore) New(_ *http.Request, name string) (*Session, error) {
	session := &Session{
		ID:      "", // Not used for cookie store
		Values:  make(map[interface{}]interface{}),
		Options: c.options,
		IsNew:   true,
		store:   c,
		name:    name,
	}

	return session, nil
}

// Save encrypts the session data and stores it as a cookie.
// The session values are serialized using gob encoding and then
// encrypted using AES-GCM before being base64 encoded and stored
// in the cookie.
func (c *CookieStore) Save(_ *http.Request, w http.ResponseWriter, session *Session) error {
	if len(session.Values) == 0 {
		// Clear cookie if session is empty
		cookie := &http.Cookie{
			Name:     session.name,
			Value:    "",
			Path:     session.Options.Path,
			Domain:   session.Options.Domain,
			MaxAge:   -1, // Delete cookie
			Secure:   session.Options.Secure,
			HttpOnly: session.Options.HttpOnly,
			SameSite: session.Options.SameSite,
		}
		http.SetCookie(w, cookie)
		return nil
	}

	// Encrypt and encode session data
	encryptedData, err := c.encryptSessionData(session.Values)
	if err != nil {
		return fmt.Errorf("failed to encrypt session data: %w", err)
	}

	// Encode as base64 for cookie storage
	cookieValue := base64.URLEncoding.EncodeToString(encryptedData)

	// Check cookie size limit (browsers typically limit to ~4KB)
	if len(cookieValue) > 4000 {
		return errors.New("session data too large for cookie storage (>4KB)")
	}

	// Set cookie
	cookie := &http.Cookie{
		Name:     session.name,
		Value:    cookieValue,
		Path:     session.Options.Path,
		Domain:   session.Options.Domain,
		Secure:   session.Options.Secure,
		HttpOnly: session.Options.HttpOnly,
		SameSite: session.Options.SameSite,
	}

	if session.Options.MaxAge > 0 {
		cookie.MaxAge = session.Options.MaxAge
		cookie.Expires = time.Now().Add(time.Duration(session.Options.MaxAge) * time.Second)
	}

	http.SetCookie(w, cookie)
	return nil
}

// encryptSessionData serializes and encrypts session values using AES-GCM.
func (c *CookieStore) encryptSessionData(values map[interface{}]interface{}) ([]byte, error) {
	// Serialize session values using gob
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(values); err != nil {
		return nil, fmt.Errorf("failed to encode session data: %w", err)
	}

	plaintext := buf.Bytes()

	// Generate random nonce
	nonce := make([]byte, c.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt with AES-GCM (includes authentication)
	ciphertext := c.cipher.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptSessionData decrypts and deserializes session values.
func (c *CookieStore) decryptSessionData(encryptedData []byte) (map[interface{}]interface{}, error) {
	if len(encryptedData) < c.cipher.NonceSize() {
		return nil, errors.New("encrypted data too short")
	}

	// Extract nonce and ciphertext
	nonce := encryptedData[:c.cipher.NonceSize()]
	ciphertext := encryptedData[c.cipher.NonceSize():]

	// Decrypt with AES-GCM (includes authentication verification)
	plaintext, err := c.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session data: %w", err)
	}

	// Deserialize using gob
	var values map[interface{}]interface{}
	decoder := gob.NewDecoder(bytes.NewReader(plaintext))
	if err := decoder.Decode(&values); err != nil {
		return nil, fmt.Errorf("failed to decode session data: %w", err)
	}

	return values, nil
}
