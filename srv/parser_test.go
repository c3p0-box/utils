package srv

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Test structures for various parsing scenarios
type UserRequest struct {
	Name     string  `json:"name" form:"name"`
	Email    string  `json:"email" form:"email"`
	Age      int     `json:"age" form:"age"`
	Active   bool    `json:"active" form:"active"`
	Score    float64 `json:"score" form:"score"`
	Page     int     `query:"page"`
	Sort     string  `query:"sort"`
	FilterBy string  `query:"filter_by"`
	Limit    int     `query:"limit"`
	Debug    bool    `query:"debug"`
	Rating   float64 `query:"rating"`
}

type QueryOnlyRequest struct {
	Page      int     `query:"page"`
	Sort      string  `query:"sort"`
	Limit     int     `query:"limit"`
	Active    bool    `query:"active"`
	MinScore  float64 `query:"min_score"`
	SkipField string  `query:"-"`
	NoTag     string
}

type FormOnlyRequest struct {
	Username string `form:"username"`
	Password string `form:"password"`
	Remember bool   `form:"remember"`
}

// Custom types implementing encoding.TextUnmarshaler for testing

// UserID represents a 16-byte user identifier
type UserID [16]byte

func (u *UserID) UnmarshalText(text []byte) error {
	if len(text) != 32 { // hex encoded 16 bytes = 32 characters
		return fmt.Errorf("invalid UserID length: expected 32, got %d", len(text))
	}
	_, err := hex.Decode(u[:], text)
	return err
}

func (u UserID) String() string {
	return hex.EncodeToString(u[:])
}

// CustomTime represents a custom time format
type CustomTime time.Time

func (ct *CustomTime) UnmarshalText(text []byte) error {
	t, err := time.Parse("2006-01-02", string(text))
	if err != nil {
		return err
	}
	*ct = CustomTime(t)
	return nil
}

func (ct CustomTime) String() string {
	return time.Time(ct).Format("2006-01-02")
}

// CustomInt wraps an int with custom parsing logic
type CustomInt int

func (ci *CustomInt) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "zero" {
		*ci = 0
		return nil
	}
	if s == "one" {
		*ci = 1
		return nil
	}
	return fmt.Errorf("unsupported CustomInt value: %s", s)
}

// PointerReceiver demonstrates a type that implements TextUnmarshaler only on pointer receiver
type PointerReceiver string

func (pr *PointerReceiver) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "" {
		return errors.New("empty string not allowed")
	}
	*pr = PointerReceiver("parsed:" + s)
	return nil
}

// TestRequestWithCustomTypes includes various custom types for comprehensive testing
type TestRequestWithCustomTypes struct {
	// Form/Query fields with custom types
	ID          UserID          `form:"id" query:"id"`
	CreatedAt   CustomTime      `form:"created_at" query:"created_at"`
	Priority    CustomInt       `form:"priority" query:"priority"`
	PtrReceiver PointerReceiver `form:"ptr_receiver" query:"ptr_receiver"`

	// Pointer versions
	IDPtr        *UserID     `form:"id_ptr" query:"id_ptr"`
	CreatedAtPtr *CustomTime `form:"created_at_ptr" query:"created_at_ptr"`
	PriorityPtr  *CustomInt  `form:"priority_ptr" query:"priority_ptr"`

	// Regular fields for comparison
	Name  string `form:"name" query:"name"`
	Count int    `form:"count" query:"count"`
}

func TestParseRequest_QueryParams_Basic(t *testing.T) {
	tests := []struct {
		name        string
		queryParams string
		want        QueryOnlyRequest
		wantErr     bool
	}{
		{
			name:        "Valid query parameters",
			queryParams: "page=2&sort=name&limit=50&active=true&min_score=85.5",
			want: QueryOnlyRequest{
				Page:     2,
				Sort:     "name",
				Limit:    50,
				Active:   true,
				MinScore: 85.5,
			},
			wantErr: false,
		},
		{
			name:        "Missing some parameters",
			queryParams: "page=1&sort=email",
			want: QueryOnlyRequest{
				Page: 1,
				Sort: "email",
			},
			wantErr: false,
		},
		{
			name:        "Empty values",
			queryParams: "page=&sort=&limit=",
			want:        QueryOnlyRequest{},
			wantErr:     false,
		},
		{
			name:        "No query parameters",
			queryParams: "",
			want:        QueryOnlyRequest{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameters
			req := httptest.NewRequest("GET", "http://example.com/test?"+tt.queryParams, nil)

			var result QueryOnlyRequest
			err := ParseRequest(req, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.Page != tt.want.Page {
					t.Errorf("Page = %v, want %v", result.Page, tt.want.Page)
				}
				if result.Sort != tt.want.Sort {
					t.Errorf("Sort = %v, want %v", result.Sort, tt.want.Sort)
				}
				if result.Limit != tt.want.Limit {
					t.Errorf("Limit = %v, want %v", result.Limit, tt.want.Limit)
				}
				if result.Active != tt.want.Active {
					t.Errorf("Active = %v, want %v", result.Active, tt.want.Active)
				}
				if result.MinScore != tt.want.MinScore {
					t.Errorf("MinScore = %v, want %v", result.MinScore, tt.want.MinScore)
				}
				// SkipField should always be empty (query:"-")
				if result.SkipField != "" {
					t.Errorf("SkipField should be empty but got %v", result.SkipField)
				}
				// NoTag should use lowercase field name
				if result.NoTag != "" {
					t.Errorf("NoTag should be empty without query param")
				}
			}
		})
	}
}

func TestParseRequest_QueryParams_InvalidTypes(t *testing.T) {
	tests := []struct {
		name        string
		queryParams string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Invalid integer",
			queryParams: "page=invalid",
			wantErr:     true,
			errMsg:      "invalid request",
		},
		{
			name:        "Invalid boolean",
			queryParams: "active=maybe",
			wantErr:     true,
			errMsg:      "invalid request",
		},
		{
			name:        "Invalid float",
			queryParams: "min_score=not_a_number",
			wantErr:     true,
			errMsg:      "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/test?"+tt.queryParams, nil)

			var result QueryOnlyRequest
			err := ParseRequest(req, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message should contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestParseRequest_JSON_WithQueryParams(t *testing.T) {
	// Test combining JSON body parsing with query parameters
	jsonData := map[string]interface{}{
		"name":   "John Doe",
		"email":  "john@example.com",
		"age":    30,
		"active": true,
		"score":  95.5,
	}

	jsonBytes, _ := json.Marshal(jsonData)
	req := httptest.NewRequest("POST", "http://example.com/users?page=2&sort=name&filter_by=active&limit=10&debug=true&rating=4.5", bytes.NewReader(jsonBytes))
	req.Header.Set(HeaderContentType, MIMEApplicationJSON)

	var result UserRequest
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	// Check body data
	if result.Name != "John Doe" {
		t.Errorf("Name = %v, want John Doe", result.Name)
	}
	if result.Email != "john@example.com" {
		t.Errorf("Email = %v, want john@example.com", result.Email)
	}
	if result.Age != 30 {
		t.Errorf("Age = %v, want 30", result.Age)
	}
	if result.Active != true {
		t.Errorf("Active = %v, want true", result.Active)
	}
	if result.Score != 95.5 {
		t.Errorf("Score = %v, want 95.5", result.Score)
	}

	// Check query parameters
	if result.Page != 2 {
		t.Errorf("Page = %v, want 2", result.Page)
	}
	if result.Sort != "name" {
		t.Errorf("Sort = %v, want name", result.Sort)
	}
	if result.FilterBy != "active" {
		t.Errorf("FilterBy = %v, want active", result.FilterBy)
	}
	if result.Limit != 10 {
		t.Errorf("Limit = %v, want 10", result.Limit)
	}
	if result.Debug != true {
		t.Errorf("Debug = %v, want true", result.Debug)
	}
	if result.Rating != 4.5 {
		t.Errorf("Rating = %v, want 4.5", result.Rating)
	}
}

func TestParseRequest_Form_WithQueryParams(t *testing.T) {
	// Test combining form data parsing with query parameters
	formData := url.Values{}
	formData.Set("name", "Jane Smith")
	formData.Set("email", "jane@example.com")
	formData.Set("age", "25")
	formData.Set("active", "false")
	formData.Set("score", "88.7")

	req := httptest.NewRequest("POST", "http://example.com/users?page=3&sort=email&filter_by=inactive&limit=20&debug=false&rating=3.2", strings.NewReader(formData.Encode()))
	req.Header.Set(HeaderContentType, MIMEApplicationForm)

	var result UserRequest
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	// Check form data
	if result.Name != "Jane Smith" {
		t.Errorf("Name = %v, want Jane Smith", result.Name)
	}
	if result.Email != "jane@example.com" {
		t.Errorf("Email = %v, want jane@example.com", result.Email)
	}
	if result.Age != 25 {
		t.Errorf("Age = %v, want 25", result.Age)
	}
	if result.Active != false {
		t.Errorf("Active = %v, want false", result.Active)
	}
	if result.Score != 88.7 {
		t.Errorf("Score = %v, want 88.7", result.Score)
	}

	// Check query parameters
	if result.Page != 3 {
		t.Errorf("Page = %v, want 3", result.Page)
	}
	if result.Sort != "email" {
		t.Errorf("Sort = %v, want email", result.Sort)
	}
	if result.FilterBy != "inactive" {
		t.Errorf("FilterBy = %v, want inactive", result.FilterBy)
	}
	if result.Limit != 20 {
		t.Errorf("Limit = %v, want 20", result.Limit)
	}
	if result.Debug != false {
		t.Errorf("Debug = %v, want false", result.Debug)
	}
	if result.Rating != 3.2 {
		t.Errorf("Rating = %v, want 3.2", result.Rating)
	}
}

func TestParseRequest_MultipartForm_WithQueryParams(t *testing.T) {
	// Test combining multipart form data parsing with query parameters
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	_ = writer.WriteField("name", "Bob Wilson")
	_ = writer.WriteField("email", "bob@example.com")
	_ = writer.WriteField("age", "35")
	_ = writer.WriteField("active", "true")
	_ = writer.WriteField("score", "92.1")
	_ = writer.Close()

	req := httptest.NewRequest("POST", "http://example.com/users?page=1&sort=age&filter_by=all&limit=5&debug=true&rating=5.0", &buf)
	req.Header.Set(HeaderContentType, writer.FormDataContentType())

	var result UserRequest
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	// Check multipart form data
	if result.Name != "Bob Wilson" {
		t.Errorf("Name = %v, want Bob Wilson", result.Name)
	}
	if result.Email != "bob@example.com" {
		t.Errorf("Email = %v, want bob@example.com", result.Email)
	}
	if result.Age != 35 {
		t.Errorf("Age = %v, want 35", result.Age)
	}
	if result.Active != true {
		t.Errorf("Active = %v, want true", result.Active)
	}
	if result.Score != 92.1 {
		t.Errorf("Score = %v, want 92.1", result.Score)
	}

	// Check query parameters
	if result.Page != 1 {
		t.Errorf("Page = %v, want 1", result.Page)
	}
	if result.Sort != "age" {
		t.Errorf("Sort = %v, want age", result.Sort)
	}
	if result.FilterBy != "all" {
		t.Errorf("FilterBy = %v, want all", result.FilterBy)
	}
	if result.Limit != 5 {
		t.Errorf("Limit = %v, want 5", result.Limit)
	}
	if result.Debug != true {
		t.Errorf("Debug = %v, want true", result.Debug)
	}
	if result.Rating != 5.0 {
		t.Errorf("Rating = %v, want 5.0", result.Rating)
	}
}

func TestParseRequest_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (*http.Request, interface{})
		wantErr bool
		errMsg  string
	}{
		{
			name: "Nil request",
			setup: func() (*http.Request, interface{}) {
				var result UserRequest
				return nil, &result
			},
			wantErr: true,
			errMsg:  "invalid request",
		},
		{
			name: "Non-pointer target",
			setup: func() (*http.Request, interface{}) {
				req := httptest.NewRequest("GET", "http://example.com/test?page=1", nil)
				var result UserRequest
				return req, result // Not a pointer
			},
			wantErr: true,
			errMsg:  "invalid request",
		},
		{
			name: "Non-struct target",
			setup: func() (*http.Request, interface{}) {
				req := httptest.NewRequest("GET", "http://example.com/test?page=1", nil)
				var result string
				return req, &result // Pointer to non-struct
			},
			wantErr: true,
			errMsg:  "invalid request",
		},
		{
			name: "Malformed Content-Type",
			setup: func() (*http.Request, interface{}) {
				req := httptest.NewRequest("POST", "http://example.com/test", bytes.NewReader([]byte("{}")))
				req.Header.Set(HeaderContentType, "invalid/content/type/format")
				var result UserRequest
				return req, &result
			},
			wantErr: true,
			errMsg:  "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, target := tt.setup()
			err := ParseRequest(req, target)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message should contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestParseRequest_JSON_Only(t *testing.T) {
	jsonData := map[string]interface{}{
		"username": "testuser",
		"password": "secret123",
		"remember": true,
	}

	jsonBytes, _ := json.Marshal(jsonData)
	req := httptest.NewRequest("POST", "http://example.com/login", bytes.NewReader(jsonBytes))
	req.Header.Set(HeaderContentType, MIMEApplicationJSON)

	var result FormOnlyRequest
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	// JSON parsing should work - Go's json decoder matches field names case-insensitively
	// even without explicit json tags
	if result.Username != "testuser" {
		t.Errorf("Username = %v, want testuser", result.Username)
	}
	if result.Password != "secret123" {
		t.Errorf("Password = %v, want secret123", result.Password)
	}
	if result.Remember != true {
		t.Errorf("Remember = %v, want true", result.Remember)
	}
}

func TestParseRequest_Form_Only(t *testing.T) {
	formData := url.Values{}
	formData.Set("username", "formuser")
	formData.Set("password", "formpass")
	formData.Set("remember", "true")

	req := httptest.NewRequest("POST", "http://example.com/login", strings.NewReader(formData.Encode()))
	req.Header.Set(HeaderContentType, MIMEApplicationForm)

	var result FormOnlyRequest
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if result.Username != "formuser" {
		t.Errorf("Username = %v, want formuser", result.Username)
	}
	if result.Password != "formpass" {
		t.Errorf("Password = %v, want formpass", result.Password)
	}
	if result.Remember != true {
		t.Errorf("Remember = %v, want true", result.Remember)
	}
}

func TestParseRequest_EmptyBody_WithQueryParams(t *testing.T) {
	// Test GET request with only query parameters (no body)
	req := httptest.NewRequest("GET", "http://example.com/search?page=5&sort=date&limit=100", nil)

	var result QueryOnlyRequest
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if result.Page != 5 {
		t.Errorf("Page = %v, want 5", result.Page)
	}
	if result.Sort != "date" {
		t.Errorf("Sort = %v, want date", result.Sort)
	}
	if result.Limit != 100 {
		t.Errorf("Limit = %v, want 100", result.Limit)
	}
}

func TestParseRequest_UnsupportedContentType(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.com/test", bytes.NewReader([]byte("some data")))
	req.Header.Set(HeaderContentType, MIMETextPlain)

	var result UserRequest
	err := ParseRequest(req, &result)

	if err == nil {
		t.Fatalf("Expected error for unsupported content type")
	}

	if !strings.Contains(err.Error(), "invalid request") {
		t.Errorf("Error should mention unsupported content type, got: %v", err)
	}
}

func TestParseRequest_EmptyJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.com/test", bytes.NewReader([]byte("")))
	req.Header.Set(HeaderContentType, MIMEApplicationJSON)

	var result UserRequest
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("Not expecting any error, but got %v", err)
	}
}

func TestParseRequest_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.com/test", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set(HeaderContentType, MIMEApplicationJSON)

	var result UserRequest
	err := ParseRequest(req, &result)

	if err == nil {
		t.Fatalf("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "invalid request") {
		t.Errorf("Error should mention invalid character, got: %v", err)
	}
}

func TestMapQueryToStruct_FieldNameFallback(t *testing.T) {
	// Test that fields without query tags use lowercase field name
	type TestStruct struct {
		Page int
		Sort string
	}

	queryParams := url.Values{
		"page": {"42"},
		"sort": {"name"},
	}

	var result TestStruct
	err := mapQueryToStruct(queryParams, &result)

	if err != nil {
		t.Fatalf("mapQueryToStruct() error = %v", err)
	}

	if result.Page != 42 {
		t.Errorf("Page = %v, want 42", result.Page)
	}
	if result.Sort != "name" {
		t.Errorf("Sort = %v, want name", result.Sort)
	}
}

func TestMapQueryToStruct_SkipUnexportedFields(t *testing.T) {
	// Test that unexported fields are skipped
	type TestStruct struct {
		Page       int
		sort       string // unexported
		FilterBy   string `query:"filter_by"`
		privateVal int    // unexported
	}

	queryParams := url.Values{
		"page":       {"10"},
		"sort":       {"should_be_ignored"},
		"filter_by":  {"active"},
		"privateval": {"should_be_ignored"},
	}

	var result TestStruct
	err := mapQueryToStruct(queryParams, &result)

	if err != nil {
		t.Fatalf("mapQueryToStruct() error = %v", err)
	}

	if result.Page != 10 {
		t.Errorf("Page = %v, want 10", result.Page)
	}
	if result.sort != "" {
		t.Errorf("sort should remain empty (unexported), got %v", result.sort)
	}
	if result.FilterBy != "active" {
		t.Errorf("FilterBy = %v, want active", result.FilterBy)
	}
	if result.privateVal != 0 {
		t.Errorf("privateVal should remain 0 (unexported), got %v", result.privateVal)
	}
}

func TestParseRequest_TextUnmarshaler_QueryParams(t *testing.T) {
	tests := []struct {
		name        string
		queryParams string
		want        TestRequestWithCustomTypes
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Valid custom types in query parameters",
			queryParams: "id=0123456789abcdef0123456789abcdef&created_at=2023-12-25&priority=one&ptr_receiver=test&name=John&count=42",
			want: TestRequestWithCustomTypes{
				ID: func() UserID {
					var id UserID
					_, _ = hex.Decode(id[:], []byte("0123456789abcdef0123456789abcdef"))
					return id
				}(),
				CreatedAt:   CustomTime(time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)),
				Priority:    CustomInt(1),
				PtrReceiver: PointerReceiver("parsed:test"),
				Name:        "John",
				Count:       42,
			},
			wantErr: false,
		},
		{
			name:        "Valid custom types with pointers",
			queryParams: "id_ptr=fedcba9876543210fedcba9876543210&created_at_ptr=2024-01-01&priority_ptr=zero",
			want: TestRequestWithCustomTypes{
				IDPtr: func() *UserID {
					var id UserID
					_, _ = hex.Decode(id[:], []byte("fedcba9876543210fedcba9876543210"))
					return &id
				}(),
				CreatedAtPtr: func() *CustomTime { ct := CustomTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)); return &ct }(),
				PriorityPtr:  func() *CustomInt { ci := CustomInt(0); return &ci }(),
			},
			wantErr: false,
		},
		{
			name:        "Invalid UserID format",
			queryParams: "id=invalid_hex",
			wantErr:     true,
			errMsg:      "invalid request",
		},
		{
			name:        "Invalid UserID length",
			queryParams: "id=123",
			wantErr:     true,
			errMsg:      "invalid request",
		},
		{
			name:        "Invalid date format",
			queryParams: "created_at=invalid-date",
			wantErr:     true,
			errMsg:      "invalid request",
		},
		{
			name:        "Invalid CustomInt value",
			queryParams: "priority=invalid",
			wantErr:     true,
			errMsg:      "invalid request",
		},
		{
			name:        "Empty PointerReceiver (should fail)",
			queryParams: "ptr_receiver=",
			wantErr:     true,
			errMsg:      "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/test?"+tt.queryParams, nil)

			var result TestRequestWithCustomTypes
			err := ParseRequest(req, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err != nil && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message should contain %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			// Validate results for successful cases
			if result.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", result.ID, tt.want.ID)
			}
			if result.CreatedAt != tt.want.CreatedAt {
				t.Errorf("CreatedAt = %v, want %v", result.CreatedAt, tt.want.CreatedAt)
			}
			if result.Priority != tt.want.Priority {
				t.Errorf("Priority = %v, want %v", result.Priority, tt.want.Priority)
			}
			if result.PtrReceiver != tt.want.PtrReceiver {
				t.Errorf("PtrReceiver = %v, want %v", result.PtrReceiver, tt.want.PtrReceiver)
			}
			if result.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", result.Name, tt.want.Name)
			}
			if result.Count != tt.want.Count {
				t.Errorf("Count = %v, want %v", result.Count, tt.want.Count)
			}

			// Check pointer fields
			if tt.want.IDPtr != nil {
				if result.IDPtr == nil {
					t.Errorf("IDPtr should not be nil")
				} else if *result.IDPtr != *tt.want.IDPtr {
					t.Errorf("IDPtr = %v, want %v", *result.IDPtr, *tt.want.IDPtr)
				}
			}
			if tt.want.CreatedAtPtr != nil {
				if result.CreatedAtPtr == nil {
					t.Errorf("CreatedAtPtr should not be nil")
				} else if *result.CreatedAtPtr != *tt.want.CreatedAtPtr {
					t.Errorf("CreatedAtPtr = %v, want %v", *result.CreatedAtPtr, *tt.want.CreatedAtPtr)
				}
			}
			if tt.want.PriorityPtr != nil {
				if result.PriorityPtr == nil {
					t.Errorf("PriorityPtr should not be nil")
				} else if *result.PriorityPtr != *tt.want.PriorityPtr {
					t.Errorf("PriorityPtr = %v, want %v", *result.PriorityPtr, *tt.want.PriorityPtr)
				}
			}
		})
	}
}

func TestParseRequest_TextUnmarshaler_FormData(t *testing.T) {
	// Test form data parsing with custom types
	formData := url.Values{}
	formData.Set("id", "0123456789abcdef0123456789abcdef")
	formData.Set("created_at", "2023-06-15")
	formData.Set("priority", "one")
	formData.Set("ptr_receiver", "form_test")
	formData.Set("name", "Jane")
	formData.Set("count", "99")

	req := httptest.NewRequest("POST", "http://example.com/test", strings.NewReader(formData.Encode()))
	req.Header.Set(HeaderContentType, MIMEApplicationForm)

	var result TestRequestWithCustomTypes
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	// Verify custom types were parsed correctly
	expectedID := UserID{}
	_, _ = hex.Decode(expectedID[:], []byte("0123456789abcdef0123456789abcdef"))
	if result.ID != expectedID {
		t.Errorf("ID = %v, want %v", result.ID, expectedID)
	}

	expectedTime := CustomTime(time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC))
	if result.CreatedAt != expectedTime {
		t.Errorf("CreatedAt = %v, want %v", result.CreatedAt, expectedTime)
	}

	if result.Priority != CustomInt(1) {
		t.Errorf("Priority = %v, want %v", result.Priority, CustomInt(1))
	}

	if result.PtrReceiver != PointerReceiver("parsed:form_test") {
		t.Errorf("PtrReceiver = %v, want %v", result.PtrReceiver, PointerReceiver("parsed:form_test"))
	}

	// Verify regular fields still work
	if result.Name != "Jane" {
		t.Errorf("Name = %v, want Jane", result.Name)
	}
	if result.Count != 99 {
		t.Errorf("Count = %v, want 99", result.Count)
	}
}

func TestParseRequest_TextUnmarshaler_MultipartForm(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields with custom types
	_ = writer.WriteField("id", "abcdef0123456789abcdef0123456789")
	_ = writer.WriteField("created_at", "2024-03-10")
	_ = writer.WriteField("priority", "zero")
	_ = writer.WriteField("ptr_receiver", "multipart_test")
	_ = writer.WriteField("name", "Bob")
	_ = writer.WriteField("count", "75")
	_ = writer.Close()

	req := httptest.NewRequest("POST", "http://example.com/test", &buf)
	req.Header.Set(HeaderContentType, writer.FormDataContentType())

	var result TestRequestWithCustomTypes
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	// Verify custom types were parsed correctly
	expectedID := UserID{}
	_, _ = hex.Decode(expectedID[:], []byte("abcdef0123456789abcdef0123456789"))
	if result.ID != expectedID {
		t.Errorf("ID = %v, want %v", result.ID, expectedID)
	}

	expectedTime := CustomTime(time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC))
	if result.CreatedAt != expectedTime {
		t.Errorf("CreatedAt = %v, want %v", result.CreatedAt, expectedTime)
	}

	if result.Priority != CustomInt(0) {
		t.Errorf("Priority = %v, want %v", result.Priority, CustomInt(0))
	}

	if result.PtrReceiver != PointerReceiver("parsed:multipart_test") {
		t.Errorf("PtrReceiver = %v, want %v", result.PtrReceiver, PointerReceiver("parsed:multipart_test"))
	}
}

func TestParseRequest_TextUnmarshaler_CombinedWithJSON(t *testing.T) {
	// Test combining JSON body with query parameters containing custom types
	jsonData := map[string]interface{}{
		"name":  "Alice",
		"count": 123,
	}

	jsonBytes, _ := json.Marshal(jsonData)
	req := httptest.NewRequest("POST", "http://example.com/test?id=1234567890abcdef1234567890abcdef&priority=one", bytes.NewReader(jsonBytes))
	req.Header.Set(HeaderContentType, MIMEApplicationJSON)

	var result TestRequestWithCustomTypes
	err := ParseRequest(req, &result)

	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	// Check JSON fields
	if result.Name != "Alice" {
		t.Errorf("Name = %v, want Alice", result.Name)
	}
	if result.Count != 123 {
		t.Errorf("Count = %v, want 123", result.Count)
	}

	// Check query parameters with custom types
	expectedID := UserID{}
	_, _ = hex.Decode(expectedID[:], []byte("1234567890abcdef1234567890abcdef"))
	if result.ID != expectedID {
		t.Errorf("ID = %v, want %v", result.ID, expectedID)
	}

	if result.Priority != CustomInt(1) {
		t.Errorf("Priority = %v, want %v", result.Priority, CustomInt(1))
	}
}
