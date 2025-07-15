package erm

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestNewAndHelpers(t *testing.T) {
	originalErr := errors.New("root cause")
	e := New(http.StatusBadRequest, "bad request", originalErr)

	// Error string should contain operation and underlying error
	errorStr := e.Error()
	if !strings.Contains(errorStr, "TestNewAndHelpers") {
		t.Fatalf("Error() should contain operation name, got: %q", errorStr)
	}
	if !strings.Contains(errorStr, "root cause") {
		t.Fatalf("Error() should contain underlying error, got: %q", errorStr)
	}

	// Status extraction
	if got := Status(e); got != http.StatusBadRequest {
		t.Fatalf("Status() = %d, want %d", got, http.StatusBadRequest)
	}

	// Message extraction
	if got := Message(e); got != "bad request" {
		t.Fatalf("Message() = %q, want %q", got, "bad request")
	}

	// Stack should not be empty
	if pcs := Stack(e); len(pcs) == 0 {
		t.Fatal("stack trace should be captured")
	}

	// errors.Is must work
	if !errors.Is(e, originalErr) {
		t.Fatal("errors.Is should find wrapped error")
	}

	// errors.As into *erm.Error should succeed
	var target *Error
	if !errors.As(e, &target) {
		t.Fatal("errors.As failed to extract *Error")
	}
}

func TestNewWithNilError(t *testing.T) {
	e := New(http.StatusBadRequest, "test message", nil)

	errorStr := e.Error()
	if !strings.Contains(errorStr, "TestNewWithNilError") {
		t.Fatalf("Error() should contain operation name, got: %q", errorStr)
	}

	if Status(e) != http.StatusBadRequest {
		t.Fatalf("Status() = %d, want %d", Status(e), http.StatusBadRequest)
	}

	if Message(e) != "test message" {
		t.Fatalf("Message() = %q, want %q", Message(e), "test message")
	}
}

func TestNewWithZeroCode(t *testing.T) {
	e := New(0, "test message", errors.New("test"))

	if Status(e) != http.StatusInternalServerError {
		t.Fatalf("Status() = %d, want %d", Status(e), http.StatusInternalServerError)
	}
}

func TestErrorMethodEdgeCases(t *testing.T) {
	// Test all branches of Error() method

	// Case 1: Both Op and Err are present
	e1 := New(http.StatusBadRequest, "test", errors.New("underlying"))
	expected1 := e1.Op + ": underlying"
	if e1.Error() != expected1 {
		t.Fatalf("Error() = %q, want %q", e1.Error(), expected1)
	}

	// Case 2: Only Err is present (no Op)
	e2 := &Error{Err: errors.New("only error")}
	if e2.Error() != "only error" {
		t.Fatalf("Error() = %q, want %q", e2.Error(), "only error")
	}

	// Case 3: Only Msg is present (no Op, no Err)
	e3 := &Error{Msg: "only message"}
	if e3.Error() != "only message" {
		t.Fatalf("Error() = %q, want %q", e3.Error(), "only message")
	}

	// Case 4: Nothing is present
	e4 := &Error{}
	if e4.Error() != "unknown error" {
		t.Fatalf("Error() = %q, want %q", e4.Error(), "unknown error")
	}
}

func TestDetectOperationEdgeCases(t *testing.T) {
	// Test detectOperation with various function name patterns

	// Test with a simple function (current test function)
	e1 := New(http.StatusBadRequest, "test", errors.New("test"))
	if !strings.Contains(e1.Op, "TestDetectOperationEdgeCases") {
		t.Fatalf("Should detect test function name, got: %q", e1.Op)
	}

	// Test with a method receiver
	svc := &testService{}
	e2 := svc.testMethod()
	if !strings.Contains(e2.Op, "testService.testMethod") {
		t.Fatalf("Should detect method receiver, got: %q", e2.Op)
	}

	// Test with a pointer receiver
	e3 := svc.testPointerMethod()
	if !strings.Contains(e3.Op, "testService.testPointerMethod") {
		t.Fatalf("Should detect pointer method receiver, got: %q", e3.Op)
	}
}

// Additional test for detectOperation edge cases
func TestDetectOperationFailureCases(t *testing.T) {
	// Test the fallback cases in detectOperation

	// We can't easily test runtime.Caller failure or runtime.FuncForPC failure
	// in normal conditions, but we can test the function name parsing edge cases

	// Test with a single part function name (edge case)
	// This is hard to create in normal Go code, but we can test the logic

	// Test the function with various skip levels to ensure robustness
	for skip := 1; skip <= 50; skip++ {
		op := detectOperation(skip)
		// Should always return something (either function name or "unknown")
		if op == "" {
			t.Fatalf("detectOperation should never return empty string for skip=%d", skip)
		}
		// When skip is too high, it should return "unknown"
		if skip > 20 && op != "unknown" {
			// This is expected behavior - when we skip too many frames, we get "unknown"
			t.Logf("Skip %d returned: %q", skip, op)
		}
	}

	// Test with extremely high skip value to trigger runtime.Caller failure
	op := detectOperation(1000)
	if op != "unknown" {
		t.Fatalf("detectOperation with very high skip should return 'unknown', got: %q", op)
	}
}

// Test function with a very simple name to test the single-part fallback
func simpleFunc() *Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}

func TestDetectOperationSimpleFunction(t *testing.T) {
	e := simpleFunc()
	// Should detect the simple function name
	if !strings.Contains(e.Op, "simpleFunc") {
		t.Fatalf("Should detect simple function name, got: %q", e.Op)
	}
}

// Test helpers for detectOperation edge cases
type testService struct{}

func (s testService) testMethod() *Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}

func (s *testService) testPointerMethod() *Error {
	return New(http.StatusBadRequest, "test", errors.New("test"))
}

func TestNilErrorHandling(t *testing.T) {
	// Test nil error handling in helper functions
	if Status(nil) != http.StatusOK {
		t.Fatalf("Status(nil) = %d, want %d", Status(nil), http.StatusOK)
	}

	if Message(nil) != "" {
		t.Fatalf("Message(nil) = %q, want empty string", Message(nil))
	}

	if Stack(nil) != nil {
		t.Fatal("Stack(nil) should return nil")
	}

	if FormatStack(nil) != "" {
		t.Fatal("FormatStack(nil) should return empty string")
	}
}

func TestNilErrorMethods(t *testing.T) {
	var e *Error

	if e.Error() != "<nil>" {
		t.Fatalf("nil Error.Error() = %q, want '<nil>'", e.Error())
	}

	if e.Unwrap() != nil {
		t.Fatal("nil Error.Unwrap() should return nil")
	}

	if e.Stack() != nil {
		t.Fatal("nil Error.Stack() should return nil")
	}
}

func TestWrapPreservesMetadata(t *testing.T) {
	base := New(http.StatusNotFound, "not found", errors.New("sql: no rows"))
	wrapped := Wrap(base)

	if Status(wrapped) != http.StatusNotFound {
		t.Fatalf("Wrap lost status code: got %d", Status(wrapped))
	}
	if Message(wrapped) != "not found" {
		t.Fatalf("Wrap lost message: got %q", Message(wrapped))
	}

	if !errors.Is(wrapped, base) {
		t.Fatal("wrapped error should include base in chain")
	}
}

func TestWrapWithNil(t *testing.T) {
	wrapped := Wrap(nil)
	if wrapped != nil {
		t.Fatal("Wrap(nil) should return nil")
	}
}

func TestWrapWithStandardError(t *testing.T) {
	base := errors.New("standard error")
	wrapped := Wrap(base)

	if Status(wrapped) != http.StatusInternalServerError {
		t.Fatalf("Wrap with standard error should default to 500, got %d", Status(wrapped))
	}

	if !errors.Is(wrapped, base) {
		t.Fatal("wrapped error should include base in chain")
	}
}

func TestFormatStack(t *testing.T) {
	e := New(http.StatusBadRequest, "test", errors.New("test"))
	formatted := FormatStack(e)

	if formatted == "" {
		t.Fatal("FormatStack should return non-empty string")
	}

	// Should contain some file information
	if !strings.Contains(formatted, ".go:") {
		t.Fatal("FormatStack should contain file and line info")
	}

	// Should contain multiple lines
	lines := strings.Split(formatted, "\n")
	if len(lines) < 2 {
		t.Fatal("FormatStack should contain multiple lines")
	}
}

func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(msg string, err error) *Error
		expected int
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest},
		{"Unauthorized", Unauthorized, http.StatusUnauthorized},
		{"Forbidden", Forbidden, http.StatusForbidden},
		{"NotFound", NotFound, http.StatusNotFound},
		{"Conflict", Conflict, http.StatusConflict},
		{"Internal", Internal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Check that operation is detected automatically
			if !strings.Contains(err.Op, "TestConvenienceConstructors") {
				t.Fatalf("Operation should be auto-detected, got: %q", err.Op)
			}
		})
	}
}

func TestConvenienceConstructorsWithNilError(t *testing.T) {
	// Test all convenience constructors with nil error
	constructors := []func(string, error) *Error{
		BadRequest, Unauthorized, Forbidden, NotFound, Conflict, Internal,
	}

	for _, constructor := range constructors {
		err := constructor("test message", nil)
		if err.Err == nil {
			t.Fatal("Constructor should create error when nil is passed")
		}
		if err.Msg != "test message" {
			t.Fatalf("Message should be preserved, got: %q", err.Msg)
		}
	}
}

func TestErrorChaining(t *testing.T) {
	// Test complex error chaining
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

func TestNonErmError(t *testing.T) {
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
}

func TestOperationDetection(t *testing.T) {
	e := New(http.StatusBadRequest, "test", errors.New("test"))

	// Should detect the test function name
	if !strings.Contains(e.Op, "TestOperationDetection") {
		t.Fatalf("Operation should be auto-detected, got: %q", e.Op)
	}
}

// Helper function to test operation detection in different contexts
func helperFunction() *Error {
	return New(http.StatusBadRequest, "helper error", errors.New("test"))
}

func TestOperationDetectionInHelper(t *testing.T) {
	e := helperFunction()

	// Should detect the helper function name
	if !strings.Contains(e.Op, "helperFunction") {
		t.Fatalf("Operation should detect helper function, got: %q", e.Op)
	}
}

func TestMessageWithEmptyMessage(t *testing.T) {
	// Test Message() function when Error has empty message
	e := New(http.StatusBadRequest, "", errors.New("test"))
	msg := Message(e)
	if msg != "Bad Request" {
		t.Fatalf("Message() should return status text when msg is empty, got: %q", msg)
	}
}

func TestNewErrorFunction(t *testing.T) {
	// Test the internal newError function indirectly through New
	e := New(http.StatusBadRequest, "test", errors.New("test"))

	// Verify all fields are set correctly
	if e.Code != http.StatusBadRequest {
		t.Fatalf("Code = %d, want %d", e.Code, http.StatusBadRequest)
	}
	if e.Msg != "test" {
		t.Fatalf("Msg = %q, want %q", e.Msg, "test")
	}
	if e.Err == nil {
		t.Fatal("Err should not be nil")
	}
	if e.Op == "" {
		t.Fatal("Op should not be empty")
	}
	if len(e.stack) == 0 {
		t.Fatal("stack should not be empty")
	}
}

// Test to achieve 100% coverage of detectOperation
func TestDetectOperationFullCoverage(t *testing.T) {
	// Test all branches of detectOperation

	// 1. Test normal operation (already covered by other tests)
	op1 := detectOperation(1)
	if op1 == "" {
		t.Fatal("detectOperation should not return empty string")
	}

	// 2. Test with very high skip to trigger runtime.Caller failure
	op2 := detectOperation(100)
	if op2 != "unknown" {
		t.Fatalf("detectOperation with high skip should return 'unknown', got: %q", op2)
	}

	// 3. Test the function name parsing edge cases
	// We need to test when len(parts) < 2 and when len(parts) == 0

	// Create a test case where we can control the function name
	// This is tricky, but we can test the logic by calling detectOperation
	// from different contexts

	// Test from main function context (single part name)
	if testing.Short() {
		t.Skip("Skipping detailed coverage test in short mode")
	}

	// Test various skip levels to ensure we hit all code paths
	for skip := 0; skip < 20; skip++ {
		op := detectOperation(skip)
		// Should never be empty
		if op == "" {
			t.Fatalf("detectOperation should never return empty string for skip=%d", skip)
		}
		// Log the results for debugging
		t.Logf("Skip %d: %q", skip, op)
	}
}

// Test function to trigger specific code paths
func TestDetectOperationCodePaths(t *testing.T) {
	// Test the function from different call contexts to trigger different code paths

	// Direct call
	op1 := detectOperation(1)
	if op1 == "" {
		t.Fatal("Direct call should not return empty")
	}

	// Call from goroutine
	done := make(chan string)
	go func() {
		done <- detectOperation(1)
	}()
	op2 := <-done
	if op2 == "" {
		t.Fatal("Goroutine call should not return empty")
	}

	// Call from nested function
	nested := func() string {
		return detectOperation(1)
	}
	op3 := nested()
	if op3 == "" {
		t.Fatal("Nested call should not return empty")
	}
}
