# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- `make build`: builds `./cmd/web` into `bin/web` and `./cmd/mcp` into `bin/mcp` with `-mod=vendor`
- `make run-web`: runs `bin/web -config=./config/config-web.yaml` (build first). `config/config-web.yaml` is git-ignored, so create it first with `cp config/config-web-example.yaml config/config-web.yaml`.
- `make run-mcp`: runs `bin/mcp -config=./config/config-mcp.yaml`. That file is also git-ignored, so create it from `config/config-mcp-example.yaml`.
- `make test`: `go test ./...`
- Single test: `go test -mod=vendor ./path/to/pkg -run '^TestName$'`
- `make bench`: runs the benchmarks with `-benchmem`
- `make lint` / `make fmt`: golangci-lint v2.14.0 with `.golangci.yml`. The Makefile installs it under `tools/bin/<version>/` the first time you run either target. CI runs the same linter version on pushes to master and on PRs, and `build_and_test.yml` runs `make build` and `make test`.
- `make vendor`: runs `go mod tidy` and then `go mod vendor`. Dependencies are vendored and committed, so run this after any change to `go.mod`.
- Frontend (`web/`, needs Node): `make web-install` (`npm ci`), `make web-dev` (Vite dev server, proxies `/files/`, `/download/` and `/dirs/` to `localhost:8080`; override the target with `API_TARGET`), `make web-build`, `make web-lint` (oxlint), `make web-gen`.

- `make compose-up` / `make compose-down`: build and run the full stack from `script/docker/docker-compose.yml`. The UI is served by nginx at `localhost:3000` and proxies `/files/`, `/download/` and `/dirs/` to the backend. The API is also exposed on `:8080`, the MCP server on `:8081`, and the Jaeger UI on `:16686`. The backend uses `script/docker/config-web.yaml`, the MCP server uses `script/docker/config-mcp.yaml`, and both share files through the `storage` named volume. `.git` is excluded from the build context, so the image is built with `-buildvcs=false` and reports an empty service version.

## Linting conventions

`.golangci.yml` turns on **all** linters except a small set (`depguard`, `err113`, `exhaustruct`, `ireturn`, `revive`, `wsl`, …), so the rules are strict. Rules the existing code follows:
- Wrap every returned error with context: `fmt.Errorf("doing x: %w", err)`.
- When a linter has to be silenced, use a targeted `//nolint:<linter>` comment, as in `gochecknoglobals`, `contextcheck` and `interfacebloat`.
- Formatting uses `gofmt` + `goimports`. Imports go in three groups: stdlib, third-party, then `github.com/Mikhalevich/filestorage/...`.

## Architecture

Module: `github.com/Mikhalevich/filestorage` (Go 1.27). The project is at an early stage. There are two binaries:
- `cmd/web` serves the file storage HTTP endpoints: `GET /files/{path...}` (list), `POST /files/{path...}` (multipart upload), `DELETE /files/{path...}`, `POST /dirs/{path...}` (create a directory) and `GET /download/{path...}`.
- `cmd/mcp` is an MCP server (see below).

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

**Domain** (`internal/domain`), shared by both binaries:
- `service/filesvc.Service` does all filesystem access. Every path is relative to `storage.root_dir`. Each method goes through `withRoot` or `withRootNoReturn`, which open an `os.Root` so a path can't escape the root, and convert the relative path with `toFSPath`. New operations should follow the same pattern.
- `model/perror` holds the domain errors (`NotFound`, `InvalidArgument`, `AlreadyExists`). The service wraps them around the underlying OS error, and the message is safe to show to clients. The web handlers map them in `serviceError` (`handler.go`) to 404/400/409, and the MCP `handle` passes the message through. Any other error becomes a generic 500 or `internal error`.

**HTTP server** (`cmd/web/internal/app`):
- Uses a stdlib `http.ServeMux`. The port and read/write timeouts come from the `http_server` config section (`config.HTTPServer`, passed in via `setup.Start`). Timeouts are written as Go duration strings (`10s`), which yaml.v3 decodes into `time.Duration`.
- Shuts down gracefully when the context is cancelled. The timeout is `http_server.shutdown_timeout` (default `60s`, must be positive); connections still open when it expires are closed forcibly.
- The API is described in `api/openapi.yaml`. Update it when you add or change a handler.
- Handlers live in `cmd/web/internal/app/handler` and have the signature `func(w http.ResponseWriter, r *http.Request) error` (`app.HandlerFunc`). Register them with `registerHandler`, which adapts them through `ToHTTPHandlerFunc`:
  - Return `*httperror.HTTPError` (e.g. `httperror.InternalServerError(msg)`) to send that status and message. Use the pointer: `HTTPError` has a pointer receiver, and the adapter matches only `*HTTPError`. A status outside 400–599 is replaced with 500.
  - Any other error is sent as a generic 500 "Internal server error".
  - If the handler has already written headers or body before returning an error, nothing more is written to the response.

**MCP server** (`cmd/mcp/internal/app`):
- Built on `github.com/modelcontextprotocol/go-sdk`. It serves streamable HTTP at `/mcp`, in stateless mode because it exposes only tools. The HTTP server has no write timeout, since SSE responses can stay open. It calls `filesvc` directly, not the web API.
- Tools live in `cmd/mcp/internal/app/tool`: `list_files`, `read_file` (UTF-8 content comes back as text, anything else as a blob resource, capped by `limits.max_read_bytes`), `write_file` (optional base64), `delete_file` and `create_directory`. They are typed `mcp.ToolHandlerFor` handlers registered through `handle`. When a handler returns an error, `handle` logs it, and the client sees only the `perror` message, or `internal error` for anything else, so the host's paths never leak.
- `app.Middleware` is an `mcp.Middleware` that starts a span (named after the method, plus the tool name for `tools/call`) and writes an access log line for each request.
- Tests drive the tools through `mcp.NewInMemoryTransports` and a real `filesvc` on a temp dir.

Config files live in `config/`. Only the `*-example.yaml` files are committed: when you add a config field, add it there too. The tracing endpoint `jaeger:4318` assumes a docker-compose network.

**Frontend** (`web/`): Vite + React + TypeScript single-page app over the HTTP API.
- `src/api/schema.d.ts` is generated from `api/openapi.yaml` by `scripts/gen-api.mjs` (openapi-typescript, `format: binary` mapped to `Blob`). It's committed and regenerated by `predev`/`prebuild`; CI fails if it's stale, so run `make web-gen` after changing the spec.
- `src/api/client.ts` is the only place that talks to the API, through the typed `openapi-fetch` client. The storage root is the separate `/files/` route because OpenAPI path params can't be empty. Downloads are plain `<a href>` links built by `downloadURL`, not fetches.
- The current directory lives in the URL hash (`#/sub/dir`). The list API returns subdirectories (first, `type: directory`) and files; `FileTable` renders subdirectories as rows that navigate into them. `CreateDirPanel` creates a subdirectory of the current one.
- `package.json` overrides `openapi-typescript`'s TypeScript peer dependency to the project's TS version.
