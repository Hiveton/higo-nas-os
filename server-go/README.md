# HiGoOS Go Backend

`server-go` is the Go control plane for the HiGoOS AI NAS desktop. Its job is to turn the current `web-pc` prototype into a real NAS backend: REST APIs, task/event streams, Linux adapters, audit, ACL, AI visibility, and safe Agent execution.

The current repository slice is a backend foundation. It already contains a Go module, API/worker/CLI entrypoints, HTTP router tests, request envelope helpers, secure headers, CORS/preflight handling, dev/test-friendly session guarding, request IDs, deterministic `devstub` adapters, and a JSON state layer for mutable development data. The fuller OpenAPI contract, database-backed domain services, background jobs, and Linux adapters are the next implementation layers described by the backend development plan.

## Current Layout

```text
server-go/
  go.mod                         Go module higoos/server-go
  .env.example                   Current local environment keys
  README.md                      Backend development and deployment guide
  api/                           API contract directory
  cmd/
    higo-api/                    HTTP API entrypoint
    higo-worker/                 Devstub worker heartbeat entrypoint
    higoctl/                     Local admin CLI with doctor command
  db/
    migrations/                  Database migration directory
    seeds/                       Seed data directory
  fixtures/
    nas-root/                    Mac-safe fixture NAS filesystem root
      backup-archive/
      downloads/
      finance-receipts/
      home-space/
      photos-and-media/
      team-space/
  internal/
    devstub/                     Deterministic Mac/test adapters and seed desktop data
    httpapi/                     HTTP router, middleware, JSON response handling
    platform/                    Config, logger, request ID, response envelope
```

Planned package boundaries beyond the current foundation are: auth, iam, audit, settings, files, storage, monitoring, backup, downloads, media, docker, remote, aiindex, assistant, agents, apps, and Linux adapters.

## Project Goals

- Serve the Vue desktop under `web-pc` with stable `/api/v1` contracts.
- Keep Mac development safe by default through deterministic devstub adapters and read-only host storage discovery.
- Use Linux adapters only on NAS hosts for filesystem, SMART, storage pools, Docker, network, systemd, SMB/NFS/WebDAV, ffmpeg, hardware telemetry, and logs.
- Enforce identity, ACL, AI visibility, risk confirmation, audit, rollback, and model policy before AI or Agent actions touch user data.
- Keep NAS fundamentals independent from AI services so files, storage, backup, and monitoring stay usable when AI indexing or model providers fail.

## Mac Development

Mac development runs in `devstub` mode. It must not execute destructive host commands, scan arbitrary user directories, manage Docker/systemd, or change local network settings. Storage pool and disk APIs read mounted host filesystems through `df -kP`; fixture file data lives under `server-go/fixtures/nas-root`, while mutable API state is written to `HIGO_STATE_DIR` or the default user cache directory.

The JSON state layer currently persists:

- desktop session layout and dock preferences
- file metadata updates such as tags on fixture-backed NAS files
- system settings and normalized model/privacy policy
- storage tasks and monitoring alerts
- download tasks and active speed profile
- Docker dev runtime state, logs, and resource limits
- remote access channel, MFA, policy, token, devices, and share scan state
- backup job runtime state
- app center install/update/start/stop state
- security identities, AI policies, risk actions, share revocation, and audit state
- media albums, people merges, generated memories, subtitles, transcode jobs, and shares
- assistant threads, messages, pending actions, and confirmations
- Agent workflow runs and event history
- AI file steward suggestion previews, confirmations, dismissals, rollbacks, and audit state

Current checks:

```sh
cd server-go
go test ./...
```

Current HTTP surface available through the router tests:

- `GET /healthz`
- `GET /readyz`
- `GET /api/v1/system/info`
- `GET /api/v1/desktop/apps`
- `GET /api/v1/desktop/windows`
- `GET /api/v1/desktop/session`
- `PUT /api/v1/desktop/session`

Current local service commands:

```sh
cd server-go
go run ./cmd/higo-api
go run ./cmd/higo-worker
go run ./cmd/higoctl doctor
```

Planned local workflow:

```sh
cd server-go
make dev
```

`make dev` should start PostgreSQL, the API server, the worker, and the `web-pc` dev server after the Makefile and database layer are added. Until then, use `go test ./...` as the reliable backend verification command and run frontend checks from `web-pc`.

## Linux Deployment

Linux deployment is the production target. A release should install the API service, worker service, admin CLI, database migrations, environment file, log rotation, health checks, reverse proxy config, storage path allowlist, and backup/export procedure.

Production service commands after binaries are built:

```sh
higo-api --config /etc/higoos/server.env
higo-worker --config /etc/higoos/server.env
higoctl migrate up
higoctl bootstrap-admin
higoctl doctor
higoctl backup export --output /var/backups/higoos
```

Expected systemd units:

- `higo-api.service`: HTTP API, `/healthz`, `/readyz`, `/api/v1`, event stream, static frontend handoff through reverse proxy.
- `higo-worker.service`: indexing, media, backup, downloads, monitoring, Agent execution, task event publication.
- `higoos.target`: optional grouping unit for ordered startup and shutdown.

Linux-only behavior includes direct access to `/sys`, `/proc`, `lsblk`, `smartctl`, storage pool tools, Docker Engine, `systemctl`, journal logs, network config, SMB/NFS/WebDAV service config, ffmpeg, GPU/NPU probes, UPS telemetry, and hardware sensors. These must stay behind adapter interfaces and capability checks.

## Environment Variables

The current config loader supports:

| Name | Default | Purpose |
| --- | --- | --- |
| `HIGO_APP_NAME` | `HiGoOS` | Display/service name returned by system info. |
| `HIGO_ENV` | `dev` | Runtime environment such as `dev`, `test`, or `prod`. |
| `HIGO_VERSION` | `dev` | Build/version string surfaced to clients and logs. |
| `HIGO_HTTP_ADDR` | `:8080` | API listen address when the API binary is added. |
| `HIGO_STATE_DIR` | user cache `higoos/state` | JSON state directory for mutable devstub-backed APIs. |

Planned production variables:

| Name | Purpose |
| --- | --- |
| `HIGO_DATABASE_URL` | PostgreSQL connection string. |
| `HIGO_REDIS_URL` | Optional Redis cache/pubsub connection. |
| `HIGO_ADAPTER_MODE` | `devstub` on Mac/test, `linux` on NAS hosts. |
| `HIGO_NAS_ROOTS` | Comma-separated allowlist of file roots exposed to NAS services. |
| `HIGO_PUBLIC_ORIGIN` | Frontend origin for CORS, cookies, and generated links. |
| `HIGO_SESSION_KEY_FILE` | Path to session signing/encryption key material. |
| `HIGO_MODEL_POLICY` | Default AI routing policy: local, hybrid, private, or cloud-enhanced. |
| `HIGO_AUDIT_RETENTION_DAYS` | Minimum audit retention window. |
| `HIGO_LOG_LEVEL` | Structured log level. |

## API and Events

All domain APIs live under `/api/v1`. JSON responses use the envelope already present in `internal/platform`:

```json
{
  "data": {},
  "requestId": "req_..."
}
```

Errors use:

```json
{
  "error": {
    "code": "permission_denied",
    "message": "permission denied"
  },
  "requestId": "req_..."
}
```

Long-running operations should return a task ID and publish progress through the event stream. The frontend consumes task and event state for SMART scans, repairs, snapshots, download progress, media jobs, Docker restarts, diagnostics, assistant actions, and Agent workflow runs.

## Test Commands

Backend:

```sh
cd server-go
go test ./...
```

Frontend smoke checks, run from the repository root after frontend dependencies are installed:

```sh
cd web-pc
npm run test:interactions
npm run build
```

Contract checks should compare handler registration, the OpenAPI contract, and the generated TypeScript client once those artifacts are added.
