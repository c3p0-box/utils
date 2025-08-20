package srv

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
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

func TestMiddleware_Type(t *testing.T) {
	// Test that Middleware type can be used
	var middleware Middleware = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "true")
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("X-Test") != "true" {
		t.Errorf("Expected X-Test header to be 'true', got '%s'", rec.Header().Get("X-Test"))
	}
}

func TestMiddlewareChain_SingleMiddleware(t *testing.T) {
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "middleware")
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := MiddlewareChain(middleware)
	wrapped := chain(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("X-Test") != "middleware" {
		t.Errorf("Expected X-Test header to be 'middleware', got '%s'", rec.Header().Get("X-Test"))
	}
}

func TestMiddlewareChain_MultipleMiddleware(t *testing.T) {
	// Test that middleware are applied in correct order (first in chain is outermost)
	var order []string
	mutex := &sync.Mutex{}

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mutex.Lock()
			order = append(order, "middleware1-before")
			mutex.Unlock()
			next.ServeHTTP(w, r)
			mutex.Lock()
			order = append(order, "middleware1-after")
			mutex.Unlock()
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mutex.Lock()
			order = append(order, "middleware2-before")
			mutex.Unlock()
			next.ServeHTTP(w, r)
			mutex.Lock()
			order = append(order, "middleware2-after")
			mutex.Unlock()
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		order = append(order, "handler")
		mutex.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	chain := MiddlewareChain(middleware1, middleware2)
	wrapped := chain(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(order) != len(expected) {
		t.Fatalf("Expected %d elements in order, got %d: %v", len(expected), len(order), order)
	}

	for i, exp := range expected {
		if order[i] != exp {
			t.Errorf("Expected order[%d] to be '%s', got '%s'", i, exp, order[i])
		}
	}
}

func TestMiddlewareChain_EmptyChain(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "handler")
		w.WriteHeader(http.StatusOK)
	})

	chain := MiddlewareChain()
	wrapped := chain(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("X-Test") != "handler" {
		t.Errorf("Expected X-Test header to be 'handler', got '%s'", rec.Header().Get("X-Test"))
	}
}

func TestMiddlewareWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	mw := &MiddlewareWriter{ResponseWriter: rec, StatusCode: http.StatusOK}

	mw.WriteHeader(http.StatusNotFound)

	if mw.StatusCode != http.StatusNotFound {
		t.Errorf("Expected StatusCode to be %d, got %d", http.StatusNotFound, mw.StatusCode)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected underlying ResponseWriter Code to be %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestMiddlewareWriter_DefaultStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	mw := &MiddlewareWriter{ResponseWriter: rec, StatusCode: http.StatusOK}

	// Don't call WriteHeader explicitly
	_, _ = mw.Write([]byte("test"))

	if mw.StatusCode != http.StatusOK {
		t.Errorf("Expected StatusCode to remain %d, got %d", http.StatusOK, mw.StatusCode)
	}
}

func TestLogging_CapturesRequestInfo(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("test response"))
	})

	wrapped := Logging(handler)
	req := httptest.NewRequest("POST", "/api/test?param=value", strings.NewReader("test body"))
	req.Header.Set("User-Agent", "test-agent/1.0")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	logOutput := captureLogs(t, func() {
		wrapped.ServeHTTP(rec, req)
	})

	// Check that log contains expected information
	expectedParts := []string{
		"views.Logging",
		"status=201",
		"method=POST",
		"path=/api/test",
		"user-agent=test-agent/1.0",
		"remote-addr=192.168.1.1:12345",
		"request completed",
	}

	for _, part := range expectedParts {
		if !strings.Contains(logOutput, part) {
			t.Errorf("Expected log output to contain '%s', but it didn't. Log output: %s", part, logOutput)
		}
	}
}

func TestLogging_DefaultStatusCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't explicitly set status code
		_, _ = w.Write([]byte("test"))
	})

	wrapped := Logging(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	logOutput := captureLogs(t, func() {
		wrapped.ServeHTTP(rec, req)
	})

	if !strings.Contains(logOutput, "status=200") {
		t.Errorf("Expected log output to contain 'status=200' for default status code. Log output: %s", logOutput)
	}
}

func TestRecover_HandlesPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrapped := Recover(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	logOutput := captureLogs(t, func() {
		// This should not panic and should log the error
		wrapped.ServeHTTP(rec, req)
	})

	expectedParts := []string{
		"views.Recover",
		"error=\"test panic\"",
		"recovered from panic",
	}

	for _, part := range expectedParts {
		if !strings.Contains(logOutput, part) {
			t.Errorf("Expected log output to contain '%s', but it didn't. Log output: %s", part, logOutput)
		}
	}
}

func TestRecover_NormalOperation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	wrapped := Recover(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	logOutput := captureLogs(t, func() {
		wrapped.ServeHTTP(rec, req)
	})

	// Should not log anything when no panic occurs
	if strings.Contains(logOutput, "recovered from panic") {
		t.Errorf("Expected no panic recovery log, but got: %s", logOutput)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "success" {
		t.Errorf("Expected body 'success', got '%s'", rec.Body.String())
	}
}

// Integration test for middleware chain with logging and recovery
func TestMiddlewareIntegration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panic" {
			panic("test panic in handler")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Chain both middleware
	wrapped := MiddlewareChain(Logging, Recover)(handler)

	t.Run("normal request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		logOutput := captureLogs(t, func() {
			wrapped.ServeHTTP(rec, req)
		})

		// Should have logging but no panic recovery
		if !strings.Contains(logOutput, "request completed") {
			t.Error("Expected logging output")
		}
		if strings.Contains(logOutput, "recovered from panic") {
			t.Error("Should not have panic recovery log for normal request")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("panic request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/panic", nil)
		rec := httptest.NewRecorder()

		logOutput := captureLogs(t, func() {
			wrapped.ServeHTTP(rec, req)
		})

		// Should have both logging and panic recovery
		if !strings.Contains(logOutput, "request completed") {
			t.Error("Expected logging output")
		}
		if !strings.Contains(logOutput, "recovered from panic") {
			t.Error("Expected panic recovery log")
		}
	})
}

// =============================================================================
// CORS Middleware Tests
// =============================================================================

func TestCORS_DefaultConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	corsMiddleware := CORS(DefaultCORSConfig)
	wrapped := corsMiddleware(handler)

	t.Run("with origin header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

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

	t.Run("without origin header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		// No Origin header
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		// Should still process the request normally
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		if rec.Body.String() != "success" {
			t.Errorf("Expected body 'success', got '%s'", rec.Body.String())
		}

		// Should have Vary header even without Origin
		if rec.Header().Get("Vary") != "Origin" {
			t.Errorf("Expected Vary header to include 'Origin', got '%s'", rec.Header().Get("Vary"))
		}
	})
}

func TestCORS_CustomConfig(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"https://example.com", "https://app.example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Total-Count"},
		MaxAge:           3600,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsMiddleware := CORS(config)
	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check CORS headers
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin to be 'https://example.com', got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Expected Access-Control-Allow-Credentials to be 'true', got '%s'", rec.Header().Get("Access-Control-Allow-Credentials"))
	}

	if rec.Header().Get("Access-Control-Expose-Headers") != "X-Total-Count" {
		t.Errorf("Expected Access-Control-Expose-Headers to be 'X-Total-Count', got '%s'", rec.Header().Get("Access-Control-Expose-Headers"))
	}
}

func TestCORS_OriginValidation(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"https://example.com", "https://*.app.com"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsMiddleware := CORS(config)
	wrapped := corsMiddleware(handler)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
		shouldAllow    bool
	}{
		{
			name:           "exact match allowed",
			origin:         "https://example.com",
			expectedOrigin: "https://example.com",
			shouldAllow:    true,
		},
		{
			name:           "wildcard pattern match",
			origin:         "https://test.app.com",
			expectedOrigin: "https://test.app.com",
			shouldAllow:    true,
		},
		{
			name:        "not allowed origin",
			origin:      "https://malicious.com",
			shouldAllow: false,
		},
		{
			name:        "subdomain not matching pattern",
			origin:      "https://app.com",
			shouldAllow: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test regular requests
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", test.origin)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if test.shouldAllow {
				if rec.Header().Get("Access-Control-Allow-Origin") != test.expectedOrigin {
					t.Errorf("Expected Access-Control-Allow-Origin to be '%s', got '%s'", test.expectedOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
				}
			} else {
				if rec.Header().Get("Access-Control-Allow-Origin") != "" {
					t.Errorf("Expected no Access-Control-Allow-Origin header, got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
				}
			}

			// Test preflight requests for disallowed origins
			if !test.shouldAllow {
				preflightReq := httptest.NewRequest("OPTIONS", "/test", nil)
				preflightReq.Header.Set("Origin", test.origin)
				preflightReq.Header.Set("Access-Control-Request-Method", "POST")
				preflightRec := httptest.NewRecorder()

				preflightHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Error("Handler should not be called for disallowed preflight requests")
				})
				preflightWrapped := corsMiddleware(preflightHandler)
				preflightWrapped.ServeHTTP(preflightRec, preflightReq)

				// Should return 204 but without CORS headers
				if preflightRec.Code != http.StatusNoContent {
					t.Errorf("Expected preflight status 204, got %d", preflightRec.Code)
				}

				if preflightRec.Header().Get("Access-Control-Allow-Origin") != "" {
					t.Errorf("Expected no Access-Control-Allow-Origin header for disallowed preflight origin, got '%s'", preflightRec.Header().Get("Access-Control-Allow-Origin"))
				}
			}
		})
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		AllowMethods: []string{"GET", "POST", "PUT"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:       3600,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should not be called for preflight requests
		t.Error("Handler should not be called for preflight requests")
	})

	corsMiddleware := CORS(config)
	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check preflight response
	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin to be 'https://example.com', got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PUT" {
		t.Errorf("Expected Access-Control-Allow-Methods to be 'GET, POST, PUT', got '%s'", rec.Header().Get("Access-Control-Allow-Methods"))
	}

	if rec.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization" {
		t.Errorf("Expected Access-Control-Allow-Headers to be 'Content-Type, Authorization', got '%s'", rec.Header().Get("Access-Control-Allow-Headers"))
	}

	if rec.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("Expected Access-Control-Max-Age to be '3600', got '%s'", rec.Header().Get("Access-Control-Max-Age"))
	}

	// Check Vary headers
	varyHeaders := rec.Header().Values("Vary")
	varyHeaderString := strings.Join(varyHeaders, ", ")
	if !strings.Contains(varyHeaderString, "Origin") || !strings.Contains(varyHeaderString, "Access-Control-Request-Method") || !strings.Contains(varyHeaderString, "Access-Control-Request-Headers") {
		t.Errorf("Expected Vary header to include CORS headers, got '%s'", varyHeaderString)
	}

	// Test preflight without origin
	t.Run("preflight without origin", func(t *testing.T) {
		preflightHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for preflight without origin")
		})

		wrapped := corsMiddleware(preflightHandler)
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Access-Control-Request-Method", "POST")
		// No Origin header
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		// Should return 204 for preflight without origin
		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", rec.Code)
		}
	})
}

func TestCORS_PreflightWithEchoHeaders(t *testing.T) {
	// Test that when no specific AllowHeaders are configured,
	// the middleware echoes the requested headers
	config := CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		AllowMethods: []string{"GET", "POST"},
		// AllowHeaders is empty - should echo requested headers
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for preflight requests")
	})

	corsMiddleware := CORS(config)
	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "X-Custom-Header, Authorization")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Headers") != "X-Custom-Header, Authorization" {
		t.Errorf("Expected Access-Control-Allow-Headers to echo requested headers 'X-Custom-Header, Authorization', got '%s'", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCORS_CredentialsWithWildcard(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsMiddleware := CORS(config)
	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// When credentials are allowed with wildcard, should echo the specific origin
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin to echo specific origin 'https://example.com', got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Expected Access-Control-Allow-Credentials to be 'true', got '%s'", rec.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestCORS_Integration(t *testing.T) {
	// Test CORS middleware integration with other middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-App", "test")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Chain CORS with other middleware
	wrapped := MiddlewareChain(Logging, CORS(DefaultCORSConfig), Recover)(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("User-Agent", "test-client")
	rec := httptest.NewRecorder()

	logOutput := captureLogs(t, func() {
		wrapped.ServeHTTP(rec, req)
	})

	// Check that CORS headers are set
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS headers to be set")
	}

	// Check that logging middleware still works
	if !strings.Contains(logOutput, "request completed") {
		t.Error("Expected logging middleware to work")
	}

	// Check that the handler was executed
	if rec.Header().Get("X-App") != "test" {
		t.Error("Expected handler to be executed")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}
