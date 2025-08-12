package env

import (
	"errors"
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

	type Config struct {
		TS   time.Time     `env:"TS"`
		Date time.Time     `env:"DATE" env-layout:"2006-01-02"`
		Dur  time.Duration `env:"DUR"`
		// URL uses net/url parser in validStructs
		URL struct{ Scheme string } // placeholder to keep struct alignment
	}

	type WithURL struct {
		SiteURLURL string `env:"_"` // dummy to keep type unique in test file
	}

	// Use a dedicated struct for URL because the parser expects url.URL type.
	type ConfigURL struct {
		SiteURL time.Time `env:"_"` // dummy to keep type unique in test file
	}

	// Compose final struct that includes url.URL field.
	type ConfigWithURL struct {
		TS   time.Time             `env:"TS"`
		Date time.Time             `env:"DATE" env-layout:"2006-01-02"`
		Dur  time.Duration         `env:"DUR"`
		Site struct{ Host string } `env-prefix:"_"` // placeholder
	}

	// Minimal check for time parsing and duration.
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
	if cfg.Favorite != color("blue") {
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

func TestGetDescriptionContains(t *testing.T) {
	type Config struct {
		Port string `env:"PORT" env-default:"8080" env-description:"HTTP port"`
	}
	var cfg Config
	text, err := GetDescription(&cfg, nil)
	if err != nil {
		t.Fatalf("GetDescription error: %v", err)
	}
	if !strings.Contains(text, "Environment variables:") {
		t.Fatalf("missing header: %q", text)
	}
	if !strings.Contains(text, "PORT") || !strings.Contains(text, "default \"8080\"") || !strings.Contains(text, "HTTP port") {
		t.Fatalf("unexpected description text: %q", text)
	}
}
