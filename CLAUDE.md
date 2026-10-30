# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- `make build`: builds `./cmd/web` into `bin/web`, `./cmd/mcp` into `bin/mcp` and `./cmd/bot` into `bin/bot` with `-mod=vendor`
- `make run-web`: runs `bin/web -config=./config/config-web.yaml` (build first). `config/config-web.yaml` is git-ignored, so create it first with `cp config/config-web-example.yaml config/config-web.yaml`.
- `make run-mcp`: runs `bin/mcp -config=./config/config-mcp.yaml`. That file is also git-ignored, so create it from `config/config-mcp-example.yaml`.
- `make run-bot`: runs `bin/bot -config=./config/config-bot.yaml`. Also git-ignored, so create it from `config/config-bot-example.yaml`. The Telegram token can come from the `BOT_TELEGRAM_TOKEN` env var instead.
- `make test`: `go test ./...`
- Single test: `go test -mod=vendor ./path/to/pkg -run '^TestName$'`
- `make bench`: runs the benchmarks with `-benchmem`
- `make lint` / `make fmt`: golangci-lint v2.14.0 with `.golangci.yml`. The Makefile installs it under `tools/bin/<version>/` the first time you run either target. CI runs the same linter version on pushes to master and on PRs, and `build_and_test.yml` runs `make build` and `make test`.
- `make vendor`: runs `go mod tidy` and then `go mod vendor`. Dependencies are vendored and committed, so run this after any change to `go.mod`.
- Frontend (`web/`, needs Node): `make web-install` (`npm ci`), `make web-dev` (Vite dev server, proxies `/files/`, `/download/`, `/dirs/` and `/auth/` to `localhost:8080`; override the target with `API_TARGET`), `make web-build`, `make web-lint` (oxlint), `make web-gen`.

- `make compose-up` / `make compose-down`: build and run the full stack from `script/docker/docker-compose.yml`. The UI is served by nginx at `localhost:3000` and proxies `/files/`, `/download/`, `/dirs/` and `/auth/` to the backend. The API is also exposed on `:8080`, the MCP server on `:8081`, PostgreSQL on `:5432` (user/password/db `filestorage`, `pgdata` volume), and the Jaeger UI on `:16686`. The one-shot `sql-migrate-user` service applies the DB migrations once postgres is healthy; the backend starts only after it exits successfully. The backend uses `script/docker/config-web.yaml`, the MCP server uses `script/docker/config-mcp.yaml`, the Telegram bot uses `script/docker/config-bot.yaml`, and all three share files through the `storage` named volume. The bot is part of the stack and its token is required (`${BOT_TELEGRAM_TOKEN:?...}`). Compose interpolates the whole file for every command, so `compose-up` and `compose-down` both fail unless `BOT_TELEGRAM_TOKEN` is set in the environment. `.git` is excluded from the build context, so the image is built with `-buildvcs=false` and reports an empty service version.

## Linting conventions

`.golangci.yml` turns on **all** linters except a small set (`depguard`, `err113`, `exhaustruct`, `ireturn`, `revive`, `wsl`, …), so the rules are strict. Rules the existing code follows:
- Wrap every returned error with context: `fmt.Errorf("doing x: %w", err)`.
- When a linter has to be silenced, use a targeted `//nolint:<linter>` comment, as in `gochecknoglobals`, `contextcheck` and `interfacebloat`.
- Formatting uses `gofmt` + `goimports`. Imports go in three groups: stdlib, third-party, then `github.com/Mikhalevich/filestorage/...`.

## Architecture

Module: `github.com/Mikhalevich/filestorage` (Go 1.27). The project is at an early stage. There are three binaries:
- `cmd/web` serves the file storage HTTP endpoints: `GET /files/{user}/{path...}` (list), `POST /files/{user}/{path...}` (multipart upload), `DELETE /files/{user}/{path...}`, `POST /dirs/{user}/{path...}` (create a directory) and `GET /download/{user}/{path...}`, plus `POST /auth/register` and `POST /auth/login` (JSON `{"username","password"}` → `{"token"}`).
- `cmd/mcp` is an MCP server (see below).
- `cmd/bot` is a Telegram bot that saves received files into storage (see below).

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

**Domain** (`internal/domain`), shared by all binaries:
- `service/filesvc.Service` does all filesystem access. Every path is relative to `storage.root_dir`. Each method goes through `withRoot` or `withRootNoReturn`, which open an `os.Root` so a path can't escape the root, and convert the relative path with `toFSPath`. New operations should follow the same pattern.
- `service/authsvc.Service` issues (`IssueToken(subject)`) and verifies (`VerifyToken`) Ed25519-signed (EdDSA) JWTs. `New(publicKeyPEM, privateKeyPEM, opts...)` takes PKIX/PKCS8 PEM keys (`openssl genpkey -algorithm ed25519`) and fails on anything else. `WithIssuer` sets `iss` and requires it on verify, `WithTTL` sets the lifetime (default 1h). Verification accepts only EdDSA, requires `exp` and `sub`, and wraps every failure in `perror.Unauthorized("invalid token")`. Verified claims are domain `model/auth.Claims`.
- `service/usersvc.Service` owns users, stored through a `Repository` (`CreateUser`, `UserByName`).
  - `Create(ctx, name, password, createdAt)` validates and bcrypt-hashes the password. Names must match `^[a-z0-9][a-z0-9_-]{2,31}$` (they become storage directories) and can't be `resource.PublicUser`; passwords are 1–72 bytes (bcrypt ignores the rest). Violations are `perror.InvalidArgument`.
  - `UserByNameAndPwd` returns `perror.NotFound` for an unknown user (after comparing against a dummy hash, so timing doesn't reveal it) and `perror.Unauthorized("invalid credentials")` for a wrong password.
- `usecase/register` and `usecase/login` combine `usersvc` with the token issuer (`authsvc`); the web handlers call them, not the services. `register` stamps `createdAt` from a `TimeProvider` (`internal/adapter/timeprovider`) and issues the token after the user is created (not atomic: if issuing fails the user still exists). `login` maps `NotFound` to `perror.Unauthorized("invalid credentials")`, so unknown user and wrong password look the same.
- Interfaces a package consumes are declared in that package, and their gomock mocks are generated next to them (`<file>_mock.go`, same package) with `go run go.uber.org/mock/mockgen@v0.6.0 -source=<file> -destination=<dir>/<file>_mock.go -package=<pkg>`. Regenerate them when the interface changes.
- `model/perror` holds the domain errors (`NotFound`, `InvalidArgument`, `AlreadyExists`, `Unauthorized`). The service wraps them around the underlying OS error, and the message is safe to show to clients. The web handlers map them in `serviceError` (`handler.go`) to 404/400/409/401, and the MCP `handle` passes the message through. Any other error becomes a generic 500 or `internal error`.

**PostgreSQL** (`internal/adapter/repository/postgres`):
- `cmd/web` setup (`MakePGXConnection`) opens `postgres.connection` with the pgx stdlib driver through `otelsql` (spans per query), pings it, and wraps it in `sqlx`.
- `driver.Pgx` names the driver and reports constraint violations (`IsConstraintError` matches `pgconn.PgError.ConstraintName`).
- `transaction.Transaction` runs a func in a DB transaction carried in the ctx (nested calls reuse it); repositories get their executor with `ExtContext(ctx)`, so they work inside and outside a transaction.
- `pguser.PgUser` implements `usersvc.Repository` with DB models in `pguser/internal/model`. A unique violation on `users_name_key` (the name Postgres gives the `UNIQUE` on `users.name`) becomes `perror.AlreadyExists("user already exists")`, no rows becomes `perror.NotFound`.
- Migrations live in `script/db/user/migrations` in `rubenv/sql-migrate` format (`-- +migrate Up/Down`). They are not applied by the app: the `sql-migrate-user` compose service (`Dockerfile.sqlmigrate.user`, config `script/docker/dbconfig-user.yml`) runs `sql-migrate up`. Add schema changes as new numbered files, never edit applied ones.
- Nothing in `make test` needs a database; `pguser` has no tests yet.

**HTTP server** (`cmd/web/internal/app`):
- Uses a stdlib `http.ServeMux`. The port and read/write timeouts come from the `http_server` config section (`config.HTTPServer`, passed in via `setup.Start`). Timeouts are written as Go duration strings (`10s`), which yaml.v3 decodes into `time.Duration`.
- Shuts down gracefully when the context is cancelled. The timeout is `http_server.shutdown_timeout` (default `60s`, must be positive); connections still open when it expires are closed forcibly.
- Auth handlers (`auth_register.go`, `auth_login.go`) read JSON `{"username","password"}` limited to 4 KiB. A body that doesn't decode is 400 `invalid credentials`; usecase errors go through `serviceError`. Register answers 201, login 200, both with `{"token"}`.
- `router.CreateMux(hdl, common, resourceMiddlewares)`: `common` (tracing, logging) wraps every route; `resourceMiddlewares` (`middleware.Resource`, `middleware.Auth`) additionally wrap the `{user}` routes only. The `/auth/*` routes have no `{user}`, so they must not get the resource middlewares (`Auth` answers 400 without a resource).
- Authorization: `setup` builds `authsvc` from `auth.public_key`/`auth.private_key` (PEM), `auth.issuer` and `auth.token_ttl`. `middleware.Auth` requires `Authorization: Bearer <jwt>` whose subject equals the path `{user}` (401 with `WWW-Authenticate: Bearer`, 403 for another user), except for the `public` user. The token's subject is added to the request logger. Tokens come from `/auth/register` and `/auth/login`.
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

**Telegram bot** (`cmd/bot/internal/app`):
- Built on `github.com/Mikhalevich/tgbot`, a wrapper over `go-telegram/bot`. If `bot.webhook_token` is empty, the bot uses long polling. If it's set, tgbot runs in webhook mode with its own HTTP server on a hardcoded `:80`, which also serves `/live` and `/ready`. `bot.read_timeout`, `bot.write_timeout` and `bot.shutdown_timeout` configure that server.
- Routes are registered in `routes` (`app.go`). There's one default handler, `handler.Upload` (`handler.Handler` is built in `handler/handler.go` from a `FileService` and an HTTP `Doer`). It downloads a document, or the largest size of a photo (saved as `<YYYYMMDD_HHMMSS UTC>_<last 10 chars of file_id>.jpg`), from the `GetDownloadLink` URL. It saves the file into the storage root with `filesvc.SaveFile`, so a taken name gets a `_N` suffix, then replies with the stored name and its size in SI units (`saved as a.txt (15 B)`). Other message types are ignored silently.
- The download `http.Client` has no timeout, and file size isn't checked up front. The Bot API refuses downloads over 20 MB, and that surfaces as `internal error`.
- There's no access control: anyone who messages the bot can upload.
- `perror` messages are sent to the chat. Any other error is returned, and tgbot replies `internal error`.
- The file download link contains the bot token. `hideURL` (`upload.go`) replaces the URL inside any `*url.Error` with `<telegram file link>` before wrapping with `%w`, so the token never reaches logs or spans. Apply it to every error that can carry the link. Errors from tgbot's own API calls (e.g. `GetDownloadLink`) are already redacted by go-telegram.
- `app/tracer.go` adapts `tgbot.Tracer`, which is created per update, so it keeps per-update state (start time, span). `Before` starts the span and puts a request logger (pattern, chat_id, message_type, username) into the ctx. `OnSuccess` writes the access log line at Info and `OnError` at Error, both with the duration; `OnError` also records the error on the span. `After` only ends the span.
- Tests (`handler/upload_test.go`) use a fake `tgbot.Operations` whose download links point at an `httptest` server, and a real `filesvc` on a temp dir. `TestUploadErrorHidesLink` checks that a fake token in the link never appears in returned errors.

Config files live in `config/`. Only the `*-example.yaml` files are committed: when you add a config field, add it there too. The tracing endpoint `jaeger:4318` assumes a docker-compose network.

Config: `cmd/web` needs `postgres.connection` (a pgx connection string) and real PEM keys in `auth.public_key`/`auth.private_key`; the placeholders in the example and docker configs make startup fail.

**Frontend** (`web/`): Vite + React + TypeScript single-page app over the HTTP API.
- `src/api/schema.d.ts` is generated from `api/openapi.yaml` by `scripts/gen-api.mjs` (openapi-typescript, `format: binary` mapped to `Blob`). It's committed and regenerated by `predev`/`prebuild`; CI fails if it's stale, so run `make web-gen` after changing the spec.
- `src/api/client.ts` is the only place that talks to the API, through the typed `openapi-fetch` client. The storage root is the separate `/files/` route because OpenAPI path params can't be empty.
- Auth: `src/auth.ts` is a small store for the session (token plus `sub`/`exp` read from the JWT payload, not verified), persisted in `localStorage` and synced across tabs; components read it with `useAuth` (`useSyncExternalStore`). A client middleware adds `Authorization: Bearer` to every request except `/auth/*`; a 401 on a request that carried the token ends the session with a notice, and so does reaching `exp`. `AuthPanel` signs in or registers and starts the session with the returned token.
- Signed out, the UI shows the `public` user's files. Signed in, it shows the user's own files, with a tab to switch to the public ones; switching users resets the directory to the root.
- Downloads of public files are plain `<a href>` links built by `downloadURL`. Links can't send the token, so a signed-in user's own files go through `downloadFile`, which fetches with the token and saves a blob: the whole file is held in browser memory.
- The current directory lives in the URL hash (`#/sub/dir`). The list API returns subdirectories (first, `type: directory`) and files; `FileTable` renders subdirectories as rows that navigate into them. `CreateDirPanel` creates a subdirectory of the current one.
- `package.json` overrides `openapi-typescript`'s TypeScript peer dependency to the project's TS version.
