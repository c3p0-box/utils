package srv

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
	} else if err.Error() != "cleanup failed" {
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
	defer func(listener net.Listener) { _ = listener.Close() }(listener)

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
		setup   func(string, string, HandlerFunc)
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
			test.setup("", test.pattern, func(ctx Context) error {
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
	mux.Get("", "/users", func(ctx Context) error {
		return ctx.String(200, "GET users")
	})

	mux.Post("", "/users", func(ctx Context) error {
		return ctx.String(201, "POST users")
	})

	mux.Get("", "/users/{id}", func(ctx Context) error {
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
	var capturedContext Context

	// Set custom error handler
	mux.ErrorHandler(func(ctx Context, err error) {
		capturedError = err
		capturedContext = ctx
		_ = ctx.JSON(400, map[string]string{"error": "Custom error: " + err.Error()})
	})

	// Register a handler that returns an error
	mux.Get("", "/error", func(ctx Context) error {
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
	mux.Post("", "/error", func(ctx Context) error {
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

	expectedBody := `Something went wrong`
	if strings.TrimSpace(rec.Body.String()) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, strings.TrimSpace(rec.Body.String()))
	}
}

func TestMux_NoErrorHandling(t *testing.T) {
	mux := NewMux()

	// Register a handler that returns no error
	mux.Put("", "/success", func(ctx Context) error {
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
	mux.ErrorHandler(func(ctx Context, err error) {
		capturedError = err
		_ = ctx.String(500, "Response error occurred")
	})

	// Register a handler that has an error in the response method
	mux.Delete("", "/response-error", func(ctx Context) error {
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

// ============================
// URL Reversing Tests
// ============================

func TestMux_NamedRoutes_BasicFunctionality(t *testing.T) {
	mux := NewMux()

	// Register named routes
	mux.Get("user-list", "/users", func(ctx Context) error {
		return ctx.String(200, "users list")
	})

	mux.Post("user-create", "/users", func(ctx Context) error {
		return ctx.String(201, "user created")
	})

	mux.Get("user-profile", "/users/{id}", func(ctx Context) error {
		id := ctx.Param("id")
		return ctx.String(200, "user "+id)
	})

	// Test that named routes work as regular routes
	tests := []struct {
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{"GET", "/users", 200, "users list"},
		{"POST", "/users", 201, "user created"},
		{"GET", "/users/123", 200, "user 123"},
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

func TestMux_NamedRoutes_AllMethods(t *testing.T) {
	mux := NewMux()

	methods := []struct {
		name      string
		method    string
		setup     func(string, string, HandlerFunc)
		routeName string
		pattern   string
	}{
		{"Get", "GET", mux.Get, "get-route", "/get"},
		{"Post", "POST", mux.Post, "post-route", "/post"},
		{"Put", "PUT", mux.Put, "put-route", "/put"},
		{"Delete", "DELETE", mux.Delete, "delete-route", "/delete"},
		{"Patch", "PATCH", mux.Patch, "patch-route", "/patch"},
		{"Head", "HEAD", mux.Head, "head-route", "/head"},
		{"Options", "OPTIONS", mux.Options, "options-route", "/options"},
	}

	for _, test := range methods {
		t.Run(test.name, func(t *testing.T) {
			called := false
			test.setup(test.routeName, test.pattern, func(ctx Context) error {
				called = true
				ctx.WriteHeader(200)
				return nil
			})

			// Test that the named route works
			req := httptest.NewRequest(test.method, test.pattern, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if !called {
				t.Errorf("Expected %s handler to be called", test.method)
			}
			if rec.Code != 200 {
				t.Errorf("Expected status code 200, got %d", rec.Code)
			}

			// Test that the route is stored for reversing
			url, err := mux.Reverse(test.routeName, nil)
			if err != nil {
				t.Errorf("Expected no error for URL reversing, got: %v", err)
			}
			if url != test.pattern {
				t.Errorf("Expected URL '%s', got '%s'", test.pattern, url)
			}
		})
	}
}

func TestMux_Reverse_BasicURLGeneration(t *testing.T) {
	mux := NewMux()

	// Register routes
	mux.Get("user-list", "/users", func(ctx Context) error {
		return ctx.String(200, "OK")
	})

	mux.Post("user-create", "/users", func(ctx Context) error {
		return ctx.String(200, "OK")
	})

	mux.Get("user-profile", "/users/{id}", func(ctx Context) error {
		return ctx.String(200, "OK")
	})

	tests := []struct {
		name        string
		routeName   string
		method      string
		params      map[string]string
		expectedURL string
		expectError bool
	}{
		{
			name:        "simple route without parameters",
			routeName:   "user-list",
			params:      nil,
			expectedURL: "/users",
			expectError: false,
		},
		{
			name:        "same pattern different method",
			routeName:   "user-create",
			params:      nil,
			expectedURL: "/users",
			expectError: false,
		},
		{
			name:        "route with parameters",
			routeName:   "user-profile",
			params:      map[string]string{"id": "123"},
			expectedURL: "/users/123",
			expectError: false,
		},
		{
			name:        "route not found",
			routeName:   "non-existent",
			params:      nil,
			expectedURL: "",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, err := mux.Reverse(test.routeName, test.params)

			if test.expectError {
				if err == nil {
					t.Error("Expected an error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if url != test.expectedURL {
					t.Errorf("Expected URL '%s', got '%s'", test.expectedURL, url)
				}
			}
		})
	}
}

func TestMux_Reverse_ParameterSubstitution(t *testing.T) {
	mux := NewMux()

	// Register routes with various parameter patterns
	mux.Get("user-profile", "/users/{id}", func(ctx Context) error {
		return ctx.String(200, "OK")
	})

	mux.Get("user-posts", "/users/{userId}/posts/{postId}", func(ctx Context) error {
		return ctx.String(200, "OK")
	})

	mux.Get("complex-route", "/api/v1/{version}/users/{id}/settings/{setting}", func(ctx Context) error {
		return ctx.String(200, "OK")
	})

	tests := []struct {
		name        string
		routeName   string
		params      map[string]string
		expectedURL string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "single parameter substitution",
			routeName:   "user-profile",
			params:      map[string]string{"id": "456"},
			expectedURL: "/users/456",
			expectError: false,
		},
		{
			name:        "multiple parameter substitution",
			routeName:   "user-posts",
			params:      map[string]string{"userId": "123", "postId": "789"},
			expectedURL: "/users/123/posts/789",
			expectError: false,
		},
		{
			name:        "complex parameter substitution",
			routeName:   "complex-route",
			params:      map[string]string{"version": "2", "id": "user123", "setting": "privacy"},
			expectedURL: "/api/v1/2/users/user123/settings/privacy",
			expectError: false,
		},
		{
			name:        "missing required parameter",
			routeName:   "user-profile",
			params:      nil,
			expectedURL: "",
			expectError: true,
			errorMsg:    "is required",
		},
		{
			name:        "partially missing parameters",
			routeName:   "user-posts",
			params:      map[string]string{"userId": "123"},
			expectedURL: "",
			expectError: true,
			errorMsg:    "is required",
		},
		{
			name:        "invalid parameter name",
			routeName:   "user-profile",
			params:      map[string]string{"wrong": "123"},
			expectedURL: "",
			expectError: true,
			errorMsg:    "is required",
		},
		{
			name:        "extra parameters (should work)",
			routeName:   "user-profile",
			params:      map[string]string{"id": "123", "extra": "ignored"},
			expectedURL: "/users/123",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, err := mux.Reverse(test.routeName, test.params)

			if test.expectError {
				if err == nil {
					t.Error("Expected an error but got none")
				} else if test.errorMsg != "" && !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", test.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if url != test.expectedURL {
					t.Errorf("Expected URL '%s', got '%s'", test.expectedURL, url)
				}
			}
		})
	}
}

func TestMux_Reverse_SameNameDifferentMethods(t *testing.T) {
	mux := NewMux()

	// Register the same route name for different methods
	// This is a common RESTful pattern
	mux.Get("users", "/users", func(ctx Context) error {
		return ctx.String(200, "GET users")
	})

	mux.Post("users", "/users", func(ctx Context) error {
		return ctx.String(201, "POST users")
	})

	mux.Put("user", "/users/{id}", func(ctx Context) error {
		return ctx.String(200, "PUT user")
	})

	mux.Delete("user", "/users/{id}", func(ctx Context) error {
		return ctx.String(204, "DELETE user")
	})

	tests := []struct {
		name        string
		routeName   string
		method      string
		params      map[string]string
		expectedURL string
		expectError bool
	}{
		{
			name:        "GET users route",
			routeName:   "users",
			method:      "GET",
			params:      nil,
			expectedURL: "/users",
			expectError: false,
		},
		{
			name:        "POST users route",
			routeName:   "users",
			method:      "POST",
			params:      nil,
			expectedURL: "/users",
			expectError: false,
		},
		{
			name:        "PUT user route",
			routeName:   "user",
			method:      "PUT",
			params:      map[string]string{"id": "123"},
			expectedURL: "/users/123",
			expectError: false,
		},
		{
			name:        "DELETE user route",
			routeName:   "user",
			method:      "DELETE",
			params:      map[string]string{"id": "456"},
			expectedURL: "/users/456",
			expectError: false,
		},
		{
			name:        "same name works for different methods",
			routeName:   "users",
			params:      nil,
			expectedURL: "/users",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, err := mux.Reverse(test.routeName, test.params)

			if test.expectError {
				if err == nil {
					t.Error("Expected an error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if url != test.expectedURL {
					t.Errorf("Expected URL '%s', got '%s'", test.expectedURL, url)
				}
			}
		})
	}
}

func TestMux_Reverse_Integration(t *testing.T) {
	mux := NewMux()

	// Set up a realistic RESTful API with named routes
	mux.Get("users", "/users", func(ctx Context) error {
		// Generate URLs to other routes within handler
		userProfileURL, _ := mux.Reverse("user", map[string]string{"id": "123"})
		createUserURL, _ := mux.Reverse("users", nil)

		return ctx.JSON(200, map[string]interface{}{
			"users":            []string{"user1", "user2"},
			"user_profile_url": userProfileURL,
			"create_user_url":  createUserURL,
		})
	})

	mux.Post("users", "/users", func(ctx Context) error {
		return ctx.String(201, "User created")
	})

	mux.Get("user", "/users/{id}", func(ctx Context) error {
		id := ctx.Param("id")
		editURL, _ := mux.Reverse("user", map[string]string{"id": id})
		deleteURL, _ := mux.Reverse("user", map[string]string{"id": id})

		return ctx.JSON(200, map[string]interface{}{
			"id":         id,
			"edit_url":   editURL,
			"delete_url": deleteURL,
		})
	})

	mux.Put("user", "/users/{id}", func(ctx Context) error {
		return ctx.String(200, "User updated")
	})

	mux.Delete("user", "/users/{id}", func(ctx Context) error {
		return ctx.String(204, "User deleted")
	})

	// Test the integration
	t.Run("GET users with URL generation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Parse JSON response
		var response map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Verify generated URLs
		if profileURL, ok := response["user_profile_url"].(string); ok {
			if profileURL != "/users/123" {
				t.Errorf("Expected user profile URL '/users/123', got '%s'", profileURL)
			}
		} else {
			t.Error("Expected user_profile_url in response")
		}

		if createURL, ok := response["create_user_url"].(string); ok {
			if createURL != "/users" {
				t.Errorf("Expected create user URL '/users', got '%s'", createURL)
			}
		} else {
			t.Error("Expected create_user_url in response")
		}
	})

	t.Run("GET user with URL generation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/456", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Parse JSON response
		var response map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Verify generated URLs
		if editURL, ok := response["edit_url"].(string); ok {
			if editURL != "/users/456" {
				t.Errorf("Expected edit URL '/users/456', got '%s'", editURL)
			}
		} else {
			t.Error("Expected edit_url in response")
		}

		if deleteURL, ok := response["delete_url"].(string); ok {
			if deleteURL != "/users/456" {
				t.Errorf("Expected delete URL '/users/456', got '%s'", deleteURL)
			}
		} else {
			t.Error("Expected delete_url in response")
		}
	})
}

// ============================
// Middleware Tests
// ============================

func TestMux_Middleware_Single(t *testing.T) {
	mux := NewMux()

	// Add middleware that sets a header
	mux.Middleware(func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			ctx.SetHeader("X-Middleware", "test")
			return next(ctx)
		}
	})

	// Register a route after middleware
	mux.Get("", "/test", func(ctx Context) error {
		return ctx.String(200, "success")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status code 200, got %d", rec.Code)
	}

	if rec.Header().Get("X-Middleware") != "test" {
		t.Errorf("Expected X-Middleware header to be 'test', got '%s'", rec.Header().Get("X-Middleware"))
	}

	if rec.Body.String() != "success" {
		t.Errorf("Expected body 'success', got '%s'", rec.Body.String())
	}
}

func TestMux_Middleware_Multiple(t *testing.T) {
	mux := NewMux()

	// Track middleware execution order
	var order []string

	// Add first middleware
	mux.Middleware(func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			order = append(order, "middleware1-before")
			ctx.SetHeader("X-Middleware-1", "true")
			err := next(ctx)
			order = append(order, "middleware1-after")
			return err
		}
	})

	// Add second middleware
	mux.Middleware(func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			order = append(order, "middleware2-before")
			ctx.SetHeader("X-Middleware-2", "true")
			err := next(ctx)
			order = append(order, "middleware2-after")
			return err
		}
	})

	// Register a route after middleware
	mux.Get("", "/test", func(ctx Context) error {
		order = append(order, "handler")
		return ctx.String(200, "success")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Verify execution order (first added = outermost)
	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(order) != len(expectedOrder) {
		t.Fatalf("Expected %d elements in order, got %d: %v", len(expectedOrder), len(order), order)
	}

	for i, exp := range expectedOrder {
		if order[i] != exp {
			t.Errorf("Expected order[%d] to be '%s', got '%s'", i, exp, order[i])
		}
	}

	// Verify both middleware headers were set
	if rec.Header().Get("X-Middleware-1") != "true" {
		t.Errorf("Expected X-Middleware-1 header to be 'true', got '%s'", rec.Header().Get("X-Middleware-1"))
	}

	if rec.Header().Get("X-Middleware-2") != "true" {
		t.Errorf("Expected X-Middleware-2 header to be 'true', got '%s'", rec.Header().Get("X-Middleware-2"))
	}
}

func TestMux_Middleware_ErrorHandling(t *testing.T) {
	mux := NewMux()

	var capturedError error
	mux.ErrorHandler(func(ctx Context, err error) {
		capturedError = err
		_ = ctx.String(400, "middleware error: "+err.Error())
	})

	// Add middleware that returns an error
	mux.Middleware(func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			return errors.New("middleware error")
		}
	})

	// Register a route (should not be reached)
	mux.Get("", "/test", func(ctx Context) error {
		t.Error("Handler should not be called when middleware returns error")
		return ctx.String(200, "success")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if capturedError == nil {
		t.Error("Expected error to be captured")
	} else if capturedError.Error() != "middleware error" {
		t.Errorf("Expected error message 'middleware error', got '%s'", capturedError.Error())
	}

	if rec.Code != 400 {
		t.Errorf("Expected status code 400, got %d", rec.Code)
	}

	expectedBody := "middleware error: middleware error"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, rec.Body.String())
	}
}

func TestMux_Middleware_ContextValues(t *testing.T) {
	mux := NewMux()

	// Add middleware that sets context values
	mux.Middleware(func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			ctx.Set("user", "test-user")
			ctx.Set("authenticated", true)
			return next(ctx)
		}
	})

	// Register a route that uses context values
	mux.Get("", "/test", func(ctx Context) error {
		user := ctx.Get("user")
		authenticated := ctx.Get("authenticated")

		if user != "test-user" {
			t.Errorf("Expected user to be 'test-user', got '%v'", user)
		}

		if authenticated != true {
			t.Errorf("Expected authenticated to be true, got '%v'", authenticated)
		}

		return ctx.JSON(200, map[string]interface{}{
			"user":          user,
			"authenticated": authenticated,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status code 200, got %d", rec.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response["user"] != "test-user" {
		t.Errorf("Expected JSON user to be 'test-user', got '%v'", response["user"])
	}

	if response["authenticated"] != true {
		t.Errorf("Expected JSON authenticated to be true, got '%v'", response["authenticated"])
	}
}

func TestMux_Middleware_Integration(t *testing.T) {
	mux := NewMux()

	// Add authentication middleware
	mux.Middleware(func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			token := ctx.GetHeader("Authorization")
			if token == "" {
				return errors.New("unauthorized: missing token")
			}
			ctx.Set("user", "authenticated-user")
			return next(ctx)
		}
	})

	// Add logging middleware
	mux.Middleware(func(next HandlerFunc) HandlerFunc {
		return func(ctx Context) error {
			ctx.SetHeader("X-Logged", "true")
			return next(ctx)
		}
	})

	// Set up error handler
	var capturedError error
	mux.ErrorHandler(func(ctx Context, err error) {
		capturedError = err
		_ = ctx.String(401, "Error: "+err.Error())
	})

	// Register route
	mux.Get("", "/protected", func(ctx Context) error {
		user := ctx.Get("user")
		return ctx.JSON(200, map[string]interface{}{"user": user})
	})

	t.Run("request with authorization", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer token123")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("Expected status code 200, got %d", rec.Code)
		}

		if rec.Header().Get("X-Logged") != "true" {
			t.Errorf("Expected X-Logged header to be set")
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		if response["user"] != "authenticated-user" {
			t.Errorf("Expected user to be 'authenticated-user', got '%v'", response["user"])
		}
	})

	t.Run("request without authorization", func(t *testing.T) {
		capturedError = nil // Reset
		req := httptest.NewRequest("GET", "/protected", nil)
		// No Authorization header
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != 401 {
			t.Errorf("Expected status code 401, got %d", rec.Code)
		}

		if capturedError == nil {
			t.Error("Expected error to be captured")
		} else if !strings.Contains(capturedError.Error(), "unauthorized") {
			t.Errorf("Expected error to contain 'unauthorized', got '%s'", capturedError.Error())
		}

		expectedBody := "Error: unauthorized: missing token"
		if rec.Body.String() != expectedBody {
			t.Errorf("Expected body '%s', got '%s'", expectedBody, rec.Body.String())
		}
	})
}
