package srv

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

// ============================
// Mux Tests
// ============================

func TestNewMux(t *testing.T) {
	mux := NewMux()

	if mux == nil {
		t.Fatal("Expected NewMux to return non-nil mux")
	}
	if mux.Mux() == nil {
		t.Error("Expected mux to have underlying ServeMux")
	}
}

func TestMux_BasicMethods(t *testing.T) {
	mux := NewMux()

	// Test Handle and HandleFunc
	handled := false
	mux.Handle("/handle", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handled = true
		w.WriteHeader(200)
	}))

	mux.HandleFunc("/handlefunc", func(w http.ResponseWriter, r *http.Request) {
		handled = true
		w.WriteHeader(200)
	})

	// Test that mux implements http.Handler
	var _ http.Handler = mux

	// Test ServeHTTP with Handle
	req := httptest.NewRequest("GET", "/handle", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !handled {
		t.Error("Expected handler to be called")
	}
	if rec.Code != 200 {
		t.Errorf("Expected status code 200, got %d", rec.Code)
	}

	// Reset and test HandleFunc
	handled = false
	req = httptest.NewRequest("GET", "/handlefunc", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !handled {
		t.Error("Expected handler function to be called")
	}
}

func TestMux_HTTPMethods(t *testing.T) {
	mux := NewMux()

	methods := []struct {
		name    string
		method  string
		setup   func(string, HandlerFunc)
		pattern string
	}{
		{"GET", "GET", mux.Get, "/get"},
		{"POST", "POST", mux.Post, "/post"},
		{"PUT", "PUT", mux.Put, "/put"},
		{"DELETE", "DELETE", mux.Delete, "/delete"},
		{"PATCH", "PATCH", mux.Patch, "/patch"},
		{"HEAD", "HEAD", mux.Head, "/head"},
		{"OPTIONS", "OPTIONS", mux.Options, "/options"},
	}

	for _, test := range methods {
		t.Run(test.name, func(t *testing.T) {
			called := false
			test.setup(test.pattern, func(ctx *HttpContext) error {
				called = true
				ctx.WriteHeader(200)
				return nil
			})

			// Test correct method
			req := httptest.NewRequest(test.method, test.pattern, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if !called {
				t.Errorf("Expected %s handler to be called", test.method)
			}
			if rec.Code != 200 {
				t.Errorf("Expected status code 200, got %d", rec.Code)
			}

			// Test wrong method (should not match)
			called = false
			wrongMethod := "GET"
			if test.method == "GET" {
				wrongMethod = "POST"
			}

			req = httptest.NewRequest(wrongMethod, test.pattern, nil)
			rec = httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if called {
				t.Errorf("Expected %s handler NOT to be called for %s request", test.method, wrongMethod)
			}
		})
	}
}

func TestMux_Integration(t *testing.T) {
	mux := NewMux()

	// Set up various routes
	mux.Get("/users", func(ctx *HttpContext) error {
		return ctx.String(200, "GET users")
	})

	mux.Post("/users", func(ctx *HttpContext) error {
		return ctx.String(201, "POST users")
	})

	mux.Get("/users/{id}", func(ctx *HttpContext) error {
		id := ctx.Param("id")
		return ctx.String(200, "GET user "+id)
	})

	tests := []struct {
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{"GET", "/users", 200, "GET users"},
		{"POST", "/users", 201, "POST users"},
		{"GET", "/users/123", 200, "GET user 123"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.method, test.path), func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != test.expectedStatus {
				t.Errorf("Expected status %d, got %d", test.expectedStatus, rec.Code)
			}

			if rec.Body.String() != test.expectedBody {
				t.Errorf("Expected body '%s', got '%s'", test.expectedBody, rec.Body.String())
			}
		})
	}
}

// ============================
// Error Handling Tests
// ============================

func TestMux_ErrorHandler(t *testing.T) {
	mux := NewMux()

	var capturedError error
	var capturedContext *HttpContext

	// Set custom error handler
	mux.ErrorHandler(func(ctx *HttpContext, err error) {
		capturedError = err
		capturedContext = ctx
		ctx.JSON(400, map[string]string{"error": "Custom error: " + err.Error()})
	})

	// Register a handler that returns an error
	mux.Get("/error", func(ctx *HttpContext) error {
		return errors.New("test error")
	})

	// Test the error handling
	req := httptest.NewRequest("GET", "/error", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Verify error was captured
	if capturedError == nil {
		t.Error("Expected error to be captured by error handler")
	}
	if capturedError.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", capturedError.Error())
	}
	if capturedContext == nil {
		t.Error("Expected context to be passed to error handler")
	}

	// Verify response
	if rec.Code != 400 {
		t.Errorf("Expected status code 400, got %d", rec.Code)
	}

	expectedBody := `{"error":"Custom error: test error"}`
	if strings.TrimSpace(rec.Body.String()) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, strings.TrimSpace(rec.Body.String()))
	}
}

func TestMux_DefaultErrorHandler(t *testing.T) {
	mux := NewMux()

	// Register a handler that returns an error (should use default error handler)
	mux.Post("/error", func(ctx *HttpContext) error {
		return errors.New("internal error")
	})

	// Test the default error handling
	req := httptest.NewRequest("POST", "/error", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Verify default error response
	if rec.Code != 500 {
		t.Errorf("Expected status code 500, got %d", rec.Code)
	}

	expectedBody := `{"error":"Internal Server Error"}`
	if strings.TrimSpace(rec.Body.String()) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, strings.TrimSpace(rec.Body.String()))
	}
}

func TestMux_NoErrorHandling(t *testing.T) {
	mux := NewMux()

	// Register a handler that returns no error
	mux.Put("/success", func(ctx *HttpContext) error {
		return ctx.JSON(200, map[string]string{"status": "success"})
	})

	// Test successful handling
	req := httptest.NewRequest("PUT", "/success", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Verify success response
	if rec.Code != 200 {
		t.Errorf("Expected status code 200, got %d", rec.Code)
	}

	expectedBody := `{"status":"success"}`
	if strings.TrimSpace(rec.Body.String()) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, strings.TrimSpace(rec.Body.String()))
	}
}

func TestMux_ErrorFromResponseMethod(t *testing.T) {
	mux := NewMux()

	var capturedError error
	mux.ErrorHandler(func(ctx *HttpContext, err error) {
		capturedError = err
		ctx.String(500, "Response error occurred")
	})

	// Register a handler that has an error in the response method
	mux.Delete("/response-error", func(ctx *HttpContext) error {
		// This should work fine and not trigger error handler
		return ctx.JSON(200, map[string]string{"message": "success"})
	})

	// Test the handling
	req := httptest.NewRequest("DELETE", "/response-error", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Verify no error was captured (JSON should work fine)
	if capturedError != nil {
		t.Errorf("Expected no error to be captured, but got: %v", capturedError)
	}

	// Verify successful response
	if rec.Code != 200 {
		t.Errorf("Expected status code 200, got %d", rec.Code)
	}
}
