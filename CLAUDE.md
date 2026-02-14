<instructions>
- NEVER commit or push without explicit user instruction. No `git commit`, `git push`, `git add` unless the user says so.
- NEVER amend commits or force-push.
- Language: German comments/docs where existing, English for code identifiers.
- Keep code simple. No over-engineering, no speculative abstractions.
- All Go modules use `replace` directives for local cross-references — never publish to a registry.
- Run `pnpm turbo build` and `pnpm turbo test` to verify changes across all packages.
- When adding dependencies: `go mod tidy` in the affected app directory.
- Shared models/client live in `packages/shared` — both `api` and `cli` depend on it.
</instructions>

<project>
# max-cloud — Deutsche Cloud Run Alternative

Serverless container platform with scale-to-zero for the German market. MVP stage.

## Monorepo Structure

```text
max-cloud/
├── apps/api/          # REST API server (Go, chi router)
├── apps/cli/          # CLI tool (Go, cobra)
├── packages/shared/   # Shared Go module (models, API client)
├── turbo.json         # Build orchestration
├── pnpm-workspace.yaml
└── package.json       # pnpm@9.15.4, turbo@^2.3.0
```

Workspaces: `apps/*`, `packages/*`
Scripts: `pnpm turbo build|test|lint`

## Go Modules

| Module | Path | Key deps |
|--------|------|----------|
| `github.com/max-cloud/api` | `apps/api/` | chi/v5, uuid, shared (replace) |
| `github.com/max-cloud/cli` | `apps/cli/` | cobra, shared (replace) |
| `github.com/max-cloud/shared` | `packages/shared/` | (none) |

All use Go 1.25.7.

## API (`apps/api/`)

Entry: `main.go` → `config.Load()` → `server.New()` → `server.Router()` → `http.Server` with graceful shutdown.

### Key files
- `internal/config/config.go` — PORT env var (default 8080), LogLevel
- `internal/server/server.go` — chi router, middleware (RequestID, RealIP, Recoverer), creates Store + Handler
- `internal/store/store.go` — in-memory `sync.RWMutex` store, CRUD for services, UUID generation
- `internal/handler/health.go` — all HTTP handlers (Health, CreateService, ListServices, GetService, DeleteService)

### Routes
```sql
GET    /healthz              → {"status":"ok"}
GET    /api/v1/services      → []Service
POST   /api/v1/services      → Service (201) — body: DeployRequest
GET    /api/v1/services/{id} → Service | 404
DELETE /api/v1/services/{id} → 204 | 404
```

### Validation
CreateService requires `name` + `image` in JSON body. Returns 400 on missing fields or invalid JSON.

## CLI (`apps/cli/`)

Entry: `main.go` → `cmd.Execute()`

### Commands
- `maxcloud deploy [image] --name NAME [--env KEY=VALUE...]` — POST to API
- `maxcloud list` — tabwriter table (NAME, IMAGE, STATUS, URL)
- `maxcloud delete [service-id]` — DELETE by ID
- `maxcloud logs [service]` — stub (not implemented)
- `maxcloud version` — prints version string

Global flag: `--api-url` (default `http://localhost:8080`)
Client initialized in `PersistentPreRun` from root command.

## Shared (`packages/shared/`)

### Models (`pkg/models/models.go`)
- `Service` — ID, Name, Image, Status, URL, EnvVars, MinScale, MaxScale, CreatedAt, UpdatedAt
- `ServiceStatus` — ready, pending, failed, deleting
- `DeployRequest` — Name, Image, EnvVars
- `Revision` — ID, ServiceID, Image, Traffic, CreatedAt
- `LogEntry` — Timestamp, Message, Stream

### API Client (`pkg/api/client.go`)
- `NewClient(baseURL)` → `*Client` (30s timeout)
- `Deploy(DeployRequest)` → `*Service, error`
- `ListServices()` → `[]Service, error`
- `GetService(id)` → `*Service, error`
- `DeleteService(id)` → `error`
- `APIError{StatusCode, Message}` for structured error handling

## Tests

| File | What |
|------|------|
| `apps/api/internal/store/store_test.go` | CRUD + not-found cases |
| `apps/api/internal/handler/handler_test.go` | HTTP tests via httptest + chi router |
| `packages/shared/pkg/api/client_test.go` | Client tests with httptest.NewServer mock |

Run: `pnpm turbo test`

## Build

```sh
pnpm turbo build   # → apps/api/bin/api, apps/cli/bin/maxcloud
pnpm turbo test    # → go test ./... in all packages
```
</project>

<example>
# Manual E2E test
./apps/api/bin/api &
./apps/cli/bin/maxcloud deploy --name myapp nginx:latest
./apps/cli/bin/maxcloud list
./apps/cli/bin/maxcloud delete <service-id>
kill %1
</example>

<formatting>
- API error responses: `{"error":"message"}` with appropriate HTTP status
- Service URLs: `https://{name}.maxcloud.dev`
- JSON field naming: snake_case (`created_at`, `env_vars`, `min_scale`)
- CLI output: tabwriter for tables, fmt.Printf for single-resource display
</formatting>

<roadmap>
Current: MVP with in-memory store, single-node, no auth.
Next steps (not yet implemented):
- Authentication/multi-tenancy
- Persistent storage (PostgreSQL)
- Knative integration for real container orchestration
- Billing/metering
- Web console
- Custom domains
- Container log streaming
Target stack: Hetzner bare metal → Talos Linux → K8s → Knative Serving → Kourier → Harbor
</roadmap>
