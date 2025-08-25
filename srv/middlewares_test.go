package srv

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test helper to capture slog output
func captureLogs(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{}))
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	fn()
	return buf.String()
}

// =============================================================================
// HandlerFunc Middleware Tests
// =============================================================================

func TestLoggingMiddleware(t *testing.T) {
	mux := NewMux()

	// Add logging middleware
	mux.Middleware(LoggingMiddleware)

	// Register a route
	mux.Post("", "/test", func(ctx Context) error {
		return ctx.String(http.StatusCreated, "test response")
	})

	req := httptest.NewRequest("POST", "/test?param=value", strings.NewReader("test body"))
	req.Header.Set("User-Agent", "test-agent/1.0")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	logOutput := captureLogs(t, func() {
		mux.ServeHTTP(rec, req)
	})

	// Check that log contains expected information
	expectedParts := []string{
		"srv.Logging",
		"method=POST",
		"path=/test",
		"user-agent=test-agent/1.0",
		"remote-addr=192.168.1.1:12345",
		"duration=",
		"request completed",
	}

	for _, part := range expectedParts {
		if !strings.Contains(logOutput, part) {
			t.Errorf("Expected log output to contain '%s', but it didn't. Log output: %s", part, logOutput)
		}
	}
}

func TestRecoverMiddleware(t *testing.T) {
	mux := NewMux()

	// Set up error handler to capture panic-converted error
	var capturedError error
	mux.ErrorHandler(func(ctx Context, err error) {
		capturedError = err
		_ = ctx.String(500, "Panic handled")
	})

	// Add recover middleware
	mux.Middleware(RecoverMiddleware)

	// Register a route that panics
	mux.Get("", "/panic", func(ctx Context) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()

	logOutput := captureLogs(t, func() {
		mux.ServeHTTP(rec, req)
	})

	// Check that panic was logged
	expectedParts := []string{
		"srv.Recover",
		"error=\"test panic\"",
		"path=/panic",
		"method=GET",
		"recovered from panic",
	}

	for _, part := range expectedParts {
		if !strings.Contains(logOutput, part) {
			t.Errorf("Expected log output to contain '%s', but it didn't. Log output: %s", part, logOutput)
		}
	}

	// Check that error was captured by error handler
	if capturedError == nil {
		t.Error("Expected panic to be converted to error")
	} else if !strings.Contains(capturedError.Error(), "panic") {
		t.Errorf("Expected error to contain 'panic', got '%s'", capturedError.Error())
	}

	if rec.Code != 500 {
		t.Errorf("Expected status code 500, got %d", rec.Code)
	}
}

func TestCORSMiddleware_DefaultConfig(t *testing.T) {
	mux := NewMux()

	// Add CORS middleware
	mux.Middleware(CORSMiddleware(DefaultCORSConfig))

	// Register a route
	mux.Get("", "/test", func(ctx Context) error {
		return ctx.String(http.StatusOK, "success")
	})

	t.Run("with origin header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		// Check CORS headers
		if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin to be '*', got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
		}

		if rec.Header().Get("Vary") != "Origin" {
			t.Errorf("Expected Vary header to include 'Origin', got '%s'", rec.Header().Get("Vary"))
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		if rec.Body.String() != "success" {
			t.Errorf("Expected body 'success', got '%s'", rec.Body.String())
		}
	})

	t.Run("CORS headers set for regular requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		// CORS headers should be set
		if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin to be '*', got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})
}

func TestAddTrailingSlashMiddleware_DefaultConfig(t *testing.T) {
	mux := NewMux()

	// Add trailing slash middleware
	mux.Middleware(AddTrailingSlashMiddleware(DefaultTrailingSlashConfig))

	// Register routes both with and without trailing slashes to test the middleware
	mux.Get("", "/users", func(ctx Context) error {
		ctx.SetHeader("X-Final-Path", ctx.Request().URL.Path)
		return ctx.String(http.StatusOK, "users without slash")
	})

	mux.Get("", "/users/", func(ctx Context) error {
		ctx.SetHeader("X-Final-Path", ctx.Request().URL.Path)
		return ctx.String(http.StatusOK, "users with slash")
	})

	t.Run("preserves trailing slash when already present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		finalPath := rec.Header().Get("X-Final-Path")
		if finalPath != "/users/" {
			t.Errorf("Expected path to remain '/users/', got '%s'", finalPath)
		}

		if !strings.Contains(rec.Body.String(), "with slash") {
			t.Errorf("Expected body to indicate slash route, got '%s'", rec.Body.String())
		}
	})

	t.Run("handles path modification internally", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		// The middleware should either redirect or forward internally
		// In this test setup, it may redirect since both routes exist
		if rec.Code != http.StatusOK && rec.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 200 or 301, got %d", rec.Code)
		}
	})
}

func TestHandlerFuncMiddleware_Integration(t *testing.T) {
	mux := NewMux()

	// Chain multiple HandlerFunc middleware
	mux.Middleware(LoggingMiddleware)
	mux.Middleware(RecoverMiddleware)
	mux.Middleware(CORSMiddleware(DefaultCORSConfig))

	// Register a route
	mux.Get("", "/test", func(ctx Context) error {
		return ctx.JSON(200, map[string]string{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("User-Agent", "test-client")
	rec := httptest.NewRecorder()

	logOutput := captureLogs(t, func() {
		mux.ServeHTTP(rec, req)
	})

	// Check that CORS headers are set
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS headers to be set")
	}

	// Check that logging middleware works
	if !strings.Contains(logOutput, "request completed") {
		t.Error("Expected logging middleware to work")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response["message"] != "success" {
		t.Errorf("Expected message 'success', got '%v'", response["message"])
	}
}

// =============================================================================
// Session Management Tests
// =============================================================================

func TestNewOptions(t *testing.T) {
	opts := NewOptions()

	if opts.Path != "/" {
		t.Errorf("Expected Path to be '/', got '%s'", opts.Path)
	}
	if opts.MaxAge != 86400 {
		t.Errorf("Expected MaxAge to be 86400, got %d", opts.MaxAge)
	}
	if !opts.Secure {
		t.Error("Expected Secure to be true")
	}
	if !opts.HttpOnly {
		t.Error("Expected HttpOnly to be true")
	}
	if opts.SameSite != http.SameSiteStrictMode {
		t.Errorf("Expected SameSite to be Strict, got %v", opts.SameSite)
	}
}

func TestSession_BasicOperations(t *testing.T) {
	session := &Session{
		ID:     "test-session-id",
		Values: make(map[interface{}]interface{}),
	}

	// Test Set and Get
	session.Set("userID", 12345)
	session.Set("username", "testuser")
	session.Set("isAdmin", true)

	if session.Get("userID") != 12345 {
		t.Errorf("Expected userID to be 12345, got %v", session.Get("userID"))
	}
	if session.Get("username") != "testuser" {
		t.Errorf("Expected username to be 'testuser', got %v", session.Get("username"))
	}
	if session.Get("isAdmin") != true {
		t.Errorf("Expected isAdmin to be true, got %v", session.Get("isAdmin"))
	}

	// Test non-existent key
	if session.Get("nonexistent") != nil {
		t.Errorf("Expected nonexistent key to return nil, got %v", session.Get("nonexistent"))
	}

	// Test Delete
	session.Delete("username")
	if session.Get("username") != nil {
		t.Errorf("Expected deleted key to return nil, got %v", session.Get("username"))
	}

	// Test Clear
	session.Clear()
	if session.Get("userID") != nil {
		t.Error("Expected all keys to be cleared")
	}
	if session.Get("isAdmin") != nil {
		t.Error("Expected all keys to be cleared")
	}
}

func TestSession_NilValues(t *testing.T) {
	session := &Session{ID: "test"}

	// Test operations on nil Values
	if session.Get("key") != nil {
		t.Error("Expected Get on nil Values to return nil")
	}

	// Set should initialize Values
	session.Set("key", "value")
	if session.Values == nil {
		t.Error("Expected Values to be initialized after Set")
	}
	if session.Get("key") != "value" {
		t.Errorf("Expected value to be 'value', got %v", session.Get("key"))
	}

	// Delete on nil Values should not panic
	session2 := &Session{ID: "test2"}
	session2.Delete("key") // Should not panic
	session2.Clear()       // Should not panic
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := generateSessionID()
	if err != nil {
		t.Fatalf("Expected no error generating session ID, got: %v", err)
	}
	if len(id1) == 0 {
		t.Error("Expected session ID to be non-empty")
	}

	id2, err := generateSessionID()
	if err != nil {
		t.Fatalf("Expected no error generating session ID, got: %v", err)
	}

	// IDs should be unique
	if id1 == id2 {
		t.Error("Expected session IDs to be unique")
	}

	// IDs should be URL-safe
	if strings.Contains(id1, "+") || strings.Contains(id1, "/") {
		t.Error("Expected session ID to be URL-safe")
	}
}

func TestInMemoryStore_NewStore(t *testing.T) {
	store := NewInMemoryStore("test-session", NewOptions())
	defer store.Close()

	if store == nil {
		t.Fatal("Expected store to be non-nil")
	}
	if store.name != "test-session" {
		t.Errorf("Expected store name to be 'test-session', got '%s'", store.name)
	}
	if store.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}
	if store.options == nil {
		t.Error("Expected options to be set")
	}

	// Test with nil options
	store2 := NewInMemoryStore("test2", nil)
	defer store2.Close()
	if store2.options == nil {
		t.Error("Expected default options to be set when nil provided")
	}
}

func TestInMemoryStore_NewSession(t *testing.T) {
	store := NewInMemoryStore("test-session", NewOptions())
	defer store.Close()

	req := httptest.NewRequest("GET", "/test", nil)

	session, err := store.New(req, "test-session")
	if err != nil {
		t.Fatalf("Expected no error creating session, got: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be non-nil")
	}
	if session.ID == "" {
		t.Error("Expected session to have an ID")
	}
	if !session.IsNew {
		t.Error("Expected new session to have IsNew = true")
	}
	if session.Values == nil {
		t.Error("Expected session Values to be initialized")
	}
	if session.name != "test-session" {
		t.Errorf("Expected session name to be 'test-session', got '%s'", session.name)
	}
	if session.store != store {
		t.Error("Expected session store to be set correctly")
	}
}

func TestInMemoryStore_SaveAndGet(t *testing.T) {
	store := NewInMemoryStore("test-session", &Options{
		Path:     "/",
		MaxAge:   3600,
		Secure:   false, // For testing
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	defer store.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Create and save session
	session, err := store.New(req, "test-session")
	if err != nil {
		t.Fatalf("Expected no error creating session, got: %v", err)
	}

	session.Set("userID", 12345)
	session.Set("role", "admin")

	err = store.Save(req, rec, session)
	if err != nil {
		t.Fatalf("Expected no error saving session, got: %v", err)
	}

	// Check cookie was set
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "test-session" {
		t.Errorf("Expected cookie name 'test-session', got '%s'", cookie.Name)
	}
	if cookie.Value != session.ID {
		t.Errorf("Expected cookie value to match session ID")
	}
	if cookie.Path != "/" {
		t.Errorf("Expected cookie path '/', got '%s'", cookie.Path)
	}
	if cookie.MaxAge != 3600 {
		t.Errorf("Expected cookie MaxAge 3600, got %d", cookie.MaxAge)
	}
	if cookie.Secure {
		t.Error("Expected cookie Secure to be false for test")
	}
	if !cookie.HttpOnly {
		t.Error("Expected cookie HttpOnly to be true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("Expected cookie SameSite to be Lax, got %v", cookie.SameSite)
	}

	// Create new request with cookie
	req2 := httptest.NewRequest("GET", "/test2", nil)
	req2.AddCookie(cookie)

	// Get existing session
	retrievedSession, err := store.Get(req2, "test-session")
	if err != nil {
		t.Fatalf("Expected no error retrieving session, got: %v", err)
	}

	if retrievedSession == nil {
		t.Fatal("Expected retrieved session to be non-nil")
	}
	if retrievedSession.ID != session.ID {
		t.Errorf("Expected retrieved session ID to match original")
	}
	if retrievedSession.IsNew {
		t.Error("Expected retrieved session to have IsNew = false")
	}
	if retrievedSession.Get("userID") != 12345 {
		t.Errorf("Expected userID to be 12345, got %v", retrievedSession.Get("userID"))
	}
	if retrievedSession.Get("role") != "admin" {
		t.Errorf("Expected role to be 'admin', got %v", retrievedSession.Get("role"))
	}
}

func TestInMemoryStore_GetNonExistentSession(t *testing.T) {
	store := NewInMemoryStore("test-session", NewOptions())
	defer store.Close()

	req := httptest.NewRequest("GET", "/test", nil)

	// Try to get session without cookie
	_, err := store.Get(req, "test-session")
	if err != http.ErrNoCookie {
		t.Errorf("Expected ErrNoCookie, got: %v", err)
	}

	// Try to get session with invalid cookie
	req.AddCookie(&http.Cookie{Name: "test-session", Value: "invalid-id"})
	_, err = store.Get(req, "test-session")
	if err != http.ErrNoCookie {
		t.Errorf("Expected ErrNoCookie for invalid session, got: %v", err)
	}
}

func TestInMemoryStore_SessionExpiration(t *testing.T) {
	store := NewInMemoryStore("test-session", &Options{
		Path:     "/",
		MaxAge:   1, // 1 second expiration
		Secure:   false,
		HttpOnly: true,
	})
	defer store.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Create and save session
	session, err := store.New(req, "test-session")
	if err != nil {
		t.Fatalf("Expected no error creating session, got: %v", err)
	}

	session.Set("data", "test")
	err = store.Save(req, rec, session)
	if err != nil {
		t.Fatalf("Expected no error saving session, got: %v", err)
	}

	// Get cookie
	cookie := rec.Result().Cookies()[0]

	// Immediately retrieve session (should work)
	req2 := httptest.NewRequest("GET", "/test2", nil)
	req2.AddCookie(cookie)
	retrievedSession, err := store.Get(req2, "test-session")
	if err != nil {
		t.Fatalf("Expected no error retrieving fresh session, got: %v", err)
	}
	if retrievedSession.Get("data") != "test" {
		t.Error("Expected to retrieve session data")
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Try to retrieve expired session
	req3 := httptest.NewRequest("GET", "/test3", nil)
	req3.AddCookie(cookie)
	_, err = store.Get(req3, "test-session")
	if err != http.ErrNoCookie {
		t.Errorf("Expected ErrNoCookie for expired session, got: %v", err)
	}
}

func TestInMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryStore("test-session", NewOptions())
	defer store.Close()

	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent session creation and access
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				rec := httptest.NewRecorder()

				// Create session
				session, err := store.New(req, "test-session")
				if err != nil {
					t.Errorf("Error creating session: %v", err)
					return
				}

				session.Set("goroutine", id)
				session.Set("operation", j)

				// Save session
				err = store.Save(req, rec, session)
				if err != nil {
					t.Errorf("Error saving session: %v", err)
					return
				}

				// Try to retrieve it
				cookie := rec.Result().Cookies()[0]
				req2 := httptest.NewRequest("GET", "/test2", nil)
				req2.AddCookie(cookie)

				retrieved, err := store.Get(req2, "test-session")
				if err != nil {
					t.Errorf("Error retrieving session: %v", err)
					return
				}

				if retrieved.Get("goroutine") != id {
					t.Errorf("Expected goroutine %d, got %v", id, retrieved.Get("goroutine"))
				}
			}
		}(i)
	}

	wg.Wait()
}

// =============================================================================
// Session Middleware Tests
// =============================================================================

func TestSessionMiddleware_NewSession(t *testing.T) {
	store := NewInMemoryStore("app-session", &Options{
		Path:     "/",
		MaxAge:   3600,
		Secure:   false, // For testing
		HttpOnly: true,
	})
	defer store.Close()

	mux := NewMux()
	mux.Middleware(SessionMiddleware(store, "app-session"))

	// Register handler that uses session
	mux.Get("", "/test", func(ctx Context) error {
		session := ctx.Get("session").(*Session)
		if session == nil {
			t.Error("Expected session to be available in context")
			return ctx.String(500, "No session")
		}

		// Set some session data
		session.Set("visited", true)
		session.Set("timestamp", time.Now().Unix())

		return ctx.JSON(200, map[string]interface{}{
			"sessionID": session.ID,
			"isNew":     session.IsNew,
			"visited":   session.Get("visited"),
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Check response
	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response["isNew"] != true {
		t.Errorf("Expected isNew to be true for new session, got %v", response["isNew"])
	}
	if response["visited"] != true {
		t.Errorf("Expected visited to be true, got %v", response["visited"])
	}
	if response["sessionID"] == nil || response["sessionID"] == "" {
		t.Error("Expected sessionID to be set")
	}

	// Check that cookie was set by examining Set-Cookie header
	setCookieHeaders := rec.Header()["Set-Cookie"]
	if len(setCookieHeaders) != 1 {
		t.Errorf("Expected 1 Set-Cookie header, got %d", len(setCookieHeaders))
	} else {
		// Parse the cookie string to extract the name
		cookieStr := setCookieHeaders[0]
		if !strings.HasPrefix(cookieStr, "app-session=") {
			t.Errorf("Expected cookie name 'app-session', got '%s'", cookieStr)
		}
	}
}

func TestSessionMiddleware_ExistingSession(t *testing.T) {
	store := NewInMemoryStore("app-session", &Options{
		Path:     "/",
		MaxAge:   3600,
		Secure:   false,
		HttpOnly: true,
	})
	defer store.Close()

	mux := NewMux()
	mux.Middleware(SessionMiddleware(store, "app-session"))

	// First handler to create session
	mux.Get("", "/create", func(ctx Context) error {
		session := ctx.Get("session").(*Session)
		session.Set("userID", 12345)
		session.Set("username", "testuser")
		return ctx.JSON(200, map[string]string{"status": "created"})
	})

	// Second handler to read session
	mux.Get("", "/read", func(ctx Context) error {
		session := ctx.Get("session").(*Session)
		return ctx.JSON(200, map[string]interface{}{
			"sessionID": session.ID,
			"isNew":     session.IsNew,
			"userID":    session.Get("userID"),
			"username":  session.Get("username"),
		})
	})

	// Create session first
	req1 := httptest.NewRequest("GET", "/create", nil)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	if rec1.Code != 200 {
		t.Fatalf("Expected status 200 for create, got %d", rec1.Code)
	}

	// Get the session cookie from Set-Cookie header
	setCookieHeaders := rec1.Header()["Set-Cookie"]
	if len(setCookieHeaders) != 1 {
		t.Fatalf("Expected 1 cookie from create request, got %d", len(setCookieHeaders))
	}

	// Parse the cookie manually
	cookieStr := setCookieHeaders[0]
	parts := strings.Split(cookieStr, ";")
	cookiePart := strings.TrimSpace(parts[0])
	cookieNameValue := strings.SplitN(cookiePart, "=", 2)
	if len(cookieNameValue) != 2 {
		t.Fatalf("Invalid cookie format: %s", cookieStr)
	}

	cookie := &http.Cookie{
		Name:  cookieNameValue[0],
		Value: cookieNameValue[1],
	}

	// Use existing session
	req2 := httptest.NewRequest("GET", "/read", nil)
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != 200 {
		t.Errorf("Expected status 200 for read, got %d", rec2.Code)
	}

	// Check response
	var response map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response["isNew"] != false {
		t.Errorf("Expected isNew to be false for existing session, got %v", response["isNew"])
	}
	if response["userID"] != float64(12345) { // JSON decode converts to float64
		t.Errorf("Expected userID to be 12345, got %v", response["userID"])
	}
	if response["username"] != "testuser" {
		t.Errorf("Expected username to be 'testuser', got %v", response["username"])
	}
	if response["sessionID"] != cookie.Value {
		t.Error("Expected sessionID to match cookie value")
	}
}

func TestSessionMiddleware_ErrorInHandler(t *testing.T) {
	store := NewInMemoryStore("app-session", NewOptions())
	defer store.Close()

	mux := NewMux()

	// Set custom error handler to capture errors
	var capturedError error
	mux.ErrorHandler(func(ctx Context, err error) {
		capturedError = err
		_ = ctx.String(500, "Handler error: "+err.Error())
	})

	mux.Middleware(SessionMiddleware(store, "app-session"))

	// Handler that returns an error
	mux.Get("", "/error", func(ctx Context) error {
		session := ctx.Get("session").(*Session)
		session.Set("data", "should be saved even with error")
		return fmt.Errorf("test handler error")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Error should be handled
	if capturedError == nil {
		t.Error("Expected error to be captured")
	} else if capturedError.Error() != "test handler error" {
		t.Errorf("Expected error message 'test handler error', got '%s'", capturedError.Error())
	}

	if rec.Code != 500 {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}

	// Session should still be saved despite handler error
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Error("Expected session cookie to be set despite handler error")
	}
}

func TestSessionMiddleware_Integration(t *testing.T) {
	// Test integration with other middleware
	store := NewInMemoryStore("app-session", &Options{
		Path:     "/",
		MaxAge:   3600,
		Secure:   false,
		HttpOnly: true,
	})
	defer store.Close()

	mux := NewMux()

	// Add multiple middleware
	mux.Middleware(LoggingMiddleware)
	mux.Middleware(RecoverMiddleware)
	mux.Middleware(SessionMiddleware(store, "app-session"))
	mux.Middleware(CORSMiddleware(DefaultCORSConfig))

	// Handler that uses session
	mux.Get("", "/profile", func(ctx Context) error {
		session := ctx.Get("session").(*Session)

		// Simulate user authentication check
		userID := session.Get("userID")
		if userID == nil {
			// Not authenticated - set some data and redirect
			session.Set("returnTo", "/profile")
			return ctx.JSON(401, map[string]string{"error": "not authenticated"})
		}

		// Authenticated - return profile
		return ctx.JSON(200, map[string]interface{}{
			"userID":  userID,
			"profile": "user profile data",
		})
	})

	mux.Post("", "/login", func(ctx Context) error {
		session := ctx.Get("session").(*Session)

		// Simulate login
		session.Set("userID", 12345)
		session.Set("loginTime", time.Now().Unix())

		returnTo := session.Get("returnTo")
		if returnTo != nil {
			session.Delete("returnTo")
		}

		return ctx.JSON(200, map[string]interface{}{
			"status":   "logged in",
			"returnTo": returnTo,
		})
	})

	// Test unauthenticated access
	req1 := httptest.NewRequest("GET", "/profile", nil)
	req1.Header.Set("Origin", "https://example.com")
	rec1 := httptest.NewRecorder()

	mux.ServeHTTP(rec1, req1)

	if rec1.Code != 401 {
		t.Errorf("Expected status 401 for unauthenticated access, got %d", rec1.Code)
	}

	// Check CORS headers are set
	if rec1.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS headers to be set")
	}

	// Get session cookie from Set-Cookie header
	setCookieHeaders := rec1.Header()["Set-Cookie"]
	if len(setCookieHeaders) != 1 {
		t.Fatalf("Expected 1 session cookie, got %d", len(setCookieHeaders))
	}

	// Parse the cookie manually
	cookieStr := setCookieHeaders[0]
	parts := strings.Split(cookieStr, ";")
	cookiePart := strings.TrimSpace(parts[0])
	cookieNameValue := strings.SplitN(cookiePart, "=", 2)
	if len(cookieNameValue) != 2 {
		t.Fatalf("Invalid cookie format: %s", cookieStr)
	}

	cookie := &http.Cookie{
		Name:  cookieNameValue[0],
		Value: cookieNameValue[1],
	}

	// Test login
	req2 := httptest.NewRequest("POST", "/login", nil)
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()

	mux.ServeHTTP(rec2, req2)

	if rec2.Code != 200 {
		t.Errorf("Expected status 200 for login, got %d", rec2.Code)
	}

	var loginResponse map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	if loginResponse["returnTo"] != "/profile" {
		t.Errorf("Expected returnTo '/profile', got %v", loginResponse["returnTo"])
	}

	// Test authenticated access
	req3 := httptest.NewRequest("GET", "/profile", nil)
	req3.AddCookie(cookie)
	rec3 := httptest.NewRecorder()

	mux.ServeHTTP(rec3, req3)

	if rec3.Code != 200 {
		t.Errorf("Expected status 200 for authenticated access, got %d", rec3.Code)
	}

	var profileResponse map[string]interface{}
	if err := json.NewDecoder(rec3.Body).Decode(&profileResponse); err != nil {
		t.Fatalf("Failed to decode profile response: %v", err)
	}

	if profileResponse["userID"] != float64(12345) {
		t.Errorf("Expected userID 12345, got %v", profileResponse["userID"])
	}
	if profileResponse["profile"] != "user profile data" {
		t.Errorf("Expected profile data, got %v", profileResponse["profile"])
	}
}

func TestSessionMiddleware_StoreErrors(t *testing.T) {
	// Test behavior when store operations fail
	store := NewInMemoryStore("test-session", NewOptions())
	defer store.Close()

	mux := NewMux()

	// Custom error handler to capture middleware errors
	var capturedError error
	mux.ErrorHandler(func(ctx Context, err error) {
		capturedError = err
		_ = ctx.String(500, "Error: "+err.Error())
	})

	mux.Middleware(SessionMiddleware(store, "test-session"))

	// Normal handler
	mux.Get("", "/test", func(ctx Context) error {
		return ctx.String(200, "OK")
	})

	// Test with store error during session creation by using a mock store that returns errors
	errorStoreInstance := &mockErrorStore{}

	// Create new mux with error store
	mux2 := NewMux()
	mux2.ErrorHandler(func(ctx Context, err error) {
		capturedError = err
		_ = ctx.String(500, "Store error")
	})
	mux2.Middleware(SessionMiddleware(errorStoreInstance, "test-session"))
	mux2.Get("", "/test", func(ctx Context) error {
		return ctx.String(200, "Should not reach here")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// This should cause an error during session creation
	mux2.ServeHTTP(rec, req)

	// Should have captured an error
	if capturedError == nil {
		t.Error("Expected error to be captured when store operations fail")
	}

	if rec.Code != 500 {
		t.Errorf("Expected status 500 when store fails, got %d", rec.Code)
	}
}

// mockErrorStore is a store implementation that always returns errors for testing
type mockErrorStore struct{}

func (e *mockErrorStore) Get(_ *http.Request, _ string) (*Session, error) {
	return nil, fmt.Errorf("simulated Get error")
}

func (e *mockErrorStore) New(_ *http.Request, _ string) (*Session, error) {
	return nil, fmt.Errorf("simulated New error")
}

func (e *mockErrorStore) Save(_ *http.Request, _ http.ResponseWriter, _ *Session) error {
	return fmt.Errorf("simulated Save error")
}

// =============================================================================
// CookieStore Tests
// =============================================================================

func TestNewCookieStore(t *testing.T) {
	t.Run("valid keys", func(t *testing.T) {
		// Test AES-128 (16 bytes)
		key16 := make([]byte, 16)
		store, err := NewCookieStore("test", key16, nil)
		if err != nil {
			t.Errorf("Expected no error for 16-byte key, got: %v", err)
		}
		if store == nil {
			t.Error("Expected store to be created")
		}

		// Test AES-192 (24 bytes)
		key24 := make([]byte, 24)
		store, err = NewCookieStore("test", key24, nil)
		if err != nil {
			t.Errorf("Expected no error for 24-byte key, got: %v", err)
		}
		if store == nil {
			t.Error("Expected store to be created")
		}

		// Test AES-256 (32 bytes)
		key32 := make([]byte, 32)
		store, err = NewCookieStore("test", key32, NewOptions())
		if err != nil {
			t.Errorf("Expected no error for 32-byte key, got: %v", err)
		}
		if store == nil {
			t.Error("Expected store to be created")
		}
	})

	t.Run("invalid keys", func(t *testing.T) {
		invalidKeys := [][]byte{
			make([]byte, 15), // Too short
			make([]byte, 17), // Invalid length
			make([]byte, 31), // Invalid length
			make([]byte, 33), // Too long
			{},               // Empty
		}

		for _, key := range invalidKeys {
			store, err := NewCookieStore("test", key, nil)
			if err == nil {
				t.Errorf("Expected error for key length %d, got none", len(key))
			}
			if store != nil {
				t.Errorf("Expected no store for invalid key, got store")
			}
		}
	})

	t.Run("default options", func(t *testing.T) {
		key := make([]byte, 32)
		store, err := NewCookieStore("test", key, nil)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check that default options are applied
		if store.options == nil {
			t.Error("Expected default options to be set")
		}
	})
}

func TestCookieStore_NewSession(t *testing.T) {
	key := make([]byte, 32)
	store, err := NewCookieStore("test-session", key, &Options{
		Path:     "/",
		MaxAge:   3600,
		Secure:   false, // For testing
		HttpOnly: true,
	})
	if err != nil {
		t.Fatalf("Expected no error creating store, got: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)

	session, err := store.New(req, "test-session")
	if err != nil {
		t.Fatalf("Expected no error creating session, got: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be non-nil")
	}
	if session.ID != "" {
		t.Error("Expected session ID to be empty for cookie store")
	}
	if !session.IsNew {
		t.Error("Expected new session to have IsNew = true")
	}
	if session.Values == nil {
		t.Error("Expected session Values to be initialized")
	}
	if session.name != "test-session" {
		t.Errorf("Expected session name to be 'test-session', got '%s'", session.name)
	}
	if session.store != store {
		t.Error("Expected session store to be set correctly")
	}
}

func TestCookieStore_SaveAndGet(t *testing.T) {
	key := make([]byte, 32)
	// Use random key for better security testing
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	store, err := NewCookieStore("test-session", key, &Options{
		Path:     "/",
		MaxAge:   3600,
		Secure:   false, // For testing
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	if err != nil {
		t.Fatalf("Expected no error creating store, got: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Create and save session
	session, err := store.New(req, "test-session")
	if err != nil {
		t.Fatalf("Expected no error creating session, got: %v", err)
	}

	// Add various data types to session
	session.Set("string", "hello world")
	session.Set("int", 42)
	session.Set("bool", true)
	session.Set("float", 3.14)
	session.Set("slice", []string{"a", "b", "c"})
	session.Set("map", map[string]int{"x": 1, "y": 2})

	err = store.Save(req, rec, session)
	if err != nil {
		t.Fatalf("Expected no error saving session, got: %v", err)
	}

	// Check cookie was set
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "test-session" {
		t.Errorf("Expected cookie name 'test-session', got '%s'", cookie.Name)
	}
	if cookie.Value == "" {
		t.Error("Expected cookie value to be non-empty")
	}
	if cookie.Path != "/" {
		t.Errorf("Expected cookie path '/', got '%s'", cookie.Path)
	}
	if cookie.MaxAge != 3600 {
		t.Errorf("Expected cookie MaxAge 3600, got %d", cookie.MaxAge)
	}
	if cookie.Secure {
		t.Error("Expected cookie Secure to be false for test")
	}
	if !cookie.HttpOnly {
		t.Error("Expected cookie HttpOnly to be true")
	}

	// Create new request with cookie to test Get
	req2 := httptest.NewRequest("GET", "/test2", nil)
	req2.AddCookie(cookie)

	// Get existing session
	retrievedSession, err := store.Get(req2, "test-session")
	if err != nil {
		t.Fatalf("Expected no error retrieving session, got: %v", err)
	}

	if retrievedSession == nil {
		t.Fatal("Expected retrieved session to be non-nil")
	}
	if retrievedSession.IsNew {
		t.Error("Expected retrieved session to have IsNew = false")
	}

	// Verify all session data was preserved
	if retrievedSession.Get("string") != "hello world" {
		t.Errorf("Expected string 'hello world', got %v", retrievedSession.Get("string"))
	}
	if retrievedSession.Get("int") != 42 {
		t.Errorf("Expected int 42, got %v", retrievedSession.Get("int"))
	}
	if retrievedSession.Get("bool") != true {
		t.Errorf("Expected bool true, got %v", retrievedSession.Get("bool"))
	}
	if retrievedSession.Get("float") != 3.14 {
		t.Errorf("Expected float 3.14, got %v", retrievedSession.Get("float"))
	}

	// Test slice (need to compare elements since gob may change slice type)
	retrievedSlice := retrievedSession.Get("slice")
	if retrievedSlice == nil {
		t.Error("Expected slice to be retrieved")
	} else {
		if slice, ok := retrievedSlice.([]string); ok {
			expected := []string{"a", "b", "c"}
			if len(slice) != len(expected) {
				t.Errorf("Expected slice length %d, got %d", len(expected), len(slice))
			} else {
				for i, v := range expected {
					if slice[i] != v {
						t.Errorf("Expected slice[%d] = '%s', got '%s'", i, v, slice[i])
					}
				}
			}
		} else {
			t.Errorf("Expected slice type []string, got %T", retrievedSlice)
		}
	}

	// Test map
	retrievedMap := retrievedSession.Get("map")
	if retrievedMap == nil {
		t.Error("Expected map to be retrieved")
	} else {
		if m, ok := retrievedMap.(map[string]int); ok {
			expected := map[string]int{"x": 1, "y": 2}
			for k, v := range expected {
				if m[k] != v {
					t.Errorf("Expected map['%s'] = %d, got %d", k, v, m[k])
				}
			}
		} else {
			t.Errorf("Expected map type map[string]int, got %T", retrievedMap)
		}
	}
}

func TestCookieStore_GetNonExistentSession(t *testing.T) {
	key := make([]byte, 32)
	store, err := NewCookieStore("test-session", key, NewOptions())
	if err != nil {
		t.Fatalf("Expected no error creating store, got: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)

	// Try to get session without cookie
	_, err = store.Get(req, "test-session")
	if err != http.ErrNoCookie {
		t.Errorf("Expected ErrNoCookie, got: %v", err)
	}

	// Try to get session with invalid cookie value
	req.AddCookie(&http.Cookie{Name: "test-session", Value: "invalid-data"})
	_, err = store.Get(req, "test-session")
	if err != http.ErrNoCookie {
		t.Errorf("Expected ErrNoCookie for invalid cookie, got: %v", err)
	}
}

func TestCookieStore_EmptySession(t *testing.T) {
	key := make([]byte, 32)
	store, err := NewCookieStore("test-session", key, NewOptions())
	if err != nil {
		t.Fatalf("Expected no error creating store, got: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Create empty session
	session, err := store.New(req, "test-session")
	if err != nil {
		t.Fatalf("Expected no error creating session, got: %v", err)
	}

	// Save empty session (should clear cookie)
	err = store.Save(req, rec, session)
	if err != nil {
		t.Fatalf("Expected no error saving empty session, got: %v", err)
	}

	// Check that cookie was cleared (MaxAge = -1)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.MaxAge != -1 {
		t.Errorf("Expected cookie MaxAge -1 for empty session, got %d", cookie.MaxAge)
	}
	if cookie.Value != "" {
		t.Errorf("Expected empty cookie value, got '%s'", cookie.Value)
	}
}

func TestCookieStore_SizeLimit(t *testing.T) {
	key := make([]byte, 32)
	store, err := NewCookieStore("test-session", key, NewOptions())
	if err != nil {
		t.Fatalf("Expected no error creating store, got: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Create session with large data
	session, err := store.New(req, "test-session")
	if err != nil {
		t.Fatalf("Expected no error creating session, got: %v", err)
	}

	// Add large data that should exceed cookie size limit
	largeData := strings.Repeat("x", 5000) // 5KB should exceed 4KB limit
	session.Set("large", largeData)

	// Save should fail due to size limit
	err = store.Save(req, rec, session)
	if err == nil {
		t.Error("Expected error for large session data, got none")
	}
	if err != nil && !strings.Contains(err.Error(), "too large") {
		t.Errorf("Expected 'too large' error, got: %v", err)
	}
}

func TestCookieStore_EncryptionSecurity(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)

	// Generate different keys
	if _, err := rand.Read(key1); err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}

	store1, err := NewCookieStore("test", key1, NewOptions())
	if err != nil {
		t.Fatalf("Expected no error creating store1, got: %v", err)
	}

	store2, err := NewCookieStore("test", key2, NewOptions())
	if err != nil {
		t.Fatalf("Expected no error creating store2, got: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Create and save session with store1
	session, err := store1.New(req, "test")
	if err != nil {
		t.Fatalf("Expected no error creating session, got: %v", err)
	}

	session.Set("secret", "sensitive data")
	err = store1.Save(req, rec, session)
	if err != nil {
		t.Fatalf("Expected no error saving session, got: %v", err)
	}

	// Get cookie
	cookie := rec.Result().Cookies()[0]

	// Try to decrypt with store2 (different key) - should fail
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.AddCookie(cookie)

	_, err = store2.Get(req2, "test")
	if err != http.ErrNoCookie {
		t.Error("Expected decryption to fail with different key")
	}

	// Verify that same store can decrypt
	_, err = store1.Get(req2, "test")
	if err != nil {
		t.Errorf("Expected same store to decrypt successfully, got: %v", err)
	}
}

func TestCookieStore_ConcurrentAccess(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	store, err := NewCookieStore("test", key, NewOptions())
	if err != nil {
		t.Fatalf("Expected no error creating store, got: %v", err)
	}

	const numGoroutines = 50
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent encryption/decryption operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				rec := httptest.NewRecorder()

				// Create session
				session, err := store.New(req, "test")
				if err != nil {
					t.Errorf("Error creating session: %v", err)
					return
				}

				// Add unique data
				session.Set("goroutine", id)
				session.Set("operation", j)
				session.Set("data", fmt.Sprintf("data-%d-%d", id, j))

				// Save session
				err = store.Save(req, rec, session)
				if err != nil {
					t.Errorf("Error saving session: %v", err)
					return
				}

				// Retrieve and verify
				cookie := rec.Result().Cookies()[0]
				req2 := httptest.NewRequest("GET", "/test", nil)
				req2.AddCookie(cookie)

				retrieved, err := store.Get(req2, "test")
				if err != nil {
					t.Errorf("Error retrieving session: %v", err)
					return
				}

				if retrieved.Get("goroutine") != id {
					t.Errorf("Expected goroutine %d, got %v", id, retrieved.Get("goroutine"))
				}
				if retrieved.Get("operation") != j {
					t.Errorf("Expected operation %d, got %v", j, retrieved.Get("operation"))
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestCookieStore_Integration(t *testing.T) {
	// Test integration with SessionMiddleware
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	store, err := NewCookieStore("app-session", key, &Options{
		Path:     "/",
		MaxAge:   3600,
		Secure:   false, // For testing
		HttpOnly: true,
	})
	if err != nil {
		t.Fatalf("Expected no error creating store, got: %v", err)
	}

	mux := NewMux()
	mux.Middleware(SessionMiddleware(store, "app-session"))

	// Handler that uses session
	mux.Get("", "/test", func(ctx Context) error {
		session := ctx.Get("session").(*Session)

		// Check if returning user
		visits := session.Get("visits")
		if visits == nil {
			visits = 0
		}

		// Increment visit count
		visitCount := visits.(int) + 1
		session.Set("visits", visitCount)
		session.Set("lastVisit", time.Now().Unix())

		result := map[string]interface{}{
			"visits":    visitCount,
			"isNew":     session.IsNew,
			"lastVisit": session.Get("lastVisit"),
		}

		return ctx.JSON(200, result)
	})

	// First request - new session
	req1 := httptest.NewRequest("GET", "/test", nil)
	rec1 := httptest.NewRecorder()

	mux.ServeHTTP(rec1, req1)

	if rec1.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec1.Code)
	}

	// Check response
	var response1 map[string]interface{}
	if err := json.NewDecoder(rec1.Body).Decode(&response1); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response1["visits"] != float64(1) { // JSON decodes numbers as float64
		t.Errorf("Expected visits 1, got %v", response1["visits"])
	}
	if response1["isNew"] != true {
		t.Errorf("Expected isNew true, got %v", response1["isNew"])
	}

	// Get cookie from first response (parse Set-Cookie header manually since value may be long)
	setCookieHeaders := rec1.Header()["Set-Cookie"]
	if len(setCookieHeaders) != 1 {
		t.Fatalf("Expected 1 Set-Cookie header, got %d", len(setCookieHeaders))
	}

	// Parse the cookie manually (like we did in other tests)
	cookieStr := setCookieHeaders[0]
	parts := strings.Split(cookieStr, ";")
	cookiePart := strings.TrimSpace(parts[0])
	cookieNameValue := strings.SplitN(cookiePart, "=", 2)
	if len(cookieNameValue) != 2 {
		t.Fatalf("Invalid cookie format: %s", cookieStr)
	}

	cookie := &http.Cookie{
		Name:  cookieNameValue[0],
		Value: strings.ReplaceAll(cookieNameValue[1], "\n", ""), // Remove any newlines
	}

	// Second request - existing session
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()

	mux.ServeHTTP(rec2, req2)

	if rec2.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec2.Code)
	}

	var response2 map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&response2); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response2["visits"] != float64(2) {
		t.Errorf("Expected visits 2, got %v", response2["visits"])
	}
	if response2["isNew"] != false {
		t.Errorf("Expected isNew false, got %v", response2["isNew"])
	}

	// Verify lastVisit timestamp was preserved and updated
	// Note: timestamps might be the same if test runs very quickly, so check if they are close
	lastVisit1, ok1 := response1["lastVisit"].(float64)
	lastVisit2, ok2 := response2["lastVisit"].(float64)

	if !ok1 || !ok2 {
		t.Errorf("Expected lastVisit to be numeric, got %T and %T", response1["lastVisit"], response2["lastVisit"])
	} else if lastVisit2 < lastVisit1 {
		t.Errorf("Expected lastVisit to be updated (lastVisit2 >= lastVisit1), got %f < %f", lastVisit2, lastVisit1)
	}
}
