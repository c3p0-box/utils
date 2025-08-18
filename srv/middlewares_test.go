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
	mw.Write([]byte("test"))

	if mw.StatusCode != http.StatusOK {
		t.Errorf("Expected StatusCode to remain %d, got %d", http.StatusOK, mw.StatusCode)
	}
}

func TestLogging_CapturesRequestInfo(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("test response"))
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
		w.Write([]byte("test"))
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
		w.Write([]byte("success"))
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
		w.Write([]byte("success"))
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
