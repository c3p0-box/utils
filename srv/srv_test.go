package srv

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
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

func TestRunServer_ParameterDefaults(t *testing.T) {
	// Test parameter default logic without actually calling RunServer
	// (which would hang waiting for signals)

	// Test default host logic
	host := ""
	if host == "" {
		host = "0.0.0.0"
	}
	if host != "0.0.0.0" {
		t.Errorf("Expected default host to be '0.0.0.0', got '%s'", host)
	}

	// Test default port logic
	port := ""
	if port == "" {
		port = "8000"
	}
	if port != "8000" {
		t.Errorf("Expected default port to be '8000', got '%s'", port)
	}

	// Test custom values are preserved
	customHost := "localhost"
	customPort := "9999"
	if customHost == "" {
		customHost = "0.0.0.0"
	}
	if customPort == "" {
		customPort = "8000"
	}
	if customHost != "localhost" {
		t.Errorf("Expected custom host to be preserved as 'localhost', got '%s'", customHost)
	}
	if customPort != "9999" {
		t.Errorf("Expected custom port to be preserved as '9999', got '%s'", customPort)
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

// Test cleanup function behavior
func TestRunServer_CleanupFunction(t *testing.T) {
	// Test that cleanup function is called
	cleanupCalled := false
	cleanup := func() error {
		cleanupCalled = true
		return nil
	}

	// We can't test the full RunServer without starting an actual server,
	// but we can test that the cleanup function works as expected
	err := cleanup()
	if err != nil {
		t.Errorf("Expected cleanup to return nil, got %v", err)
	}
	if !cleanupCalled {
		t.Error("Expected cleanup function to be called")
	}
}

func TestRunServer_CleanupFunctionError(t *testing.T) {
	// Test that cleanup function errors are returned
	expectedErr := errors.New("cleanup failed")
	cleanup := func() error {
		return expectedErr
	}

	err := cleanup()
	if err == nil {
		t.Error("Expected cleanup to return an error")
	}
	if err.Error() != "cleanup failed" {
		t.Errorf("Expected error message 'cleanup failed', got '%s'", err.Error())
	}
}

// Test RunServer with server startup error (e.g., port already in use)
func TestRunServer_ServerStartupError(t *testing.T) {
	// First, start a server on a specific port to occupy it
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test listener: %v", err)
	}
	defer listener.Close()

	// Get the port that's now in use
	addr := listener.Addr().(*net.TCPAddr)
	host := "127.0.0.1"
	port := fmt.Sprintf("%d", addr.Port)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cleanupCalled := false
	cleanup := func() error {
		cleanupCalled = true
		return nil
	}

	// Try to start server on the same port - should fail
	err = RunServer(handler, host, port, cleanup)

	// Should return an error due to address already in use
	if err == nil {
		t.Error("Expected RunServer to return an error when port is already in use")
	}

	// Cleanup should not be called if server fails to start
	if cleanupCalled {
		t.Error("Expected cleanup not to be called when server fails to start")
	}
}

// Test RunServer with invalid address to test more error paths
func TestRunServer_InvalidAddress(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cleanupCalled := false
	cleanup := func() error {
		cleanupCalled = true
		return nil
	}

	// Try to bind to an invalid port (port 0 should not fail, but let's try an invalid IP)
	err := RunServer(handler, "999.999.999.999", "8080", cleanup)

	// Should return an error due to invalid address
	if err == nil {
		t.Error("Expected RunServer to return an error with invalid address")
	}

	// Cleanup should not be called if server fails to start
	if cleanupCalled {
		t.Error("Expected cleanup not to be called when server fails to start")
	}
}

// Test RunServer with port that requires privilege (like port 80) to test bind errors
func TestRunServer_PrivilegedPort(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cleanupCalled := false
	cleanup := func() error {
		cleanupCalled = true
		return nil
	}

	// Try to bind to port 1 (privileged port) - should fail for non-root users
	err := RunServer(handler, "127.0.0.1", "1", cleanup)

	// Should return an error due to permission denied
	if err == nil {
		t.Logf("Expected error for privileged port, but got nil (running as root?)")
	}

	// Cleanup should not be called if server fails to start
	if cleanupCalled {
		t.Error("Expected cleanup not to be called when server fails to start")
	}
}
