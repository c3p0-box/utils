package srv

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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
