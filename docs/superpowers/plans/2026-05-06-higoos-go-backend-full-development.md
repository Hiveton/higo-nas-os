# HiGoOS Go Backend Full Development Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the complete Go backend for HiGoOS so the current `web-pc` desktop becomes a real NAS control plane backed by Linux system services, storage, files, AI indexing, Agent execution, audit, and real-time events.

**Architecture:** The backend is a modular Go monolith with clear internal packages and Linux adapters at the edge. The API is contract-first through OpenAPI, with REST for resource operations, WebSocket/SSE for desktop status and task updates, and background workers for NAS, AI, download, media, backup, and Agent jobs. Mac development uses safe stub adapters and fixture storage; Linux deployment uses real adapters for filesystem, SMART, mdadm/ZFS/Btrfs, Docker, systemd, networking, SMB/NFS/WebDAV, and hardware telemetry.

**Tech Stack:** Go 1.23+, `net/http` with `chi` or Gin, OpenAPI 3.1, PostgreSQL 16 + pgvector, Redis optional for cache/pubsub, `sqlc` or Ent, goose migrations, structured logging with `slog`/zap, WebSocket/SSE, systemd, Docker Engine API, Linux CLI adapters, Vue 3 generated TypeScript API client.

---

## 1. Current Project Findings

The repository currently contains two product documents and one PC web frontend:

- `docs/superpowers/specs/2026-05-05-higoos-ai-nas-architecture-design.md`: complete AI NAS architecture covering NAS base services, AI data intelligence, Agent execution, security governance, multi-device UX, ecosystem, data flows, and success criteria.
- `docs/superpowers/plans/2026-05-06-higoos-web-pc-implementation.md`: previous plan for building the Vue PC desktop prototype.
- `web-pc`: Vue 3 + Vite + TypeScript desktop shell with local seed data and component-local state.

The frontend has no real HTTP client yet. It imports static data from `web-pc/src/data/higoos.ts` and stores most interaction state inside Vue components. The existing validation passed:

- `npm run test:interactions`: `Interaction coverage OK: 39 checks`
- `npm run build`: Vite production build succeeded

Backend planning must therefore start by converting static UI state into stable API contracts instead of changing visual layout first.

## 2. Frontend Surface That Must Be Backed

`web-pc/src/data/higoos.ts` defines global desktop data:

- Dock apps and window configs.
- File rows, folders, storage pools, disks.
- AI steward suggestions and audit entries.
- Agent templates and workflow nodes.
- Assistant messages.
- Top metrics and alerts.

Window components define additional domain state:

- `FileManagerWindow.vue`: folder tree, semantic search, file details, preview, share settings, smart tags.
- `StorageMonitorWindow.vue`: storage pools, disks, SMART actions, repair/snapshot actions.
- `AiStewardWindow.vue`: suggestions, risk levels, execution/dismissal log.
- `AgentWorkbenchWindow.vue`: templates, workflow simulation, tool permissions, confirmation.
- `PhotoMediaWindow.vue`: timeline, people, places, devices, albums, memories, subtitle/transcode jobs, sharing.
- `DownloadCenterWindow.vue`: BT/HTTP/magnet/RSS tasks, speed modes, archive integration.
- `DockerWindow.vue`: compose stacks, containers, ports, mounts, env, resource limits, logs, start/stop/restart.
- `SecurityCenterWindow.vue`: identities, AI policies, share links, risk actions, audit, rollback.
- `DeviceMonitorWindow.vue`: metrics, service states, logs, alerts, diagnostics.
- `SystemSettingsWindow.vue`: account, network, model, AI, notification, update, privacy, audit, backup settings.
- `RemoteAccessWindow.vue`: remote channel, MFA, tunnel mode, bound devices, policies, login alerts, share scans.
- `AiAssistantPanel.vue` and `TopBar.vue`: semantic search, assistant chat, model policies, notifications, suggested actions.
- `App.vue`: desktop session state, open/minimized/maximized windows, dock layout, context menus, utility launching.

## 3. Backend Repository Structure

Create a backend service under `server-go` and keep frontend integration explicit:

- `server-go/cmd/higo-api/main.go`: HTTP API, WebSocket/SSE, health checks.
- `server-go/cmd/higo-worker/main.go`: background workers for indexing, backup, media, downloads, Agent execution.
- `server-go/cmd/higoctl/main.go`: local admin CLI for Linux diagnostics, migrations, bootstrap user, service checks.
- `server-go/internal/platform`: config, logging, errors, clock, IDs, transactions, event bus.
- `server-go/internal/httpapi`: routers, middleware, OpenAPI handlers, request validation, response envelopes.
- `server-go/internal/auth`: login, sessions, MFA, device binding, WebAuthn-ready hooks, API tokens.
- `server-go/internal/iam`: users, roles, groups, spaces, ACL, application/Agent permissions.
- `server-go/internal/audit`: append-only audit log, risk actions, rollback registry.
- `server-go/internal/settings`: system settings, model policy, privacy policy, notification preferences.
- `server-go/internal/files`: files, folders, metadata, tags, preview, shares, recycle bin, versions.
- `server-go/internal/storage`: disks, pools, volumes, RAID/ZFS/Btrfs adapters, SMART, snapshots.
- `server-go/internal/monitoring`: metrics, logs, alerts, diagnostics, notification dispatch.
- `server-go/internal/backup`: backup plans, runs, snapshots, restore points, integrity checks.
- `server-go/internal/downloads`: download tasks, RSS, magnet/BT/HTTP adapters, speed profiles, archive hooks.
- `server-go/internal/media`: photos, videos, music, albums, people, places, scraping, subtitles, transcoding.
- `server-go/internal/docker`: Docker Engine and Compose management, resource limits, logs, ports, mounts.
- `server-go/internal/remote`: DDNS, tunnels, MFA policies, bound devices, login alerts, share safety scans.
- `server-go/internal/aiindex`: parse/OCR/transcribe/summarize/tag/vector/index pipelines.
- `server-go/internal/assistant`: chat, semantic search, tool routing, permission-filtered answers.
- `server-go/internal/agents`: templates, workflows, tool registry, execution plans, confirmations, rollback.
- `server-go/internal/apps`: app center, plugin registry, permission declarations.
- `server-go/internal/linux`: production adapters for systemd, SMART, filesystem, network, Docker, SMB/NFS/WebDAV.
- `server-go/internal/devstub`: Mac-safe adapters returning deterministic fixtures for local development.
- `server-go/api/openapi.yaml`: single source of truth for frontend/backend contracts.
- `server-go/db/migrations`: SQL migrations.
- `server-go/db/queries`: `sqlc` queries if `sqlc` is selected.
- `server-go/tests`: integration, contract, adapter, and e2e test harness.
- `web-pc/src/api`: generated TypeScript client and API runtime.
- `web-pc/src/stores`: frontend stores replacing local seed state.

## 4. Data Model Domains

Core tables:

- Identity: `users`, `groups`, `roles`, `sessions`, `mfa_factors`, `trusted_devices`, `api_tokens`.
- Authorization: `spaces`, `folder_acl`, `app_permissions`, `agent_permissions`, `permission_snapshots`.
- Audit/security: `audit_events`, `risk_actions`, `rollback_operations`, `share_links`, `security_findings`.
- System settings: `settings`, `model_policies`, `privacy_policies`, `notification_rules`, `system_backups`.
- Files: `file_nodes`, `file_versions`, `file_tags`, `file_favorites`, `file_previews`, `recycle_items`.
- Search/index: `index_jobs`, `document_chunks`, `semantic_embeddings`, `entity_links`, `knowledge_edges`.
- Storage: `disks`, `storage_pools`, `volumes`, `snapshots`, `smart_reports`, `storage_tasks`.
- Monitoring: `metrics_samples`, `system_logs`, `alerts`, `diagnostic_runs`.
- Backup: `backup_plans`, `backup_runs`, `backup_items`, `restore_points`, `integrity_checks`.
- Downloads: `download_tasks`, `download_sources`, `rss_subscriptions`, `speed_profiles`, `archive_rules`.
- Media: `media_items`, `albums`, `people`, `places`, `memory_runs`, `subtitle_jobs`, `transcode_jobs`.
- Docker/apps: `compose_stacks`, `containers`, `container_events`, `app_catalog`, `app_installs`.
- Remote: `remote_channels`, `tunnel_sessions`, `ddns_records`, `login_alerts`, `bound_devices`.
- Agents: `agent_templates`, `agent_instances`, `workflow_definitions`, `workflow_runs`, `tool_calls`, `confirmations`.
- Assistant: `conversation_threads`, `messages`, `assistant_actions`, `retrieval_citations`.

## 5. API Contract Map For `web-pc`

Desktop shell:

- `GET /api/v1/desktop/apps`
- `GET /api/v1/desktop/windows`
- `GET /api/v1/desktop/session`
- `PUT /api/v1/desktop/session`
- `GET /api/v1/events/stream` for metrics, alerts, task progress, assistant updates.

File Manager:

- `GET /api/v1/files/tree?space=`
- `GET /api/v1/files/search?q=&space=&type=&tags=`
- `GET /api/v1/files/{id}`
- `GET /api/v1/files/{id}/preview`
- `POST /api/v1/files/{id}/tags`
- `POST /api/v1/files/{id}/shares`
- `POST /api/v1/files/batch/move`
- `POST /api/v1/files/batch/rename`
- `POST /api/v1/files/batch/delete`
- `POST /api/v1/files/{id}/restore`

Storage Monitor:

- `GET /api/v1/storage/pools`
- `GET /api/v1/storage/disks`
- `GET /api/v1/storage/smart`
- `POST /api/v1/storage/tasks/smart-scan`
- `POST /api/v1/storage/tasks/repair`
- `POST /api/v1/storage/tasks/snapshot`
- `GET /api/v1/storage/tasks/{id}`

AI File Steward:

- `GET /api/v1/steward/suggestions`
- `POST /api/v1/steward/suggestions/{id}/preview`
- `POST /api/v1/steward/suggestions/{id}/confirm`
- `POST /api/v1/steward/suggestions/{id}/dismiss`
- `GET /api/v1/steward/audit`
- `POST /api/v1/steward/audit/{id}/rollback`

Agent Workbench:

- `GET /api/v1/agents/templates`
- `POST /api/v1/agents`
- `GET /api/v1/agents/{id}/tools`
- `POST /api/v1/workflows/preview`
- `POST /api/v1/workflows/runs`
- `POST /api/v1/workflows/runs/{id}/confirm`
- `POST /api/v1/workflows/runs/{id}/cancel`
- `GET /api/v1/workflows/runs/{id}/events`

Assistant and semantic search:

- `POST /api/v1/search/semantic`
- `POST /api/v1/assistant/threads`
- `GET /api/v1/assistant/threads/{id}`
- `POST /api/v1/assistant/threads/{id}/messages`
- `POST /api/v1/assistant/actions/{id}/confirm`

Photo and media:

- `GET /api/v1/media/items`
- `GET /api/v1/media/albums`
- `POST /api/v1/media/albums`
- `POST /api/v1/media/memories`
- `POST /api/v1/media/people/merge`
- `POST /api/v1/media/subtitles/jobs`
- `POST /api/v1/media/transcode/jobs`
- `POST /api/v1/media/shares`

Download Center:

- `GET /api/v1/downloads/tasks`
- `POST /api/v1/downloads/tasks`
- `POST /api/v1/downloads/tasks/{id}/pause`
- `POST /api/v1/downloads/tasks/{id}/resume`
- `POST /api/v1/downloads/tasks/{id}/archive`
- `DELETE /api/v1/downloads/tasks/{id}`
- `GET /api/v1/downloads/speed-profiles`
- `PUT /api/v1/downloads/speed-profile`

Docker:

- `GET /api/v1/docker/stacks`
- `GET /api/v1/docker/containers`
- `GET /api/v1/docker/containers/{id}/logs`
- `POST /api/v1/docker/containers/{id}/start`
- `POST /api/v1/docker/containers/{id}/stop`
- `POST /api/v1/docker/containers/{id}/restart`
- `PUT /api/v1/docker/containers/{id}/limits`

Security Center:

- `GET /api/v1/security/identities`
- `PUT /api/v1/security/identities/{id}/permissions`
- `GET /api/v1/security/ai-policies`
- `PUT /api/v1/security/ai-policies/{id}`
- `GET /api/v1/security/risk-actions`
- `POST /api/v1/security/risk-actions/{id}/confirm`
- `POST /api/v1/security/risk-actions/{id}/block`
- `GET /api/v1/security/audit`
- `POST /api/v1/security/audit/{id}/rollback`
- `GET /api/v1/shares`
- `DELETE /api/v1/shares/{id}`

Device Monitor:

- `GET /api/v1/monitoring/metrics/current`
- `GET /api/v1/monitoring/metrics/trend?range=`
- `GET /api/v1/monitoring/logs`
- `GET /api/v1/monitoring/alerts`
- `POST /api/v1/monitoring/alerts`
- `POST /api/v1/monitoring/alerts/{id}/mute`
- `POST /api/v1/monitoring/diagnostics`

System Settings:

- `GET /api/v1/settings`
- `PUT /api/v1/settings`
- `POST /api/v1/settings/defaults`
- `GET /api/v1/system/updates`
- `POST /api/v1/system/updates/check`
- `POST /api/v1/system/backups`

Remote Access:

- `GET /api/v1/remote/status`
- `POST /api/v1/remote/channel/start`
- `POST /api/v1/remote/channel/stop`
- `PUT /api/v1/remote/tunnel-mode`
- `POST /api/v1/remote/domain-token`
- `POST /api/v1/remote/domain-token/rotate`
- `GET /api/v1/remote/devices`
- `POST /api/v1/remote/devices/{id}/bind`
- `POST /api/v1/remote/devices/{id}/unbind`
- `GET /api/v1/remote/login-alerts`
- `POST /api/v1/remote/share-scan`

## 6. Development Milestones

### Milestone 1: Backend Foundation

- [ ] Create `server-go` Go module, command layout, internal package boundaries, config loader, logger, error model, health endpoint.
- [ ] Add PostgreSQL migrations, seed data matching `web-pc/src/data/higoos.ts`, and a reproducible Mac dev database setup.
- [ ] Add OpenAPI file with response envelope, auth errors, pagination, task IDs, event schemas, and all endpoints listed above.
- [ ] Generate TypeScript client into `web-pc/src/api/generated`.
- [ ] Add CORS, secure headers, request ID, panic recovery, structured access logs.
- [ ] Add auth bootstrap: first admin creation, password login, session cookie, CSRF-safe mutation strategy, logout.
- [ ] Add test harness: unit tests, HTTP contract tests, migration tests, frontend client generation check.

### Milestone 2: Security Governance Base

- [ ] Implement users, roles, groups, spaces, trusted devices, MFA-ready data model.
- [ ] Implement ACL evaluator for spaces, folders, files, app permissions, AI visibility, and Agent tool access.
- [ ] Implement audit events as append-only records with actor, subject, scope, risk, reason, request ID, rollback reference.
- [ ] Implement risk action workflow: low risk executes or records, medium/high risk requires confirmation, blocked actions prevent tool execution.
- [ ] Implement rollback registry for share revocation, tag changes, file moves, renames, permission edits, Agent actions.
- [ ] Expose Security Center APIs and wire frontend state to backend.

### Milestone 3: Desktop and Global Status API

- [ ] Move dock apps, window configs, desktop session, pinned dock apps, open windows, icon positions, and window geometry into backend-backed session APIs.
- [ ] Add current metrics, alerts, notifications, model policy, and desktop status endpoints.
- [ ] Add WebSocket/SSE event stream for task progress, alerts, monitoring, downloads, Docker events, assistant updates.
- [ ] Replace `web-pc/src/data/higoos.ts` imports with generated API client and stores while keeping visual components unchanged.

### Milestone 4: File Service

- [ ] Implement file tree scanning with Linux root path allowlist and Mac fixture filesystem.
- [ ] Implement metadata CRUD, folder listing, semantic-ready search fallback, tags, favorites, recent files.
- [ ] Implement preview generation for text, PDF metadata, images, and safe unsupported-file responses.
- [ ] Implement share links with password, expiry, download limits, audit, and revocation.
- [ ] Implement recycle bin, restore, version metadata, and batch move/rename/delete with rollback records.
- [ ] Wire File Manager to real APIs and remove local `files`/`folders` dependency.

### Milestone 5: Storage Service

- [ ] Implement disk inventory adapter using Linux `/sys`, `lsblk`, `smartctl`, and devstub fixtures on Mac.
- [ ] Implement storage pools and volumes with adapter interface for mdadm, Btrfs, and ZFS.
- [ ] Implement SMART scan tasks, repair tasks, snapshot tasks, task progress, failure states, and alerts.
- [ ] Persist disk temperature, health, capacity, slot, role, pool membership, and historical metrics.
- [ ] Wire Storage Monitor to real APIs.

### Milestone 6: Monitoring, Logs, Alerts

- [ ] Implement metric collectors for CPU, memory, network, disk I/O, temperature, fan, services, backup, downloads, Docker.
- [ ] Implement log ingestion from application logs, systemd journal on Linux, and devstub logs on Mac.
- [ ] Implement alert rules, alert creation, mute/unmute, severity, source, and notification delivery.
- [ ] Implement diagnostics runs and attach outputs to audit/events.
- [ ] Wire Device Monitor, TopBar metrics, widgets, and notification popovers to backend.

### Milestone 7: System Settings

- [ ] Implement settings schema for accounts, network/DDNS, model strategy, AI routing, notifications, updates, privacy, audit retention, and system backup.
- [ ] Add validation rules: enterprise/local mode disables cloud AI, privacy modes enforce local-only sensitive data, audit retention cannot undercut active compliance policy.
- [ ] Implement settings audit, defaults restore, system backup creation, and update-check adapter.
- [ ] Wire System Settings window to backend, including optimistic UI and error rollback.

### Milestone 8: Remote Access

- [ ] Implement remote channel state, DDNS records, tunnel mode, domain token creation/rotation, token expiry.
- [ ] Implement trusted device binding/unbinding and remote session audit.
- [ ] Implement MFA policy toggles and login alert records.
- [ ] Implement share scan checks against active share links, file sensitivity, expiry, and public access.
- [ ] Wire Remote Access window to backend.

### Milestone 9: Docker and App Center

- [ ] Implement Docker Engine adapter for Linux and devstub adapter for Mac.
- [ ] Implement stack, container, image, network, port, mount, env, resource, and log APIs.
- [ ] Implement start, stop, restart, limit update, event audit, and risk confirmation for sensitive mounts/ports.
- [ ] Implement app catalog and installed app model with permission declarations.
- [ ] Wire Docker window and later App Center window to backend.

### Milestone 10: Download Center

- [ ] Implement task model for BT, HTTP, magnet, and RSS sources.
- [ ] Integrate a production download adapter behind an interface; use devstub on Mac.
- [ ] Implement queue status, speed profiles, pause/resume, delete, completion hooks, and archive rules.
- [ ] Add file service integration so completed items can be tagged, moved, scanned, and indexed.
- [ ] Wire Download Center window to backend.

### Milestone 11: Backup and Snapshot Service

- [ ] Implement backup plans for local folders, client devices, cloud sync, remote NAS, and snapshot policies.
- [ ] Implement backup runs, progress, retry, verification, integrity checks, restore points, and restore jobs.
- [ ] Expose backup status in widgets, monitoring, assistant, and alerts.
- [ ] Connect backup events to AI steward suggestions when important folders are unprotected.

### Milestone 12: Photo and Media

- [ ] Implement media library scan, EXIF extraction, timeline, people, places, devices, albums, and shared albums.
- [ ] Implement memory generation jobs with audit-safe derived assets.
- [ ] Implement subtitle and transcode job queues with Linux adapter for ffmpeg and Mac devstub.
- [ ] Implement media scraping metadata boundaries and manual correction.
- [ ] Wire Photo Media window to backend.

### Milestone 13: AI Indexing

- [ ] Implement indexing jobs for filesystem metadata, text extraction, OCR, image metadata, audio/video transcription hooks, summaries, tags, entities, and embeddings.
- [ ] Enforce AI visibility through ACL snapshots before indexing, retrieval, assistant answers, and vector search.
- [ ] Implement local/cloud/private model providers with policy routing and audit.
- [ ] Implement keyword search plus vector search and relation graph writes.
- [ ] Implement index rebuild, pause, resume, retry, and purge on permission changes.

### Milestone 14: Assistant

- [ ] Implement permission-filtered semantic search endpoint.
- [ ] Implement conversation threads, messages, citations, suggested actions, and tool-call previews.
- [ ] Implement model policy selection per request: local-only, cloud-enhanced with redaction, private endpoint.
- [ ] Implement confirmation flow for assistant-generated actions.
- [ ] Wire TopBar search and `AiAssistantPanel.vue` to assistant APIs.

### Milestone 15: Agent Platform

- [ ] Implement tool registry with explicit schemas for file, backup, media, download, Docker, monitoring, share, notification, and external webhook tools.
- [ ] Implement Agent templates, Agent instances, workflow definitions, workflow runs, and tool-call logs.
- [ ] Implement planner preview, impact analysis, risk classification, confirmation checkpoints, execution, cancellation, retry, and rollback.
- [ ] Implement event triggers: new file, backup failure, space low, SMART warning, photo import, download complete, container unhealthy.
- [ ] Wire Agent Workbench and AI File Steward to backend.

### Milestone 16: Frontend Integration Standards

- [ ] Add `web-pc/src/api/runtime.ts` for base URL, auth, request ID, error normalization, and retry policy.
- [ ] Add generated client from `server-go/api/openapi.yaml`.
- [ ] Add stores per domain: `desktop`, `files`, `storage`, `security`, `monitoring`, `settings`, `remote`, `docker`, `downloads`, `media`, `assistant`, `agents`.
- [ ] Replace local seeds incrementally by window, keeping fallback fixture mode only for development.
- [ ] Add loading, empty, error, permission denied, confirmation required, task running, and event-updated states.
- [ ] Add contract tests ensuring frontend API calls match OpenAPI.

### Milestone 17: Linux Deployment

- [ ] Build Linux binaries for `higo-api`, `higo-worker`, and `higoctl`.
- [ ] Add systemd units, environment files, log rotation, health checks, and graceful shutdown.
- [ ] Add migration runner and first-admin bootstrap command.
- [ ] Add reverse proxy config for serving `web-pc/dist` and `/api/v1`.
- [ ] Add storage path allowlist and Linux capability requirements for hardware adapters.
- [ ] Add backup/export of database, settings, keys, and app state.
- [ ] Add release packaging: tarball or deb/rpm later, with checksum and rollback procedure.

### Milestone 18: Mac Development Workflow

- [ ] Add `make dev` to start Postgres, backend, worker, and `web-pc` dev server.
- [ ] Add devstub adapters for disks, storage pools, SMART, Docker, system logs, network, remote tunnel, downloads, and media jobs.
- [ ] Add seed command that reproduces the current `web-pc` demo data.
- [ ] Add fixture filesystem under `server-go/fixtures/nas-root`.
- [ ] Add docs for Mac setup, Linux-only behavior, and how to switch adapters.

### Milestone 19: Testing and Quality Gates

- [ ] Unit tests for services, ACL decisions, risk classification, rollback builders, model policy routing, and adapter parsing.
- [ ] Integration tests for migrations, HTTP handlers, event stream, task queues, and background workers.
- [ ] Contract tests comparing `openapi.yaml`, generated TypeScript client, and handler registration.
- [ ] Linux adapter tests using fixture command outputs for `lsblk`, `smartctl`, `docker`, `systemctl`, and journal logs.
- [ ] Frontend smoke tests: `npm run test:interactions`, `npm run build`, API fixture mode, and live-backend mode.
- [ ] End-to-end flows: file search/share/revoke, storage SMART scan, Agent preview/confirm/rollback, Docker restart, download archive, settings policy update, remote share scan.

### Milestone 20: Documentation

- [ ] Write `server-go/README.md` with architecture, commands, environment variables, Mac dev, Linux deployment.
- [ ] Write `docs/api.md` generated from OpenAPI with examples for each `web-pc` window.
- [ ] Write `docs/security-governance.md` covering ACL, AI visibility, risk levels, audit, rollback.
- [ ] Write `docs/linux-adapters.md` covering command dependencies and permissions.
- [ ] Write `docs/frontend-backend-integration.md` mapping each Vue component to stores and APIs.

## 7. Implementation Order

Build in this order to avoid rework:

1. Backend skeleton, OpenAPI, migrations, seed data.
2. Auth, IAM, audit, settings, event stream.
3. Desktop status, frontend generated client, global stores.
4. Files and security center because they define permission behavior for everything else.
5. Monitoring and storage because Linux device truth needs to appear early.
6. Remote, Docker, downloads, backup, media.
7. AI indexing, assistant, Agent platform.
8. Full frontend replacement of local component state.
9. Linux packaging and deployment hardening.

## 8. Completion Definition

The backend is complete when:

- Every visible `web-pc` window uses backend APIs or event streams for primary state.
- All medium/high-risk actions go through risk confirmation, audit, and rollback.
- Mac dev can run with devstub adapters and seed data.
- Linux deployment can run real adapters for storage, monitoring, Docker, networking, filesystem, and system services.
- AI search, assistant, and Agent tools are permission-filtered and auditable.
- Frontend build, interaction checks, backend tests, contract tests, migrations, and Linux adapter fixture tests pass.
- Documentation explains setup, API contracts, security governance, deployment, and frontend/backend mapping.
