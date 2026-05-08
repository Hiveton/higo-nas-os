# HiGoOS Web PC Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Vue 3 PC web desktop prototype for HiGoOS in `web-pc`, faithfully using the architecture document and Dock cut assets.

**Architecture:** The app is a Vite + Vue 3 single page desktop shell. Data, navigation, and app definitions live in small config modules; visual regions are split into focused components for the top bar, Dock, desktop widgets, app windows, AI assistant, and module-specific panels.

**Tech Stack:** Vue 3, Vite, TypeScript, CSS variables, local PNG assets from `assets/design/higoos-dock`.

---

## File Structure

- `web-pc/package.json`: npm scripts and dependencies.
- `web-pc/index.html`: Vite HTML entry.
- `web-pc/src/main.ts`: Vue app bootstrap.
- `web-pc/src/App.vue`: screen composition and window state.
- `web-pc/src/data/higoos.ts`: app icons, windows, storage, files, agents, assistant, alerts.
- `web-pc/src/components/TopBar.vue`: global status bar.
- `web-pc/src/components/DesktopDock.vue`: macOS-inspired Dock using generated PNG icons.
- `web-pc/src/components/DesktopWindow.vue`: reusable desktop window frame.
- `web-pc/src/components/DesktopWidgets.vue`: storage, backup, system, security widgets.
- `web-pc/src/components/AiAssistantPanel.vue`: persistent AI assistant side panel.
- `web-pc/src/components/windows/*.vue`: File Manager, AI File Steward, Agent Workbench, Storage Monitor.
- `web-pc/src/styles/*.css`: design tokens, base reset, component styles, responsive behavior.
- `web-pc/src/assets/higoos-dock/*`: copied generated assets.

## Tasks

### Task 1: Scaffold Vue 3 App and Data Model

**Files:**
- Create: `web-pc/package.json`
- Create: `web-pc/index.html`
- Create: `web-pc/src/main.ts`
- Create: `web-pc/src/data/higoos.ts`
- Create: `web-pc/src/styles/tokens.css`
- Create: `web-pc/src/styles/base.css`
- Copy assets into: `web-pc/src/assets/higoos-dock/`

- [ ] Create a Vite Vue 3 TypeScript project skeleton under `web-pc`.
- [ ] Copy wallpaper, Dock base, preview, icon sheet, and 14 independent icon PNGs from `assets/design/higoos-dock`.
- [ ] Define app metadata for 文件管理, 存储管理, AI 文件管家, Agent 工作台, AI 助手, 备份同步, 相册媒体, 下载中心, 应用中心, Docker, 安全中心, 设备监控, 系统设置, 远程访问.
- [ ] Define seeded data for files, storage pools, Agent workflows, audit entries, assistant messages, alerts, and system metrics.

### Task 2: Desktop Shell and Dock Navigation

**Files:**
- Create: `web-pc/src/components/TopBar.vue`
- Create: `web-pc/src/components/DesktopDock.vue`
- Modify: `web-pc/src/App.vue`
- Modify: `web-pc/src/styles/base.css`

- [ ] Build full-screen desktop using the generated wallpaper.
- [ ] Build top system bar with HiGoOS name, semantic search, CPU/RAM/network, notifications, model mode, and avatar.
- [ ] Build floating bottom Dock with generated `dock-base.png`, generated icons, active indicators, badges, hover labels, and click-to-open behavior.
- [ ] Preserve PC desktop interaction feel: windows remain above Dock, Dock stays visually prominent.

### Task 3: Application Windows and AI-NAS Panels

**Files:**
- Create: `web-pc/src/components/DesktopWindow.vue`
- Create: `web-pc/src/components/DesktopWidgets.vue`
- Create: `web-pc/src/components/AiAssistantPanel.vue`
- Create: `web-pc/src/components/windows/FileManagerWindow.vue`
- Create: `web-pc/src/components/windows/AiStewardWindow.vue`
- Create: `web-pc/src/components/windows/AgentWorkbenchWindow.vue`
- Create: `web-pc/src/components/windows/StorageMonitorWindow.vue`
- Modify: `web-pc/src/App.vue`

- [ ] Implement reusable window chrome with title, status badge, traffic-light controls, and content slot.
- [ ] Implement File Manager with folder tree, breadcrumb, semantic search, table rows, tags, permission badges, and preview panel.
- [ ] Implement AI 文件管家 with smart整理 suggestions, duplicate insight, risk cards, rollback/audit records.
- [ ] Implement Agent 工作台 with templates, workflow nodes, tool permissions, risk level, execution log, and confirmation checkpoint.
- [ ] Implement Storage Monitor with RAID 5, disk bays, SMART/temperature, and capacity charts.
- [ ] Implement persistent AI assistant panel tied to architecture concepts: files, storage, backups, permissions, local/cloud model strategy.

### Task 4: Visual Fidelity, Responsiveness, and Verification

**Files:**
- Modify: `web-pc/src/styles/tokens.css`
- Modify: `web-pc/src/styles/base.css`
- Modify: relevant components only for polish.

- [ ] Match the generated Dock design: light OS wallpaper, frosted Dock, colorful icons, active glows, calm blue/cyan/green palette.
- [ ] Ensure desktop layout works at 1440px, 1920px, and tablet-width fallback.
- [ ] Add meaningful hover, selected, and click states without fake dead controls.
- [ ] Run `npm install`, `npm run build`, and launch the dev server.
- [ ] Capture/inspect the browser result and compare against `assets/design/higoos-dock/dock-preview.png`.
