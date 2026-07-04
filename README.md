## Readme and Go Code written with AI

# Config Service

A small configuration management service built in Go, deployed on a local Kubernetes cluster with PostgreSQL.

## Architecture

```
┌─────────────────────────────────────────────────┐
│                 Kind Cluster                     │
│                                                  │
│  ┌──────────────┐       ┌─────────────────────┐  │
│  │ config-service│──────▶│  PostgreSQL (Bitnami)│  │
│  │  (Deployment) │       │    (StatefulSet)     │  │
│  │  port: 8080   │       │    port: 5432        │  │
│  └──────┬───────┘       └─────────────────────┘  │
│         │                                        │
│  ┌──────┴───────┐                                │
│  │   Service     │                                │
│  │  (ClusterIP)  │                                │
│  └──────┬───────┘                                │
│         │                                        │
└─────────┼────────────────────────────────────────┘
          │ kubectl port-forward
          ▼
    localhost:8080
```

### Components

| Component | Tool | Purpose |
|-----------|------|---------|
| Kubernetes cluster | Kind | Local single-node cluster |
| Application | Go + chi router | HTTP API for config CRUD |
| Database | PostgreSQL 16 (Bitnami Helm chart) | Persistent storage |
| IaC | Terraform (Helm + Kubernetes providers) | Provision PostgreSQL and app deployment |
| Packaging | Helm chart | Application deployment to K8s |
| Automation | Shell scripts | End-to-end setup, deploy, validate, teardown |

### Application Structure

```
cmd/server/          main entrypoint
internal/
  handler/           HTTP handlers (thin, validation + response shaping)
  service/           business logic
  repository/        PostgreSQL persistence (sqlx)
  model/             domain types and DTOs
migrations/          SQL migration files (embedded, run on startup)
```

Handlers delegate to services, services delegate to repositories. No persistence logic in handlers.

---

## API Contracts

### Health Check

```
GET /ping
```

Response: `200 OK`
```
pong
```

Used as the Kubernetes **liveness** probe. Confirms the process is alive.

### Readiness Check

```
GET /healthz
```

Response: `200 OK`
```json
{"status": "ok"}
```

Response on failure: `503 Service Unavailable`
```json
{"status": "unavailable", "error": "database connection failed"}
```

Used as the Kubernetes **readiness** probe. Verifies the app can reach PostgreSQL.

### Get Config

```
GET /configs/{id}
```

- `id` is a user-defined string identifier (e.g. `cfg_1`, `my-app-config`). Max 255 characters, alphanumeric plus `-` and `_`.
- Returns `200 OK` with the config record.
- Returns `404 Not Found` if no config exists with that ID.
- Returns `400 Bad Request` if `id` fails validation.

Response:
```json
{
  "id": "cfg_1",
  "host": "localhost",
  "port": 8080,
  "app_name": "config-service",
  "log_level": "INFO",
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

### Upsert Config

```
POST /configs
```

Creates a new config or updates an existing one (matched by `id`).

Request body:
```json
{
  "id": "cfg_1",
  "host": "localhost",
  "port": 8080,
  "app_name": "config-service",
  "log_level": "INFO"
}
```

**Upsert behavior:** Uses PostgreSQL `INSERT ... ON CONFLICT (id) DO UPDATE`. If a record with the given `id` exists, all fields are overwritten and `updated_at` is refreshed. This makes the operation idempotent — the same request produces the same result regardless of how many times it runs.

Response on create: `201 Created`
Response on update: `200 OK`

Both return the full config record.

Validation:
- `id`: required, 1-255 chars, pattern `^[a-zA-Z0-9_-]+$`
- `host`: required, non-empty string
- `port`: required, integer 1-65535
- `app_name`: required, non-empty string
- `log_level`: required, one of `DEBUG`, `INFO`, `WARN`, `ERROR`

Returns `400 Bad Request` with error details on validation failure.

### Metrics

```
GET /metrics
```

Prometheus-compatible metrics endpoint exposing request counts, latencies, and Go runtime metrics.

---

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS configs (
    id         VARCHAR(255) PRIMARY KEY,
    host       VARCHAR(255) NOT NULL,
    port       INTEGER NOT NULL CHECK (port >= 1 AND port <= 65535),
    app_name   VARCHAR(255) NOT NULL,
    log_level  VARCHAR(10) NOT NULL CHECK (log_level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Design decisions:**
- `id` as VARCHAR primary key (not auto-increment) because the assignment specifies user-provided IDs like `cfg_1`.
- `CHECK` constraints on `port` range and `log_level` enum to enforce validity at the database level.
- `created_at` / `updated_at` timestamps for auditability.
- No additional indexes beyond the primary key — the only query pattern is lookup by `id`, which the PK index covers.

**Migrations** are embedded in the Go binary using `embed.FS` and run automatically on startup via `golang-migrate`. This keeps schema management simple and ensures the database is always at the correct version when the app starts.

---

## Configuration and Secrets

### How config is supplied

The application reads configuration from environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP listen port | `8080` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database user | `configservice` |
| `DB_PASSWORD` | Database password | (required) |
| `DB_NAME` | Database name | `configservice` |
| `DB_SSLMODE` | SSL mode | `disable` |
| `LOG_LEVEL` | Application log level | `info` |

### How secrets are managed locally

- PostgreSQL password is stored in a Kubernetes Secret created by Terraform.
- The same secret is referenced by both the PostgreSQL StatefulSet (via Bitnami chart values) and the config-service Deployment.
- Secrets are injected as environment variables into pods.

### What changes for production

- Use an external secret manager (AWS Secrets Manager, HashiCorp Vault, or Kubernetes External Secrets Operator).
- Enable SSL for database connections (`DB_SSLMODE=require`).
- Use separate credentials for app vs admin database access.
- Rotate secrets on a schedule.

---

## Deployment Flow

### Prerequisites

- Docker Desktop (running)
- kubectl, Helm, Terraform, Kind, Go (installed via `brew`)

### Quick Start

```bash
# Full setup: cluster + infrastructure + app + validation
./scripts/setup.sh

# Validate everything is working
./scripts/validate.sh

# Access the service
kubectl port-forward svc/config-service 8080:8080

# Tear down
./scripts/teardown.sh
```

### Step-by-step

1. **`scripts/setup.sh`** — Creates Kind cluster, builds Docker image, loads it into the cluster.
2. **`scripts/deploy.sh`** — Runs `terraform init` and `terraform apply` to provision PostgreSQL (via Bitnami Helm chart) and deploy the config-service (via local Helm chart).
3. **`scripts/validate.sh`** — Waits for pods to be ready, runs smoke tests against the API.
4. **`scripts/teardown.sh`** — Runs `terraform destroy`, deletes the Kind cluster.

### Infrastructure as Code Split

Terraform manages:
- Kubernetes namespace
- PostgreSQL deployment (via Helm provider + Bitnami chart)
- Database secret (Kubernetes Secret)
- Config-service deployment (via Helm provider + local chart)

This keeps all infrastructure in one declarative place. The Helm charts handle Kubernetes resource templating, while Terraform orchestrates the overall provisioning.

---

## Health Checking

| Probe | Endpoint | Checks | Failure behavior |
|-------|----------|--------|-----------------|
| Liveness | `GET /ping` | Process is alive | Pod is restarted |
| Readiness | `GET /healthz` | DB connection works | Pod removed from Service endpoints |

**Startup behavior:** The app attempts to connect to PostgreSQL with retries (5 attempts, 3s backoff). If all retries fail, the process exits with a non-zero code. Kubernetes restartPolicy handles restart.

**Liveness vs readiness separation:** A liveness failure means the process is hung and should be killed. A readiness failure means the app is alive but can't serve traffic (e.g., DB is temporarily down) — it stays running and gets re-checked.

---

## Observability

- **Structured logging:** JSON format via `zerolog`. Every log line includes timestamp, level, and component.
- **Request logging:** Each HTTP request logs method, path, status, duration, and a request ID.
- **Startup logs:** Database connection status, migration results, server listen address.
- **Prometheus metrics:** Request count and duration histograms by method/path/status at `/metrics`.
- **Probes:** Liveness and readiness endpoints for Kubernetes health monitoring.

### Troubleshooting

```bash
# Check pod status
kubectl get pods

# Check app logs
kubectl logs -l app=config-service

# Check PostgreSQL logs
kubectl logs -l app.kubernetes.io/name=postgresql

# Describe pod for events (useful for crash loops)
kubectl describe pod -l app=config-service

# Check if service endpoints are populated
kubectl get endpoints config-service
```

---

## Testing Strategy

### Application Tests
- Unit tests for handler validation logic
- Integration tests for repository layer (against a real PostgreSQL via Docker in CI)

### Infrastructure Validation
- `scripts/validate.sh` runs after deployment:
  - Checks all pods are Running/Ready
  - `GET /ping` returns `pong`
  - `POST /configs` creates a record
  - `GET /configs/{id}` retrieves it
  - Verifies response structure

### CI Pipeline
- `.github/workflows/ci.yml` runs on push:
  - Go build and test
  - Lint (golangci-lint)
  - Docker image build

---

## Known Limitations

- **Single replica:** The deployment runs one replica. For production, increase replicas and consider connection pooling (pgbouncer).
- **No TLS:** The service runs plain HTTP. In production, use an ingress controller with TLS termination.
- **Local secrets:** Kubernetes Secrets are base64-encoded, not encrypted at rest. Production needs an external secret manager.
- **No backup:** PostgreSQL data lives in a Kind cluster PV. No backup strategy. Production needs automated backups.
- **Port forwarding:** Service is accessed via `kubectl port-forward`. Production would use an ingress or load balancer.
- **Schema migrations on startup:** Simple for this use case, but in production with multiple replicas, use a separate migration job to avoid race conditions.

---

## Repository Structure

```
config-service/
├── cmd/server/              # Application entrypoint
│   └── main.go
├── internal/
│   ├── handler/             # HTTP handlers
│   ├── service/             # Business logic
│   ├── repository/          # Database access
│   └── model/               # Domain types
├── migrations/              # SQL migration files
├── helm/
│   └── config-service/      # Helm chart for the app
├── terraform/               # IaC (PostgreSQL + app provisioning)
├── scripts/                 # Automation scripts
│   ├── setup.sh
│   ├── deploy.sh
│   ├── validate.sh
│   └── teardown.sh
├── .github/workflows/       # CI pipeline
├── Dockerfile               # Multi-stage build
├── go.mod
└── README.md
```
