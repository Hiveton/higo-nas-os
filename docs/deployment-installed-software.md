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
