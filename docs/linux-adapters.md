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

## Protocols adapter (`internal/protocols`)

`HostAdapter` (Linux) drives the sharing-protocol stack; `DevAdapter` (Mac) is an optimistic devstub. All shell-outs go through an injectable `commandRunner` (unit-tested with a fake), and every config write is an atomic temp-file + rename of a HiGoOS-owned file.

- **Status**: `systemctl list-unit-files <unit>` (installed), `systemctl is-active <unit>` (running) per protocol. A missing unit yields `Installed:false` without failing the whole list.
- **Enable/Disable**: `systemctl enable|disable --now <unit>` (`smb`→`smbd`+`nmbd`, `nfs`→`nfs-kernel-server`, `webdav`→`higoos-webdav`, `dlna`→`minidlna`).
- **Apply shares** (whole-file regenerate from the desired list, then reload):
  - SMB → rewrite `/etc/samba/smb.conf.d/higoos.conf` (one `[name]` stanza per share) + `smbcontrol all reload-config`. The user's `smb.conf` is only ever appended-to once (the `include =` line).
  - NFS → rewrite `/etc/exports.d/higoos.exports` + `exportfs -ra`. `/etc/exports` is never touched.
  - DLNA → replace the delimited `# >>> HiGoOS media_dir >>>` block in `/etc/minidlna.conf` (user lines outside it survive) + `systemctl restart minidlna`.
  - WebDAV → rewrite `/etc/higoos/webdav/higoos-dav.conf` + `systemctl reload higoos-webdav` (a dedicated Apache instance on port 8081).

Devstub: returns success for all mutations; `Status` reports every protocol installed with `Running` tracking the desired toggle.

## Accounts / identity adapter (`internal/accounts`)

User identity is OS-authoritative on the NAS host: HiGoOS accounts *are* Linux system users. A `Directory` interface fronts the backend — `HostDirectory` (Linux) drives the real system-user database; `DevDirectory` (Mac) is a sidecar-only devstub. All host shell-outs go through an injectable runner (unit-tested with a fake). App-level metadata HiGoOS adds (role, quota, space grants, sessions, MFA) lives in sidecar JSON keyed by username — the OS never stores it.

Boundary:

- **Provision / mutate**: create = `useradd -m -g higoos -s /usr/sbin/nologin -u <uid≥base> -c <gecos>`; update = `usermod`; delete = `userdel -r`; lock/unlock = `usermod -L`/`-U`; group membership = `gpasswd -a/-d`, `groupadd`.
- **Credentials**: set = `chpasswd -c SHA512` (forces a `$6$` hash regardless of the host default), plus best-effort `smbpasswd -s -a` so SMB shares share the credential. Verify = read `/etc/shadow` and recompute the hash in pure Go — yescrypt (`$y$`/`$gy$`, Ubuntu 24.04's default) and crypt(3) `$6$`/`$5$`/`$1$` — so any pre-existing OS password authenticates, with no cgo/PAM dependency.
- **Enumerate**: `getent passwd` lists real human accounts (UID ∈ [1000, 60000], excluding `root`/daemons/`nobody`); `getent group <higoos-admins|sudo|wheel>` resolves the admin role. The service reconciles these into its user list at start-up and on every account listing, so a pre-existing sudo user (e.g. `hiveton`) appears and can log in without HiGoOS having created it.
- **Managed scope & safety**: HiGoOS-created users live in group `higoos`, UID ≥ `HIGO_ACCOUNTS_UID_BASE` (3000), shell `/usr/sbin/nologin`. Deletion is refused for accounts below the managed UID base, so an installer/sudo user can never be removed through the API. Requires the process to run as root (the `higo-api.service` unit does) to manage users and read `/etc/shadow`.

Linux dependencies:

- `useradd` / `usermod` / `userdel` / `chpasswd` / `gpasswd` / `groupadd` (shadow-utils)
- `getent` for the passwd/group databases and `/etc/shadow` read access (root)
- optional `smbpasswd` (Samba) — absent hosts simply skip the SMB sync

Devstub: keeps SHA-512 credentials in `credentials.json`, never touches host users, and `List` returns nothing — so on Mac the accounts service stays sidecar-authoritative and development is unaffected.

### Filesystem provisioner

A companion `Provisioner` (`HostProvisioner` on Linux, `DevProvisioner` no-op elsewhere) realizes the user/group/grant model on the real filesystem under the NAS root, so identity actually owns storage:

- **Personal folder**: creating a user (or reconciling an existing OS user) provisions `<NAS_ROOT>/homes/<username>`, `chown <user>:<group>`, `chmod 0700`. A per-user block quota is applied best-effort via `setquota` (only takes effect when the filesystem is mounted with `usrquota`).
- **Group folder**: creating a HiGoOS group provisions a real system group (named by the stable group ID) plus `<NAS_ROOT>/groups/<groupID>`, `chown root:<group>`, `chmod 2770` (setgid so new files inherit the group); membership is synced with `gpasswd -M`.
- **Grants → ACLs**: a `SpaceGrant` is projected onto the target directory with `setfacl` — a recursive ACL plus a default ACL so new children inherit (`read`→`rX`, `read_write`/`manage`→`rwX`, for `u:<user>` or `g:<group>`). Revoking clears it with `setfacl -x`.
- All paths are confined to the NAS root; every shell-out goes through the injectable runner (unit-tested with a fake). Requires `acl` (setfacl) and, for quotas, `quota` (setquota).

The web layer additionally scopes the file tree to the caller (`files.TreeFor`/`CanAccess`): a non-admin sees only their personal folder, their group folders, and granted shared spaces, and is denied reads outside that set — mirroring the on-disk ownership/ACLs for the API (which runs as root and would otherwise bypass filesystem permissions). SMB/NFS/WebDAV honor the ownership/ACLs directly.
