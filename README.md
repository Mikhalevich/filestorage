# filestorage

A small file storage service written in Go, with a browser UI. It serves a directory on disk over HTTP: list the files in a directory, upload files into it and download them.

## Features

- **List** the regular files in a directory, sorted by modification time (`GET /files/{path}`).
- **Upload** one or more files as `multipart/form-data` (`POST /files/{path}`). Existing files are never overwritten: if the name is taken, a numeric suffix is added (`a.txt` → `a_1.txt`).
- **Download** a file as an attachment (`GET`/`HEAD /download/{path}`), with support for range and conditional requests.
- All paths are relative to the configured storage root and cannot escape it.
- Structured logging (logrus) and OpenTelemetry tracing over OTLP/HTTP.
- Graceful shutdown on `SIGINT`/`SIGTERM`.
- A React web UI (`web/`) to browse directories, upload files (drag & drop or file picker) and download them.
- An [MCP](https://modelcontextprotocol.io) server (`cmd/mcp`) that lets LLM clients list, read, write and delete files.

There is no authentication yet.

## Requirements

- Go 1.27+
- Node.js (only for the web UI)

## Quick start

```sh
cp config/config-web-example.yaml config/config-web.yaml
mkdir -p storage          # storage.root_dir must exist
make build
make run-web              # API on http://localhost:8080
```

To run the web UI:

```sh
make web-install
make web-dev              # UI on http://localhost:5173
```

The Vite dev server proxies `/files/` and `/download/` to `http://localhost:8080`. To use another backend, set `API_TARGET=http://host:port make web-dev`.

### Docker

```sh
make compose-up           # UI on http://localhost:3000, API on :8080, MCP on :8081, Jaeger UI on :16686
make compose-down
```

The stack is defined in `script/docker/docker-compose.yml`. The server reads `script/docker/config-web.yaml`, and uploaded files are kept in the `storage` volume.

## Configuration

The server reads a YAML file passed with the required `-config` flag. All fields are required:

```yaml
log_level: "info"
tracing:
  endpoint: "jaeger:4318"   # OTLP/HTTP collector
  service_name: "web"
http_server:
  port: 8080
  read_timeout: 10s         # Go duration strings
  write_timeout: 10s
storage:
  root_dir: "./storage"     # directory served by the API
```

## MCP server

`cmd/mcp` serves the same storage over the Model Context Protocol, using the streamable HTTP transport at `/mcp`:

```sh
cp config/config-mcp-example.yaml config/config-mcp.yaml
make build
make run-mcp              # MCP on http://localhost:8081/mcp
claude mcp add --transport http filestorage http://localhost:8081/mcp
```

| Tool          | Description                                                                 |
|---------------|-----------------------------------------------------------------------------|
| `list_files`  | List files in a directory                                                   |
| `read_file`   | Read a file: UTF-8 content as text, anything else as a base64 resource (size capped by `limits.max_read_bytes`) |
| `write_file`  | Save a new file (text or base64). Taken names get a numeric suffix          |
| `delete_file` | Delete a file                                                               |
| `create_directory` | Create a directory; the parent must exist                              |

Its config matches the web one except that `http_server` has no `write_timeout` and there is an extra `limits.max_read_bytes` field.

## API

The API is described in [`api/openapi.yaml`](api/openapi.yaml) (OpenAPI 3.1).

| Method       | Path                | Description                         |
|--------------|---------------------|-------------------------------------|
| `GET`        | `/files/{path}`     | List files in a directory (JSON)    |
| `POST`       | `/files/{path}`     | Upload files into a directory       |
| `GET`/`HEAD` | `/download/{path}`  | Download a file                     |

Use `/files/` for the storage root. Errors are returned as plain text with an HTTP status code.

```sh
curl -F file=@a.txt http://localhost:8080/files/
curl http://localhost:8080/files/
curl -OJ http://localhost:8080/download/a.txt
```

## Development

| Command            | Description                                               |
|--------------------|-----------------------------------------------------------|
| `make build`       | Build `./cmd/web` and `./cmd/mcp` into `bin/`             |
| `make test`        | Run Go tests                                              |
| `make bench`       | Run benchmarks                                            |
| `make lint`        | Run golangci-lint (installed into `tools/bin` on first use) |
| `make fmt`         | Format code with golangci-lint formatters                 |
| `make vendor`      | `go mod tidy` + `go mod vendor` (dependencies are vendored) |
| `make web-install` | Install frontend dependencies                             |
| `make web-dev`     | Start the Vite dev server                                 |
| `make web-build`   | Build the frontend into `web/dist`                        |
| `make web-lint`    | Lint the frontend with oxlint                             |
| `make web-gen`     | Regenerate `web/src/api/schema.d.ts` from the OpenAPI spec |

Run `make web-gen` after changing `api/openapi.yaml`. CI fails if the generated types are out of date.

## Project layout

```
api/              OpenAPI specification
cmd/web/          HTTP server binary (config, setup, handlers)
cmd/mcp/          MCP server binary (config, setup, tools)
config/           Example configuration
internal/domain/  File storage service
internal/infra/   Application bootstrap, logging, tracing
web/              React + TypeScript web UI
```

## License

[MIT](LICENSE)
