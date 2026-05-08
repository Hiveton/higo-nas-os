# Linux Adapter Design

Linux adapters are the only layer allowed to touch host operating-system capabilities. Domain services call typed adapter interfaces; adapters perform command execution, parse outputs, enforce allowlists, attach audit/task context, and translate host failures into stable API errors.

Mac development uses `devstub` adapters backed by deterministic fixtures under `server-go/fixtures/nas-root`. Devstub must never change the developer machine, start/stop host services, edit network settings, mount filesystems, or manage Docker/systemd.

## Adapter Rules

- All host commands run with explicit arguments, bounded timeouts, and no shell interpolation.
- Every adapter receives request/task/audit context and returns structured results.
- Destructive operations require a prior governance decision and confirmation reference.
- File operations are restricted to configured NAS root allowlists.
- Linux-only dependencies are checked by `higoctl doctor` before services are marked ready.
- Parsing is covered by fixture command outputs so tests do not require real hardware.

## systemd Adapter

Boundary:

- Read service state for HiGoOS services, app services, SMB/NFS/WebDAV, Docker, and configured NAS daemons.
- Start/stop/restart only allowlisted services.
- Read journal slices for diagnostics and Device Monitor logs.
- Expose graceful shutdown and readiness checks for deployment.

Linux dependencies:

- `systemctl`
- `journalctl`
- systemd unit files for `higo-api.service` and `higo-worker.service`

Devstub:

- Returns stable service states and synthetic journal rows.
- Restart/stop operations become task events without touching macOS launch services.

## Filesystem Adapter

Boundary:

- Scan configured NAS roots.
- Read metadata, extended attributes where available, safe preview bytes, directory listings.
- Move, rename, recycle, restore, version metadata, and archive completed downloads/media.
- Enforce path allowlist, symlink escape protection, max preview size, and recycle retention.

Linux dependencies:

- Standard filesystem syscalls.
- Optional `stat`, `findmnt`, `rsync`, and filesystem-specific snapshot tools through storage adapter.

Devstub:

- Uses `server-go/fixtures/nas-root` directories for family, team, photos, downloads, finance receipts, and backup archive spaces.

## SMART and Disk Adapter

Boundary:

- Inventory disks, slots, serials, capacity, temperature, health, SMART attributes.
- Start SMART short/long scans.
- Publish disk health alerts and storage task progress.

Linux dependencies:

- `/sys/block`
- `/dev/disk/by-id`
- `lsblk --json`
- `smartctl`
- optional `nvme` for NVMe health

Devstub:

- Emits deterministic disks, temperatures, and scan tasks matching the Storage Monitor UI.

## Storage Pool Adapter

Boundary:

- Represent storage pools, volumes, RAID/ZFS/Btrfs state, snapshots, repair/rebuild tasks.
- Normalize mdadm, Btrfs, and ZFS concepts into pool/volume/snapshot/task models.
- Block destructive pool operations unless governance confirms high risk.

Linux dependencies:

- `mdadm`
- `btrfs`
- `zpool` and `zfs`
- `lsblk --json`
- `findmnt --json`

Devstub:

- Returns fixed pool/volume data and simulates SMART scan, repair, and snapshot task progress.

## Docker Adapter

Boundary:

- Read stacks, containers, images, ports, mounts, env, networks, logs, resource usage.
- Start, stop, restart, and update CPU/memory limits for allowlisted app containers.
- Deny privileged mounts, host network, or sensitive paths unless explicitly permitted and confirmed.

Linux dependencies:

- Docker Engine socket or API endpoint.
- Docker Compose metadata labels when available.

Devstub:

- Returns deterministic Compose stacks and containers.
- Start/stop/restart create task events only.

## Network Adapter

Boundary:

- Read interfaces, IP addresses, DNS, gateway, DDNS state, remote tunnel state.
- Apply NAS-managed network settings only through validated settings APIs.
- Keep remote access state observable for Remote Access and Device Monitor windows.

Linux dependencies:

- `ip --json`
- `resolvectl` or system resolver files
- reverse proxy configuration hooks
- tunnel/DDNS provider client

Devstub:

- Returns synthetic LAN/WAN/DDNS/tunnel state and login alerts.

## SMB, NFS, WebDAV Adapters

Boundary:

- Represent share services, share roots, protocol status, user/group mapping, and active exports.
- Apply share configuration only from file/share/security services after ACL and risk checks.
- Feed share scan and public exposure checks.

Linux dependencies:

- Samba config and `smbstatus`
- NFS exports and service state
- WebDAV service configuration
- systemd adapter for service reloads

Devstub:

- Reports services as healthy and validates share operations against fixture roots.

## ffmpeg and Media Adapter

Boundary:

- Probe media metadata.
- Create subtitle extraction/transcode tasks.
- Generate thumbnails or derived media in managed cache paths.
- Respect file ACL and AI visibility when media content is used by assistant or Agent flows.

Linux dependencies:

- `ffmpeg`
- `ffprobe`
- optional hardware acceleration probes for GPU/NPU.

Devstub:

- Creates simulated subtitle/transcode job events and stable media metadata.

## Monitoring and Hardware Telemetry

Boundary:

- CPU, memory, disk I/O, network I/O, temperature, fan, UPS, service state, app state.
- Feed `GET /api/v1/monitoring/metrics/current`, trends, alerts, diagnostics, and event stream updates.

Linux dependencies:

- `/proc`
- `/sys`
- `sensors` when available
- UPS tools when configured
- Docker/systemd/storage adapters for cross-domain status.

Devstub:

- Emits deterministic metrics and alerts suitable for `web-pc/src/components/windows/DeviceMonitorWindow.vue` and `web-pc/src/components/TopBar.vue`.
