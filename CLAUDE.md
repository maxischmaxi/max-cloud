<instructions>
- NEVER commit or push without explicit user instruction. No `git commit`, `git push`, `git add` unless the user says so.
- NEVER amend commits or force-push.
- Language: German comments/docs where existing, English for code identifiers.
- Keep code simple. No over-engineering, no speculative abstractions.
- All Go modules use `replace` directives for local cross-references — never publish to a registry.
- Run `pnpm turbo build` and `pnpm turbo test` to verify changes across all packages.
- When adding dependencies: `go mod tidy` in the affected app directory.
- Shared models/client live in `packages/shared` — both `api` and `cli` depend on it.
- CLI errors: Use `formatError()` from `cmd/errors.go` for user-friendly error messages.
- Reconciler: Checks Knative status first, only deploys if service missing (avoids unnecessary updates).
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

| Module                        | Path               | Key deps                       |
| ----------------------------- | ------------------ | ------------------------------ |
| `github.com/max-cloud/api`    | `apps/api/`        | chi/v5, uuid, shared (replace) |
| `github.com/max-cloud/cli`    | `apps/cli/`        | cobra, shared (replace)        |
| `github.com/max-cloud/shared` | `packages/shared/` | (none)                         |

All use Go 1.25.7.

## API (`apps/api/`)

Entry: `main.go` → `config.Load()` → `server.New()` → `server.Router()` → `http.Server` with graceful shutdown.

### Key files

- `internal/config/config.go` — PORT env var (default 8080), LogLevel, DatabaseURL
- `internal/server/server.go` — chi router, middleware (RequestID, RealIP, Recoverer, Auth), creates Store + Handler
- `internal/store/store.go` — Store interface
- `internal/store/memory.go` — in-memory `sync.RWMutex` store (for dev)
- `internal/store/postgres.go` — PostgreSQL store with migrations
- `internal/store/postgres_auth.go` — Auth store (Register, API-Keys, Invites)
- `internal/handler/*.go` — HTTP handlers (Health, Services, Auth, Invites, Logs)
- `internal/auth/middleware.go` — API-Key authentication middleware
- `internal/orchestrator/knative.go` — Knative deployment orchestrator
- `internal/email/resend.go` — Email sending via Resend

### Routes

```sql
GET    /healthz                    → {"status":"ok"}
POST   /api/v1/auth/register       → User registration
POST   /api/v1/auth/accept-invite  → Accept invite
POST   /api/v1/auth/invites        → Create invite (admin, auth)
GET    /api/v1/auth/invites        → List invites (admin, auth)
DELETE /api/v1/auth/invites/{id}   → Revoke invite (admin, auth)
GET    /api/v1/services            → []Service (auth)
POST   /api/v1/services            → Service (201) — body: DeployRequest (auth)
GET    /api/v1/services/{id}       → Service | 404 (auth)
DELETE /api/v1/services/{id}       → 204 | 404 (auth)
GET    /api/v1/services/{id}/logs  → Stream logs (auth, SSE)
POST   /api/v1/auth/api-keys       → Create API key (auth)
GET    /api/v1/auth/api-keys       → List API keys (auth)
DELETE /api/v1/auth/api-keys/{id}  → Delete API key (auth)
GET    /api/v1/auth/status         → Auth status (auth)
GET    /api/v1/registry/token      → JWT for Docker Registry (auth)
```

### Validation

CreateService requires `name` + `image` in JSON body. Returns 400 on missing fields or invalid JSON.

## CLI (`apps/cli/`)

Entry: `main.go` → `cmd.Execute()`

### Commands

- `maxcloud deploy [image] --name NAME [--port PORT] [--command CMD] [--args ARGS] [--env KEY=VALUE...]` — POST to API
- `maxcloud list` — tabwriter table (NAME, IMAGE, STATUS, URL)
- `maxcloud delete [service-id]` — DELETE by ID or name
- `maxcloud logs [service] [--follow] [--tail N]` — Stream container logs (SSE)
- `maxcloud version` — prints version string
- `maxcloud auth register --email EMAIL --org NAME` — Register new account
- `maxcloud auth status` — Show current auth info
- `maxcloud auth api-keys` — List API keys
- `maxcloud auth api-keys create --name NAME` — Create new API key
- `maxcloud auth api-keys delete ID` — Delete API key
- `maxcloud invite create --email EMAIL [--role ROLE]` — Create invite (admin)
- `maxcloud invite list` — List pending invites
- `maxcloud invite revoke ID` — Revoke invite
- `maxcloud push <image> --name NAME [--tag TAG]` — Push image to registry.maxcloud.dev
- `maxcloud images` — List registry images

### Deploy Flags

| Flag        | Description                                                           |
| ----------- | --------------------------------------------------------------------- |
| `--name`    | Service name (required)                                               |
| `--port`    | Container port, `0` = auto-detect from Dockerfile EXPOSE (default: 0) |
| `--command` | Override ENTRYPOINT, comma-separated (e.g., `python,app.py`)          |
| `--args`    | Override CMD, comma-separated (e.g., `--port,3000,--debug`)           |
| `--env`     | Environment variables, repeatable (e.g., `--env KEY=VALUE`)           |

### CLI Error Handling

- `SilenceUsage: true` — No usage help on errors
- `formatError()` in `cmd/errors.go` provides user-friendly messages:
  - 401 → "authentication required" with hint to `auth register`
  - 409 → "conflict: service name already exists"
  - Connection errors → "cannot connect to API server" with hint

Global flags: `--api-url` (default `http://localhost:8080`), `--api-key`
Config file: `~/.config/maxcloud/credentials` (stores API key and URL)

## Shared (`packages/shared/`)

### Models (`pkg/models/models.go`)

- `Service` — ID, Name, Image, Status, URL, Port, Command, Args, EnvVars, MinScale, MaxScale, CreatedAt, UpdatedAt, OrgID
- `ServiceStatus` — ready, pending, failed, deleting
- `DeployRequest` — Name, Image, Port, Command, Args, EnvVars
- `Revision` — ID, ServiceID, Image, Traffic, CreatedAt
- `LogEntry` — Timestamp, Message, Stream
- `User`, `Organization`, `APIKeyInfo`, `AuthInfo` — Auth models
- `Invitation`, `InviteRequest`, `InviteResponse` — Invite models

### API Client (`pkg/api/client.go`)

- `NewClient(baseURL)` → `*Client` (30s timeout)
- `Deploy(DeployRequest)` → `*Service, error`
- `ListServices()` → `[]Service, error`
- `GetService(id)` → `*Service, error`
- `DeleteService(id)` → `error`
- `StreamLogs(ctx, id, follow, tail)` → `*LogStream, error`
- `Register(RegisterRequest)` → `*RegisterResponse, error`
- `CreateAPIKey(CreateAPIKeyRequest)` → `*CreateAPIKeyResponse, error`
- `ListAPIKeys()` → `[]APIKeyInfo, error`
- `DeleteAPIKey(id)` → `error`
- `AuthStatus()` → `*AuthInfo, error`
- `CreateInvite(InviteRequest)` → `*InviteResponse, error`
- `ListInvites()` → `[]Invitation, error`
- `RevokeInvite(id)` → `error`
- `AcceptInvite(AcceptInviteRequest)` → `*AcceptInviteResponse, error`
- `GetRegistryToken(scope)` → `*RegistryTokenResponse, error`
- `APIError{StatusCode, Message}` for structured error handling

## Tests

| File                                            | What                                      |
| ----------------------------------------------- | ----------------------------------------- |
| `apps/api/internal/store/store_test.go`         | CRUD + not-found cases                    |
| `apps/api/internal/store/auth_test.go`          | Auth store tests (memory)                 |
| `apps/api/internal/store/postgres_test.go`      | PostgreSQL store tests                    |
| `apps/api/internal/store/postgres_auth_test.go` | PostgreSQL auth tests                     |
| `apps/api/internal/handler/handler_test.go`     | HTTP tests via httptest + chi router      |
| `apps/api/internal/handler/logs_test.go`        | Log streaming tests (SSE)                 |
| `apps/api/internal/handler/auth_test.go`        | Auth handler tests                        |
| `apps/api/internal/handler/invite_test.go`      | Invite handler tests                      |
| `apps/api/internal/auth/middleware_test.go`     | Auth middleware tests                     |
| `packages/shared/pkg/api/client_test.go`        | Client tests with httptest.NewServer mock |

Run: `pnpm turbo test`

## Build

```sh
pnpm turbo build   # → apps/api/bin/api, apps/cli/bin/maxcloud
pnpm turbo test    # → go test ./... in all packages
pnpm turbo lint    # → golangci-lint ./...
```

## CI/CD Pipeline

### GitHub Actions

| Workflow | Trigger              | Beschreibung               |
| -------- | -------------------- | -------------------------- |
| `ci.yml` | push/PR main,develop | Build, Test, Security Scan |
| `cd.yml` | push main            | Deploy zu Hetzner          |

### Benötigte Secrets

Im GitHub Repository unter Settings → Secrets:

| Secret                  | Beschreibung                              |
| ----------------------- | ----------------------------------------- |
| `KUBECONFIG`            | Kubernetes kubeconfig für Hetzner Cluster |
| `HETZNER_API_TOKEN`     | Hetzner Cloud API Token                   |
| `DNS_ZONE`              | DNS Zone (z.B. maxcloud.dev)              |
| `DATABASE_URL`          | PostgreSQL Connection String              |
| `RESEND_API_KEY`        | Resend API Key für Emails (erforderlich)  |
| `HETZNER_S3_ACCESS_KEY` | Hetzner Object Storage Access Key         |
| `HETZNER_S3_SECRET_KEY` | Hetzner Object Storage Secret Key         |
| `REGISTRY_JWT_SECRET`   | HMAC Secret für Registry JWT-Signierung   |

### Environment Variables

| Variable                | Beschreibung                                        | Erforderlich |
| ----------------------- | --------------------------------------------------- | ------------ |
| `PORT`                  | Server Port (default: 8080)                         | Nein         |
| `DATABASE_URL`          | PostgreSQL Connection String                        | Nein\*       |
| `KUBECONFIG`            | Pfad zur kubeconfig Datei                           | Nein         |
| `KNATIVE_NAMESPACE`     | Kubernetes Namespace für Knative (default: default) | Nein         |
| `RESEND_API_KEY`        | Resend API Key für Email-Versand                    | **Ja**       |
| `EMAIL_FROM`            | Absender-Email (default: noreply@maxcloud.dev)      | Nein         |
| `DEV_MODE`              | `true` deaktiviert Auth-Middleware                  | Nein         |
| `REGISTRY_URL`          | Registry Domain (default: registry.maxcloud.dev)    | Nein         |
| `REGISTRY_JWT_SECRET`   | HMAC Secret für Registry JWT-Signierung             | Für Registry |
| `REGISTRY_TOKEN_EXPIRY` | Token-Gültigkeit (default: 1h)                      | Nein         |

\*In-Memory Store wird verwendet, wenn DATABASE_URL nicht gesetzt ist.

### Deployment Flow

1. **CI Pipeline** (automatisch bei PR/push):
   - Build → Test → Security Scan

2. **CD Pipeline** (automatisch bei push auf main):
   - Kubernetes Cluster Setup
   - ExternalDNS + cert-manager deployen
   - Knative installieren
   - max-cloud API deployen
   - API Health Check

## Deployment

### Voraussetzungen

- Hetzner Cloud Account
- Hetzner API Token mit Read/Write Rechten
- SSH Key in Hetzner hinterlegt

### Manueller Erst-Setup

```bash
# 1. Hetzner CLI installieren
brew install hcloud

# 2. Environment setzen
export HETZNER_API_TOKEN="your-token"
export DNS_ZONE="maxcloud.dev"

# 3. Cluster erstellen
chmod +x deploy/hetzner-setup.sh
./deploy/hetzner-setup.sh

# 4. Kubeconfig exportieren
export KUBECONFIG=./kubeconfig
```

### CI/CD Deployment (empfohlen)

1. **Secrets konfigurieren** in GitHub Repository Settings
2. **Auf main pushen** → Pipeline deployed automatisch

## Kubernetes Deploy Files

| File                              | Beschreibung                     |
| --------------------------------- | -------------------------------- |
| `deploy/external-dns.yaml`        | ExternalDNS mit Hetzner Provider |
| `deploy/cert-manager.yaml`        | cert-manager für TLS Zertifikate |
| `deploy/hetzner-setup.sh`         | Script für Cluster-Erstellung    |
| `deploy/registry-namespace.yaml`  | Registry Namespace               |
| `deploy/registry-secret.yaml`     | S3/JWT Secrets                   |
| `deploy/registry-config.yaml`     | Distribution Config              |
| `deploy/registry-deployment.yaml` | Registry:3 Pods                  |
| `deploy/registry-ingress.yaml`    | registry.maxcloud.dev            |

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

<current-status>
✅ **MVP Code Vollständig implementiert**

**Funktionierende Features:**

- Knative Serving + Kourier Ingress auf Kubernetes
- Container-Deployment mit Scale-to-Zero via Knative (minScale=0)
- Auto-Scaling: 0 bis N Instanzen basierend auf Request-Concurrency
- Reconciler synchronisiert API-Store mit Knative-Status (10s Intervall)
  - Prüft erst Knative-Status, deployt nur wenn Service fehlt (vermeidet unnötige Updates)
- Container Log Streaming via SSE (`GET /api/v1/services/{id}/logs`)
- Auth-System: User-Registrierung, API-Keys, Invites
- Email-Versand für Invites via Resend (erforderlich)
- Multi-Tenant Isolation: `org_id` in allen DB-Queries, `auth.OrgIDFromContext()`
- Kubernetes Namespace pro Organisation (`mc-org-{org_id}`)
- PostgreSQL Store mit Migrationen
- CLI Commands: `deploy`, `list`, `delete`, `logs`, `version`, `auth`, `invite`
- Auth-Middleware: Produktiv mit API-Key, Dev-Mode mit Fake-Auth
- Volle Docker-Image-Unterstützung: Port, Command, Args konfigurierbar

**Lokales Setup:** `./setup-local-fixed.sh`

- Automatische Knative Installation für Entwicklung
- Minikube Konfiguration (8GB RAM, 4 CPUs)
- Knative Domain: `minikube.local` (konfiguriert in `config-domain` ConfigMap)
- Zugriff via: `curl -H "Host: {name}.{namespace}.minikube.local" http://$(kubectl get svc -n kourier-system kourier -o jsonpath='{.spec.clusterIP}')`

**Produktionsbereite Komponenten:**

- Knative Orchestrator mit vollständiger Container-Lifecycle
- Namespace-Management pro Organisation (On-Demand Erstellung)
- Graceful Shutdown für API Server
- Logging mit slog (JSON Structured Logging)
- Error Handling für Kubernetes-Operationen

**Database Migrations:**

- `001_create_services.sql` — Services Tabelle
- `002_create_auth_tables.sql` — Users, Organizations, API Keys
- `003_add_tenant_to_services.sql` — org_id für Multi-Tenant
- `004_create_invitations.sql` — Invitations Tabelle
- `005_add_port_command_args.sql` — Port, Command, Args für Docker-Images
- `006_create_registry_images.sql` — Registry Images Tracking

### Models

- `Service` — ID, Name, Image, Status, URL, Port, Command, Args, EnvVars, MinScale, MaxScale, CreatedAt, UpdatedAt, OrgID
- `ServiceStatus` — ready, pending, failed, deleting
- `DeployRequest` — Name, Image, Port, Command, Args, EnvVars
- `Revision` — ID, ServiceID, Image, Traffic, CreatedAt
- `LogEntry` — Timestamp, Message, Stream
- `User`, `Organization`, `APIKeyInfo`, `AuthInfo` — Auth models
- `Invitation`, `InviteRequest`, `InviteResponse` — Invite models
- `RegistryTokenResponse` — Token, ExpiresIn, IssuedAt
- `Image`, `Repository` — Registry models

<production-readiness>
## Pre-Deployment Checklist

### Kritisch (Vor erstem Deployment behoben)

| Issue                        | Status  | Beschreibung                                                              |
| ---------------------------- | ------- | ------------------------------------------------------------------------- |
| Dockerfiles erstellen        | ✅ Done | `apps/api/Dockerfile` und `apps/cli/Dockerfile` aktualisiert für Monorepo |
| CD: Container Images bauen   | ✅ Done | `build-images` Job baut und pushed Images vor Deploy                      |
| CD: Kubeconfig Secret        | ✅ Done | Secret wird mit Dateiinhalt erstellt (nicht Pfad)                         |
| Migration Version Tracking   | ✅ Done | `schema_migrations` Tabelle trackt ausgeführte Migrationen                |
| Health/Readiness Probes      | ✅ Done | Liveness und Readiness Probes in Deployment Manifest                      |
| Panic Recovery in Goroutines | ✅ Done | `recover()` in Auth Middleware Goroutine                                  |

### Wichtig (Vor erstem Deployment behoben)

| Issue                    | Status  | Beschreibung                                            |
| ------------------------ | ------- | ------------------------------------------------------- |
| Service Name Validation  | ✅ Done | DNS-kompatible Validierung (lowercase, max 63 chars)    |
| Namespace Creation Fatal | ✅ Done | Namespace-Fehler blockiert Service-Erstellung           |
| Request ID in Errors     | ✅ Done | `request_id` in allen Internal Server Error Responses   |
| Duplicate Service Error  | ✅ Done | API returns 409 Conflict instead of 500                 |
| CLI Error Messages       | ✅ Done | User-friendly errors via `formatError()`, no usage spam |
| Reconciler Loop          | ✅ Done | Only deploys if service missing, status-first check     |

### Optional (Post-Deployment)

- Rate Limiting auf Auth-Endpunkten
- ResourceQuotas pro Namespace
- GDPR Account Deletion Path
- API Key Plaintext Storage in CLI
  </production-readiness>

<roadmap>
Current: ✅ Produktionsbereit - Alle kritischen Issues behoben

Done:

- ✅ HTTPS/TLS mit Let's Encrypt (cert-manager)
- ✅ Public Domain `.maxcloud.dev` (ExternalDNS)
- ✅ Container-Log Streaming via SSE
- ✅ Multi-Tenant Isolation (org_id in DB-Schema)
- ✅ PostgreSQL Store mit Migrationen
- ✅ Email-Versand für Invites (Resend API)
- ✅ Kubernetes Namespace pro Organisation
- ✅ Auth-Middleware für Produktion
- ✅ Dockerfiles für Monorepo-Struktur
- ✅ CI/CD Pipeline mit Container Image Build
- ✅ Migration Version Tracking
- ✅ Health/Readiness Probes
- ✅ Panic Recovery
- ✅ Service Name Validation
- ✅ Request ID in Error Responses
- ✅ User-friendly CLI Error Messages
- ✅ Reconciler: Status-first, only deploy if missing
- ✅ Full Docker Image Support: Port, Command, Args configurable
- ✅ Knative Domain Config for local dev (`minikube.local`)
- ✅ Scale-to-Zero (minScale=0 default)
- ✅ Private Docker Registry mit Hetzner S3 Backend
- ✅ Registry Token Auth via JWT

Ready for:

- Erstes Produktions-Deployment

Post-MVP:

- Billing/Metering (per-second)
- Web Console UI
- Custom Domains
- Monitoring (Prometheus/Grafana)
  </roadmap>

<last-session>
## Session 2026-02-15 (Fortsetzung): Docker Registry mit Hetzner S3

### Implementiert: Private Docker Registry

**Architektur:**

```
CLI push -> API Token -> JWT -> CNCF Distribution -> Hetzner S3 Bucket
                                   |
Knative Service <- pull <- imagePullSecret (JWT)
```

**Neue API Route:**

```
GET /api/v1/registry/token?scope=repository:{org_id}/{name}:push,pull
```

**Neue CLI Commands:**

- `maxcloud push <image> --name <name> [--tag latest]` — Tag + push to registry.maxcloud.dev/{org}/{name}:{tag}
- `maxcloud images` — List registry images

**Knative Integration:**

- `imagePullSecrets` wird gesetzt, wenn Image mit Registry-URL beginnt
- Token-server auth für Registry via JWT (HMAC-SHA256)

### Neue Files

| File                                                                | Beschreibung                       |
| ------------------------------------------------------------------- | ---------------------------------- |
| `apps/api/internal/handler/registry.go`                             | Token Endpoint, Scope-Validierung  |
| `apps/cli/cmd/push.go`                                              | Docker push mit JWT-Auth           |
| `apps/cli/cmd/images.go`                                            | Images list (stub)                 |
| `apps/api/internal/store/migrations/006_create_registry_images.sql` | Tracking-Tabelle                   |
| `deploy/registry-namespace.yaml`                                    | Namespace maxcloud-registry        |
| `deploy/registry-secret.yaml`                                       | S3 Creds, JWT Secret               |
| `deploy/registry-config.yaml`                                       | Distribution Config (S3 Backend)   |
| `deploy/registry-deployment.yaml`                                   | Registry:3 Deployment (2 Replicas) |
| `deploy/registry-ingress.yaml`                                      | registry.maxcloud.dev              |

### Geänderte Files

| File                                        | Änderung                                  |
| ------------------------------------------- | ----------------------------------------- |
| `packages/shared/pkg/models/models.go`      | +RegistryTokenResponse, Image, Repository |
| `packages/shared/pkg/api/client.go`         | +GetRegistryToken()                       |
| `apps/api/internal/config/config.go`        | +RegistryURL, JWTSecret, TokenExpiry      |
| `apps/api/internal/handler/health.go`       | Handler struct erweitert                  |
| `apps/api/internal/server/server.go`        | Server struct, /registry/token Route      |
| `apps/api/internal/orchestrator/knative.go` | +imagePullSecrets, usesPrivateRegistry()  |
| `apps/api/main.go`                          | NewKnative() mit Registry-Params          |
| `apps/cli/cmd/root.go`                      | +pushCmd, imagesCmd                       |

### Neue Env-Vars

| Variable                | Default                 | Beschreibung                                           |
| ----------------------- | ----------------------- | ------------------------------------------------------ |
| `REGISTRY_URL`          | `registry.maxcloud.dev` | Registry Domain                                        |
| `REGISTRY_JWT_SECRET`   | —                       | HMAC Secret für JWT-Signierung (required für Registry) |
| `REGISTRY_TOKEN_EXPIRY` | `1h`                    | Token-Gültigkeit                                       |

### Neue GitHub Secrets (für CD)

| Secret                  | Beschreibung                      |
| ----------------------- | --------------------------------- |
| `HETZNER_S3_ACCESS_KEY` | Hetzner Object Storage Access Key |
| `HETZNER_S3_SECRET_KEY` | Hetzner Object Storage Secret Key |

### Workflow

```bash
# Image pushen
maxcloud push nginx:latest --name myapp
# Output: Pushed: registry.maxcloud.dev/{org-id}/myapp:latest

# Deployen
maxcloud deploy registry.maxcloud.dev/{org-id}/myapp:latest --name myapp
```

### Dependencies

- `github.com/golang-jwt/jwt/v5` in apps/api

</last-session>
