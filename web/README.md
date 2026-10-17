# filestorage web

Browser UI for the `cmd/web` HTTP API (`../api/openapi.yaml`): browse a directory, upload files (drag & drop or picker), download files.

```sh
make web-install        # from the repo root
make build run-web      # backend on :8080
make web-dev            # UI on http://localhost:5173
```

`API_TARGET=http://host:port make web-dev` points the dev proxy at another backend. API types in `src/api/schema.d.ts` are generated from the spec with `npm run gen`.
