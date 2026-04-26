# c3p0-box/utils

[![Go Reference](https://pkg.go.dev/badge/github.com/c3p0-box/utils.svg)](https://pkg.go.dev/github.com/c3p0-box/utils)
[![Go Report Card](https://goreportcard.com/badge/github.com/c3p0-box/utils)](https://goreportcard.com/report/github.com/c3p0-box/utils)

A collection of reusable Go packages with minimal external dependencies, designed for building robust applications following clean architecture principles.

## Overview

This repository provides essential utility packages for Go development, each focused on a specific domain with minimal dependencies. All packages are production-ready, well-tested, and follow idiomatic Go patterns.

## Packages

| Package | Description | Key Features |
|---------|-------------|--------------|
| [`env`](./env) | Environment variable configuration | Struct tag-based parsing, nested structures, type conversion, updatable fields |
| [`erm`](./erm) | Enhanced error management | HTTP status codes, stack traces, i18n support, validation error collection |
| [`i18n`](./i18n) | Lightweight internationalization | Template support, pluralization, thread-safe, singleton pattern |
| [`set`](./set) | Generic set data structure | Type-safe with Go generics, add/remove/contains operations, string utilities |
| [`srv`](./srv) | HTTP server utilities | Middleware system, graceful shutdown, sessions, request parsing, URL reversing |
| [`vix`](./vix) | Type-safe validation library | Fluent API, ERM integration, conditional validation, multi-field validation |

## Installation

Install individual packages as needed:

```bash
go get github.com/c3p0-box/utils/env
go get github.com/c3p0-box/utils/erm
go get github.com/c3p0-box/utils/i18n
go get github.com/c3p0-box/utils/set
go get github.com/c3p0-box/utils/srv
go get github.com/c3p0-box/utils/vix
```

Or use all packages:

```bash
go get github.com/c3p0-box/utils/...
```

## Dependencies

This project maintains minimal external dependencies:

- `golang.org/x/text` - Language tag support for i18n
All other functionality uses Go standard library.

## Go Version

Requires Go 1.26 or later.

## Quick Example

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/c3p0-box/utils/env"
    "github.com/c3p0-box/utils/vix"
    "github.com/c3p0-box/utils/erm"
)

type Config struct {
    Port string `env:"PORT" env-default:"8080"`
    Host string `env:"HOST" env-default:"localhost"`
}

func main() {
    // Load configuration from environment
    var cfg Config
    if err := env.ReadEnv(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Validate input
    validator := vix.Is(
        vix.String(cfg.Host, "host").Required(),
        vix.String(cfg.Port, "port").Required().Numeric(),
    )
    
    if !validator.Valid() {
        log.Fatal(validator.Error())
    }
    
    fmt.Printf("Server: %s:%s\n", cfg.Host, cfg.Port)
}
```

## Package Relationships

```
┌─────────────────────────────────────────────────────────┐
│                        srv                              │
│              (HTTP Server Utilities)                    │
└─────────────────────────────────────────────────────────┘
                            │
           ┌────────────────┼────────────────┐
           ▼                ▼                ▼
    ┌──────────┐     ┌──────────┐     ┌──────────┐
    │   erm    │     │   vix    │     │   env    │
    │ (Errors) │────▶│(Validate)│     │ (Config) │
    └────┬─────┘     └──────────┘     └──────────┘
         │
         ▼
    ┌──────────┐
    │   i18n   │
    │(Messages)│
    └──────────┘
         │
    ┌──────────┐
    │   set    │
    │(Generic) │
    └──────────┘
```

- **srv** uses **erm** for error handling and **env** for configuration
- **vix** uses **erm** for validation error reporting
- **erm** uses **i18n** for internationalization
- **set** is a standalone utility package

## Testing

All packages include comprehensive test coverage:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## License

This project is licensed under the Mozilla Public License 2.0. See [LICENSE](./LICENSE) for details.

## Contributing

Contributions are welcome! Please ensure:

1. Code follows Go best practices
2. All tests pass
3. New features include tests
4. Documentation is updated

## Architecture Principles

All packages follow these design principles:

- **KISS**: Keep It Simple, Stupid - minimal complexity
- **SOLID**: Single responsibility, open/closed, dependency inversion
- **Clean Architecture**: Clear separation of concerns
- **Minimal Dependencies**: Prefer standard library
- **Type Safety**: Leverage Go generics where appropriate
- **Thread Safety**: Safe for concurrent use

## Acknowledgments

Built with ❤️ for the Go community.
