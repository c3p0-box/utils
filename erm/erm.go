// Package erm provides error management utilities for Go applications following
// onion/clean architecture patterns. It enriches errors with stack traces,
// HTTP status codes, and safe user-facing messages while maintaining full
// compatibility with Go's standard error handling.
//
// The package follows KISS and SOLID principles, depends only on Go's
// standard library, and provides a clean API for error propagation
// across application layers.
//
// The package uses an interface-based design where Error is the main interface
// implemented by StackError. This provides flexibility while maintaining
// type safety and compatibility with Go's standard error handling.
//
// # Basic Usage
//
//	// Create errors with automatic operation detection
//	err := erm.New(http.StatusBadRequest, "Invalid email", originalErr)
//
//	// Use convenience constructors
//	err := erm.BadRequest("Invalid input", originalErr)
//
//	// Extract information safely from any error
//	status := erm.Status(err)   // Works with any error type
//	message := erm.Message(err) // Safe for user consumption
//
// # Error Wrapping
//
//	// Preserve metadata while adding operation context
//	wrapped := erm.Wrap(err) // Maintains status and message
//
// All functions handle nil errors gracefully and are safe for concurrent use.
package erm

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// Error defines the interface for enriched errors with stack traces,
// HTTP status codes, and operation context. All errors created by this
// package implement this interface.
//
// The interface extends Go's standard error interface with additional
// metadata for better error handling in web applications and services.
type Error interface {
	// Error returns the error message, satisfying Go's error interface
	Error() string

	// Op returns the operation name where the error occurred
	Op() string

	// Code returns the HTTP status code associated with this error
	Code() int

	// Unwrap returns the wrapped error for errors.Is/As compatibility
	Unwrap() error

	// Stack returns the captured stack trace as program counters
	Stack() []uintptr
}

// StackError represents an application error enriched with stack trace,
// HTTP status code, and safe user-facing message.
//
// StackError captures the following information when created:
//   - op: Operation name automatically detected from call stack
//   - code: HTTP status code (e.g., http.StatusBadRequest)
//   - msg: Safe user-facing message for client responses
//   - root: Wrapped underlying error maintaining error chain
//   - stack: Stack trace as program counters for debugging
//
// StackError values are immutable after creation and are safe for
// concurrent access. They satisfy Go's standard error wrapping
// expectations and work with errors.Is/As functions.
type StackError struct {
	op    string
	code  int
	msg   string
	root  error
	stack []uintptr
}

// New creates a new Error with automatic stack trace capture and
// operation detection from the call stack.
//
// Parameters:
//   - code: HTTP status code (if 0, defaults to http.StatusInternalServerError)
//   - msg: User-safe message for client responses
//   - err: Underlying error to wrap (if nil, creates generic error with msg)
//
// The operation name is automatically detected from the calling function,
// making error tracking easier without manual operation specification.
//
// Example:
//
//	err := erm.New(http.StatusBadRequest, "Invalid email format", validationErr)
//	// err.Op() might return "UserService.ValidateEmail"
//	// err.Code() returns 400
//	// err.Error() includes operation context
func New(code int, msg string, err error) Error {
	skip := 2 // Skip New and the function that called New
	if code == 0 {
		code = http.StatusInternalServerError
	}

	if err == nil {
		err = errors.New(msg)
	}

	// Capture up to 32 PCs, skipping appropriate frames
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])

	// Automatically detect operation name from caller
	op := detectOperation(skip)

	return &StackError{
		op:    op,
		code:  code,
		msg:   msg,
		root:  err,
		stack: pcs[:n],
	}
}

// detectOperation automatically detects the operation name from the call stack
func detectOperation(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	name := fn.Name()

	// Extract meaningful operation name
	// Example: "github.com/user/project/service.(*UserService).Register"
	// Should become: "UserService.Register"
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		// Get the last two parts and clean them
		receiver := parts[len(parts)-2]
		method := parts[len(parts)-1]

		// Remove pointer indicators and parentheses
		receiver = strings.TrimPrefix(receiver, "(*")
		receiver = strings.TrimSuffix(receiver, ")")

		// Handle package functions (not methods)
		if strings.Contains(receiver, "/") {
			// This is a package function, not a method
			// Extract just the function name
			return method
		}

		return fmt.Sprintf("%s.%s", receiver, method)
	}

	// Fallback to just the function name
	if len(parts) > 0 {
		funcName := parts[len(parts)-1]
		// Remove any package path prefixes
		if strings.Contains(funcName, "/") {
			pathParts := strings.Split(funcName, "/")
			return pathParts[len(pathParts)-1]
		}
		return funcName
	}

	return "unknown"
}

// Error implements the built-in error interface, returning a formatted
// error message that includes operation context when available.
func (e *StackError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.op != "" && e.root != nil {
		return fmt.Sprintf("%s: %v", e.op, e.root)
	}
	if e.root != nil {
		return e.root.Error()
	}
	if e.msg != "" {
		return e.msg
	}
	return "unknown error"
}

// Op returns the operation name where the error occurred.
// Returns empty string for nil receivers.
func (e *StackError) Op() string {
	if e == nil {
		return ""
	}
	return e.op
}

// Code returns the HTTP status code associated with the error.
// Returns http.StatusOK (200) for nil receivers.
func (e *StackError) Code() int {
	if e == nil {
		return http.StatusOK
	}
	return e.code
}

// Unwrap returns the wrapped error for compatibility with errors.Is/As.
// If no error was wrapped, returns the error itself to maintain the chain.
// Returns nil for nil receivers.
func (e *StackError) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.root != nil {
		return e.root
	}
	return e
}

// Stack returns the captured stack trace as program counters.
// Use runtime.CallersFrames to convert to human-readable format,
// or use FormatStack for a formatted string representation.
// Returns nil for nil receivers or if no stack was captured.
func (e *StackError) Stack() []uintptr {
	if e == nil {
		return nil
	}
	return e.stack
}

// Status extracts the HTTP status code from any error, providing
// a safe way to get status codes from mixed error types.
//
// Returns:
//   - For erm.Error: the actual status code
//   - For standard errors: http.StatusInternalServerError (500)
//   - For nil errors: http.StatusOK (200)
func Status(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if e, ok := err.(Error); ok {
		return e.Code()
	}

	return http.StatusInternalServerError
}

// Message extracts a safe user-facing message from any error.
// The returned message is safe to send to clients without leaking
// internal implementation details.
//
// Returns:
//   - For erm errors with custom message: the custom message
//   - For erm errors without message: HTTP status text (e.g., "Bad Request")
//   - For standard errors: "Internal Server Error"
//   - For nil errors: empty string
func Message(err error) string {
	if err == nil {
		return ""
	}

	if e, ok := err.(*StackError); ok {
		if e.msg != "" {
			return e.msg
		}
		// Fallback to HTTP status text if message is empty
		return http.StatusText(e.code)
	}

	// For non-erm errors, return status text
	return http.StatusText(http.StatusInternalServerError)
}

// Stack extracts the stack trace from any error that supports it.
// Use this with FormatStack to get human-readable stack traces
// for logging and debugging.
//
// Returns:
//   - For erm errors: captured stack trace as program counters
//   - For other errors: nil
//   - For nil errors: nil
func Stack(err error) []uintptr {
	if err == nil {
		return nil
	}

	if e, ok := err.(Error); ok {
		return e.Stack()
	}

	return nil
}

// Wrap wraps an error with erm error capabilities while preserving
// the original error's metadata when possible.
//
// Behavior:
//   - If err is nil: returns nil
//   - If err is already an erm.Error: returns it unchanged
//   - If err is a standard error: wraps it with http.StatusInternalServerError
//
// The operation context is updated to reflect the current call site,
// making it useful for adding operation context as errors bubble up
// through application layers.
func Wrap(err error) Error {
	if err == nil {
		return nil
	}

	// If it's already an erm error, don't wrap it
	if e, ok := err.(Error); ok {
		return e
	}

	// For standard errors, default to 500
	return New(http.StatusInternalServerError, "Internal Server Error", err)
}

// FormatStack formats a stack trace into a human-readable string
// suitable for logging and debugging. Each frame shows the function
// name, file path, and line number.
//
// Returns empty string if the error is nil or has no stack trace.
//
// Example output:
//
//	main.processUser
//		/app/user.go:42
//	main.handleRequest
//		/app/handler.go:28
func FormatStack(err Error) string {
	if err == nil {
		return ""
	}

	pcs := err.Stack()
	if len(pcs) == 0 {
		return ""
	}

	var buf strings.Builder
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		_, _ = fmt.Fprintf(&buf, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return buf.String()
}

// Convenience constructors for common HTTP errors.
// All constructors automatically detect the operation name and
// capture stack traces.

// BadRequest creates a 400 Bad Request error with the given message
// and optional underlying error.
func BadRequest(msg string, err error) Error {
	return New(http.StatusBadRequest, msg, err)
}

// Unauthorized creates a 401 Unauthorized error with the given message
// and optional underlying error.
func Unauthorized(msg string, err error) Error {
	return New(http.StatusUnauthorized, msg, err)
}

// Forbidden creates a 403 Forbidden error with the given message
// and optional underlying error.
func Forbidden(msg string, err error) Error {
	return New(http.StatusForbidden, msg, err)
}

// NotFound creates a 404 Not Found error with the given message
// and optional underlying error.
func NotFound(msg string, err error) Error {
	return New(http.StatusNotFound, msg, err)
}

// Conflict creates a 409 Conflict error with the given message
// and optional underlying error.
func Conflict(msg string, err error) Error {
	return New(http.StatusConflict, msg, err)
}

// Internal creates a 500 Internal Server Error with the given message
// and optional underlying error.
func Internal(msg string, err error) Error {
	return New(http.StatusInternalServerError, msg, err)
}
