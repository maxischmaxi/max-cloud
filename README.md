# max-cloud — Deutsche Cloud Run Alternative

Serverlose Container-Plattform mit Scale-to-Zero für den deutschen Markt. Open Source. DSGVO-konform. Gehostet in Deutschland.

Europa strebt nach digitaler Souveränität und Unabhängigkeit von US-Cloud-Anbietern. Im deutschen Markt gibt es **keine** serverlose Container-Plattform mit Scale-to-Zero (wie Google Cloud Run). Scaleway (Frankreich) ist der einzige europäische Anbieter mit vergleichbarem Angebot. **Das ist eine klare Marktlücke.**

---

## Was max-cloud leistet

| Feature | Beschreibung |
|---|---|
| Container Deployment | Beliebiges Docker/OCI-Image deployen, bekommt HTTPS-Endpoint |
| Auto-Scaling | 0 bis N Instanzen basierend auf Request-Concurrency |
| Scale-to-Zero | Keine laufenden Instanzen wenn kein Traffic |
| HTTPS + TLS | Automatisch für jede Service-URL |
| Pay-per-Use | Abrechnung pro vCPU-Sekunde und GiB-Sekunde |
| Container Registry | Images speichern und verwalten |
| Logging/Monitoring | Strukturierte Logs, Metriken |
| Env Vars & Secrets | Konfiguration zur Laufzeit injizieren |
| Custom Domains | Eigene Domain mit automatischem Zertifikat |
| Revisions | Immutable Snapshots, Traffic-Splitting zwischen Versionen |

---

## Architektur

```text
[User Request]
       │
[DNS (PowerDNS + ExternalDNS)]
       │
[L4/L7 Load Balancer (MetalLB)]
       │
[TLS Termination (cert-manager + Let's Encrypt)]
       │
[Ingress Controller (Kourier/Envoy)]
       │
[Knative Serving]
[Activator ↔ Autoscaler]
       │
[Container Instance (containerd + gVisor)]
       │
[Harbor Registry]  [Vault/Secrets]  [Prometheus + Loki + Grafana]
```

### Tech-Stack (Ziel)

- **Bare Metal:** Hetzner Dedicated, DE-Rechenzentrum
- **OS:** Talos Linux (immutable)
- **Orchestrierung:** Kubernetes (k3s oder k8s mit Cluster API)
- **Serverless-Kern:** Knative Serving (CNCF graduated)
- **Ingress:** Kourier (leichtgewichtiger Knative-Ingress)
- **Registry:** Harbor (CNCF graduated)
- **TLS:** cert-manager + Let's Encrypt
- **DNS:** ExternalDNS + PowerDNS
- **Observability:** Prometheus + Loki + Grafana
- **API Server:** Go (chi Router)
- **CLI:** Go (Cobra)

---

## Developer-Dokumentation

### Voraussetzungen

| Tool | Version |
|---|---|
| Go | ≥ 1.25 |
| Node.js | ≥ 24 |
| pnpm | ≥ 9.15 |
| Turbo | wird via pnpm installiert |

### Setup

```bash
git clone <repo-url> && cd max-cloud
pnpm install          # installiert turbo
pnpm turbo build      # kompiliert alle Go-Binaries
pnpm turbo test       # führt alle Tests aus
```

### Projektstruktur

```sql
max-cloud/
├── apps/
│   ├── api/                          # REST API Server
│   │   ├── main.go                   # Entry, graceful shutdown
│   │   └── internal/
│   │       ├── config/config.go      # PORT env, LogLevel
│   │       ├── server/server.go      # chi Router, Middleware, DI
│   │       ├── store/store.go        # In-Memory Store (sync.RWMutex)
│   │       ├── store/store_test.go
│   │       ├── handler/health.go     # HTTP Handler (CRUD + Health)
│   │       └── handler/handler_test.go
│   └── cli/                          # CLI Tool
│       ├── main.go                   # Entry
│       └── cmd/
│           ├── root.go               # Root-Command, --api-url Flag
│           ├── deploy.go             # deploy --name NAME [--env K=V]
│           ├── list.go               # list (Tabellen-Output)
│           ├── delete.go             # delete <service-id>
│           └── logs.go               # logs (Stub)
├── packages/
│   └── shared/                       # Geteiltes Go-Modul
│       └── pkg/
│           ├── models/models.go      # Service, DeployRequest, etc.
│           └── api/
│               ├── client.go         # HTTP-Client für die API
│               └── client_test.go
├── turbo.json
├── pnpm-workspace.yaml
└── package.json
```

### Go-Module

Das Projekt besteht aus drei Go-Modulen, die über `replace`-Direktiven lokal verknüpft sind:

| Modul | Pfad | Abhängigkeiten |
|---|---|---|
| `github.com/max-cloud/api` | `apps/api/` | chi/v5, uuid, shared |
| `github.com/max-cloud/cli` | `apps/cli/` | cobra, shared |
| `github.com/max-cloud/shared` | `packages/shared/` | — |

Neue Dependency hinzufügen:

```bash
cd apps/api    # oder apps/cli
go get github.com/example/pkg
go mod tidy
```

### API starten & testen

```bash
# API starten (Port 8080)
./apps/api/bin/api &

# Service deployen
./apps/cli/bin/maxcloud deploy --name myapp nginx:latest

# Services auflisten
./apps/cli/bin/maxcloud list

# Service löschen
./apps/cli/bin/maxcloud delete <service-id>

# API stoppen
kill %1
```

Oder direkt mit curl:

```bash
# Health Check
curl http://localhost:8080/healthz

# Service erstellen
curl -X POST http://localhost:8080/api/v1/services \
  -H "Content-Type: application/json" \
  -d '{"name":"myapp","image":"nginx:latest","env_vars":{"PORT":"3000"}}'

# Alle Services auflisten
curl http://localhost:8080/api/v1/services

# Einzelnen Service abrufen
curl http://localhost:8080/api/v1/services/<id>

# Service löschen
curl -X DELETE http://localhost:8080/api/v1/services/<id>
```

### API-Endpunkte

| Methode | Pfad | Beschreibung | Response |
|---|---|---|---|
| GET | `/healthz` | Health Check | `{"status":"ok"}` |
| POST | `/api/v1/services` | Service erstellen | 201 + Service |
| GET | `/api/v1/services` | Alle Services | 200 + Service[] |
| GET | `/api/v1/services/{id}` | Einzelner Service | 200 + Service / 404 |
| DELETE | `/api/v1/services/{id}` | Service löschen | 204 / 404 |

### Tests

```bash
pnpm turbo test                           # alle Tests
cd apps/api && go test ./...              # nur API-Tests
cd packages/shared && go test ./...       # nur Client-Tests
```

---

## MVP-Scope

### P0 — Must Have

- [x] Docker-Image deployen
- [ ] Auto-Scaling (1 bis N) basierend auf Request-Concurrency
- [ ] Scale-to-Zero
- [ ] HTTPS-Endpoint mit automatischem TLS
- [ ] Plattform-Domain (`*.maxcloud.dev`)
- [x] Environment Variables
- [x] CLI (deploy, list, delete)
- [x] REST API
- [ ] Container Logs (stdout/stderr)

### P1 — Wichtig für frühe Kunden

- [ ] Private Container Registry (Harbor)
- [ ] Secrets
- [ ] Custom Domains mit automatischem TLS
- [ ] Einfache Web-Konsole
- [ ] Billing (per-Sekunde)

### P2 — Kann warten

- [ ] Revision Management / Rollback
- [ ] Traffic Splitting
- [ ] Detaillierte Metriken-Dashboards

### P3 — Post-MVP

- [ ] Event Triggers (Knative Eventing)
- [ ] VPC / Private Networking
- [ ] GPU Support
- [ ] Multi-Region

### MVP-Vereinfachungen gegenüber Google Cloud Run

1. **Billing:** Per-Minute statt per-100ms — dramatisch einfacher
2. **Isolation:** Namespace-Isolation + Network Policies statt gVisor (kommt später)
3. **Region:** Ein Standort (z.B. Nürnberg/Falkenstein)
4. **SLA:** Kein formelles SLA während Beta
5. **UI:** CLI-first, minimale Web-Oberfläche

---

## Komplexität pro Komponente

### Verfügbar als Open Source (Konfiguration + Integration)

| Komponente | Aufwand | Komplexität |
|---|---|---|
| Kubernetes Cluster | HA, etcd, Upgrades, Lifecycle | Hoch |
| Knative Serving | Setup, Tuning, Cold-Start-Optimierung | Mittel |
| Container Runtime + Sandboxing (gVisor) | Sicherheits-Hardening | Mittel |
| Ingress + Load Balancing (Kourier, MetalLB) | Health Checks, Rate Limiting | Mittel |
| TLS (cert-manager + Let's Encrypt) | Wildcard-Zertifikate, Renewal | Niedrig |
| Container Registry (Harbor) | Multi-Tenant, Vulnerability Scanning | Niedrig–Mittel |
| DNS (PowerDNS + ExternalDNS) | Wildcard DNS, Domain-Verifizierung | Mittel |
| Secrets (Vault + External Secrets Operator) | Tenant-Isolation | Mittel |
| Observability (Prometheus, Loki, Grafana) | Per-Tenant Log-Isolation | Hoch |
| Revision Mgmt / Traffic Splitting | Knative-nativ vorhanden | Niedrig |

### Muss selbst gebaut werden

| Komponente | Aufwand | Komplexität |
|---|---|---|
| Control Plane API (REST/gRPC) | 3–6 Monate, 3–4 Devs | Sehr Hoch |
| Web Console | 3–6 Monate, 2–3 Frontend Devs | Sehr Hoch |
| CLI Tool | 4–6 Wochen, 1 Dev | Mittel |
| Billing / Metering Engine | 3–6 Monate, 2–3 Devs | Sehr Hoch |
| Multi-Tenant Isolation | 2–4 Monate, 2 Devs | Sehr Hoch |

**Fazit:** ~70% der Infrastruktur ist als Open Source verfügbar (Knative ist der Kern). Die restlichen ~30% (API, Billing, Console, Multi-Tenancy) sind der schwierigste Teil.

---

## Regulatorik & Compliance (DE/EU)

| Thema | Beschreibung | Komplexität |
|---|---|---|
| DSGVO | Datenresidenz in DE, AVV für Kunden, Löschkonzept, Breach Notification (72h) | Mittel |
| BSI C5 | Deutscher Cloud-Sicherheitsstandard. Type 2 Attestierung kostet 100–300k EUR. Pflicht für Healthcare seit Juli 2025 | Sehr Hoch |
| Gaia-X | Maschinenlesbare Self-Descriptions. Marketing-Vorteil, nicht rechtlich verpflichtend | Niedrig |
| NIS2 | Cybersecurity-Anforderungen falls "wesentliche Einrichtungen" bedient werden | Hoch |
| EU Data Act | Workload-Portabilität & Anbieterwechsel. Knative/OCI = gut positioniert | Niedrig |

**Empfehlung:** BSI C5 von Tag 1 mitdenken (Audit-Logging, Zugriffskontrolle), auch wenn formelle Attestierung erst post-MVP kommt.

---

## Europäische Wettbewerber

| Anbieter | Land | Serverless Containers? | Scale-to-Zero? |
|---|---|---|---|
| **Scaleway** | FR | Ja | Ja — einziger EU-Anbieter mit Cloud-Run-Äquivalent |
| OVHcloud | FR | Teilweise (Knative auf MKS) | Via Knative |
| Hetzner | DE | Nein | Nein |
| IONOS | DE | Nein | Nein |
| gridscale | DE | Nein | Nein (aber BSI C5) |
| Sliplane | EU | Managed Container | Nein |

**Marktlücke:** Kein deutscher Anbieter bietet serverlose Container mit Scale-to-Zero an.

---

## Team & Timeline

### MVP-Team (Minimum)

| Rolle | Anzahl |
|---|---|
| Platform/Infra Engineers (K8s + Knative) | 3–4 |
| Backend Engineers (API + Billing) | 3–4 |
| Frontend Engineer | 1–2 |
| SRE / Operations | 2–3 |
| Security Engineer | 1 |
| Product Manager | 1 |
| **Gesamt** | **11–15 Personen** |

### Timeline bis Public Beta

| Phase | Dauer | Ergebnis |
|---|---|---|
| Phase 0: Infrastruktur | 2–3 Monate | K8s auf Bare Metal in DE-Rechenzentrum |
| Phase 1: Core Platform | 3–4 Monate | Knative, Harbor, cert-manager, Basis-API |
| Phase 2: Product Layer | 3–4 Monate | Control Plane API, CLI, Web-Konsole, Tenant-Isolation |
| Phase 3: Billing + Polish | 2–3 Monate | Metering, Billing, Custom Domains, Observability |
| Phase 4: Beta | 2–3 Monate | Lasttests, Security Audit, Beta-Nutzer |
| **Gesamt** | **12–17 Monate** | |

### MVP-Infrastrukturkosten

| Position | Monatlich |
|---|---|
| 3× Hetzner AX102 (K8s Worker) | ~900 EUR |
| 2× Hetzner AX52 (Control Plane, Registry, Monitoring) | ~300 EUR |
| Load Balancer, DNS | ~80 EUR |
| **Gesamt** | **~1.300 EUR/Monat** |

---

## Risiken

| Risiko | Wahrscheinlichkeit | Impact | Mitigation |
|---|---|---|---|
| Cold-Start-Latenz zu hoch | Mittel | Hoch | Knative Activator tunen, Minimum-Instances Option |
| Multi-Tenant Security Breach | Niedrig | Sehr Hoch | gVisor, Pentesting, Namespace-Isolation |
| Kundengewinnung | Hoch | Sehr Hoch | "Datensouveränität"-Messaging, kompetitive Preise |
| Hetzner/IONOS baut selbst | Mittel | Hoch | Schnell sein, Community aufbauen |
| BSI C5 Kosten/Aufwand | Hoch | Mittel | Design-for-Compliance von Tag 1 |

---

## Strategische Empfehlungen

1. **Hetzner Bare Metal** als Infrastruktur-Basis — bestes Preis/Leistungsverhältnis in DE
2. **Knative Serving** als Kern — CNCF graduated, liefert Scale-to-Zero out of the box
3. **CLI-first** — Developer-fokussierte Plattformen gewinnen über CLI, Web-Console kommt später
4. **Per-Sekunde abrechnen** statt per-Request — 10× einfacher zu implementieren
5. **BSI C5 von Anfang an mitdenken** — auch ohne formelle Attestierung
6. **Deutscher Mittelstand als Zielgruppe** — braucht DSGVO-konforme Container-Plattform, findet Hyperscaler zu komplex
7. **CLI und SDK Open Source** — baut Vertrauen und Community auf

---

## Gesamtbewertung

**Machbar, aber kein Wochenendprojekt.**

- Die Infrastruktur-Basis (Knative + K8s + Harbor) ist als Open Source verfügbar und gut dokumentiert
- Der eigentliche Aufwand liegt in der Produktschicht (API, Billing, Multi-Tenancy, Console)
- Ein erfahrenes Team von 11–15 Personen braucht 12–17 Monate bis zur Public Beta
- Infrastrukturkosten sind mit ~1.300 EUR/Monat für MVP überschaubar
- Größte Herausforderung: Nicht die Technik, sondern Kundengewinnung und operativer Betrieb 24/7
