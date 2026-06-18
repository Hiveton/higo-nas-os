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

The 2026-06-10 Docker Center update installed these packages on `192.168.81.3`:

- `docker.io` `29.1.3-0ubuntu3~24.04.2`: Docker Engine and CLI used by the Docker Center backend for real container, stack, log, port, mount, environment, and resource-limit operations.
- `docker-compose-v2` `2.40.3+ds1-0ubuntu1~24.04.1`: Docker Compose v2 plugin for Compose stack management and future compose-file operations.
- `containerd` `2.2.1-0ubuntu1~24.04.2`: container runtime dependency installed with Docker.
- `runc` `1.3.4-0ubuntu1~24.04.1`: OCI runtime dependency installed with Docker.
- `bridge-utils` `1.7.1-1ubuntu2`: bridge network helper installed with Docker networking support.
- `dnsmasq-base` `2.90-2ubuntu0.3`: DNS helper dependency installed with Docker networking support.
- `dns-root-data` `2024071801~ubuntu0.24.04.1`: DNS root data dependency installed with Docker networking support.
- `pigz` `2.8-1`: parallel gzip helper installed with Docker image layer handling.
- `ubuntu-fan` `0.12.16+24.04.1`: Ubuntu FAN networking support installed with Docker.

Docker was enabled through `systemctl enable --now docker`, and user `xuehui` was added to the `docker` group for interactive CLI access after the next login.

The Docker Center validation used the real Docker Engine API path through the HiGoOS backend:

- Verified `/api/v1/docker/containers`, `/api/v1/docker/stacks`, `/api/v1/docker/images`, `/api/v1/docker/volumes`, and `/api/v1/docker/networks` against the local Docker daemon.
- Created a temporary `hello-world:latest` container through `/api/v1/docker/containers`.
- Deleted the temporary container through `DELETE /api/v1/docker/containers/{id}`.
- Removed the temporary `hello-world:latest` image after the smoke test so the Docker Center does not retain demo data.

The 2026-06-15 UI layout deployment to `192.168.81.3` verified these packages with `apt-get install -y`; apt reported `0 newly installed` because the required runtime packages were already present:

- `ca-certificates` `20240203`: TLS trust store for outbound HTTPS calls.
- `curl` `8.5.0-2ubuntu10.9`: health checks and operational diagnostics.
- `smartmontools` `7.4-2build1`: SMART disk health probing through `smartctl`.
- `util-linux` `2.39.3-9ubuntu6.5`: block device, signature wipe, UUID, and mount tools such as `lsblk`, `findmnt`, `wipefs`, `blkid`, and `mount`.
- `e2fsprogs` `1.47.0-2.4~exp1ubuntu4.1`: ext4 formatting support through `mkfs.ext4`.
- `btrfs-progs` `6.6.3-1.1build2`: Btrfs formatting support through `mkfs.btrfs`.
- `zfsutils-linux` `2.2.2-0ubuntu9.4`: ZFS pool and filesystem creation through `zpool` and `zfs`.
- `zfs-zed` `2.2.2-0ubuntu9.4`: ZFS event daemon installed with Ubuntu ZFS tooling.
- `lm-sensors` `1:3.6.0-9build1`: host temperature sensor probing when hardware exposes it.
- `rsync` `3.2.7-1ubuntu1.5`: deployment file sync plus future file copy, backup, and sync workflows.
- `ffmpeg` `7:6.1.1-3ubuntu5`: video probing through `ffprobe` and future video transcoding through `ffmpeg`.
- `aria2` `1.37.0+debian-1build3`: BT, torrent, magnet, and HTTP download execution through `aria2c`.
- `libaria2-0` `1.37.0+debian-1build3`: runtime library dependency for `aria2`.
- `libcares2` `1.27.0-1.0ubuntu1`: asynchronous DNS runtime dependency installed with `aria2`.
- `libssh2-1t64` `1.11.0-4.1ubuntu0.24.04.1`: SSH2/SFTP runtime dependency installed with `aria2`.
- `docker.io` `29.1.3-0ubuntu3~24.04.2`: Docker Engine and CLI used by the Docker Center backend.
- `docker-compose-v2` `2.40.3+ds1-0ubuntu1~24.04.1`: Docker Compose v2 plugin.
- `containerd` `2.2.1-0ubuntu1~24.04.2`: container runtime dependency installed with Docker.
- `runc` `1.3.4-0ubuntu1~24.04.1`: OCI runtime dependency installed with Docker.
- `bridge-utils` `1.7.1-1ubuntu2`: bridge network helper installed with Docker networking support.
- `dnsmasq-base` `2.90-2ubuntu0.3`: DNS helper dependency installed with Docker networking support.
- `dns-root-data` `2024071801~ubuntu0.24.04.1`: DNS root data dependency installed with Docker networking support.
- `pigz` `2.8-1`: parallel gzip helper installed with Docker image layer handling.
- `ubuntu-fan` `0.12.16+24.04.1`: Ubuntu FAN networking support installed with Docker.

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
