# AGENTS.md

## Repository Snapshot
- Module: `github.com/c3p0-box/utils` (`go.mod`)
- Target Go version: **1.26**
- Primary external dependency: `golang.org/x/text` (language tags/i18n)
- Package set: `env`, `erm`, `i18n`, `set`, `srv`, `vix`

## Essential Commands
```bash
# full test suite
go test ./...

# commonly used focused runs (documented in package READMEs)
go test ./srv
go test -cover ./srv
go test -v ./srv
go test -bench=. ./srv

go test -v ./vix
go test -bench=. ./vix

go test -cover ./...
```

No Makefile, no CI workflow files, and no repo-documented lint command were found.

## Code Organization & Dependency Direction
- `env/`: reflection-based env-to-struct loader (`ReadEnv`, `UpdateEnv`) via struct tags.
- `erm/`: central error abstraction (`erm.Error`) with HTTP code, optional stack traces, validation metadata, localization.
- `i18n/`: lightweight singleton translation manager used by `erm`.
- `set/`: generic set + string utilities.
- `srv/`: HTTP context abstraction, mux/router, middleware, request parser, sessions.
- `vix/`: fluent validation built on `erm`.

Observed package dependency flow:
- `vix -> erm`
- `srv -> erm`
- `erm -> i18n`
- `env` and `set` are standalone utilities

## Repository Conventions

### General
- Prefer table-driven tests and subtests.
- Keep APIs small and composable; fluent chains in `vix` are the norm.
- Normalize public error behavior through `erm.Error` where package patterns already do so.

### ERM (`erm/`)
- `erm.New(code, msg, err)` captures stack traces only for HTTP 500.
- Validation errors are message-key based (`erm.Msg*`) with template params.
- Error collection uses `AddError`/`AddErrors` and flattens nested collections.
- Localization defaults to English fallback behavior.

### VIX (`vix/`)
- Validators accumulate into `ValidationResult`.
- Multi-field orchestration through `vix.Is(...)` / `vix.V()`.
- Namespaced paths (`In`, `InRow`) rewrite field keys for error maps.
- New validation rules must use `erm.Msg*` keys (not ad-hoc text).

### SRV (`srv/`)
- `Mux` patterns are registered as `"METHOD /path"` with handler signature `func(ctx Context) error`.
- Default mux error handler returns generic `500 Something went wrong`.
- Reverse lookup keys routes by name (`map[string]Route`), not by method.
- `ParseRequest` parses query first, then body by content type; parse failures intentionally collapse to generic invalid-request validation errors.
- Keep using `srv.Context` abstraction instead of direct `http.ResponseWriter`/`*http.Request` in handlers/middleware.

### ENV (`env/`)
- Tag surface includes `env`, `env-default`, `env-required`, `env-upd`, `env-prefix`, `env-layout`, etc.
- `UpdateEnv` only mutates fields marked `env-upd`.
- Supports `encoding.TextUnmarshaler`, custom `Setter`, optional `Updater`.
- Nested struct traversal is recursive for struct-typed fields.

### I18N (`i18n/`)
- Singleton manager with map-backed translations and template caching.
- Resolution fallback: requested language -> default language -> key literal.

### SET (`set/`)
- `Set[T comparable]` is `map[T]Void`.
- `FromStr` strips a broad punctuation set; do not assume whitespace-only normalization.

## Go 1.20–1.26 Features Relevant to Contributions
Use these when they improve clarity and maintain repo consistency:

- **Go 1.20**: `errors.Join`, multi-`%w` in `fmt.Errorf`, multi-error-aware `errors.Is/As`; `context` cancellation causes.
- **Go 1.21**: `min`, `max`, `clear`; `log/slog`; expanded cause-aware context helpers.
- **Go 1.22**: safer loop variable semantics; integer `range`; richer `net/http` ServeMux patterns.
- **Go 1.23**: iterator-capable `range`; `iter` package and iterator helpers in stdlib.
- **Go 1.24**: fully supported generic type aliases; `runtime.AddCleanup`; `testing.B.Loop`; `os.OpenRoot`/`os.Root`.
- **Go 1.25**: runtime/tooling improvements; no major language syntax changes.
- **Go 1.26**: `new(...)` accepts expressions; expanded generic constraint expressiveness; `go fix` modernizers; `errors.AsType[E error]`.

## Go Best Practices Required for This Repo

### Context
- Pass `context.Context` as first parameter when APIs need it.
- Do not store context in long-lived structs unless there is a deliberate compatibility reason.
- Never pass nil context (`context.TODO()` if needed).
- Use context values only for request-scoped metadata.

### Error Discipline
- Never discard returned errors.
- Do not use panic for ordinary control flow.
- Keep base error strings lowercase and punctuation-light.

### Error Contract Design
- Treat wrapping policy as API design:
  - Use `%w` when callers should inspect causes.
  - Avoid exposing unstable internals via wrapping.
- Prefer `errors.Is` / `errors.As` (and `errors.AsType` on Go 1.26) over direct equality/type assertions.
- For batch/multi-failure paths, use `errors.Join` or equivalent multi-wrapped behavior.

### Concurrency
- Ensure goroutines have bounded lifetimes and clear termination paths (context cancellation/closed channels).

## Testing Approach
- Behavior-first tests over mock-heavy tests.
- Keep extensive edge-case coverage with table cases and `t.Run`.
- `srv` tests should use `httptest` with realistic middleware/handler execution.
- `env` tests should use `t.Setenv` and conversion failure coverage.
- `erm`/`vix` tests should verify both message behavior and structured outputs (`ErrMap`, aggregation, nil safety).

## Gotchas / Non-Obvious Behaviors
- `srv` CORS preflight handling still requires explicit `OPTIONS` route registration for covered endpoints.
- `srv.NewOptions()` defaults `Secure: true` for session cookies.
- `srv.CookieStore` enforces practical cookie size limits (~4KB).
- `srv.Mux.Reverse` ignores extra params and errors when placeholders remain unreplaced.
- In `srv`, route names are not method-scoped internally; same-name reuse assumes same route shape.
- `ParseRequest` deliberately returns generic invalid-request errors instead of low-level parse internals.
- `erm.StackError.Unwrap()` returning self when root is nil is intentional and test-covered.

## Implementation Guidance for New Changes
1. Preserve package boundaries and existing dependency flow.
2. Keep `erm` as canonical error model for validation/api-style errors.
3. For new validation behavior:
   - add/route keys through `erm/locale.go` constants + translation map
   - keep `vix` chain/orchestrator behavior consistent
4. For `srv.ParseRequest` changes:
   - preserve query-first/body-second parse order
   - preserve existing invalid-request normalization unless intentionally changing public API
5. Before finalizing:
   - verify exported API/error-contract impact
   - add/adjust table-driven tests
   - run `go test ./...`
