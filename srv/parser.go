package srv

import (
	"encoding"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/c3p0-box/utils/erm"
)

// ParseRequest parses HTTP request payload (JSON or form data) and query parameters into the provided target structure
// This function automatically detects the Content-Type and uses the appropriate parsing method for the request body,
// while also parsing URL query parameters regardless of Content-Type
//
// Supported Content-Types:
// - application/json: Uses json.NewDecoder for efficient streaming
// - application/x-www-form-urlencoded: Parses form data using reflection and struct tags
// - multipart/form-data: Parses multipart form data
// - Empty Content-Type: Attempts to detect based on request body
//
// Features:
// - Automatic Content-Type detection and routing
// - Efficient JSON streaming (like json/v2.UnmarshalRead pattern)
// - Form field mapping using `form` struct tags
// - Query parameter mapping using `query` struct tags
// - Type conversion for form and query values (string, int, bool, etc.)
// - Custom type support for types implementing encoding.TextUnmarshaler interface
// - Proper error handling with erm.Error types
// - Resource leak prevention with automatic cleanup
// - Combined parsing: both request body and query parameters in single call
//
// Example usage:
//
//	type UserID [16]byte
//
//	func (u *UserID) UnmarshalText(text []byte) error {
//		if len(text) != 32 { // hex encoded 16 bytes
//			return errors.New("invalid UserID length")
//		}
//		_, err := hex.Decode(u[:], text)
//		return err
//	}
//
//	type CreateUserRequest struct {
//		Name     string `json:"name" form:"name"`
//		Email    string `json:"email" form:"email"`
//		Age      int    `json:"age" form:"age"`
//		UserID   UserID `form:"user_id" query:"user_id"`  // Custom type with TextUnmarshaler
//		Page     int    `query:"page"`
//		Sort     string `query:"sort"`
//		FilterBy string `query:"filter_by"`
//	}
//
//	var req CreateUserRequest
//	if err := ParseRequest(r, &req); err != nil {
//		// Handle error
//		return err
//	}
//	// Use req.Name, req.Email, req.Age (from body)
//	// Use req.UserID (parsed via UnmarshalText), req.Page, req.Sort, req.FilterBy (from query parameters)
func ParseRequest(r *http.Request, target interface{}) erm.Error {
	if r == nil {
		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}

	// Parse query parameters first (always available regardless of Content-Type)
	if err := parseQueryParams(r, target); err != nil {
		return err
	}

	// Determine content type for body parsing
	contentType := r.Header.Get(HeaderContentType)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil && contentType != "" {
		// If Content-Type header is malformed, treat as unsupported
		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}

	// Route to appropriate parser based on content type for body parsing
	switch mediaType {
	case "application/json":
		return parseJSONRequest(r, target)
	case "application/x-www-form-urlencoded":
		return parseFormRequest(r, target)
	case "multipart/form-data":
		return parseMultipartFormRequest(r, target)
	case "":
		// No Content-Type specified, try to detect or default to JSON
		return parseRequestWithDetection(r, target)
	default:
		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}
}

// parseJSONRequest handles JSON payload parsing
func parseJSONRequest(r *http.Request, target interface{}) erm.Error {
	if r.Body == nil {
		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}

	// Ensure body is closed to prevent resource leaks
	defer func(Body io.ReadCloser) {
		rootErr := Body.Close()
		if rootErr != nil {
			slog.With(
				slog.String("name", "req.parseJSONRequest"),
				slog.Any("error", rootErr),
			).Debug("failed to close request body")
		}
	}(r.Body)

	// Check if body has any content
	if r.ContentLength == 0 {
		return nil
	}

	// Create a JSON decoder that reads directly from the request body
	// This streams the JSON input efficiently like json/v2.UnmarshalRead would
	decoder := json.NewDecoder(r.Body)

	// Decode the JSON body into the target structure
	if err := decoder.Decode(target); err != nil {
		slog.With(
			slog.String("name", "req.parseJSONRequest"),
			slog.Any("error", err),
		).Debug("failed to parse JSON request body")

		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}

	return nil
}

// parseFormRequest handles application/x-www-form-urlencoded parsing
func parseFormRequest(r *http.Request, target interface{}) erm.Error {
	if err := r.ParseForm(); err != nil {
		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}

	return mapFormToStruct(r.Form, target)
}

// parseMultipartFormRequest handles multipart/form-data parsing
func parseMultipartFormRequest(r *http.Request, target interface{}) erm.Error {
	// Set max memory for multipart parsing (32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}

	return mapFormToStruct(r.MultipartForm.Value, target)
}

// parseRequestWithDetection attempts to detect content type when not specified
func parseRequestWithDetection(r *http.Request, target interface{}) erm.Error {
	if r.Body == nil {
		return nil // No body to parse - this is fine for GET requests
	}

	// Check if the request has any content to parse
	if r.ContentLength == 0 {
		return nil // No content to parse - this is fine
	}

	// For requests without Content-Type, default to JSON parsing
	// This maintains backward compatibility with existing code
	return parseJSONRequest(r, target)
}

// parseQueryParams parses URL query parameters and maps them to struct fields with `query` tags
func parseQueryParams(r *http.Request, target interface{}) erm.Error {
	if r.URL == nil {
		return nil // No query parameters to parse
	}

	queryValues := r.URL.Query()
	if len(queryValues) == 0 {
		return nil // No query parameters to parse
	}

	return mapQueryToStruct(queryValues, target)
}

// mapQueryToStruct maps query parameters to struct fields using reflection and `query` struct tags
func mapQueryToStruct(values map[string][]string, target interface{}) erm.Error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}

	rv = rv.Elem()
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the query tag, fallback to field name if not present
		queryTag := fieldType.Tag.Get("query")
		if queryTag == "" {
			queryTag = strings.ToLower(fieldType.Name)
		}

		// Skip fields with query:"-"
		if queryTag == "-" {
			continue
		}

		// Get query values for this field
		queryVals, exists := values[queryTag]
		if !exists || len(queryVals) == 0 {
			continue
		}

		// Set the field value based on its type
		if err := setFieldValue(field, queryVals[0]); err != nil {
			slog.With(
				slog.String("name", "req.mapQueryToStruct"),
				slog.String("field", fieldType.Name),
				slog.String("queryTag", queryTag),
				slog.Any("error", err),
			).Debug("failed to set field value from query parameter")

			return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
		}
	}

	return nil
}

// mapFormToStruct maps form values to struct fields using reflection and struct tags
func mapFormToStruct(values map[string][]string, target interface{}) erm.Error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
	}

	rv = rv.Elem()
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the form tag, fallback to field name if not present
		formTag := fieldType.Tag.Get("form")
		if formTag == "" {
			formTag = strings.ToLower(fieldType.Name)
		}

		// Skip fields with form:"-"
		if formTag == "-" {
			continue
		}

		// Get form values for this field
		formValues, exists := values[formTag]
		if !exists || len(formValues) == 0 {
			continue
		}

		// Set the field value based on its type
		if err := setFieldValue(field, formValues[0]); err != nil {
			slog.With(
				slog.String("name", "req.mapFormToStruct"),
				slog.String("field", fieldType.Name),
				slog.String("formTag", formTag),
				slog.Any("error", err),
			).Debug("failed to set field value from form data")

			return erm.NewValidationError(erm.MsgErrorInvalidRequest, erm.NonFieldErrors, "", "")
		}
	}

	return nil
}

// getTextUnmarshaler checks if a reflect.Value implements encoding.TextUnmarshaler
// and returns the unmarshaler interface, or nil if not supported
func getTextUnmarshaler(field reflect.Value) encoding.TextUnmarshaler {
	// For pointer types
	if field.Kind() == reflect.Ptr {
		// Ensure pointer is not nil
		if field.IsNil() && field.CanSet() {
			field.Set(reflect.New(field.Type().Elem()))
		}

		// Check if the pointer value implements TextUnmarshaler
		if !field.IsNil() && field.CanInterface() {
			if unmarshaler, ok := field.Interface().(encoding.TextUnmarshaler); ok {
				return unmarshaler
			}
		}
	} else {
		// For non-pointer types, check if the value directly implements TextUnmarshaler
		if field.CanInterface() {
			if unmarshaler, ok := field.Interface().(encoding.TextUnmarshaler); ok {
				return unmarshaler
			}
		}

		// Check if a pointer to this value implements TextUnmarshaler
		if field.CanAddr() {
			if unmarshaler, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
				return unmarshaler
			}
		}
	}

	return nil
}

// setFieldValue sets a struct field value from a string form value
// Supports basic types (string, int, bool, float) and custom types implementing encoding.TextUnmarshaler
func setFieldValue(field reflect.Value, value string) error {
	// Try to use TextUnmarshaler interface first
	if unmarshaler := getTextUnmarshaler(field); unmarshaler != nil {
		return unmarshaler.UnmarshalText([]byte(value))
	}

	// Fall back to basic type handling
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			return nil // Skip empty values for numeric fields
		}
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == "" {
			return nil
		}
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		if value == "" {
			return nil
		}
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		if value == "" {
			return nil
		}
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	default:
		return erm.Internal("unsupported field type: "+field.Kind().String(), nil)
	}

	return nil
}
