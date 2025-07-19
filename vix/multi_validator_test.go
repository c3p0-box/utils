package vix

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/c3p0-box/utils/erm"
)

// setupLocalizer sets up the default localizer for tests
// This is a simplified version that uses the erm package's test helper
func setupLocalizer() {
	erm.SetupTestLocalizer()
}

// =============================================================================
// ValidationOrchestrator Core Tests
// =============================================================================

func TestValidationOrchestrator_Is(t *testing.T) {
	tests := []struct {
		name        string
		validators  []Validator
		expectValid bool
		expectError bool
	}{
		{
			name: "valid single string validation",
			validators: []Validator{
				String("test@example.com", "email").Required().Email(),
			},
			expectValid: true,
			expectError: false,
		},
		{
			name: "invalid single string validation",
			validators: []Validator{
				String("", "email").Required(),
			},
			expectValid: false,
			expectError: true,
		},
		{
			name: "valid multiple field validation",
			validators: []Validator{
				String("test@example.com", "email").Required().Email(),
				Int(25, "age").Required().Min(18),
			},
			expectValid: true,
			expectError: false,
		},
		{
			name: "invalid multiple field validation",
			validators: []Validator{
				String("", "email").Required(),
				Int(16, "age").Required().Min(18),
			},
			expectValid: false,
			expectError: true,
		},
		{
			name: "mixed valid and invalid fields",
			validators: []Validator{
				String("test@example.com", "email").Required().Email(),
				String("", "name").Required(),
				Int(25, "age").Required().Min(18),
			},
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orchestrator := Is(tt.validators...)

			if orchestrator.Valid() != tt.expectValid {
				t.Errorf("expected Valid() = %v, got %v", tt.expectValid, orchestrator.Valid())
			}

			hasError := orchestrator.Error() != nil
			if hasError != tt.expectError {
				t.Errorf("expected error presence = %v, got %v", tt.expectError, hasError)
			}
		})
	}
}

func TestValidationOrchestrator_IsValid(t *testing.T) {
	orchestrator := Is(
		String("test@example.com", "email").Required().Email(),
		String("", "name").Required(),
		Int(25, "age").Required().Min(18),
	)

	if !orchestrator.IsValid("email") {
		t.Error("email should be valid")
	}
	if orchestrator.IsValid("name") {
		t.Error("name should be invalid")
	}
	if !orchestrator.IsValid("age") {
		t.Error("age should be valid")
	}
	if !orchestrator.IsValid("nonexistent") {
		t.Error("nonexistent field should be valid (not tested)")
	}
}

func TestValidationOrchestrator_ToError(t *testing.T) {
	setupLocalizer() // Set up localizer for tests

	t.Run("valid orchestrator returns nil", func(t *testing.T) {
		orchestrator := Is(
			String("test@example.com", "email").Required().Email(),
			Int(25, "age").Required().Min(18),
		)

		errorMap := orchestrator.ErrMap()
		if errorMap != nil {
			t.Errorf("expected nil error map but got: %v", errorMap)
		}
	})

	t.Run("invalid orchestrator returns error map", func(t *testing.T) {
		orchestrator := Is(
			String("", "email").Required(),
			Int(16, "age").Required().Min(18),
		)

		errorMap := orchestrator.ErrMap()
		if errorMap == nil {
			t.Fatal("expected error map but got nil")
		}
		if _, exists := errorMap["email"]; !exists {
			t.Error("error map should contain 'email' key")
		}
		if _, exists := errorMap["age"]; !exists {
			t.Error("error map should contain 'age' key")
		}
		if !strings.Contains(errorMap["email"][0], "email is required") {
			t.Errorf("expected 'email is required' message, got: %v", errorMap["email"])
		}
		if !strings.Contains(errorMap["age"][0], "age must be at least 18") {
			t.Errorf("expected 'age must be at least 18' message, got: %v", errorMap["age"])
		}
	})
}

func TestValidationOrchestrator_ToJSON(t *testing.T) {
	t.Run("valid orchestrator returns empty JSON", func(t *testing.T) {
		orchestrator := Is(
			String("test@example.com", "email").Required().Email(),
		)

		jsonBytes, err := orchestrator.ToJSON()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(jsonBytes) != "{}" {
			t.Errorf("expected '{}' but got: %s", string(jsonBytes))
		}
	})

	t.Run("invalid orchestrator returns error JSON", func(t *testing.T) {
		orchestrator := Is(
			String("", "email").Required(),
			Int(16, "age").Required().Min(18),
		)

		jsonBytes, err := orchestrator.ToJSON()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		var errorMap map[string][]string
		err = json.Unmarshal(jsonBytes, &errorMap)
		if err != nil {
			t.Errorf("failed to unmarshal JSON: %v", err)
		}
		if _, exists := errorMap["email"]; !exists {
			t.Error("JSON should contain 'email' key")
		}
		if _, exists := errorMap["age"]; !exists {
			t.Error("JSON should contain 'age' key")
		}
	})
}

func TestValidationOrchestrator_In(t *testing.T) {
	addressValidation := Is(
		String("", "name").Required(),
		String("123 Main St", "street").Required(),
	)

	orchestrator := Is(String("Bob", "name").Required()).
		In("address", addressValidation)

	if orchestrator.Valid() {
		t.Error("orchestrator should be invalid")
	}
	if !orchestrator.IsValid("name") {
		t.Error("name should be valid")
	}
	if orchestrator.IsValid("address.name") {
		t.Error("address.name should be invalid")
	}
	if !orchestrator.IsValid("address.street") {
		t.Error("address.street should be valid")
	}

	errorMap := orchestrator.ErrMap()
	if _, exists := errorMap["address.name"]; !exists {
		t.Error("error map should contain 'address.name' key")
	}
	if _, exists := errorMap["address.street"]; exists {
		t.Error("error map should not contain 'address.street' key")
	}
}

func TestValidationOrchestrator_InRow(t *testing.T) {
	addresses := []struct {
		name   string
		street string
	}{
		{"", "123 Main St"},
		{"Home", ""},
	}

	orchestrator := Is(String("Bob", "name").Required())

	for i, addr := range addresses {
		addressValidation := Is(
			String(addr.name, "name").Required(),
			String(addr.street, "street").Required(),
		)
		orchestrator.InRow("addresses", i, addressValidation)
	}

	if orchestrator.Valid() {
		t.Error("orchestrator should be invalid")
	}
	if !orchestrator.IsValid("name") {
		t.Error("name should be valid")
	}
	if orchestrator.IsValid("addresses[0].name") {
		t.Error("addresses[0].name should be invalid")
	}
	if !orchestrator.IsValid("addresses[0].street") {
		t.Error("addresses[0].street should be valid")
	}
	if !orchestrator.IsValid("addresses[1].name") {
		t.Error("addresses[1].name should be valid")
	}
	if orchestrator.IsValid("addresses[1].street") {
		t.Error("addresses[1].street should be invalid")
	}

	errorMap := orchestrator.ErrMap()
	if _, exists := errorMap["addresses[0].name"]; !exists {
		t.Error("error map should contain 'addresses[0].name' key")
	}
	if _, exists := errorMap["addresses[1].street"]; !exists {
		t.Error("error map should contain 'addresses[1].street' key")
	}
	if _, exists := errorMap["addresses[0].street"]; exists {
		t.Error("error map should not contain 'addresses[0].street' key")
	}
	if _, exists := errorMap["addresses[1].name"]; exists {
		t.Error("error map should not contain 'addresses[1].name' key")
	}
}

func TestValidationOrchestrator_FieldNames(t *testing.T) {
	orchestrator := Is(
		String("test", "email").Required(),
		Int(25, "age").Required(),
	)

	fieldNames := orchestrator.FieldNames()
	hasEmail := false
	hasAge := false
	for _, name := range fieldNames {
		if name == "email" {
			hasEmail = true
		}
		if name == "age" {
			hasAge = true
		}
	}
	if !hasEmail {
		t.Error("field names should contain 'email'")
	}
	if !hasAge {
		t.Error("field names should contain 'age'")
	}
	if len(fieldNames) != 2 {
		t.Errorf("expected 2 field names, got %d", len(fieldNames))
	}
}

func TestValidationOrchestrator_GetFieldResult(t *testing.T) {
	orchestrator := Is(
		String("test@example.com", "email").Required().Email(),
		String("", "name").Required(),
	)

	emailResult := orchestrator.GetFieldResult("email")
	if emailResult == nil {
		t.Error("email result should not be nil")
	}
	if emailResult != nil && !emailResult.Valid() {
		t.Error("email result should be valid")
	}

	nameResult := orchestrator.GetFieldResult("name")
	if nameResult == nil {
		t.Error("name result should not be nil")
	}
	if nameResult != nil && nameResult.Valid() {
		t.Error("name result should be invalid")
	}

	nonExistentResult := orchestrator.GetFieldResult("nonexistent")
	if nonExistentResult != nil {
		t.Error("nonexistent result should be nil")
	}
}

func TestValidationOrchestrator_String(t *testing.T) {
	setupLocalizer() // Set up localizer for tests

	t.Run("valid orchestrator", func(t *testing.T) {
		orchestrator := Is(
			String("test@example.com", "email").Required().Email(),
		)

		if orchestrator.String() != "ValidationOrchestrator: Valid" {
			t.Errorf("expected 'ValidationOrchestrator: Valid', got: %s", orchestrator.String())
		}
	})

	t.Run("invalid orchestrator", func(t *testing.T) {
		orchestrator := Is(
			String("", "email").Required(),
			Int(16, "age").Required().Min(18),
		)

		result := orchestrator.String()
		if !strings.Contains(result, "email is required") {
			t.Errorf("expected 'email is required' in result: %s", result)
		}
		if !strings.Contains(result, "age must be at least 18") {
			t.Errorf("expected 'age must be at least 18' in result: %s", result)
		}
		if !strings.Contains(result, ", ") {
			t.Errorf("expected ', ' separator in result: %s", result)
		}
	})
}

func TestValidationOrchestrator_Error(t *testing.T) {
	setupLocalizer() // Set up localizer for tests

	orchestrator := Is(
		String("", "email").Required(),
		Int(16, "age").Required().Min(18),
	)

	// Test Error() returns a container error with all child errors
	err := orchestrator.Error()
	if err == nil {
		t.Error("expected error to be returned")
	}

	// Test that the error is an erm.Error with child errors
	if ermErr, ok := err.(erm.Error); ok {
		childErrors := ermErr.AllErrors()
		if len(childErrors) != 2 {
			t.Errorf("expected 2 child errors, got %d", len(childErrors))
		}

		errorMessages := make([]string, len(childErrors))
		for i, err := range childErrors {
			errorMessages[i] = err.Error()
		}

		hasEmailError := false
		hasAgeError := false
		for _, msg := range errorMessages {
			if strings.Contains(msg, "email is required") {
				hasEmailError = true
			}
			if strings.Contains(msg, "age must be at least 18") {
				hasAgeError = true
			}
		}
		if !hasEmailError {
			t.Error("expected 'email is required' error message")
		}
		if !hasAgeError {
			t.Error("expected 'age must be at least 18' error message")
		}
	} else {
		t.Error("expected error to be an erm.Error")
	}
}

func TestValidationOrchestrator_ComplexExample(t *testing.T) {
	// This test mimics the valgo example from the user's request
	orchestrator := Is(
		String("Bob", "full_name").Not().Required().LengthBetween(4, 20),
		Int(17, "age").GreaterThan(18),
		String("singl", "status").In("married", "single"),
	)

	if orchestrator.Valid() {
		t.Error("orchestrator should be invalid")
	}

	errorMap := orchestrator.ErrMap()
	if _, exists := errorMap["full_name"]; !exists {
		t.Error("error map should contain 'full_name' key")
	}
	if _, exists := errorMap["age"]; !exists {
		t.Error("error map should contain 'age' key")
	}
	if _, exists := errorMap["status"]; !exists {
		t.Error("error map should contain 'status' key")
	}

	jsonBytes, err := orchestrator.ToJSON()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var jsonMap map[string][]string
	err = json.Unmarshal(jsonBytes, &jsonMap)
	if err != nil {
		t.Errorf("failed to unmarshal JSON: %v", err)
	}
	if _, exists := jsonMap["full_name"]; !exists {
		t.Error("JSON should contain 'full_name' key")
	}
	if _, exists := jsonMap["age"]; !exists {
		t.Error("JSON should contain 'age' key")
	}
	if _, exists := jsonMap["status"]; !exists {
		t.Error("JSON should contain 'status' key")
	}
}

func TestValidationOrchestrator_NestedStructExample(t *testing.T) {
	// This test mimics the nested structure example from the user's request
	type Address struct {
		Name   string
		Street string
	}

	type Person struct {
		Name    string
		Address Address
	}

	p := Person{"Bob", Address{"", "1600 Amphitheatre Pkwy"}}

	orchestrator := Is(String(p.Name, "name").LengthBetween(4, 20)).
		In("address", Is(
			String(p.Address.Name, "name").Required(),
			String(p.Address.Street, "street").Required(),
		))

	if orchestrator.Valid() {
		t.Error("orchestrator should be invalid")
	}

	errorMap := orchestrator.ErrMap()
	if _, exists := errorMap["address.name"]; !exists {
		t.Error("error map should contain 'address.name' key")
	}
	if _, exists := errorMap["name"]; !exists {
		t.Error("error map should contain 'name' key")
	}

	jsonBytes, err := orchestrator.ToJSON()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var jsonMap map[string][]string
	err = json.Unmarshal(jsonBytes, &jsonMap)
	if err != nil {
		t.Errorf("failed to unmarshal JSON: %v", err)
	}
	if _, exists := jsonMap["address.name"]; !exists {
		t.Error("JSON should contain 'address.name' key")
	}
	if _, exists := jsonMap["name"]; !exists {
		t.Error("JSON should contain 'name' key")
	}
}

func TestValidationOrchestrator_ArrayExample(t *testing.T) {
	// This test mimics the array example from the user's request
	type Address struct {
		Name   string
		Street string
	}

	type Person struct {
		Name      string
		Addresses []Address
	}

	p := Person{
		"Bob",
		[]Address{
			{"", "1600 Amphitheatre Pkwy"},
			{"Home", ""},
		},
	}

	orchestrator := Is(String(p.Name, "name").LengthBetween(4, 20))

	for i, a := range p.Addresses {
		orchestrator.InRow("addresses", i, Is(
			String(a.Name, "name").Required(),
			String(a.Street, "street").Required(),
		))
	}

	if orchestrator.Valid() {
		t.Error("orchestrator should be invalid")
	}

	errorMap := orchestrator.ErrMap()
	if _, exists := errorMap["addresses[0].name"]; !exists {
		t.Error("error map should contain 'addresses[0].name' key")
	}
	if _, exists := errorMap["addresses[1].street"]; !exists {
		t.Error("error map should contain 'addresses[1].street' key")
	}
	if _, exists := errorMap["name"]; !exists {
		t.Error("error map should contain 'name' key")
	}

	jsonBytes, err := orchestrator.ToJSON()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var jsonMap map[string][]string
	err = json.Unmarshal(jsonBytes, &jsonMap)
	if err != nil {
		t.Errorf("failed to unmarshal JSON: %v", err)
	}
	if _, exists := jsonMap["addresses[0].name"]; !exists {
		t.Error("JSON should contain 'addresses[0].name' key")
	}
	if _, exists := jsonMap["addresses[1].street"]; !exists {
		t.Error("JSON should contain 'addresses[1].street' key")
	}
	if _, exists := jsonMap["name"]; !exists {
		t.Error("JSON should contain 'name' key")
	}
}

func TestValidationOrchestrator_NumericTypes(t *testing.T) {
	orchestrator := Is(
		Int(10, "int_val").Required(),
		Int8(5, "int8_val").Required(),
		Int16(100, "int16_val").Required(),
		Int32(1000, "int32_val").Required(),
		Int64(10000, "int64_val").Required(),
		Uint(10, "uint_val").Required(),
		Uint8(5, "uint8_val").Required(),
		Uint16(100, "uint16_val").Required(),
		Uint32(1000, "uint32_val").Required(),
		Uint64(10000, "uint64_val").Required(),
		Float32(3.14, "float32_val").Required(),
		Float64(3.14159, "float64_val").Required(),
	)

	if !orchestrator.Valid() {
		t.Error("orchestrator should be valid")
	}
	if orchestrator.ErrMap() != nil {
		t.Error("error map should be nil")
	}

	fieldNames := orchestrator.FieldNames()
	if len(fieldNames) != 12 {
		t.Errorf("expected 12 field names, got %d", len(fieldNames))
	}
}

func TestValidationOrchestrator_EmptyOrchestrator(t *testing.T) {
	orchestrator := V()

	if !orchestrator.Valid() {
		t.Error("empty orchestrator should be valid")
	}
	if orchestrator.Error() != nil {
		t.Error("empty orchestrator should have no error")
	}
	if orchestrator.ErrMap() != nil {
		t.Error("empty orchestrator should have nil error map")
	}
	if len(orchestrator.FieldNames()) != 0 {
		t.Error("empty orchestrator should have no field names")
	}
	if orchestrator.String() != "ValidationOrchestrator: Valid" {
		t.Errorf("expected 'ValidationOrchestrator: Valid', got: %s", orchestrator.String())
	}
}

func TestValidationOrchestrator_NewValidationOrchestrator(t *testing.T) {
	orchestrator := NewValidationOrchestrator()
	if orchestrator == nil {
		t.Error("orchestrator should not be nil")
	}
	if orchestrator.fieldResults == nil {
		t.Error("fieldResults should not be nil")
	}
	if orchestrator.fieldOrder == nil {
		t.Error("fieldOrder should not be nil")
	}
	// Note: locale field removed - now handled globally through erm.SetLocalizer()
}

func TestValidationOrchestrator_FieldOrder(t *testing.T) {
	orchestrator := Is(
		String("test", "email").Required(),
		Int(25, "age").Required(),
		String("Bob", "name").Required(),
	)

	fieldNames := orchestrator.FieldNames()
	expected := []string{"email", "age", "name"}
	if len(fieldNames) != len(expected) {
		t.Errorf("expected %d field names, got %d", len(expected), len(fieldNames))
	}
	for i, name := range expected {
		if i >= len(fieldNames) || fieldNames[i] != name {
			t.Errorf("expected field name %s at index %d, got %s", name, i, fieldNames[i])
		}
	}
}

func TestValidationOrchestrator_OverwriteField(t *testing.T) {
	orchestrator := Is(String("test", "email").Required()).
		Is(String("", "email").Required()) // Overwrite the same field

	if orchestrator.Valid() {
		t.Error("orchestrator should be invalid")
	}
	if orchestrator.IsValid("email") {
		t.Error("email should be invalid")
	}

	fieldNames := orchestrator.FieldNames()
	if len(fieldNames) != 1 {
		t.Errorf("expected 1 field, got %d", len(fieldNames))
	}
	if fieldNames[0] != "email" {
		t.Errorf("expected field name 'email', got %s", fieldNames[0])
	}
}

// TestPackageLevelFunctions verifies that the new package-level functions work correctly
func TestPackageLevelFunctions(t *testing.T) {
	t.Run("package-level Is() function", func(t *testing.T) {
		// Test vix.Is() instead of vix.V().Is()
		orchestrator := Is(
			String("test@example.com", "email").Required().Email(),
			Int(25, "age").Required().Min(18),
		)

		if !orchestrator.Valid() {
			t.Error("orchestrator should be valid")
		}
		if orchestrator.ErrMap() != nil {
			t.Error("error map should be nil")
		}
	})

	t.Run("package-level In() function", func(t *testing.T) {
		// Test vix.In() instead of vix.V().In()
		orchestrator := In("address", Is(
			String("123 Main St", "street").Required(),
			String("New York", "city").Required(),
		))

		if !orchestrator.Valid() {
			t.Error("orchestrator should be valid")
		}
		if !orchestrator.IsValid("address.street") {
			t.Error("address.street should be valid")
		}
		if !orchestrator.IsValid("address.city") {
			t.Error("address.city should be valid")
		}
	})

	t.Run("package-level InRow() function", func(t *testing.T) {
		// Test vix.InRow() instead of vix.V().InRow()
		orchestrator := InRow("items", 0, Is(
			String("Product A", "name").Required(),
			Int(100, "price").Required().Min(1),
		))

		if !orchestrator.Valid() {
			t.Error("orchestrator should be valid")
		}
		if !orchestrator.IsValid("items[0].name") {
			t.Error("items[0].name should be valid")
		}
		if !orchestrator.IsValid("items[0].price") {
			t.Error("items[0].price should be valid")
		}
	})

	t.Run("package-level functions with errors", func(t *testing.T) {
		// Test error handling with package-level functions
		orchestrator := Is(
			String("", "email").Required(),
			Int(16, "age").Required().Min(18),
		)

		if orchestrator.Valid() {
			t.Error("orchestrator should be invalid")
		}

		errorMap := orchestrator.ErrMap()
		if errorMap == nil {
			t.Error("error map should not be nil")
		}
		if _, exists := errorMap["email"]; !exists {
			t.Error("error map should contain 'email' key")
		}
		if _, exists := errorMap["age"]; !exists {
			t.Error("error map should contain 'age' key")
		}
	})
}

func TestValidationOrchestrator_LocalizedErrMap(t *testing.T) {
	// Test valid orchestrator returns nil
	validOrchestrator := Is(
		String("test@example.com", "email").Required().Email(),
		String("ValidPassword123", "password").Required().MinLength(8),
	)

	localizedMap := validOrchestrator.LocalizedErrMap(nil)
	if localizedMap != nil {
		t.Fatalf("Valid orchestrator should return nil localized error map, got: %v", localizedMap)
	}

	// Test invalid orchestrator returns proper localized error map
	invalidOrchestrator := Is(
		String("", "email").Required().Email(),
		String("123", "password").Required().MinLength(8),
	)

	localizedMap = invalidOrchestrator.LocalizedErrMap(nil)
	if localizedMap == nil {
		t.Fatal("Invalid orchestrator should return a localized error map")
	}

	// Should have errors for both fields
	if _, exists := localizedMap["email"]; !exists {
		t.Fatal("Localized error map should contain email field errors")
	}
	if _, exists := localizedMap["password"]; !exists {
		t.Fatal("Localized error map should contain password field errors")
	}
}
