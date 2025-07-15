// Package erm provides error management utilities for Go applications following
// onion/clean architecture patterns. It enriches errors with stack traces,
// HTTP status codes, and safe user-facing messages while maintaining
// compatibility with Go's standard error handling.
//
// The package follows KISS and SOLID principles, depends only on Go's
// standard library, and provides a clean API for error propagation
// across application layers.
package erm

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// Error represents an application error enriched with
// a stack trace, HTTP status code, and safe user-facing message.
//
//   - Code  holds the HTTP status that best describes the error
//     (e.g. http.StatusBadRequest).
//   - Msg   is a short, safe string that can be sent back to clients.
//   - Op    describes the operation where the error occurred, automatically
//     detected from the call stack.
//   - Err   wraps the underlying error so that errors.Is/As still work.
//   - stack stores program counters captured once when the error is
//     created; callers may format it with runtime.CallersFrames.
//
// Error values are immutable and satisfy Go's standard error wrapping
// expectations.
type Error struct {
	Op    string
	Code  int
	Msg   string
	Err   error
	stack []uintptr
}

// New creates a new *Error capturing a stack trace and automatically
// detecting the operation name from the call stack.
// If code is 0, it defaults to http.StatusInternalServerError.
// If err is nil, a generic error is created with the provided message.
func New(code int, msg string, err error) *Error {
	return newError(code, msg, err, 3)
}

// newError is the core error creation function that handles stack capture and operation detection.
// The skip parameter determines how many stack frames to skip for operation detection.
func newError(code int, msg string, err error, skip int) *Error {
	if code == 0 {
		code = http.StatusInternalServerError
	}

	if err == nil {
		err = errors.New(msg)
	}

	// Capture up to 32 PCs, skipping appropriate frames
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip-1, pcs[:])

	// Automatically detect operation name from caller
	op := detectOperation(skip)

	return &Error{
		Op:    op,
		Code:  code,
		Msg:   msg,
		Err:   err,
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

		return fmt.Sprintf("%s.%s", receiver, method)
	}

	// Fallback to just the function name
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "unknown"
}

// Error implements the built-in error interface.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Op != "" && e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Msg != "" {
		return e.Msg
	}
	return "unknown error"
}

// Unwrap makes the error compatible with errors.Is/As.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Stack returns the captured stack trace PCs (may be nil).
func (e *Error) Stack() []uintptr {
	if e == nil {
		return nil
	}
	return e.stack
}

// Status returns the HTTP status code associated with any error.
// If the provided error does not carry a status, it returns 500.
func Status(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return http.StatusInternalServerError
}

// Message returns a safe user-facing message for any error.
// If the error does not carry one, http.StatusText(Status(err)) is used.
func Message(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) && e.Msg != "" {
		return e.Msg
	}
	return http.StatusText(Status(err))
}

// Stack returns the stack trace of any error created with this package.
// It returns nil if the error has no trace.
func Stack(err error) []uintptr {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Stack()
	}
	return nil
}

// FormatStack formats the stack trace for logging or debugging.
// It returns a human-readable string representation of the stack.
func FormatStack(err error) string {
	pcs := Stack(err)
	if len(pcs) == 0 {
		return ""
	}

	var buf strings.Builder
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&buf, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return buf.String()
}

// Wrap is a convenience helper that keeps the original message & code
// but adds a new operation context from the current call stack.
func Wrap(err error) error {
	if err == nil {
		return nil
	}

	// Preserve existing metadata if present
	var e *Error
	if errors.As(err, &e) {
		return New(e.Code, e.Msg, err)
	}

	return New(http.StatusInternalServerError, "", err)
}

// Convenience constructors for common HTTP errors

// BadRequest creates a 400 Bad Request error.
func BadRequest(msg string, err error) *Error {
	return newError(http.StatusBadRequest, msg, err, 3)
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(msg string, err error) *Error {
	return newError(http.StatusUnauthorized, msg, err, 3)
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(msg string, err error) *Error {
	return newError(http.StatusForbidden, msg, err, 3)
}

// NotFound creates a 404 Not Found error.
func NotFound(msg string, err error) *Error {
	return newError(http.StatusNotFound, msg, err, 3)
}

// Conflict creates a 409 Conflict error.
func Conflict(msg string, err error) *Error {
	return newError(http.StatusConflict, msg, err, 3)
}

// Internal creates a 500 Internal Server Error.
func Internal(msg string, err error) *Error {
	return newError(http.StatusInternalServerError, msg, err, 3)
}
