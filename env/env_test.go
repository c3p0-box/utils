package env

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

type color string

// UnmarshalText implements encoding.TextUnmarshaler for tests.
func (c *color) UnmarshalText(b []byte) error {
	s := strings.TrimSpace(strings.ToLower(string(b)))
	if s == "" {
		return errors.New("empty color")
	}
	*c = color(s)
	return nil
}

func TestReadEnv_BasicDefaults(t *testing.T) {
	type Config struct {
		Port string `env:"PORT" env-default:"8080"`
		Host string `env:"HOST" env-default:"localhost"`
	}

	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if cfg.Port != "8080" || cfg.Host != "localhost" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestReadEnv_Required(t *testing.T) {
	type Config struct {
		Token string `env:"TOKEN" env-required:"true"`
	}
	var cfg Config
	if err := ReadEnv(&cfg); err == nil {
		t.Fatalf("expected error for required field, got nil")
	}
}

func TestReadEnv_WithEnvValues(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("HOST", "example")

	type Config struct {
		Port string `env:"PORT" env-default:"8080"`
		Host string `env:"HOST" env-default:"localhost"`
	}

	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if cfg.Port != "9090" || cfg.Host != "example" {
		t.Fatalf("unexpected values: %+v", cfg)
	}
}

func TestReadEnv_SliceAndMap(t *testing.T) {
	t.Setenv("ITEMS", "a,b,c")
	t.Setenv("PAIRS", "k1:1,k2:2")
	t.Setenv("ALT", "x;y;z")

	type Config struct {
		Items []string       `env:"ITEMS"`
		Pairs map[string]int `env:"PAIRS"`
		Alt   []string       `env:"ALT" env-separator:";"`
	}
	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if len(cfg.Items) != 3 || cfg.Items[0] != "a" || cfg.Items[2] != "c" {
		t.Fatalf("unexpected slice: %#v", cfg.Items)
	}
	if len(cfg.Pairs) != 2 || cfg.Pairs["k1"] != 1 || cfg.Pairs["k2"] != 2 {
		t.Fatalf("unexpected map: %#v", cfg.Pairs)
	}
	if len(cfg.Alt) != 3 || cfg.Alt[1] != "y" {
		t.Fatalf("unexpected custom-sep slice: %#v", cfg.Alt)
	}
}

func TestReadEnv_TimeURLDuration(t *testing.T) {
	t.Setenv("TS", "2021-12-31T23:59:59Z")
	t.Setenv("DATE", "2021-12-31")
	t.Setenv("SITE", "https://example.com/path?q=1")
	t.Setenv("DUR", "1h2m3s")

	// Test time.Time and time.Duration
	var cfg struct {
		TS   time.Time     `env:"TS"`
		Date time.Time     `env:"DATE" env-layout:"2006-01-02"`
		Dur  time.Duration `env:"DUR"`
	}

	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}
	if cfg.TS.UTC().Format(time.RFC3339) != "2021-12-31T23:59:59Z" {
		t.Fatalf("unexpected TS: %v", cfg.TS)
	}
	if cfg.Date.Year() != 2021 || cfg.Date.Month() != 12 || cfg.Date.Day() != 31 {
		t.Fatalf("unexpected Date: %v", cfg.Date)
	}
	if cfg.Dur != time.Hour+2*time.Minute+3*time.Second {
		t.Fatalf("unexpected Dur: %v", cfg.Dur)
	}

	// Test url.URL
	var urlCfg struct {
		URL url.URL `env:"SITE"`
	}
	if err := ReadEnv(&urlCfg); err != nil {
		t.Fatalf("ReadEnv error for URL: %v", err)
	}
	if urlCfg.URL.Scheme != "https" || urlCfg.URL.Host != "example.com" {
		t.Fatalf("unexpected URL: %+v", urlCfg.URL)
	}
}

func TestUpdateEnv_OnlyUpdatable(t *testing.T) {
	t.Setenv("F1", "a")
	t.Setenv("F2", "b")

	type Config struct {
		F1 string `env:"F1"`
		F2 string `env:"F2" env-upd:"true"`
	}
	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if cfg.F1 != "a" || cfg.F2 != "b" {
		t.Fatalf("unexpected initial values: %+v", cfg)
	}

	t.Setenv("F1", "x")
	t.Setenv("F2", "y")
	if err := UpdateEnv(&cfg); err != nil {
		t.Fatalf("UpdateEnv error: %v", err)
	}

	if cfg.F1 != "a" {
		t.Fatalf("F1 should not update: %q", cfg.F1)
	}
	if cfg.F2 != "y" {
		t.Fatalf("F2 should update: %q", cfg.F2)
	}
}

func TestReadEnv_AltEnvNames(t *testing.T) {
	t.Setenv("ALT_PORT", "10000")
	type Config struct {
		Port string `env:"PORT,ALT_PORT"`
	}
	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}
	if cfg.Port != "10000" {
		t.Fatalf("unexpected alt env value: %q", cfg.Port)
	}

	// If both are set, first wins.
	t.Setenv("PORT", "20000")
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}
	if cfg.Port != "20000" {
		t.Fatalf("PORT should win when both set: %q", cfg.Port)
	}
}

func TestReadEnv_TextUnmarshaler(t *testing.T) {
	t.Setenv("COLOR", "  Blue ")
	type Config struct {
		Favorite color `env:"COLOR"`
	}
	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}
	if cfg.Favorite != "blue" {
		t.Fatalf("unexpected unmarshaled value: %q", cfg.Favorite)
	}
}

func TestNestedStructWithPrefix(t *testing.T) {
	t.Setenv("APP_NAME", "svc")
	type Child struct {
		Name string `env:"NAME"`
	}
	type Parent struct {
		Sub Child `env-prefix:"APP_"`
	}
	var p Parent
	if err := ReadEnv(&p); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}
	if p.Sub.Name != "svc" {
		t.Fatalf("unexpected nested value: %+v", p)
	}
}

// ===== NESTED STRUCTURE TESTS =====

func TestNestedStructures_DeepNesting(t *testing.T) {
	t.Setenv("APP_DB_HOST", "localhost")
	t.Setenv("APP_DB_PORT", "5432")
	t.Setenv("APP_CACHE_REDIS_HOST", "redis.example.com")
	t.Setenv("APP_CACHE_REDIS_PORT", "6379")
	t.Setenv("APP_CACHE_TTL", "3600")

	type Redis struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	type Cache struct {
		Redis Redis `env-prefix:"REDIS_"`
		TTL   int   `env:"TTL"`
	}

	type Database struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	type Config struct {
		DB    Database `env-prefix:"DB_"`
		Cache Cache    `env-prefix:"CACHE_"`
	}

	type App struct {
		Config Config `env-prefix:"APP_"`
	}

	var app App
	if err := ReadEnv(&app); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if app.Config.DB.Host != "localhost" || app.Config.DB.Port != 5432 {
		t.Fatalf("unexpected DB config: %+v", app.Config.DB)
	}
	if app.Config.Cache.Redis.Host != "redis.example.com" || app.Config.Cache.Redis.Port != 6379 {
		t.Fatalf("unexpected Redis config: %+v", app.Config.Cache.Redis)
	}
	if app.Config.Cache.TTL != 3600 {
		t.Fatalf("unexpected Cache TTL: %d", app.Config.Cache.TTL)
	}
}

func TestNestedStructures_WithArraysAndMaps(t *testing.T) {
	t.Setenv("APP_SERVERS", "web1,web2,web3")
	t.Setenv("APP_PORTS", "8080,8081,8082")
	t.Setenv("APP_ENV_VARS", "DEBUG:true,LOG_LEVEL:info")

	type ServerConfig struct {
		Servers []string          `env:"SERVERS"`
		Ports   []int             `env:"PORTS"`
		EnvVars map[string]string `env:"ENV_VARS"`
	}

	type App struct {
		Config ServerConfig `env-prefix:"APP_"`
	}

	var app App
	if err := ReadEnv(&app); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	expectedServers := []string{"web1", "web2", "web3"}
	expectedPorts := []int{8080, 8081, 8082}

	if len(app.Config.Servers) != 3 {
		t.Fatalf("unexpected servers length: got %d, want 3", len(app.Config.Servers))
	}
	for i, expected := range expectedServers {
		if app.Config.Servers[i] != expected {
			t.Fatalf("unexpected server at index %d: got %s, want %s", i, app.Config.Servers[i], expected)
		}
	}

	if len(app.Config.Ports) != 3 {
		t.Fatalf("unexpected ports length: got %d, want 3", len(app.Config.Ports))
	}
	for i, expected := range expectedPorts {
		if app.Config.Ports[i] != expected {
			t.Fatalf("unexpected port at index %d: got %d, want %d", i, app.Config.Ports[i], expected)
		}
	}

	if app.Config.EnvVars["DEBUG"] != "true" || app.Config.EnvVars["LOG_LEVEL"] != "info" {
		t.Fatalf("unexpected env vars: %v", app.Config.EnvVars)
	}
}

// ===== ERROR HANDLING TESTS =====

func TestParseValue_NumericErrors(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		value   string
		target  interface{}
		wantErr bool
	}{
		{"int_invalid", "TEST_INT", "not_a_number", &struct {
			Field int `env:"TEST_INT"`
		}{}, true},
		{"int8_overflow", "TEST_INT8", "300", &struct {
			Field int8 `env:"TEST_INT8"`
		}{}, true},
		{"uint_negative", "TEST_UINT", "-1", &struct {
			Field uint `env:"TEST_UINT"`
		}{}, true},
		{"float_invalid", "TEST_FLOAT", "not_float", &struct {
			Field float64 `env:"TEST_FLOAT"`
		}{}, true},
		{"bool_invalid", "TEST_BOOL", "maybe", &struct {
			Field bool `env:"TEST_BOOL"`
		}{}, true},
		{"duration_invalid", "TEST_DUR", "invalid_duration", &struct {
			Field time.Duration `env:"TEST_DUR"`
		}{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.value)
			err := ReadEnv(tt.target)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %s: %v", tt.name, err)
			}
		})
	}
}

func TestParseValue_ComplexTypeErrors(t *testing.T) {
	// Test URL parsing error
	t.Setenv("BAD_URL", "://invalid-url")
	type ConfigURL struct {
		URL url.URL `env:"BAD_URL"`
	}
	var cfgURL ConfigURL
	if err := ReadEnv(&cfgURL); err == nil {
		t.Fatalf("expected error for invalid URL, got nil")
	}

	// Test time parsing error
	t.Setenv("BAD_TIME", "invalid-time-format")
	type ConfigTime struct {
		Time time.Time `env:"BAD_TIME"`
	}
	var cfgTime ConfigTime
	if err := ReadEnv(&cfgTime); err == nil {
		t.Fatalf("expected error for invalid time, got nil")
	}

	// Test location parsing error
	t.Setenv("BAD_LOCATION", "Invalid/Location")
	type ConfigLocation struct {
		Location *time.Location `env:"BAD_LOCATION"`
	}
	var cfgLocation ConfigLocation
	if err := ReadEnv(&cfgLocation); err == nil {
		t.Fatalf("expected error for invalid location, got nil")
	}
}

func TestParseValue_UnsupportedType(t *testing.T) {
	t.Setenv("UNSUPPORTED", "value")

	// Test with a genuinely unsupported type - complex numbers
	type Config struct {
		Field complex64 `env:"UNSUPPORTED"`
	}

	var cfg Config
	err := ReadEnv(&cfg)
	if err == nil {
		t.Fatalf("expected error for unsupported type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("expected 'unsupported type' error, got: %v", err)
	}
}

func TestParseValue_SliceMapErrors(t *testing.T) {
	// Test slice parsing error
	t.Setenv("BAD_SLICE", "1,not_int,3")
	var cfg1 struct {
		BadSlice []int `env:"BAD_SLICE"`
	}
	if err := ReadEnv(&cfg1); err == nil {
		t.Fatalf("expected error for invalid slice element, got nil")
	}

	// Test map value parsing error
	t.Setenv("BAD_MAP", "k1:1,k2:not_int")
	var cfg2 struct {
		BadMap map[string]int `env:"BAD_MAP"`
	}
	if err := ReadEnv(&cfg2); err == nil {
		t.Fatalf("expected error for invalid map value, got nil")
	}

	// Test malformed map
	t.Setenv("MALFORMED_MAP", "k1,k2:2") // Missing colon in first pair
	var cfg3 struct {
		BadFormat map[string]int `env:"MALFORMED_MAP"`
	}
	if err := ReadEnv(&cfg3); err == nil {
		t.Fatalf("expected error for malformed map, got nil")
	}
}

// ===== INTERFACE IMPLEMENTATION TESTS =====

// Custom Setter interface tests
type CustomString string

func (c *CustomString) SetValue(s string) error {
	if s == "" {
		return errors.New("cannot be empty")
	}
	*c = CustomString("custom:" + s)
	return nil
}

type FailingSetter string

func (f *FailingSetter) SetValue(_ string) error {
	return errors.New("setter always fails")
}

func TestSetter_Interface(t *testing.T) {
	t.Setenv("CUSTOM_VALUE", "test")

	type Config struct {
		Custom CustomString `env:"CUSTOM_VALUE"`
	}

	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if cfg.Custom != "custom:test" {
		t.Fatalf("unexpected custom value: %s", cfg.Custom)
	}
}

func TestSetter_InterfaceError(t *testing.T) {
	t.Setenv("FAILING_VALUE", "test")

	type Config struct {
		Failing FailingSetter `env:"FAILING_VALUE"`
	}

	var cfg Config
	if err := ReadEnv(&cfg); err == nil {
		t.Fatalf("expected error from failing setter, got nil")
	}
}

// Updater interface tests
type UpdatableConfig struct {
	Value   string `env:"UPDATE_VALUE"`
	updated bool
}

func (u *UpdatableConfig) Update() error {
	u.updated = true
	return nil
}

type FailingUpdater struct {
	Value string `env:"UPDATE_VALUE"`
}

func (f *FailingUpdater) Update() error {
	return errors.New("update failed")
}

func TestUpdater_Interface(t *testing.T) {
	t.Setenv("UPDATE_VALUE", "test")

	cfg := &UpdatableConfig{}
	if err := ReadEnv(cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if !cfg.updated {
		t.Fatalf("Update() method was not called")
	}
	if cfg.Value != "test" {
		t.Fatalf("unexpected value: %s", cfg.Value)
	}
}

func TestUpdater_InterfaceError(t *testing.T) {
	t.Setenv("UPDATE_VALUE", "test")

	cfg := &FailingUpdater{}
	if err := ReadEnv(cfg); err == nil {
		t.Fatalf("expected error from failing updater, got nil")
	}
}

// TextUnmarshaler error tests
type FailingUnmarshaler string

func (f *FailingUnmarshaler) UnmarshalText(_ []byte) error {
	return errors.New("unmarshal failed")
}

func TestTextUnmarshaler_Error(t *testing.T) {
	t.Setenv("FAILING_UNMARSHAL", "test")

	type Config struct {
		Failing FailingUnmarshaler `env:"FAILING_UNMARSHAL"`
	}

	var cfg Config
	if err := ReadEnv(&cfg); err == nil {
		t.Fatalf("expected error from failing unmarshaler, got nil")
	}
}

// ===== ADDITIONAL EDGE CASE TESTS =====

func TestReadEnv_NonStructError(t *testing.T) {
	var notStruct string
	if err := ReadEnv(&notStruct); err == nil {
		t.Fatalf("expected error for non-struct, got nil")
	}
}

func TestParseSlice_ByteSlice(t *testing.T) {
	t.Setenv("BYTE_DATA", "hello")

	type Config struct {
		Data []byte `env:"BYTE_DATA"`
	}

	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if string(cfg.Data) != "hello" {
		t.Fatalf("unexpected byte slice: %s", string(cfg.Data))
	}
}

func TestParseSlice_EmptyValue(t *testing.T) {
	t.Setenv("EMPTY_SLICE", "")

	type Config struct {
		Items []string `env:"EMPTY_SLICE"`
	}

	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	if len(cfg.Items) != 0 {
		t.Fatalf("expected empty slice, got: %v", cfg.Items)
	}
}

func TestNumericTypes_AllSizes(t *testing.T) {
	t.Setenv("INT8_VAL", "127")
	t.Setenv("INT16_VAL", "32767")
	t.Setenv("INT32_VAL", "2147483647")
	t.Setenv("INT64_VAL", "9223372036854775807")
	t.Setenv("UINT8_VAL", "255")
	t.Setenv("UINT16_VAL", "65535")
	t.Setenv("UINT32_VAL", "4294967295")
	t.Setenv("UINT64_VAL", "18446744073709551615")
	t.Setenv("FLOAT32_VAL", "3.14159")
	t.Setenv("FLOAT64_VAL", "2.718281828459045")

	type Config struct {
		Int8    int8    `env:"INT8_VAL"`
		Int16   int16   `env:"INT16_VAL"`
		Int32   int32   `env:"INT32_VAL"`
		Int64   int64   `env:"INT64_VAL"`
		Uint8   uint8   `env:"UINT8_VAL"`
		Uint16  uint16  `env:"UINT16_VAL"`
		Uint32  uint32  `env:"UINT32_VAL"`
		Uint64  uint64  `env:"UINT64_VAL"`
		Float32 float32 `env:"FLOAT32_VAL"`
		Float64 float64 `env:"FLOAT64_VAL"`
	}

	var cfg Config
	if err := ReadEnv(&cfg); err != nil {
		t.Fatalf("ReadEnv error: %v", err)
	}

	// Verify all values were parsed correctly
	if cfg.Int8 != 127 || cfg.Int16 != 32767 || cfg.Int32 != 2147483647 {
		t.Fatalf("unexpected int values: %+v", cfg)
	}
	if cfg.Uint8 != 255 || cfg.Uint16 != 65535 || cfg.Uint32 != 4294967295 {
		t.Fatalf("unexpected uint values: %+v", cfg)
	}
	if cfg.Float32 < 3.14 || cfg.Float32 > 3.15 {
		t.Fatalf("unexpected float32 value: %f", cfg.Float32)
	}
	if cfg.Float64 < 2.71 || cfg.Float64 > 2.72 {
		t.Fatalf("unexpected float64 value: %f", cfg.Float64)
	}
}

func TestBooleanVariations(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
	}

	for _, tt := range tests {
		t.Run("bool_"+tt.value, func(t *testing.T) {
			t.Setenv("BOOL_VAL", tt.value)

			type Config struct {
				Flag bool `env:"BOOL_VAL"`
			}

			var cfg Config
			if err := ReadEnv(&cfg); err != nil {
				t.Fatalf("ReadEnv error for %s: %v", tt.value, err)
			}

			if cfg.Flag != tt.expected {
				t.Fatalf("unexpected bool value for %s: got %v, want %v", tt.value, cfg.Flag, tt.expected)
			}
		})
	}
}
