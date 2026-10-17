# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- `make build`: builds `./cmd/web` into `bin/web` with `-mod=vendor`
- `make run-web`: runs `bin/web -config=./config/config-web.yaml` (build first). `config/config-web.yaml` is git-ignored, so create it first with `cp config/config-web-example.yaml config/config-web.yaml`.
- `make test`: `go test ./...`
- Single test: `go test -mod=vendor ./path/to/pkg -run '^TestName$'`
- `make bench`: runs the benchmarks with `-benchmem`
- `make lint` / `make fmt`: golangci-lint v2.14.0 with `.golangci.yml`. The Makefile installs it under `tools/bin/<version>/` the first time you run either target. CI runs the same linter version on pushes to master and on PRs, and `build_and_test.yml` runs `make build` and `make test`.
- `make vendor`: runs `go mod tidy` and then `go mod vendor`. Dependencies are vendored and committed, so run this after any change to `go.mod`.

`compose-up`/`compose-down` point to `script/docker/docker-compose.yml`, which doesn't exist yet.

## Linting conventions

`.golangci.yml` turns on **all** linters except a small set (`depguard`, `err113`, `exhaustruct`, `ireturn`, `revive`, `wsl`, …), so the rules are strict. Rules the existing code follows:
- Wrap every returned error with context: `fmt.Errorf("doing x: %w", err)`.
- When a linter has to be silenced, use a targeted `//nolint:<linter>` comment, as in `gochecknoglobals`, `contextcheck` and `interfacebloat`.
- Formatting uses `gofmt` + `goimports`. Imports go in three groups: stdlib, third-party, then `github.com/Mikhalevich/filestorage/...`.

## Architecture

Module: `github.com/Mikhalevich/filestorage` (Go 1.27). The project is at an early stage: the only binary is `cmd/web`, and it serves file storage endpoints: `GET /files/{path...}` (list), `POST /files/{path...}` (multipart upload) and `GET /download/{path...}`.

**Startup flow** (`cmd/web/main.go` → `internal/infra/application.Run`):
1. `LoadConfig` parses the `-config` flag, which is required and has no default, and loads YAML with `jinzhu/configor` into the binary's config struct. Struct tags `required:"true"` enforce required fields.
2. Sets up the logrus logger and installs it as the global std logger.
3. Sets up the OTLP/HTTP tracer (`internal/infra/tracing`) and installs it globally. The default is a no-op tracer.
4. Runs the binary's `RunFn` with a context that is cancelled on SIGINT/SIGTERM and carries the service logger.

The internal `run` returns an exit code, and only the outer `Run` calls `os.Exit`. This lets the deferred tracer shutdown flush pending spans before exit. Use `return 1` in `run` instead of calling `os.Exit` directly. The service version sent to the tracer is the VCS revision stamped by `go build` (`-dirty` is appended for uncommitted changes), so keep building the package `./cmd/web`, not `main.go`. Building a single file skips this stamping.

To add a new binary, follow the `cmd/web` layout:
- `cmd/<name>/internal/config`: a config struct that satisfies `application.Configer` (`Level()`, `ServiceName()`, `TracingEndpoint()`)
- `cmd/<name>/internal/setup`: wires dependencies from the config
- `cmd/<name>/internal/app`: the runtime

**Shared infra** (`internal/infra`):
- `logger`: a `Logger` interface backed by logrus. Get the logger with `logger.FromContext(ctx)`, which falls back to the std logger. A logrus hook adds OTel trace fields to log entries.
- `tracing`: a `Tracer` interface with `StartSpan`, with OTel and no-op implementations. The `semconv/vX.Y.Z` import in `tracer.go` must match the schema version used by the OTel SDK's `resource.Default()`. If they differ, startup fails with `conflicting Schema URL`, so bump the import whenever you upgrade `go.opentelemetry.io/otel`.

**HTTP server** (`cmd/web/internal/app`):
- Uses a stdlib `http.ServeMux`. The port and read/write timeouts come from the `http_server` config section (`config.HTTPServer`, passed in via `setup.Start`). Timeouts are written as Go duration strings (`10s`), which yaml.v3 decodes into `time.Duration`.
- Shuts down gracefully when the context is cancelled, with a 60s timeout.
- The API is described in `api/openapi.yaml`. Update it when you add or change a handler.
- Handlers live in `cmd/web/internal/app/handler` and have the signature `func(w http.ResponseWriter, r *http.Request) error` (`app.HandlerFunc`). Register them with `registerHandler`, which adapts them through `ToHTTPHandlerFunc`:
  - Return `*httperror.HTTPError` (e.g. `httperror.InternalServerError(msg)`) to send that status and message. Use the pointer: `HTTPError` has a pointer receiver, and the adapter matches only `*HTTPError`. A status outside 400–599 is replaced with 500.
  - Any other error is sent as a generic 500 "Internal server error".
  - If the handler has already written headers or body before returning an error, nothing more is written to the response.

Config files live in `config/`. Only `config-web-example.yaml` is committed: when you add a config field, add it there too. The tracing endpoint `jaeger:4318` assumes a docker-compose network.
