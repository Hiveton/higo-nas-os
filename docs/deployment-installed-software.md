# Deployment Installed Software

Target host: `192.168.81.3`

The HiGoOS Go backend deployment installed or verified these Ubuntu packages on 2026-05-12:

- `ca-certificates` `20240203`: TLS trust store for outbound HTTPS calls.
- `curl` `8.5.0-2ubuntu10.9`: health checks and operational diagnostics.
- `smartmontools` `7.4-2build1`: SMART disk health probing through `smartctl`.
- `util-linux` `2.39.3-9ubuntu6.5`: block device, signature wipe, UUID, and mount tools such as `lsblk`, `findmnt`, `wipefs`, `blkid`, and `mount`.
- `e2fsprogs`: ext4 formatting support through `mkfs.ext4`.
- `btrfs-progs`: Btrfs formatting support through `mkfs.btrfs`.
- `zfsutils-linux` `2.2.2-0ubuntu9.4`: ZFS pool and filesystem creation through `zpool` and `zfs`.
- `zfs-zed` `2.2.2-0ubuntu9.4`: ZFS event daemon installed with Ubuntu ZFS tooling.
- `lm-sensors` `1:3.6.0-9build1`: host temperature sensor probing when hardware exposes it.
- `rsync` `3.2.7-1ubuntu1.2`: future file copy, backup, and sync workflows.

The 2026-06-03 deployment to the freshly installed Ubuntu host verified or installed these packages:

- `ca-certificates` `20240203`: TLS trust store for outbound HTTPS calls, including metadata provider requests.
- `curl` `8.5.0-2ubuntu10.9`: health checks and operational diagnostics.
- `smartmontools` `7.4-2build1`: SMART disk health probing through `smartctl`.
- `util-linux` `2.39.3-9ubuntu6.5`: block device, signature wipe, UUID, and mount tools such as `lsblk`, `findmnt`, `wipefs`, `blkid`, and `mount`.
- `e2fsprogs` `1.47.0-2.4~exp1ubuntu4.1`: ext4 formatting support through `mkfs.ext4`.
- `btrfs-progs` `6.6.3-1.1build2`: Btrfs formatting support through `mkfs.btrfs`.
- `zfsutils-linux` `2.2.2-0ubuntu9.4`: ZFS pool and filesystem creation through `zpool` and `zfs`.
- `zfs-zed` `2.2.2-0ubuntu9.4`: ZFS event daemon installed with Ubuntu ZFS tooling.
- `lm-sensors` `1:3.6.0-9build1`: host temperature sensor probing when hardware exposes it.
- `rsync` `3.2.7-1ubuntu1.4`: deployment file sync plus future file copy, backup, and sync workflows.
- `ffmpeg` `7:6.1.1-3ubuntu5`: video probing through `ffprobe` and future video transcoding through `ffmpeg`.

Installing `ffmpeg` also upgraded `libdrm-common` and `libdrm2:amd64` to `2.4.125-1ubuntu0.1~24.04.1` as apt-managed media and hardware acceleration dependencies.

The 2026-06-09 Download Center update installed these packages on `192.168.81.3`:

- `aria2` `1.37.0`: BT, torrent, magnet, and HTTP download execution through `aria2c`.
- `libaria2-0`: runtime library dependency for `aria2`.
- `libcares2`: asynchronous DNS runtime dependency installed with `aria2`.
- `libssh2-1t64`: SSH2/SFTP runtime dependency installed with `aria2`.

The Download Center validation used real external links:

- HTTP: `https://raw.githubusercontent.com/github/gitignore/main/Go.gitignore`
- BT torrent dry-run: `https://releases.ubuntu.com/24.04/ubuntu-24.04.3-live-server-amd64.iso.torrent`
- RSS feed parse: `https://xkcd.com/rss.xml`

The deployment also creates:

- `/opt/higoos/bin/higo-api`
- `/opt/higoos/bin/higo-worker`
- `/opt/higoos/bin/higoctl`
- `/opt/higoos/web`
- `/etc/higoos/server.env`
- `/srv/higoos/nas`
- `/var/lib/higoos/state`
- `/etc/systemd/system/higo-api.service`
- `/etc/systemd/system/higo-worker.service`

`higo-api.service` now runs as `root` because storage-space creation performs controlled disk formatting, `/etc/fstab` updates, ZFS pool creation, and mount operations. The backend still rejects system disks, mounted disks, unsupported modes, and missing confirmation phrases before invoking formatting tools.
