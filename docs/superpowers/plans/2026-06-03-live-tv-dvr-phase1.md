# Live TV DVR Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Jellyfin-style first-phase Live TV DVR support: XMLTV guide import, guide browsing, DVR settings, and one-time recording reservations.

**Architecture:** Extend the existing `internal/video` service rather than adding a separate live-tv package, because current M3U source and live channel state already live there and persist through the same JSON store. Add typed API methods and handlers for settings, guide sources, programs, timers, and recordings, then expose a compact Live TV dashboard in `VideoCenterWindow.vue`.

**Tech Stack:** Go service and tests, JSON persistence, XMLTV parsing with `encoding/xml`, ffmpeg command construction for future recording execution, Vue 3 TypeScript frontend.

---

### Task 1: Backend Live TV Types And Persistence

**Files:**
- Modify: `server-go/internal/video/types.go`
- Modify: `server-go/internal/video/service.go`
- Test: `server-go/internal/video/service_test.go`

- [ ] Add `LiveGuideSource`, `LiveProgram`, `DVRSettings`, `RecordingTimer`, and `RecordingItem` JSON types.
- [ ] Extend persisted video state with guide sources, programs, DVR settings, timers, and recordings.
- [ ] Write tests that save a service with DVR state and load it again.

### Task 2: XMLTV Guide Import

**Files:**
- Modify: `server-go/internal/video/service.go`
- Test: `server-go/internal/video/service_test.go`

- [ ] Add `CreateGuideSource` and `RefreshGuideSource`.
- [ ] Parse XMLTV `channel` and `programme` entries.
- [ ] Map guide programs to live channels by `tvg-id`, channel name, and channel id.
- [ ] Write tests for local XMLTV import and mapped program queries.

### Task 3: Timers And DVR Settings

**Files:**
- Modify: `server-go/internal/video/service.go`
- Test: `server-go/internal/video/service_test.go`

- [ ] Add `DVRSettings`, `UpdateDVRSettings`, `RecordingTimers`, `CreateRecordingTimer`, and `CancelRecordingTimer`.
- [ ] Apply global pre/post padding defaults to new timers.
- [ ] Reject timers with missing channel or invalid time ranges.
- [ ] Expose timer status as `scheduled`, `cancelled`, `recording`, `completed`, or `failed`.

### Task 4: HTTP API And Frontend Client

**Files:**
- Modify: `server-go/internal/httpapi/router.go`
- Modify: `server-go/internal/httpapi/video_handlers.go`
- Modify: `web-pc/src/api/types.ts`
- Modify: `web-pc/src/api/client.ts`

- [ ] Add API routes under `/api/v1/videos/live`.
- [ ] Add frontend types and client methods for guide sources, programs, settings, timers, and recordings.

### Task 5: Live TV UI

**Files:**
- Modify: `web-pc/src/components/windows/VideoCenterWindow.vue`

- [ ] Add Live TV tabs for channels, guide, recordings, and settings.
- [ ] Show EPG as a compact time-based list grouped by channel.
- [ ] Add one-click recording reservation from guide programs.
- [ ] Add DVR settings form for recording path, padding, max concurrency, and post-processing toggles.

### Task 6: Verification And Deployment

**Files:**
- Modify if needed: `docs/deployment-installed-software.md`

- [ ] Run `go test ./internal/video -count=1`.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `npm run build`.
- [ ] Deploy static frontend and Go backend to `192.168.81.3`.
- [ ] Verify `/healthz`, `/readyz`, systemd service status, and browser Live TV UI.
