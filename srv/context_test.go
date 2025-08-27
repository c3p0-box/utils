package srv

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// ============================
// Interface Implementation Tests
// ============================

// TestHttpContextImplementsContext verifies that HttpContext implements the Context interface
func TestHttpContextImplementsContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	ctx := NewHttpContext(rec, req)

	// This should compile without errors if HttpContext implements Context
	var _ Context = ctx

	// Additional runtime verification
	if ctx == nil {
		t.Fatal("Expected HttpContext instance to implement Context interface")
	}
}

// ============================
// HttpContext Tests
// ============================

func TestNewHttpContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	ctx := NewHttpContext(rec, req)

	if ctx == nil {
		t.Fatal("Expected NewHttpContext to return non-nil context")
	}
	if ctx.Request() != req {
		t.Error("Expected context to store the request")
	}
	if ctx.Response() != rec {
		t.Error("Expected context to store the response writer")
	}
	if ctx.values == nil {
		t.Error("Expected context to initialize values map")
	}
}

func TestHttpContext_ValueStore(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	ctx := NewHttpContext(rec, req)

	// Test Set and Get
	ctx.Set("key1", "value1")
	ctx.Set("key2", 42)
	ctx.Set("key3", true)

	if val := ctx.Get("key1"); val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}
	if val := ctx.Get("key2"); val != 42 {
		t.Errorf("Expected 42, got %v", val)
	}
	if val := ctx.Get("key3"); val != true {
		t.Errorf("Expected true, got %v", val)
	}

	// Test non-existent key
	if val := ctx.Get("nonexistent"); val != nil {
		t.Errorf("Expected nil for non-existent key, got %v", val)
	}

	// Test overwrite
	ctx.Set("key1", "newvalue")
	if val := ctx.Get("key1"); val != "newvalue" {
		t.Errorf("Expected 'newvalue', got %v", val)
	}
}

func TestHttpContext_ValueStore_ThreadSafety(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	ctx := NewHttpContext(rec, req)

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // readers and writers

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				ctx.Set(key, value)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				ctx.Get(key) // Just checking it doesn't panic
			}
		}(i)
	}

	wg.Wait()
	// If we get here without panicking, the thread safety test passed
}

func TestHttpContext_RequestInformation(t *testing.T) {
	t.Run("basic request info", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/test", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		if ctx.Method() != "POST" {
			t.Errorf("Expected method 'POST', got '%s'", ctx.Method())
		}
		if ctx.Path() != "/api/test" {
			t.Errorf("Expected path '/api/test', got '%s'", ctx.Path())
		}
		if ctx.IsTLS() {
			t.Error("Expected IsTLS to be false for HTTP request")
		}
		if ctx.IsWebSocket() {
			t.Error("Expected IsWebSocket to be false for normal request")
		}
	})

	t.Run("TLS request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/secure", nil)
		req.TLS = &tls.ConnectionState{} // Simulate TLS
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		if !ctx.IsTLS() {
			t.Error("Expected IsTLS to be true for HTTPS request")
		}
	})

	t.Run("WebSocket request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set(HeaderUpgrade, "websocket")
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		if !ctx.IsWebSocket() {
			t.Error("Expected IsWebSocket to be true for WebSocket upgrade request")
		}
	})

	t.Run("SetPath", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/original/path", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		if ctx.Path() != "/original/path" {
			t.Errorf("Expected initial path '/original/path', got '%s'", ctx.Path())
		}

		ctx.SetPath("/new/path")
		if ctx.Path() != "/new/path" {
			t.Errorf("Expected path to be '/new/path' after SetPath, got '%s'", ctx.Path())
		}

		// Ensure original request is not mutated
		if req.URL.Path != "/original/path" {
			t.Errorf("Expected original request path to be unchanged, got '%s'", req.URL.Path)
		}
	})
}

func TestHttpContext_Parameters(t *testing.T) {
	t.Run("query parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?name=john&age=30&active=true", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		query := ctx.Query()
		if query.Get("name") != "john" {
			t.Errorf("Expected name 'john', got '%s'", query.Get("name"))
		}

		if ctx.QueryParam("name") != "john" {
			t.Errorf("Expected name 'john', got '%s'", ctx.QueryParam("name"))
		}
		if ctx.QueryParam("age") != "30" {
			t.Errorf("Expected age '30', got '%s'", ctx.QueryParam("age"))
		}
		if ctx.QueryParam("nonexistent") != "" {
			t.Errorf("Expected empty string for non-existent param, got '%s'", ctx.QueryParam("nonexistent"))
		}
	})

	t.Run("form values", func(t *testing.T) {
		form := url.Values{}
		form.Add("username", "testuser")
		form.Add("password", "secret")

		req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
		req.Header.Set(HeaderContentType, MIMEApplicationForm)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		if ctx.FormValue("username") != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", ctx.FormValue("username"))
		}
		if ctx.FormValue("password") != "secret" {
			t.Errorf("Expected password 'secret', got '%s'", ctx.FormValue("password"))
		}
		if ctx.FormValue("nonexistent") != "" {
			t.Errorf("Expected empty string for non-existent form value, got '%s'", ctx.FormValue("nonexistent"))
		}
	})
}

func TestHttpContext_Headers(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	rec := httptest.NewRecorder()
	ctx := NewHttpContext(rec, req)

	// Test request headers
	if ctx.GetHeader("Authorization") != "Bearer token123" {
		t.Errorf("Expected Authorization header 'Bearer token123', got '%s'", ctx.GetHeader("Authorization"))
	}
	if ctx.GetHeader("User-Agent") != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent header 'TestAgent/1.0', got '%s'", ctx.GetHeader("User-Agent"))
	}
	if ctx.GetHeader("Non-Existent") != "" {
		t.Errorf("Expected empty string for non-existent header, got '%s'", ctx.GetHeader("Non-Existent"))
	}

	headers := ctx.GetHeaders()
	if headers.Get("Authorization") != "Bearer token123" {
		t.Error("Expected GetHeaders to return all headers")
	}

	// Test response headers
	ctx.SetHeader(HeaderContentType, MIMEApplicationJSON)
	ctx.AddHeader("X-Custom", "value1")
	ctx.AddHeader("X-Custom", "value2")

	if rec.Header().Get(HeaderContentType) != MIMEApplicationJSON {
		t.Errorf("Expected Content-Type '%s', got '%s'", MIMEApplicationJSON, rec.Header().Get(HeaderContentType))
	}

	customHeaders := rec.Header().Values("X-Custom")
	if len(customHeaders) != 2 || customHeaders[0] != "value1" || customHeaders[1] != "value2" {
		t.Errorf("Expected X-Custom headers ['value1', 'value2'], got %v", customHeaders)
	}
}

func TestHttpContext_Cookies(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	req.AddCookie(&http.Cookie{Name: "preferences", Value: "dark-mode"})
	rec := httptest.NewRecorder()
	ctx := NewHttpContext(rec, req)

	// Test getting cookies
	sessionCookie, err := ctx.Cookie("session")
	if err != nil {
		t.Errorf("Expected no error getting session cookie, got %v", err)
	}
	if sessionCookie == nil || sessionCookie.Value != "abc123" {
		t.Errorf("Expected session cookie value 'abc123', got %v", sessionCookie)
	}

	// Test non-existent cookie
	_, err = ctx.Cookie("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent cookie")
	}

	// Test getting all cookies
	cookies := ctx.Cookies()
	if len(cookies) != 2 {
		t.Errorf("Expected 2 cookies, got %d", len(cookies))
	}

	// Test setting cookie
	newCookie := &http.Cookie{
		Name:     "new-cookie",
		Value:    "new-value",
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	}
	ctx.SetCookie(newCookie)

	setCookieHeader := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookieHeader, "new-cookie=new-value") {
		t.Errorf("Expected Set-Cookie header to contain new cookie, got '%s'", setCookieHeader)
	}
}

func TestHttpContext_ResponseMethods(t *testing.T) {
	t.Run("JSON response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/data", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		data := map[string]interface{}{
			"message": "success",
			"count":   42,
		}

		err := ctx.JSON(200, data)
		if err != nil {
			t.Errorf("Expected no error from JSON, got %v", err)
		}

		if rec.Code != 200 {
			t.Errorf("Expected status code 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get(HeaderContentType)
		if contentType != MIMEApplicationJSON {
			t.Errorf("Expected Content-Type '%s', got '%s'", MIMEApplicationJSON, contentType)
		}

		var result map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &result)
		if err != nil {
			t.Errorf("Expected valid JSON response, got error: %v", err)
		}
		if result["message"] != "success" || result["count"] != float64(42) {
			t.Errorf("Expected correct JSON data, got %v", result)
		}
	})

	t.Run("JSON response with invalid data", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/data", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		// Try to encode something that can't be JSON encoded
		invalidData := make(chan int)

		err := ctx.JSON(200, invalidData)
		if err == nil {
			t.Error("Expected error when encoding invalid JSON data")
		}
	})

	t.Run("String response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/text", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		err := ctx.String(201, "Hello, World!")
		if err != nil {
			t.Errorf("Expected no error from String, got %v", err)
		}

		if rec.Code != 201 {
			t.Errorf("Expected status code 201, got %d", rec.Code)
		}

		contentType := rec.Header().Get(HeaderContentType)
		if contentType != MIMETextPlain {
			t.Errorf("Expected Content-Type '%s', got '%s'", MIMETextPlain, contentType)
		}

		if rec.Body.String() != "Hello, World!" {
			t.Errorf("Expected body 'Hello, World!', got '%s'", rec.Body.String())
		}
	})

	t.Run("HTML response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		html := "<html><body><h1>Hello</h1></body></html>"
		err := ctx.HTML(200, html)
		if err != nil {
			t.Errorf("Expected no error from HTML, got %v", err)
		}

		if rec.Code != 200 {
			t.Errorf("Expected status code 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get(HeaderContentType)
		if contentType != MIMETextHTMLCharsetUTF8 {
			t.Errorf("Expected Content-Type '%s', got '%s'", MIMETextHTML, contentType)
		}

		if rec.Body.String() != html {
			t.Errorf("Expected body '%s', got '%s'", html, rec.Body.String())
		}
	})

	t.Run("HTMLBlob response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		html := []byte("<html><body><h1>Hello</h1></body></html>")
		err := ctx.HTMLBlob(200, html)
		if err != nil {
			t.Errorf("Expected no error from HTML, got %v", err)
		}

		if rec.Code != 200 {
			t.Errorf("Expected status code 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get(HeaderContentType)
		if contentType != MIMETextHTMLCharsetUTF8 {
			t.Errorf("Expected Content-Type '%s', got '%s'", MIMETextHTML, contentType)
		}

		if rec.Body.String() != string(html) {
			t.Errorf("Expected body '%s', got '%s'", html, rec.Body.String())
		}
	})

	t.Run("Redirect response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/old-page", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		_ = ctx.Redirect(302, "/new-page")

		if rec.Code != 302 {
			t.Errorf("Expected status code 302, got %d", rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "/new-page" {
			t.Errorf("Expected Location '/new-page', got '%s'", location)
		}
	})

	t.Run("WriteHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)

		ctx.WriteHeader(204)

		if rec.Code != 204 {
			t.Errorf("Expected status code 204, got %d", rec.Code)
		}
	})
}

// ============================
// Benchmark Tests
// ============================

func BenchmarkHttpContext_ValueStore(b *testing.B) {
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	ctx := NewHttpContext(rec, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		ctx.Set(key, i)
		ctx.Get(key)
	}
}

func BenchmarkHttpContext_JSON(b *testing.B) {
	req := httptest.NewRequest("GET", "/test", nil)
	data := map[string]interface{}{
		"message": "hello",
		"count":   42,
		"active":  true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		ctx := NewHttpContext(rec, req)
		_ = ctx.JSON(200, data)
	}
}
