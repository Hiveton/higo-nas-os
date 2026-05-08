# Frontend and Backend Integration

The current `web-pc` desktop uses a shared API runtime plus Go dev services for the visible desktop windows. Static seed data in `web-pc/src/data/higoos.ts` remains a fallback path for local development when the backend is unavailable.

## Runtime and Store/API Shape

Use one shared API runtime plus a store or focused API adapter per domain:

| Store/API adapter | Vue surface | API groups |
| --- | --- | --- |
| `desktop` | `web-pc/src/App.vue`, dock, windows, desktop widgets | desktop apps, windows, session, event stream |
| `files` | `web-pc/src/components/windows/FileManagerWindow.vue` | files tree/search/detail/preview/tags/shares/batch/restore |
| `storage` | `web-pc/src/components/windows/StorageMonitorWindow.vue` | storage pools/disks/SMART/tasks |
| `security` | `web-pc/src/components/windows/SecurityCenterWindow.vue` | identities, AI policies, risk actions, shares, audit, rollback |
| `monitoring` | `web-pc/src/components/windows/DeviceMonitorWindow.vue`, `web-pc/src/components/TopBar.vue` | metrics, trends, logs, alerts, diagnostics |
| `settings` | `web-pc/src/components/windows/SystemSettingsWindow.vue` | settings, defaults, updates, system backups |
| `remote` | `web-pc/src/components/windows/RemoteAccessWindow.vue` | remote status, channel, tunnel mode, domain token, devices, login alerts, share scan |
| `docker` | `web-pc/src/components/windows/DockerWindow.vue` | stacks, containers, logs, start/stop/restart, limits |
| `backup` | `web-pc/src/components/windows/BackupSyncWindow.vue` | backup jobs, run, pause, resume, verify |
| `appCenter` | `web-pc/src/components/windows/AppCenterWindow.vue` | app catalog, install, update, start, stop |
| `downloads` | `web-pc/src/components/windows/DownloadCenterWindow.vue` | tasks, pause/resume/archive/delete, speed profiles |
| `media` | `web-pc/src/components/windows/PhotoMediaWindow.vue` | media items, albums, memories, people merge, subtitles, transcode, shares |
| `assistant` | `web-pc/src/components/AiAssistantPanel.vue`, top bar search | semantic search, threads, messages, action confirmations |
| `agents` | `web-pc/src/components/windows/AgentWorkbenchWindow.vue`, AI steward | templates, tools, workflow preview/runs/events, suggestions |

The runtime should own base URL, credentials, request ID propagation, CSRF/session behavior, JSON envelope unwrapping, normalized errors, retries for idempotent reads, and event-stream reconnect.

Mutable devstub-backed APIs now use `HIGO_STATE_DIR` for JSON persistence. Desktop session, file metadata, settings, storage tasks, monitoring alerts, downloads, Docker, remote access, backup jobs, app center runtime state, security, media, assistant, Agent workflow, and AI steward state are backend-owned and survive API process restarts while the database layer is still pending.

## Migration Order

1. `desktop` store: replace `dockApps` and `desktopWindows` imports in `web-pc/src/App.vue`; persist shell session with debounced `PUT /api/v1/desktop/session`.
2. `monitoring` store: connect `TopBar.vue` metrics/notices/model badges and open the event stream once per desktop session.
3. `security` store: wire `SecurityCenterWindow.vue` before risky file/AI/Agent actions so confirmation and rollback UI are reusable.
4. `files` store: replace local `files` and `folders`; keep search, preview, tags, share, and batch action UI states.
5. `storage` store: connect pools/disks and turn SMART/repair/snapshot buttons into task creation plus event updates.
6. `settings` and `remote` stores: wire policy toggles, model strategy, privacy, remote channel, MFA, devices, token rotation, share scan.
7. `docker`, `backup`, `appCenter`, and `downloads` surfaces: use backend-backed container, backup, app catalog, and download actions with task progress and audit.
8. `media` store: connect timeline/albums/jobs/sharing; treat subtitle/transcode/memory as tasks.
9. `assistant` and `agents` stores: connect semantic search, chat, suggestions, workflow preview, confirmation, run events, rollback.
10. Remove direct imports from `web-pc/src/data/higoos.ts` after every visible window has a store-backed path; keep fixture mode through backend devstub rather than frontend-only seeds.

## Component Mapping

| Component | Current state | Store/API migration |
| --- | --- | --- |
| `web-pc/src/App.vue` | `openWindowIds`, `minimizedWindowIds`, `activeWindowId`, `dockOrder`, `pinnedDockAppIds`, icon positions, window geometry. | `desktop.loadApps()`, `desktop.loadWindows()`, `desktop.loadSession()`, `desktop.saveSessionPatch()`, `desktop.subscribeEvents()`. |
| `web-pc/src/components/TopBar.vue` | local metrics, notices, search suggestions, model policies. | `monitoring.current`, `monitoring.alerts`, `security.aiPolicies`, `assistant.semanticSearch(query)`, logout through auth. |
| `web-pc/src/components/windows/FileManagerWindow.vue` | local folders/files, selected file/folder, search, preview/share toggles, smart tags. | `files.loadTree(space)`, `files.search(filters)`, `files.loadDetail(id)`, `files.preview(id)`, `files.addTags(id)`, `files.createShare(id)`, batch actions. |
| `web-pc/src/components/windows/StorageMonitorWindow.vue` | static pool/disk data and local action text. | `storage.loadPools()`, `storage.loadDisks()`, `storage.startSmartScan()`, `storage.startSnapshot()`, `storage.startRepair()`, task event updates. |
| `web-pc/src/components/windows/AiStewardWindow.vue` | seed suggestions and local dismiss/action log. | `agents.loadStewardSuggestions()`, `agents.previewSuggestion(id)`, `agents.confirmSuggestion(id)`, `agents.dismissSuggestion(id)`, audit rollback. |
| `web-pc/src/components/windows/AgentWorkbenchWindow.vue` | seed templates/workflow nodes and local simulation state. | `agents.loadTemplates()`, `agents.loadTools(agentId)`, `agents.previewWorkflow(payload)`, `agents.startRun(payload)`, `agents.confirmRun(id)`, run events. |
| `web-pc/src/components/AiAssistantPanel.vue` | seed messages, local draft, static capabilities/model policies. | `assistant.createThread()`, `assistant.loadThread(id)`, `assistant.sendMessage(id, body)`, `assistant.confirmAction(id)`, `security.aiPolicies`. |
| `web-pc/src/components/windows/PhotoMediaWindow.vue` | local timeline/facets/albums/jobs/share state. | `media.loadItems(filters)`, `media.loadAlbums()`, `media.createMemory()`, `media.mergePeople()`, `media.createSubtitleJob()`, `media.createTranscodeJob()`, `media.createShare()`. |
| `web-pc/src/components/windows/DownloadCenterWindow.vue` | backend-backed tasks and speed profiles with local UI fallback. | `downloads.loadTasks()`, `downloads.createTask()`, `downloads.pause()`, `downloads.resume()`, `downloads.archive()`, `downloads.deleteTask()`, `downloads.updateSpeedProfile()`. |
| `web-pc/src/components/windows/DockerWindow.vue` | local stacks/containers/logs/limits/status. | `docker.loadStacks()`, `docker.loadContainers()`, `docker.loadLogs(id)`, `docker.start(id)`, `docker.stop(id)`, `docker.restart(id)`, `docker.updateLimits(id)`. |
| `web-pc/src/components/windows/BackupSyncWindow.vue` | backend-backed and restart-persistent backup jobs with local fallback. | `backup.loadJobs()`, `backup.runJob(id)`, `backup.pauseJob(id)`, `backup.resumeJob(id)`, `backup.verifyJob(id)`. |
| `web-pc/src/components/windows/AppCenterWindow.vue` | backend-backed and restart-persistent app catalog/runtime state with local fallback. | `appCenter.loadApps()`, `appCenter.installApp(id)`, `appCenter.updateApp(id)`, `appCenter.startApp(id)`, `appCenter.stopApp(id)`. |
| `web-pc/src/components/windows/SecurityCenterWindow.vue` | local identities, AI policies, share links, risk actions, audit entries. | `security.loadIdentities()`, `security.savePermissions()`, `security.loadAiPolicies()`, `security.saveAiPolicy()`, `security.confirmRisk()`, `security.blockRisk()`, `security.revokeShare()`, `security.rollbackAudit()`. |
| `web-pc/src/components/windows/DeviceMonitorWindow.vue` | local metrics/services/trends/logs/alerts/diagnostics. | `monitoring.loadCurrent()`, `monitoring.loadTrend(range)`, `monitoring.loadLogs()`, `monitoring.loadAlerts()`, `monitoring.createAlert()`, `monitoring.muteAlert()`, `monitoring.runDiagnostics()`. |
| `web-pc/src/components/windows/SystemSettingsWindow.vue` | backend-backed and restart-persistent settings with update/backup task feedback. | `settings.load()`, `settings.savePatch()`, `settings.restoreDefaults()`, `settings.checkUpdates()`, `settings.createSystemBackup()`, validation errors. |
| `web-pc/src/components/windows/RemoteAccessWindow.vue` | backend-backed and restart-persistent channel/domain/token/devices/policies/login/share scan state. | `remote.loadStatus()`, `remote.startChannel()`, `remote.stopChannel()`, `remote.setTunnelMode()`, `remote.rotateToken()`, `remote.bindDevice()`, `remote.unbindDevice()`, `remote.scanShares()`. |

## UI State Standard

Every store exposes the same state shape:

```ts
type RequestState = 'idle' | 'loading' | 'ready' | 'empty' | 'error' | 'permission_denied';

type TaskState = 'queued' | 'running' | 'waiting_confirmation' | 'succeeded' | 'failed' | 'cancelled' | 'rolled_back';
```

Store fields:

- `state`: current request state.
- `items` or `detail`: domain data.
- `error`: normalized `{ code, message, requestId }`.
- `permission`: denied action, required permission, and target scope when blocked.
- `pendingConfirmation`: confirmation ID, risk, impact summary, expiry.
- `tasks`: map of task ID to state, progress, message, audit ID, rollback ID.
- `lastEventAt`: timestamp of latest event-stream update.

## Loading, Error, Permission, Task, Event Rules

- Loading: use skeleton/disabled controls for first load; preserve previous data during refetch.
- Empty: show domain-specific empty state only after a successful ready response with no items.
- Error: show backend error message and request ID; do not clear previous data unless the request was the first load.
- Permission denied: use a distinct state from generic error and expose required permission/action scope.
- Confirmation required: open a confirmation panel/modal before executing medium/high-risk mutation.
- Task running: disable duplicate mutation buttons for the same target and show event-stream progress.
- Event updated: patch store state from `GET /api/v1/events/stream`; refetch detail when event payload says data is stale.
- Rollback available: show rollback action only when the API returns a rollback ID and the operation is still valid.
- Optimistic update: allowed for low-risk UI preferences and reversible settings; file, permission, share, storage, Docker, and Agent changes wait for backend acceptance or task event.

## Fixture Mode

Frontend fixture mode should come from backend `devstub` responses, not from component-local seed arrays. This keeps Mac development deterministic while exercising the same API runtime, stores, event handling, permission states, and error normalization that production uses.
