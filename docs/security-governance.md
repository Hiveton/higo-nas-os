# HiGoOS Security Governance

Security governance is a cross-cutting service, not a window-only feature. It constrains file access, app permissions, AI indexing, assistant retrieval, Agent tool execution, remote access, shares, Docker operations, settings changes, audit, and rollback.

The architecture rules are:

- AI can only index, summarize, embed, retrieve, and answer from data the actor can access.
- Medium and high-risk actions require explicit confirmation, audit, and rollback support.
- NAS core services must keep running even if AI providers, indexing, or Agent planning fail.
- Model routing is policy-driven by space, data sensitivity, user role, and task type.

## Identities

Actors are normalized before authorization:

| Identity | Scope |
| --- | --- |
| User | Administrator, family member, team member, guest. |
| Device | Bound browser/device, remote device, mobile backup client. |
| Session | Login session, MFA state, trusted-device state, CSRF context. |
| App | Installed app or Docker/app-center integration with declared permissions. |
| Agent | Agent template/instance with owner, space, tools, execution policy, risk level. |
| System worker | Background worker for indexing, media, backup, monitoring, downloads, and maintenance. |

Every request should carry actor ID, session/device ID when present, request ID, source IP, user agent, and selected space. Worker actions inherit a system actor plus the user/Agent/action that scheduled the work.

## Authentication

Identity is now backed by a real authentication layer, not an implicit dev actor:

- **The OS is the authoritative identity source.** On the Linux NAS host accounts are Linux system users, managed through `useradd`/`usermod`/`userdel`/`chpasswd` and authenticated by reading `/etc/shadow`. Mac/dev uses a devstub backend (`credentials.json`); `HIGO_ACCOUNTS_BACKEND` (`system`/`devstub`/empty = auto by OS) selects the backend.
- **Pre-existing system users are enumerated, not just HiGoOS-created ones.** At start-up and on every account listing the service reconciles `getent passwd` into its user list, including every real human account (UID in `[1000, 60000]`, excluding `root`/daemons/`nobody`). So the installer's sudo user (e.g. `hiveton`) appears in the user center and can log in with its existing OS password — no HiGoOS-side provisioning required. App-level metadata HiGoOS adds (role, quota, space grants, sessions, MFA) lives in sidecar JSON keyed by username; the OS never stores it.
- **Password verification is pure-Go and scheme-aware.** `/etc/shadow` hashes are verified without cgo/PAM: yescrypt (`$y$`/`$gy$`, Ubuntu 24.04's default, via `openwall/yescrypt-go`) and the crypt(3) schemes `$6$`/`$5$`/`$1$` (via `GehirnInc/crypt`). Passwords **HiGoOS sets** are forced to SHA-512 `$6$` (`chpasswd -c SHA512`) and mirrored into the Samba DB (`smbpasswd`) so SMB shares share the credential.
- **Role follows native group membership.** A user is `admin` when it belongs to `higoos-admins` (`HIGO_ACCOUNTS_ADMIN_GROUP`), `sudo`, or `wheel`; otherwise `user`. So existing sudo users are treated as administrators automatically.
- **Managed-user shape & delete protection.** HiGoOS-created accounts use primary group `higoos` (`HIGO_ACCOUNTS_GROUP`), UID ≥ 3000 (`HIGO_ACCOUNTS_UID_BASE`), and shell `/usr/sbin/nologin`. The user center refuses to delete accounts below the managed UID base, so a pre-existing system/sudo user can never be removed through the API.
- **Two-factor (TOTP).** Users may enrol RFC-6238 TOTP (`/api/v1/auth/mfa/setup|enable|disable`, secrets in `mfa.json`); when enabled, login requires a 6-digit `code` (errors `mfa_required`/`mfa_invalid`).
- **Sessions and cookies.** Login issues a server-side session persisted in `sessions.json`, delivered as an HttpOnly `higo_session` cookie. `sessionGuard` admits dev/test with an implicit admin actor; other environments require a valid session or return `401`. `HIGO_AUTH_REQUIRED=true` enforces sessions even in dev. Session TTL is `HIGO_SESSION_TTL` (default `720h`); cookie SameSite is `HIGO_COOKIE_SAMESITE` (`lax`/`strict`/`none`).
- **CSRF.** Cookie-session write requests must send an `X-CSRF-Token` header matching the readable `higo_csrf` cookie. Bearer-token requests and `login` are exempt; CSRF is disabled by default in dev/test (`HIGO_CSRF_DISABLED`).
- **Coarse-grained admin gate (`roleGate`).** Write methods on sensitive prefixes (`/api/v1/accounts/`, `/security/`, `/settings`, `/remote/`, `/protocols`, `/storage/`, `/ai/providers`, …) require the `admin` role or return `403`. This is a coarse gate layered on top of the per-resource ACL model below, not a replacement for it.
- **Login lockout.** An account locks after `HIGO_LOGIN_MAX_FAILURES` (default 5) consecutive failed logins.
- **Bootstrap admin.** On first start a seed `admin` user with no credentials is given a random password printed once to the log (`WARN initial admin password generated`); `HIGO_ADMIN_BOOTSTRAP_PASSWORD` fixes it for reproducible provisioning.

The self-service auth endpoints (`/api/v1/auth/login|logout|me|password|sessions`) are documented in `api.md`.

## File storage isolation (real)

Identity owns storage on disk, not just in metadata:

- **Personal folders** live at `<NAS_ROOT>/homes/<username>` (`chown user`, `0700`); **group folders** at `<NAS_ROOT>/groups/<groupID>` (system group, `2770` setgid). A `SpaceGrant` is written as a real POSIX ACL (`setfacl`, recursive + default) on the target space directory. See `linux-adapters.md` (filesystem provisioner).
- **Web file visibility is strictly scoped**: a non-admin sees only their personal folder, their group folders, and explicitly-granted shared spaces; reads outside that set return `403`. Admins see everything. This mirrors the on-disk ownership/ACLs that SMB/NFS/WebDAV enforce natively (the HTTP API runs as root and so enforces the same scope in-app).

## ACL Model

ACL decisions combine:

- Role: admin, family member, team member, guest, app, Agent, worker.
- Space: personal, family, team, shared, system.
- File/folder ACL: read, write, delete, share, manage permissions, restore.
- App permission: storage path, network, Docker, media, backup, notification, external webhook.
- Agent permission: allowed tools, data scopes, execution policy, confirmation policy.
- Share policy: password, expiry, download limit, public/private, sensitivity scan.

Evaluation order:

1. Authenticate user/session/device/app/Agent identity.
2. Resolve role and group membership.
3. Resolve target space and folder/file ACL.
4. Apply app or Agent declared permission ceiling.
5. Apply policy blocks such as sensitive-local-only, guest restriction, remote restriction, or disabled share.
6. Produce a decision: allow, deny, require confirmation, or allow read-only.

## AI Visibility

AI visibility is stricter than file visibility because derived data can leak content. Visibility applies to:

- Raw extracted text, OCR text, transcripts, EXIF, thumbnails, preview text.
- Summaries, tags, entities, embeddings, vector chunks, knowledge graph edges.
- Assistant citations, semantic search results, Agent planning context.

Rules:

- Index jobs receive an ACL snapshot for each item before extraction or embedding.
- Vector and keyword indexes are partitioned by space and permission snapshot.
- Sensitive files can be marked `aiExcluded` so they remain visible in file listings but absent from AI analysis.
- Permission changes enqueue visibility refresh: revoke stale chunks, summaries, embeddings, graph edges, and assistant cache entries.
- Assistant and Agent retrieval re-checks live ACL before returning citations or tool inputs.
- Cloud model calls are blocked for sensitive data unless policy explicitly allows redacted cloud routing.

## Risk Levels

| Level | Examples | Required behavior |
| --- | --- | --- |
| Low | Search, preview, summarize, classify suggestion, read-only monitoring, diagnostics preview. | Execute when ACL permits; audit when AI, Agent, remote, or admin surfaces are involved. |
| Medium | Move, rename, batch tag, create private share, pause/resume downloads, restart container, create snapshot, merge people. | Return preview and confirmation, write audit, attach rollback reference when state changes. |
| High | Delete, overwrite, public share, permission change, disable backup, repair/rebuild storage, stop remote security control, send sensitive data to external API, change cloud model policy. | Require privileged actor and explicit confirmation; block if rollback is unavailable and the action is not safely reversible. |

Risk classification uses action type, target sensitivity, affected count, external exposure, destructive potential, remote context, and whether AI/Agent initiated the action.

## Confirmation

Confirmation records must include:

- Actor and effective identity.
- Action, target scope, affected item count, risk level.
- Human-readable impact summary.
- Required permission and policy checks.
- Expiry time and single-use token.
- Preview of rollback availability.

Medium/high-risk endpoints first return `confirmationId` and no irreversible side effect. The confirmed request must repeat the action intent so the backend can detect stale or mismatched confirmations.

## Audit

Audit is append-only. Records should include:

- Timestamp, request ID, actor, device/session, source IP.
- Action, domain, target IDs, target path/scope.
- Tools used by Agent or assistant.
- Data range read by AI or Agent.
- Before/after summary for changed state.
- Risk level, confirmation ID, policy decision, model provider/routing when AI is involved.
- Task ID, event IDs, rollback operation ID.
- Result: allowed, denied, confirmed, blocked, failed, rolled back.

Audit feeds Security Center, system settings audit retention, assistant citations, Agent run detail, diagnostics, and compliance export.

## Rollback

Rollback is a registry of reversible operations, not a best-effort text note.

Supported rollback types:

- File move: move item back if source and destination still valid.
- File rename: restore previous name.
- Tag change: restore previous tag set.
- Share creation: revoke generated share link.
- Permission change: restore previous ACL snapshot.
- Archive rule: move completed download/media item back or reverse metadata changes.
- Agent workflow: replay registered compensating operations in reverse order.
- Delete: restore from recycle bin or version metadata when still retained.

Rollback records store operation type, actor, target, before/after payload, validation checks, expiry/retention, and rollback result. If rollback cannot be guaranteed, the confirmation preview must say the operation is irreversible before execution.

## Model Strategy

Model routing follows the architecture policy:

- Family hybrid mode: local models handle privacy-sensitive indexing and basic understanding; cloud models may be used for complex reasoning on non-sensitive data.
- Small team provider mode: administrator selects OpenAI, private endpoint, LAN model, or other provider per task class.
- Enterprise local mode: cloud model calls are disabled; processing stays local or private.
- Data-level routing: sensitive data is local-only; ordinary data can use cloud-enhanced routing when enabled.
- Task-level routing: OCR, transcription, summarization, question answering, and Agent planning may use different model providers.

Each model call records provider, model, policy decision, data sensitivity level, redaction state, actor, request ID, and cost/latency metadata when available. The user must be able to see which files entered AI analysis.

## Security Center Mapping

`web-pc/src/components/windows/SecurityCenterWindow.vue` maps to governance APIs:

- Identities and permissions: `GET /api/v1/security/identities`, `PUT /api/v1/security/identities/{id}/permissions`.
- AI policy cards: `GET /api/v1/security/ai-policies`, `PUT /api/v1/security/ai-policies/{id}`.
- Risk queue: `GET /api/v1/security/risk-actions`, confirm/block endpoints.
- Shares: `GET /api/v1/shares`, `DELETE /api/v1/shares/{id}`.
- Audit and rollback: `GET /api/v1/security/audit`, `POST /api/v1/security/audit/{id}/rollback`.

## Sharing protocols (SMB / NFS / WebDAV / DLNA)

The `protocols` domain manages real host sharing services and is fully governed. Every mutation is medium/high risk and follows the preview → confirm → rollback pattern (mirrors steward/security): a `*/preview` endpoint returns a `confirmationId` + impact summary with **no side effects**; the matching `*/confirm` endpoint applies the change, runs the host adapter, and writes an append-only audit entry carrying a rollback hint.

Risk classification:

- Enable / disable a protocol → **medium**.
- Create a share at `account` / `readonly` access → **medium**.
- Create a share at `public` / `password` access → **high** (more open exposure; impact summary warns to confirm no sensitive data).
- Delete a share → **medium** (reversible — rollback re-adds the directory).

`web-pc/src/components/windows/ProtocolsWindow.vue` maps to:

- List + live state: `GET /api/v1/protocols`, `GET /api/v1/protocols/{key}`.
- Enable/disable: `POST /api/v1/protocols/{key}/{enable|disable}/{preview|confirm}`.
- Shares: `GET /api/v1/protocols/{key}/shares`, `POST /api/v1/protocols/{key}/shares/{preview|confirm}`, `POST /api/v1/protocols/shares/{id}/delete/{preview|confirm}`.
- Audit + rollback: `GET /api/v1/protocols/audit`, `POST /api/v1/protocols/audit/{id}/rollback`.

Host-config safety: the Linux adapter only ever writes files HiGoOS owns (`smb.conf.d/higoos.conf` via `include=`, `exports.d/*.exports`, a delimited block in `minidlna.conf`, a dedicated Apache WebDAV config) — the user's primary `smb.conf` / `exports` are never rewritten.
