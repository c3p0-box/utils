# env

Minimal helpers to populate Go structs from OS environment variables via struct tags. This package intentionally does not parse configuration files (YAML/TOML/.env/etc.).

## Installation

```bash
go get github.com/c3p0-box/utils/env
```

## Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/c3p0-box/utils/env"
)

type Config struct {
    Port string `env:"PORT" env-default:"8080"`
    Host string `env:"HOST" env-default:"localhost"`
}

func main() {
    var cfg Config
    if err := env.ReadEnv(&cfg); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Server: %s:%s\n", cfg.Host, cfg.Port)
}
```

## Advanced Examples

### Nested Structures

```go
type DatabaseConfig struct {
    Host     string `env:"HOST" env-default:"localhost"`
    Port     int    `env:"PORT" env-default:"5432"`
    Username string `env:"USERNAME" env-required:"true"`
    Password string `env:"PASSWORD" env-required:"true"`
    Database string `env:"DATABASE" env-default:"myapp"`
}

type RedisConfig struct {
    Host string `env:"HOST" env-default:"localhost"`
    Port int    `env:"PORT" env-default:"6379"`
    TTL  int    `env:"TTL" env-default:"3600"`
}

type ServerConfig struct {
    Port      int           `env:"PORT" env-default:"8080"`
    Hosts     []string      `env:"HOSTS" env-separator:","`
    DebugMode bool          `env:"DEBUG" env-default:"false"`
    EnvVars   map[string]string `env:"ENV_VARS"`
}

type Config struct {
    Server   ServerConfig   `env-prefix:"SERVER_"`
    Database DatabaseConfig `env-prefix:"DB_"`
    Redis    RedisConfig    `env-prefix:"REDIS_"`
}

func main() {
    // Set environment variables
    os.Setenv("SERVER_PORT", "9000")
    os.Setenv("SERVER_HOSTS", "api1.example.com,api2.example.com")
    os.Setenv("SERVER_DEBUG", "true")
    os.Setenv("SERVER_ENV_VARS", "LOG_LEVEL:info,REGION:us-west-2")
    
    os.Setenv("DB_HOST", "db.example.com")
    os.Setenv("DB_USERNAME", "admin")
    os.Setenv("DB_PASSWORD", "secret123")
    
    os.Setenv("REDIS_HOST", "cache.example.com")
    os.Setenv("REDIS_TTL", "7200")

    var cfg Config
    if err := env.ReadEnv(&cfg); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Server: %+v\n", cfg.Server)
    fmt.Printf("Database: %+v\n", cfg.Database)
    fmt.Printf("Redis: %+v\n", cfg.Redis)
}
```

### Deep Nesting Example

```go
type ApplicationConfig struct {
    Name    string `env:"NAME" env-default:"myapp"`
    Version string `env:"VERSION" env-default:"1.0.0"`
}

type MonitoringConfig struct {
    Enabled  bool   `env:"ENABLED" env-default:"false"`
    Endpoint string `env:"ENDPOINT"`
    APIKey   string `env:"API_KEY"`
}

type LoggingConfig struct {
    Level  string `env:"LEVEL" env-default:"info"`
    Format string `env:"FORMAT" env-default:"json"`
}

type ObservabilityConfig struct {
    Monitoring MonitoringConfig `env-prefix:"MONITORING_"`
    Logging    LoggingConfig    `env-prefix:"LOGGING_"`
}

type EnvironmentConfig struct {
    Application   ApplicationConfig   `env-prefix:"APP_"`
    Observability ObservabilityConfig `env-prefix:"OBS_"`
}

func main() {
    // Environment variables:
    // APP_NAME=my-service
    // APP_VERSION=2.1.0
    // OBS_MONITORING_ENABLED=true
    // OBS_MONITORING_ENDPOINT=https://metrics.example.com
    // OBS_MONITORING_API_KEY=secret
    // OBS_LOGGING_LEVEL=debug
    // OBS_LOGGING_FORMAT=text

    var cfg EnvironmentConfig
    if err := env.ReadEnv(&cfg); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("App: %s v%s\n", cfg.Application.Name, cfg.Application.Version)
    fmt.Printf("Monitoring enabled: %v\n", cfg.Observability.Monitoring.Enabled)
    fmt.Printf("Log level: %s\n", cfg.Observability.Logging.Level)
}
```

### Complex Data Types

```go
type ComplexConfig struct {
    // Time parsing
    StartTime time.Time     `env:"START_TIME"`
    Date      time.Time     `env:"DATE" env-layout:"2006-01-02"`
    Timeout   time.Duration `env:"TIMEOUT" env-default:"30s"`
    
    // URL parsing
    APIEndpoint url.URL `env:"API_ENDPOINT"`
    
    // Arrays and maps
    Servers     []string          `env:"SERVERS"`
    Ports       []int             `env:"PORTS"`
    Config      map[string]string `env:"CONFIG"`
    Settings    map[string]int    `env:"SETTINGS"`
    
    // Custom separator
    Tags        []string `env:"TAGS" env-separator:";"`
    
    // Numeric types
    MaxRetries  int8    `env:"MAX_RETRIES" env-default:"3"`
    BufferSize  uint32  `env:"BUFFER_SIZE" env-default:"1024"`
    Ratio       float64 `env:"RATIO" env-default:"0.75"`
}

func main() {
    // Example environment variables:
    // START_TIME=2023-01-01T00:00:00Z
    // DATE=2023-12-31
    // TIMEOUT=45s
    // API_ENDPOINT=https://api.example.com/v1
    // SERVERS=web1,web2,web3
    // PORTS=8080,8081,8082
    // CONFIG=debug:true,region:us-east-1
    // SETTINGS=workers:10,connections:100
    // TAGS=production;backend;api
    // MAX_RETRIES=5
    // BUFFER_SIZE=2048
    // RATIO=0.85

    var cfg ComplexConfig
    if err := env.ReadEnv(&cfg); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Configuration loaded: %+v\n", cfg)
}
```

### Custom Interfaces

```go
// Custom Setter interface
type LogLevel string

func (l *LogLevel) SetValue(s string) error {
    switch strings.ToLower(s) {
    case "debug", "info", "warn", "error":
        *l = LogLevel(strings.ToUpper(s))
        return nil
    default:
        return fmt.Errorf("invalid log level: %s", s)
    }
}

// Custom Updater interface
type DynamicConfig struct {
    RefreshRate int    `env:"REFRESH_RATE" env-upd:"true"`
    CacheSize   int    `env:"CACHE_SIZE" env-upd:"true"`
    LogLevel    LogLevel `env:"LOG_LEVEL" env-upd:"true"`
    initialized bool
}

func (d *DynamicConfig) Update() error {
    fmt.Println("Configuration updated!")
    d.initialized = true
    return nil
}

func main() {
    os.Setenv("REFRESH_RATE", "30")
    os.Setenv("CACHE_SIZE", "1000")
    os.Setenv("LOG_LEVEL", "debug")

    cfg := &DynamicConfig{}
    
    // Initial load
    if err := env.ReadEnv(cfg); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Initial config: %+v\n", cfg)

    // Update only updatable fields
    os.Setenv("REFRESH_RATE", "60")
    os.Setenv("LOG_LEVEL", "info")
    
    if err := env.UpdateEnv(cfg); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Updated config: %+v\n", cfg)
}
```

## Supported Tags

- **`env`**: comma-separated list of environment variable names to read; the first found wins.
- **`env-default`**: default value if no environment is set and the field is zero.
- **`env-separator`**: custom separator for slices and maps (default `,`).
- **`env-description`**: description for documentation (informational only).
- **`env-upd`**: mark a field as updatable by `UpdateEnv`.
- **`env-required`**: require a value; returns an error if missing and the field is zero.
- **`env-prefix`**: prefix to add to nested struct fields.
- **`env-layout`**: custom time layout for `time.Time` parsing (default RFC3339).

## Supported Types

- **Basic types**: `string`, `bool`, all integer and unsigned integer types, `float32/64`
- **Time types**: `time.Duration` (as string, e.g. `1h2m3s`), `time.Time`, `*time.Location`
- **URL type**: `url.URL`
- **Collections**: slices and maps (maps expect `key:value` pairs separated by `env-separator`)
- **Custom types**: Any type that implements `encoding.TextUnmarshaler` or the package `Setter` interface

## Features

### Alternative Environment Variable Names

```go
type Config struct {
    Port string `env:"PORT,APP_PORT,SERVICE_PORT"` // First found wins
}
```

### Required Fields

```go
type Config struct {
    APIKey string `env:"API_KEY" env-required:"true"` // Will error if not set
}
```

### Updating Values at Runtime

```go
type Config struct {
    RefreshRate int `env:"REFRESH_RATE" env-upd:"true"`
    StaticValue int `env:"STATIC_VALUE"`
}

// Later in your application:
env.UpdateEnv(&cfg) // Only updates fields marked with env-upd
```

### Custom Separators

```go
type Config struct {
    Tags     []string `env:"TAGS" env-separator:";"`           // Split by semicolon
    Settings map[string]string `env:"SETTINGS" env-separator:"|"` // Split by pipe
}
```

## Error Handling

The package provides detailed error messages for debugging:

```go
var cfg Config
if err := env.ReadEnv(&cfg); err != nil {
    // Errors include field names and specific parsing issues
    log.Printf("Configuration error: %v", err)
}
```

## Best Practices

1. **Use descriptive environment variable names** with consistent naming conventions
2. **Provide sensible defaults** using `env-default` for non-critical settings
3. **Mark sensitive or critical values as required** using `env-required`
4. **Use prefixes for nested structures** to avoid naming conflicts
5. **Document your configuration** using `env-description` tags for clarity
6. **Validate custom types** when implementing the `Setter` interface

## Notes

- This package does not read or parse configuration files. If you need file-based configs, load them separately and then call `ReadEnv` to override with environment variables as needed.
- Environment variables are read from `os.LookupEnv`, respecting the current process environment.
- Nested structures are processed recursively with prefix support for clean organization.
- Zero values are preserved unless explicitly set via environment variables or defaults.