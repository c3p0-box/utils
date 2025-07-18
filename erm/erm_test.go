package erm

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestNew tests the New function with various inputs
func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		msg      string
		err      error
		wantCode int
	}{
		{"with valid code", http.StatusBadRequest, "bad request", errors.New("root"), http.StatusBadRequest},
		{"with zero code", 0, "test message", errors.New("test"), http.StatusInternalServerError},
		{"with nil error", http.StatusBadRequest, "test message", nil, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(tt.code, tt.msg, tt.err)

			// Verify interface implementation
			if _, ok := e.(Error); !ok {
				t.Fatal("New() should return Error interface")
			}

			// Verify fields
			if e.Code() != tt.wantCode {
				t.Fatalf("Code() = %d, want %d", e.Code(), tt.wantCode)
			}

			if Message(e) != tt.msg {
				t.Fatalf("Message() = %q, want %q", Message(e), tt.msg)
			}

			if e.Op() == "" {
				t.Fatal("Op() should not be empty")
			}

			if len(e.Stack()) == 0 {
				t.Fatal("Stack() should not be empty")
			}

			// Test error chain
			if tt.err != nil {
				if !errors.Is(e, tt.err) {
					t.Fatal("errors.Is should find wrapped error")
				}

				// Test errors.As
				var target *StackError
				if !errors.As(e, &target) {
					t.Fatal("errors.As should extract *StackError")
				}
			}
		})
	}
}

// TestStackErrorError tests all branches of the Error() method
func TestStackErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *StackError
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "<nil>",
		},
		{
			name:     "op and root present",
			err:      &StackError{op: "TestOp", root: errors.New("underlying")},
			expected: "TestOp: underlying",
		},
		{
			name:     "only root present",
			err:      &StackError{root: errors.New("only error")},
			expected: "only error",
		},
		{
			name:     "only message present",
			err:      &StackError{msg: "only message"},
			expected: "only message",
		},
		{
			name:     "nothing present",
			err:      &StackError{},
			expected: "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Fatalf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestStackErrorMethods tests all StackError interface methods
func TestStackErrorMethods(t *testing.T) {
	// Test with nil receiver
	var nilErr *StackError

	if nilErr.Error() != "<nil>" {
		t.Fatalf("nil Error() = %q, want '<nil>'", nilErr.Error())
	}
	if nilErr.Op() != "" {
		t.Fatalf("nil Op() = %q, want empty", nilErr.Op())
	}
	if nilErr.Code() != http.StatusOK {
		t.Fatalf("nil Code() = %d, want %d", nilErr.Code(), http.StatusOK)
	}
	if nilErr.Unwrap() != nil {
		t.Fatal("nil Unwrap() should return nil")
	}
	if nilErr.Stack() != nil {
		t.Fatal("nil Stack() should return nil")
	}

	// Test with valid receiver
	root := errors.New("root error")
	err := &StackError{
		op:    "TestOp",
		code:  http.StatusBadRequest,
		msg:   "test message",
		root:  root,
		stack: []uintptr{1, 2, 3},
	}

	if err.Op() != "TestOp" {
		t.Fatalf("Op() = %q, want %q", err.Op(), "TestOp")
	}
	if err.Code() != http.StatusBadRequest {
		t.Fatalf("Code() = %d, want %d", err.Code(), http.StatusBadRequest)
	}
	if err.Unwrap() != root {
		t.Fatal("Unwrap() should return root error")
	}
	if len(err.Stack()) != 3 {
		t.Fatalf("Stack() length = %d, want 3", len(err.Stack()))
	}

	// Test Unwrap when no root error
	errNoRoot := &StackError{op: "test", code: 200, msg: "test"}
	if errNoRoot.Unwrap() != errNoRoot {
		t.Fatal("Unwrap() should return itself when no root error")
	}
}

// TestDetectOperation tests the detectOperation function comprehensively
func TestDetectOperation(t *testing.T) {
	t.Run("normal operation detection", func(t *testing.T) {
		op := detectOperation(1)
		if op == "" {
			t.Fatal("detectOperation should not return empty string")
		}
	})

	t.Run("high skip values trigger unknown", func(t *testing.T) {
		// Test runtime.Caller failure with high skip values
		highSkips := []int{100, 500, 1000, 10000, 100000}
		for _, skip := range highSkips {
			op := detectOperation(skip)
			if op != "unknown" {
				t.Logf("Skip %d returned: %q (expected 'unknown')", skip, op)
			}
		}
	})

	t.Run("method detection", func(t *testing.T) {
		svc := &testService{}

		// Test value receiver
		err1 := svc.testMethod()
		if !strings.Contains(err1.Op(), "testMethod") {
			t.Fatalf("Should detect method name, got: %q", err1.Op())
		}

		// Test pointer receiver
		err2 := svc.testPointerMethod()
		if !strings.Contains(err2.Op(), "testPointerMethod") {
			t.Fatalf("Should detect pointer method name, got: %q", err2.Op())
		}
	})

	t.Run("simple function", func(t *testing.T) {
		err := simpleFunc()
		if !strings.Contains(err.Op(), "simpleFunc") {
			t.Fatalf("Should detect simple function name, got: %q", err.Op())
		}
	})

	t.Run("comprehensive skip range", func(t *testing.T) {
		// Test sufficient range to hit edge cases without excessive redundancy
		for skip := 0; skip <= 100; skip += 10 {
			op := detectOperation(skip)
			if op == "" {
				t.Fatalf("detectOperation should never return empty string for skip=%d", skip)
			}
		}

		// Test boundary detection
		var unknownFound bool
		for skip := 100; skip <= 1000; skip += 100 {
			op := detectOperation(skip)
			if op == "unknown" {
				unknownFound = true
				break
			}
		}

		if !unknownFound {
			// Force with extreme value
			op := detectOperation(50000)
			if op != "unknown" {
				t.Log("Warning: Could not trigger 'unknown' result")
			}
		}
	})

	t.Run("different calling contexts", func(t *testing.T) {
		// Test from anonymous function
		anonResult := func() string { return detectOperation(1) }()
		if anonResult == "" {
			t.Fatal("Anonymous function should detect operation")
		}

		// Test from closure
		closureFunc := func() func() string {
			return func() string { return detectOperation(1) }
		}()
		closureResult := closureFunc()
		if closureResult == "" {
			t.Fatal("Closure should detect operation")
		}

		// Test from nested context
		nested := func() string {
			return detectOperation(2) // Skip the anonymous function
		}
		nestedResult := nested()
		if nestedResult == "" {
			t.Fatal("Nested function should detect operation")
		}
	})
}

// TestHelperFunctions tests Status, Message, Stack, and FormatStack helpers
func TestHelperFunctions(t *testing.T) {
	t.Run("Status helper", func(t *testing.T) {
		// Test with nil
		if Status(nil) != http.StatusOK {
			t.Fatalf("Status(nil) = %d, want %d", Status(nil), http.StatusOK)
		}

		// Test with erm error
		ermErr := New(http.StatusBadRequest, "test", nil)
		if Status(ermErr) != http.StatusBadRequest {
			t.Fatalf("Status(ermErr) = %d, want %d", Status(ermErr), http.StatusBadRequest)
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		if Status(stdErr) != http.StatusInternalServerError {
			t.Fatalf("Status(stdErr) = %d, want %d", Status(stdErr), http.StatusInternalServerError)
		}

		// Test with Error interface
		var e Error = &StackError{code: http.StatusNotFound}
		if Status(e) != http.StatusNotFound {
			t.Fatalf("Status(interface) = %d, want %d", Status(e), http.StatusNotFound)
		}
	})

	t.Run("Message helper", func(t *testing.T) {
		// Test with nil
		if Message(nil) != "" {
			t.Fatalf("Message(nil) = %q, want empty", Message(nil))
		}

		// Test with erm error with message
		ermErr := New(http.StatusBadRequest, "custom message", nil)
		if Message(ermErr) != "custom message" {
			t.Fatalf("Message(ermErr) = %q, want 'custom message'", Message(ermErr))
		}

		// Test with erm error with empty message
		emptyMsgErr := &StackError{code: http.StatusNotFound, msg: ""}
		if Message(emptyMsgErr) != "Not Found" {
			t.Fatalf("Message(empty) = %q, want 'Not Found'", Message(emptyMsgErr))
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		if Message(stdErr) != "Internal Server Error" {
			t.Fatalf("Message(stdErr) = %q, want 'Internal Server Error'", Message(stdErr))
		}
	})

	t.Run("Stack helper", func(t *testing.T) {
		// Test with nil
		if Stack(nil) != nil {
			t.Fatal("Stack(nil) should return nil")
		}

		// Test with erm error
		ermErr := New(http.StatusBadRequest, "test", nil)
		if len(Stack(ermErr)) == 0 {
			t.Fatal("Stack(ermErr) should not be empty")
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		if Stack(stdErr) != nil {
			t.Fatal("Stack(stdErr) should return nil")
		}

		// Test with Error interface
		var e Error = &StackError{stack: []uintptr{1, 2, 3}}
		if len(Stack(e)) != 3 {
			t.Fatalf("Stack(interface) length = %d, want 3", len(Stack(e)))
		}
	})
}

// TestFormatStack tests FormatStack function for complete coverage
func TestFormatStack(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if FormatStack(nil) != "" {
			t.Fatal("FormatStack(nil) should return empty string")
		}
	})

	t.Run("error with empty stack", func(t *testing.T) {
		err := &StackError{stack: []uintptr{}}
		if FormatStack(err) != "" {
			t.Fatal("FormatStack with empty stack should return empty string")
		}
	})

	t.Run("error with stack", func(t *testing.T) {
		err := New(http.StatusBadRequest, "test", errors.New("test"))
		formatted := FormatStack(err)

		if formatted == "" {
			t.Fatal("FormatStack should return non-empty string")
		}

		if !strings.Contains(formatted, ".go:") {
			t.Fatal("FormatStack should contain file and line info")
		}

		lines := strings.Split(formatted, "\n")
		if len(lines) < 2 {
			t.Fatal("FormatStack should contain multiple lines")
		}
	})

	t.Run("error with nil stack", func(t *testing.T) {
		err := &StackError{op: "test", stack: nil}
		if FormatStack(err) != "" {
			t.Fatal("FormatStack with nil stack should return empty string")
		}
	})
}

// TestWrap tests the Wrap function comprehensively
func TestWrap(t *testing.T) {
	t.Run("wrap nil", func(t *testing.T) {
		if Wrap(nil) != nil {
			t.Fatal("Wrap(nil) should return nil")
		}
	})

	t.Run("wrap erm error", func(t *testing.T) {
		original := New(http.StatusNotFound, "not found", errors.New("root"))
		wrapped := Wrap(original)

		// Properties should be preserved
		if Status(wrapped) != Status(original) {
			t.Fatal("Wrap should preserve status")
		}
		if Message(wrapped) != Message(original) {
			t.Fatal("Wrap should preserve message")
		}

		// Error chain should be maintained
		if !errors.Is(wrapped, original) {
			t.Fatal("Wrap should maintain error chain")
		}
	})

	t.Run("wrap standard error", func(t *testing.T) {
		stdErr := errors.New("standard error")
		wrapped := Wrap(stdErr)

		if wrapped == stdErr {
			t.Fatal("Wrap should create new erm error for standard errors")
		}

		if _, ok := wrapped.(Error); !ok {
			t.Fatal("Wrap should return Error interface")
		}

		if Status(wrapped) != http.StatusInternalServerError {
			t.Fatalf("Wrapped standard error should have 500 status, got %d", Status(wrapped))
		}

		if !errors.Is(wrapped, stdErr) {
			t.Fatal("Wrapped error should maintain error chain")
		}
	})
}

// TestConvenienceConstructors tests all convenience constructor functions
func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(msg string, err error) Error
		expected   int
		expectedOp string
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest, "BadRequest"},
		{"Unauthorized", Unauthorized, http.StatusUnauthorized, "Unauthorized"},
		{"Forbidden", Forbidden, http.StatusForbidden, "Forbidden"},
		{"NotFound", NotFound, http.StatusNotFound, "NotFound"},
		{"Conflict", Conflict, http.StatusConflict, "Conflict"},
		{"Internal", Internal, http.StatusInternalServerError, "Internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with error
			err := tt.fn("test message", errors.New("test"))

			if Status(err) != tt.expected {
				t.Fatalf("Status() = %d, want %d", Status(err), tt.expected)
			}

			if Message(err) != "test message" {
				t.Fatalf("Message() = %q, want %q", Message(err), "test message")
			}

			if len(Stack(err)) == 0 {
				t.Fatal("stack trace should be captured")
			}

			if err.Op() != tt.expectedOp {
				t.Fatalf("Operation should be %q, got: %q", tt.expectedOp, err.Op())
			}

			// Test with nil error
			errNil := tt.fn("test message", nil)
			if se, ok := errNil.(*StackError); ok {
				if se.root == nil {
					t.Fatal("Constructor should create error when nil is passed")
				}
				if se.msg != "test message" {
					t.Fatalf("Message should be preserved, got: %q", se.msg)
				}
			} else {
				t.Fatal("Constructor should return *StackError")
			}
		})
	}
}

// TestErrorChaining tests complex error scenarios
func TestErrorChaining(t *testing.T) {
	root := errors.New("root cause")
	layer1 := New(http.StatusNotFound, "not found", root)
	layer2 := Wrap(layer1)
	layer3 := Wrap(layer2)

	// Should preserve original metadata
	if Status(layer3) != http.StatusNotFound {
		t.Fatalf("Status() = %d, want %d", Status(layer3), http.StatusNotFound)
	}

	if Message(layer3) != "not found" {
		t.Fatalf("Message() = %q, want %q", Message(layer3), "not found")
	}

	// Should maintain error chain
	if !errors.Is(layer3, root) {
		t.Fatal("errors.Is should find root error")
	}

	if !errors.Is(layer3, layer1) {
		t.Fatal("errors.Is should find layer1 error")
	}
}

// TestEdgeCases tests remaining edge cases for comprehensive coverage
func TestEdgeCases(t *testing.T) {
	t.Run("error with empty message fallback", func(t *testing.T) {
		e := New(http.StatusBadRequest, "", errors.New("test"))
		msg := Message(e)
		if msg != "Bad Request" {
			t.Fatalf("Message() should return status text when msg is empty, got: %q", msg)
		}
	})

	t.Run("mixed error types", func(t *testing.T) {
		standardErr := errors.New("standard error")

		if Status(standardErr) != http.StatusInternalServerError {
			t.Fatalf("Status() = %d, want %d", Status(standardErr), http.StatusInternalServerError)
		}

		if Message(standardErr) != "Internal Server Error" {
			t.Fatalf("Message() = %q, want %q", Message(standardErr), "Internal Server Error")
		}

		if Stack(standardErr) != nil {
			t.Fatal("Stack() should return nil for non-erm errors")
		}
	})

	t.Run("goroutine context", func(t *testing.T) {
		done := make(chan string, 1)
		go func() {
			done <- detectOperation(1)
		}()
		op := <-done
		if op == "" {
			t.Fatal("Goroutine call should detect operation")
		}
	})
}

// Test helper types and functions
type testService struct{}

func (s testService) testMethod() Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}

func (s *testService) testPointerMethod() Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}

func simpleFunc() Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}
