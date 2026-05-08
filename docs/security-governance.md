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
