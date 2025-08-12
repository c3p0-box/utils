### env

Minimal helpers to populate Go structs from OS environment variables via struct tags. This package intentionally does not parse configuration files (YAML/TOML/.env/etc.).

## Installation

```bash
go get github.com/c3p0-box/utils/env
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/c3p0-box/utils/env"
)

type Config struct{
    Port string `env:"PORT" env-default:"8080"`
    Host string `env:"HOST" env-default:"localhost"`
}

func main(){
    var cfg Config
    if err := env.ReadEnv(&cfg); err != nil {
        panic(err)
    }
    fmt.Printf("%s:%s\n", cfg.Host, cfg.Port)
}
```

## Tags

- **env**: comma-separated list of environment variable names to read; the first found wins.
- **env-default**: default value if no environment is set and the field is zero.
- **env-separator**: custom separator for slices and maps (default `,`).
- **env-description**: description used by `Usage` helpers.
- **env-upd**: mark a field as updatable by `UpdateEnv`.
- **env-required**: require a value; returns an error if missing and the field is zero.
- **env-prefix**: prefix to add to nested struct fields.

## Supported types

- Strings, booleans, all integer and unsigned integer types, float32/64
- `time.Duration` (as string, e.g. `1h2m3s`)
- `time.Time` (default layout RFC3339; override with `env-layout:"2006-01-02"`)
- `url.URL`
- Slices and maps (maps expect `key:value` pairs separated by `env-separator`)
- Any type that implements `encoding.TextUnmarshaler` or the package `Setter` interface

## Updating values

Call `UpdateEnv(&cfg)` to re-read values at runtime. Only fields tagged with `env-upd` are updated.

## Descriptions and usage

```go
header := "My Service Environment"
usage := env.FUsage(os.Stdout, &cfg, &header)
usage()
```

## Notes

- This package does not read or parse configuration files. If you need file-based configs, load them separately and then call `ReadEnv` to override with environment variables as needed.
