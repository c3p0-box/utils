package srv

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
)

type Context interface {
	Set(key string, value interface{})
	Get(key string) interface{}
	Request() *http.Request
	Response() http.ResponseWriter
	IsTLS() bool
	IsWebSocket() bool
	Method() string
	Path() string
	Param(key string) string
	Query() url.Values
	QueryParam(key string) string
	FormValue(key string) string
	GetHeader(key string) string
	GetHeaders() http.Header
	Cookie(key string) (*http.Cookie, error)
	Cookies() []*http.Cookie
	SetHeader(key, value string)
	AddHeader(key, value string)
	SetCookie(cookie *http.Cookie)
	JSON(code int, v interface{}) error
	String(code int, text string) error
	Redirect(code int, path string) error
	HTML(code int, html string) error
	HTMLBlob(code int, html []byte) error
	WriteHeader(code int)
}

// HttpContext provides a convenient wrapper around http.Request and http.ResponseWriter
// with additional functionality for storing request-scoped values, handling common
// HTTP operations, and providing helper methods for request/response processing.
//
// HttpContext is thread-safe for concurrent access to its internal value store.
type HttpContext struct {
	request        *http.Request
	responseWriter http.ResponseWriter
	mu             sync.RWMutex
	values         map[string]interface{}
	query          url.Values
	path           string
}

// NewHttpContext creates a new HttpContext instance wrapping the provided
// http.ResponseWriter and http.Request. The context includes an empty
// thread-safe value store for request-scoped data.
func NewHttpContext(w http.ResponseWriter, r *http.Request) *HttpContext {
	return &HttpContext{
		request:        r,
		responseWriter: w,
		mu:             sync.RWMutex{},
		values:         make(map[string]interface{}),
		query:          r.URL.Query(),
	}
}

// ============================
// Value Store Methods
// ============================

// Set stores a value in the context's value store with the given key.
// This method is thread-safe and can be called concurrently.
func (c *HttpContext) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// Get retrieves a value from the context's value store by key.
// Returns nil if the key doesn't exist.
// This method is thread-safe and can be called concurrently.
func (c *HttpContext) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.values[key]; ok {
		return v
	}
	return nil
}

// ============================
// Request Access Methods
// ============================

// Request returns the underlying http.Request object.
func (c *HttpContext) Request() *http.Request {
	return c.request
}

// Response returns the underlying http.ResponseWriter object.
func (c *HttpContext) Response() http.ResponseWriter {
	return c.responseWriter
}

// ============================
// Request Information Methods
// ============================

// IsTLS returns true if the request was made over HTTPS/TLS.
func (c *HttpContext) IsTLS() bool {
	return c.Request().TLS != nil
}

// IsWebSocket returns true if this is a WebSocket upgrade request.
// It checks for the "Upgrade: websocket" header.
func (c *HttpContext) IsWebSocket() bool {
	return c.Request().Header.Get("Upgrade") == "websocket"
}

// Method returns the HTTP method of the request (GET, POST, etc.).
func (c *HttpContext) Method() string {
	return c.Request().Method
}

// Path returns the URL path of the request.
func (c *HttpContext) Path() string {
	if c.path == "" {
		c.path = c.Request().URL.Path
	}

	return c.path
}

// SetPath sets the URL path of the request.
func (c *HttpContext) SetPath(path string) {
	c.path = path
}

// ============================
// Request Parameter Methods
// ============================

// Query returns all URL query parameters as url.Values.
func (c *HttpContext) Query() url.Values {
	return c.query
}

// QueryParam returns the value of the specified query parameter.
// Returns empty string if the parameter doesn't exist.
func (c *HttpContext) QueryParam(key string) string {
	return c.Query().Get(key)
}

// Param returns the value of the specified path parameter.
// This uses Go 1.22+ ServeMux path value extraction.
func (c *HttpContext) Param(key string) string {
	return c.Request().PathValue(key)
}

// FormValue returns the value of the specified form parameter.
// It parses the form data if not already parsed.
func (c *HttpContext) FormValue(key string) string {
	return c.Request().FormValue(key)
}

// ============================
// Header Methods
// ============================

// GetHeader returns the value of the specified request header.
func (c *HttpContext) GetHeader(key string) string {
	return c.Request().Header.Get(key)
}

// GetHeaders returns all request headers.
func (c *HttpContext) GetHeaders() http.Header {
	return c.Request().Header
}

// SetHeader sets a response header. If a header with the same key already
// exists, it will be replaced.
func (c *HttpContext) SetHeader(key, value string) {
	c.Response().Header().Set(key, value)
}

// AddHeader adds a response header. If a header with the same key already
// exists, the value will be appended.
func (c *HttpContext) AddHeader(key, value string) {
	c.Response().Header().Add(key, value)
}

// ============================
// Cookie Methods
// ============================

// Cookie returns the named cookie provided in the request.
// Returns ErrNoCookie if no cookie with the given name is found.
func (c *HttpContext) Cookie(key string) (*http.Cookie, error) {
	return c.Request().Cookie(key)
}

// Cookies returns all cookies provided in the request.
func (c *HttpContext) Cookies() []*http.Cookie {
	return c.Request().Cookies()
}

// SetCookie adds a Set-Cookie header to the response.
func (c *HttpContext) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response(), cookie)
}

// ============================
// Response Methods
// ============================

// JSON writes a JSON response with the specified status code.
// The Content-Type header is automatically set to "application/json".
// Returns an error if JSON encoding fails.
func (c *HttpContext) JSON(code int, v interface{}) error {
	c.SetHeader("Content-Type", MIMEApplicationJSON)
	c.Response().WriteHeader(code)
	return json.NewEncoder(c.Response()).Encode(v)
}

// String writes a plain text response with the specified status code.
// The Content-Type header is automatically set to "text/plain".
func (c *HttpContext) String(code int, text string) error {
	c.SetHeader("Content-Type", MIMETextPlain)
	c.Response().WriteHeader(code)
	_, err := c.Response().Write([]byte(text))
	return err
}

// HTML writes an HTML response with the specified status code.
// The Content-Type header is automatically set to "text/html".
func (c *HttpContext) HTML(code int, html string) error {
	c.SetHeader("Content-Type", MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(code)
	_, err := c.Response().Write([]byte(html))
	return err
}

// HTMLBlob writes an HTML response with the specified status code.
// The Content-Type header is automatically set to "text/html".
func (c *HttpContext) HTMLBlob(code int, blob []byte) error {
	c.SetHeader("Content-Type", MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(code)
	_, err := c.Response().Write(blob)
	return err
}

// Redirect sends an HTTP redirect response with the specified status code and URL.
// Common status codes are 301 (permanent), 302 (found), 303 (see other),
// 307 (temporary), and 308 (permanent redirect).
func (c *HttpContext) Redirect(code int, url string) error {
	c.SetHeader("Location", url)
	c.Response().WriteHeader(code)
	return nil
}

// WriteHeader sends an HTTP response header with the provided status code.
func (c *HttpContext) WriteHeader(code int) {
	c.Response().WriteHeader(code)
}
