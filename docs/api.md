# HiGoOS Backend API Map

This file is the human-readable contract map for wiring `web-pc` to `server-go`. The current backend exposes the visible Vue desktop API groups under `/api/v1`; Linux adapters and the database layer are still being phased in behind the same contracts.

All resource APIs use `/api/v1`. Mutations that affect files, permissions, shares, storage, Docker, remote access, model policy, or Agent tools must return audit metadata or a task/confirmation reference.

## Shared Contract Rules

- Response envelope: `{ "data": ..., "requestId": "..." }`.
- Error envelope: `{ "error": { "code": "...", "message": "..." }, "requestId": "..." }`.
- Pagination: list endpoints accept `cursor` and `limit`; responses include `items`, `nextCursor`, and `total` when total is cheap.
- Long tasks: mutating endpoints that run asynchronously return `taskId`, `state`, `risk`, and `auditId` when applicable.
- Confirmation: medium/high-risk actions return `confirmationId` before execution unless an explicit confirmed request is submitted.
- Event stream: `GET /api/v1/events/stream` publishes metrics, alerts, task progress, Docker events, downloads, assistant updates, and workflow events.
- Dev persistence: `HIGO_STATE_DIR` stores mutable devstub/API state as JSON so stateful API changes survive API process restarts before the database layer lands. This includes desktop session, file metadata, settings, storage tasks, monitoring alerts, downloads, Docker, remote access, backups, app center, security, media, assistant, Agent workflow, and AI steward state.

## Frontend Integration Order

1. Desktop shell and global status: apps, windows, session, metrics, alerts, model policy, event stream.
2. Security foundation: identities, ACL, AI policies, risk actions, shares, audit, rollback.
3. Files and storage: file tree/search/preview/share/batch operations, pools, disks, SMART, repair, snapshots.
4. Monitoring and settings: metrics, logs, alerts, diagnostics, account/network/model/privacy/update settings.
5. Remote, Docker, downloads: remote channel and devices, containers/stacks/logs/limits, download queues/profiles/archive.
6. Media, assistant, Agent: media library/jobs/shares, semantic search/chat/action confirmations, templates/workflow runs/events.

## Desktop Shell: `web-pc/src/App.vue`

Use first because it removes global dependency on `web-pc/src/data/higoos.ts`.

| API | Purpose |
| --- | --- |
| `GET /api/v1/desktop/apps` | Dock apps, badges, utility flags, icon paths. This endpoint is currently covered by router tests. |
| `GET /api/v1/desktop/windows` | Window configs: title, subtitle, status tone, default geometry, z order. |
| `GET /api/v1/desktop/session` | Open windows, minimized windows, active window, pinned dock apps, dock order, desktop icon positions, window geometry. |
| `PUT /api/v1/desktop/session` | Persist shell layout changes and compact/desktop preferences to the backend state directory. |
| `GET /api/v1/system/info` | App name, environment, version, adapter mode, host OS, readiness. This endpoint is currently covered by router tests. |
| `GET /api/v1/events/stream` | Desktop notifications, task updates, alert changes, assistant messages. |

## Top Bar: `web-pc/src/components/TopBar.vue`

| API | Purpose |
| --- | --- |
| `POST /api/v1/search/semantic` | Search suggestions and permission-filtered semantic results from the search field. |
| `GET /api/v1/monitoring/metrics/current` | CPU, memory, network, disk, temperature, fan summary. |
| `GET /api/v1/monitoring/alerts` | Notification popover and unread severity counts. |
| `GET /api/v1/security/ai-policies` | Current model policy badges. |
| `POST /api/v1/auth/logout` | Account menu logout once auth lands. |

## File Manager: `web-pc/src/components/windows/FileManagerWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/files/tree?space=` | Folder tree and space roots. |
| `GET /api/v1/files/search?q=&space=&type=&tags=` | Filtered file list and semantic fallback. |
| `GET /api/v1/files/{id}` | File metadata, tags, permissions, AI summary, version and recycle status. |
| `GET /api/v1/files/{id}/preview` | Safe preview payload or unsupported response. |
| `POST /api/v1/files/{id}/tags` | Add smart/manual tags with audit entry. |
| `POST /api/v1/files/{id}/shares` | Create share link after risk check. |
| `POST /api/v1/files/batch/move` | Batch move with rollback reference. |
| `POST /api/v1/files/batch/rename` | Batch rename with rollback reference. |
| `POST /api/v1/files/batch/delete` | Recycle/delete flow with confirmation for high risk. |
| `POST /api/v1/files/{id}/restore` | Restore from recycle bin/version metadata. |

## Storage Monitor: `web-pc/src/components/windows/StorageMonitorWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/storage/pools` | Storage pools, volumes, RAID/ZFS/Btrfs state, capacity. |
| `GET /api/v1/storage/disks` | Disk slots, role, temperature, SMART health, pool membership. |
| `GET /api/v1/storage/smart` | Latest SMART reports. |
| `POST /api/v1/storage/tasks/smart-scan` | Start SMART scan task for one disk or pool. |
| `POST /api/v1/storage/tasks/repair` | Start repair/rebuild task after confirmation. |
| `POST /api/v1/storage/tasks/snapshot` | Create snapshot task. |
| `GET /api/v1/storage/tasks/{id}` | Task detail and progress. |

## AI File Steward: `web-pc/src/components/windows/AiStewardWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/steward/suggestions` | AI governance suggestions with impact and risk. |
| `POST /api/v1/steward/suggestions/{id}/preview` | Preview file moves, renames, tags, shares, or permission effects. |
| `POST /api/v1/steward/suggestions/{id}/confirm` | Execute confirmed suggestion and create audit/rollback records. |
| `POST /api/v1/steward/suggestions/{id}/dismiss` | Dismiss suggestion without execution. |
| `GET /api/v1/steward/audit` | Steward action log shown in the window. |
| `POST /api/v1/steward/audit/{id}/rollback` | Undo supported steward action. |

## Agent Workbench: `web-pc/src/components/windows/AgentWorkbenchWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/agents/templates` | Agent templates, declared tools, default risk. |
| `POST /api/v1/agents` | Create an Agent instance scoped to user/space. |
| `GET /api/v1/agents/{id}/tools` | Permission-filtered tool registry for the Agent. |
| `POST /api/v1/workflows/preview` | Plan preview with impact analysis and confirmation checkpoints. |
| `POST /api/v1/workflows/runs` | Start a workflow run. |
| `POST /api/v1/workflows/runs/{id}/confirm` | Confirm a paused workflow step. |
| `POST /api/v1/workflows/runs/{id}/cancel` | Cancel run and trigger rollback where needed. |
| `GET /api/v1/workflows/runs/{id}/events` | Workflow event stream for the run detail panel. |

## AI Assistant Panel: `web-pc/src/components/AiAssistantPanel.vue`

| API | Purpose |
| --- | --- |
| `POST /api/v1/search/semantic` | Permission-filtered retrieval for assistant grounding. |
| `POST /api/v1/assistant/threads` | Create a thread. |
| `GET /api/v1/assistant/threads/{id}` | Load messages, citations, pending actions. |
| `POST /api/v1/assistant/threads/{id}/messages` | Send message, select model policy, stream or poll response. |
| `POST /api/v1/assistant/actions/{id}/confirm` | Confirm assistant-generated actions. |

## Photo Media: `web-pc/src/components/windows/PhotoMediaWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/media/items` | Timeline and filtered media list. |
| `GET /api/v1/media/albums` | Albums, people, places, devices, memories. |
| `POST /api/v1/media/albums` | Create album or shared album. |
| `POST /api/v1/media/memories` | Generate memory job. |
| `POST /api/v1/media/people/merge` | Merge people with rollback record. |
| `POST /api/v1/media/subtitles/jobs` | Start subtitle job. |
| `POST /api/v1/media/transcode/jobs` | Start ffmpeg transcode job. |
| `POST /api/v1/media/shares` | Share media after ACL/risk check. |

## Download Center: `web-pc/src/components/windows/DownloadCenterWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/downloads/tasks` | Download queue, category, progress, speed, archive state; dev mutations are persisted. |
| `POST /api/v1/downloads/tasks` | Create BT, HTTP, magnet, or RSS task and persist the queue. |
| `POST /api/v1/downloads/tasks/{id}/pause` | Pause active task and persist state. |
| `POST /api/v1/downloads/tasks/{id}/resume` | Resume paused task and persist state. |
| `POST /api/v1/downloads/tasks/{id}/archive` | Move completed item into file service path, enqueue indexing, and persist state. |
| `DELETE /api/v1/downloads/tasks/{id}` | Delete task or queue record after risk check and persist state. |
| `GET /api/v1/downloads/speed-profiles` | Load speed profiles. |
| `PUT /api/v1/downloads/speed-profile` | Update and persist active speed profile. |

## Docker: `web-pc/src/components/windows/DockerWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/docker/stacks` | Compose stacks, ports, volumes, network, status. |
| `GET /api/v1/docker/containers` | Container rows, health, CPU/memory, mounts, env, ports. |
| `GET /api/v1/docker/containers/{id}/logs` | Bounded log tail. |
| `POST /api/v1/docker/containers/{id}/start` | Start container and audit action. |
| `POST /api/v1/docker/containers/{id}/stop` | Stop container after risk check. |
| `POST /api/v1/docker/containers/{id}/restart` | Restart with task/event progress. |
| `POST /api/v1/docker/containers/{id}/complete-restart` | Mark restart health check complete and refresh logs. |
| `PUT /api/v1/docker/containers/{id}/limits` | Update CPU/memory limits. |

## Backup Sync: `web-pc/src/components/windows/BackupSyncWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/backups/jobs` | Backup jobs, schedule, progress, retention, health, and policy metadata; dev mutations are persisted. |
| `POST /api/v1/backups/jobs/{id}/run` | Start or rerun a backup job. |
| `POST /api/v1/backups/jobs/{id}/pause` | Pause a running backup job. |
| `POST /api/v1/backups/jobs/{id}/resume` | Resume a paused backup job. |
| `POST /api/v1/backups/jobs/{id}/verify` | Verify backup data and write audit feedback. |

## App Center: `web-pc/src/components/windows/AppCenterWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/app-center/apps` | App catalog, versions, install/runtime state, ports, risk, and resource profile; dev mutations are persisted. |
| `POST /api/v1/app-center/apps/{id}/install` | Install an app and start its service. |
| `POST /api/v1/app-center/apps/{id}/update` | Update an app to the latest version. |
| `POST /api/v1/app-center/apps/{id}/start` | Start an installed app. |
| `POST /api/v1/app-center/apps/{id}/stop` | Stop a running app. |

## Security Center: `web-pc/src/components/windows/SecurityCenterWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/security/identities` | Users, groups, roles, devices, Agent/app identities. |
| `PUT /api/v1/security/identities/{id}/permissions` | Update ACL/app/Agent permissions and refresh AI visibility snapshots. |
| `GET /api/v1/security/ai-policies` | Space/user/model routing policies. |
| `PUT /api/v1/security/ai-policies/{id}` | Update AI policy with audit entry. |
| `GET /api/v1/security/risk-actions` | Pending/handled risk actions. |
| `POST /api/v1/security/risk-actions/{id}/confirm` | Confirm risk action. |
| `POST /api/v1/security/risk-actions/{id}/block` | Block risk action. |
| `GET /api/v1/security/audit` | Audit event list. |
| `POST /api/v1/security/audit/{id}/rollback` | Roll back supported action. |
| `GET /api/v1/shares` | Active share links. |
| `DELETE /api/v1/shares/{id}` | Revoke share link. |

## Device Monitor: `web-pc/src/components/windows/DeviceMonitorWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/monitoring/metrics/current` | Current metrics summary. |
| `GET /api/v1/monitoring/metrics/trend?range=` | Trend series for selected metric. |
| `GET /api/v1/monitoring/logs` | System/app/AI/Agent/storage logs. |
| `GET /api/v1/monitoring/alerts` | Alerts list. |
| `POST /api/v1/monitoring/alerts` | Create threshold alert. |
| `POST /api/v1/monitoring/alerts/{id}/mute` | Mute/unmute alert. |
| `POST /api/v1/monitoring/diagnostics` | Start diagnostics task. |

## System Settings: `web-pc/src/components/windows/SystemSettingsWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/settings` | Account, network, model, AI, notification, update, privacy, audit, backup settings. |
| `PUT /api/v1/settings` | Save and persist settings with validation and audit. |
| `POST /api/v1/settings/defaults` | Restore and persist defaults. |
| `GET /api/v1/system/updates` | Current update status. |
| `POST /api/v1/system/updates/check` | Start update check. |
| `POST /api/v1/system/backups` | Create system config/database/key backup. |

## Remote Access: `web-pc/src/components/windows/RemoteAccessWindow.vue`

| API | Purpose |
| --- | --- |
| `GET /api/v1/remote/status` | Remote channel, DDNS, tunnel mode, MFA, token state; dev mutations are persisted. |
| `POST /api/v1/remote/channel/start` | Start remote channel. |
| `POST /api/v1/remote/channel/stop` | Stop remote channel. |
| `PUT /api/v1/remote/tunnel-mode` | Switch relay/direct/disabled mode. |
| `POST /api/v1/remote/domain-token` | Create domain token. |
| `POST /api/v1/remote/domain-token/rotate` | Rotate token and audit. |
| `GET /api/v1/remote/devices` | Bound/trusted devices. |
| `POST /api/v1/remote/devices/{id}/bind` | Bind device. |
| `POST /api/v1/remote/devices/{id}/unbind` | Unbind device. |
| `GET /api/v1/remote/login-alerts` | Login alert history. |
| `POST /api/v1/remote/share-scan` | Scan share links against public access and sensitivity rules. |
