# Reinstall Raspberry Pi with Ubuntu

## Status: PLANNING

---

## Goal

Replace Raspbian Buster (armv7l, 32-bit) with Ubuntu 24.04 LTS (arm64, 64-bit) on the Raspberry Pi 4. The new setup must be simple, low-maintenance, secure, and operable primarily by LLMs via a constrained MCP interface.

**Milestone 1 (must-have):** Ubuntu running, all current Docker containers restored, TLS connections to ENTSO-E and other external APIs working.

Everything else is phased or nice-to-have.

---

## Architecture Decisions

### Ubuntu 24.04 LTS (arm64)

Raspberry Pi 4 has a 64-bit CPU (Cortex-A72). The current OS is 32-bit Raspbian Buster. Ubuntu 24.04 LTS Server for arm64 is the right target:
- Supported until April 2029 (LTS)
- Official Raspberry Pi images from Canonical
- Modern CA bundle — fixes any TLS compatibility issues present on Buster
- Unattended security upgrades via `apt`
- All required Docker images have arm64 variants

### MCP server: configure before the upgrade

The existing `tools/mcp-pi/` MCP server runs on the developer's laptop and SSHes into the Pi. It already provides pre-approved, constrained access. Set up SSH credentials for the new Pi before reinstalling so LLM tooling can monitor and assist the restoration process from the start. Extend the MCP server with database tools as Phase 3.

---

## Compatibility: arm32v7 → arm64

Most images support multi-arch including arm64. Two require explicit attention:

| Container | Current image | Action needed |
|---|---|---|
| mariadb | `linuxserver/mariadb:arm32v7-version-10.5.12-r0` | Replace with `mariadb:10.11` (official, multi-arch) |
| zigbee2mqtt | `koenkk/zigbee2mqtt:1.18.1` | Upgrade to current release (arm64 support added in ~1.20) |
| grafana | `grafana/grafana:8.2.5` | Upgrade to 10.x or 11.x (arm64 supported) |
| prometheus | `prom/prometheus:v2.31.1` | Upgrade to v2.50+ (arm64 supported) |
| villa73 Go binaries | built for arm/v7 | Rebuild with `GOARCH=arm64` |
| Node.js frontend | pinned `22.22.0` arm/v7 | Use official Node 22 arm64 image |

All other images (InfluxDB 1.8.10, mosquitto, influxdb client libs, Python services) have arm64 variants. Home Assistant publishes its own arm64 image.

**Risk:** The MariaDB data directory is 32-bit format. A `mysqldump` (logical backup) is required — a raw volume copy will not work across architectures.

---

## Phase 0 — Preparation (do NOW, before any reinstall)

Goal: everything ready so reinstall is fast and reversible. No downtime yet.

### 0.1 Document current configuration

- [ ] Record all container env vars (redact secrets) — especially MariaDB credentials and InfluxDB database name (`homedb`)
- [ ] Record `zigbee2mqtt` coordinator model and Zigbee network config (`zigbee2mqtt/configuration.yaml`, `devices.yaml`)
- [ ] Copy all docker.env / .env files to a secure location

### 0.2 Database backups

**InfluxDB** (portable backup, stores in binary format, includes all databases):
```bash
# On the Pi
docker exec influxdb influxd backup -portable /var/lib/influxdb/backup
# Then copy out of the volume
docker cp influxdb:/var/lib/influxdb/backup ./influxdb-backup-$(date +%Y%m%d)
```

**MariaDB** (logical dump — required for cross-architecture restore):
```bash
docker exec mariadb mysqldump \
  -u root -p"${ROOT_PASSWORD}" \
  --all-databases --single-transaction --routines --triggers \
  > ./mariadb-backup-$(date +%Y%m%d).sql
```

**Verify restores before proceeding.** Test the MariaDB dump by restoring into a temporary container on the developer machine.

### 0.3 Prepare SSH access for new Pi

Generate a dedicated Ed25519 SSH key pair for MCP/LLM access (separate from your personal key):
```bash
ssh-keygen -t ed25519 -C "mcp-pi-access" -f ~/.ssh/mcp_pi_ed25519
```

Keep `~/.ssh/mcp_pi_ed25519.pub` — it will be added to `authorized_keys` during Ubuntu setup. Update `.env` in the villa73 repo to point at this new key once the Pi is reinstalled.

### 0.4 Download Ubuntu 24.04 LTS for Raspberry Pi

Download `ubuntu-24.04-preinstalled-server-arm64+raspi.img.xz` from `ubuntu.com/download/raspberry-pi`. Verify SHA256 checksum. Flash to a new SD card with Raspberry Pi Imager — use the "Advanced options" (gear icon) to:
- Set hostname (e.g. `pi73`)
- Pre-configure SSH with the `mcp-pi-access` public key above
- Set timezone `Europe/Helsinki`
- Disable password login

---

## Phase 1 — Milestone 1: Ubuntu Up and Solution Running

### 1.1 Install Ubuntu 24.04 LTS

1. Boot from the freshly flashed SD card
2. SSH in with your key: `ssh ubuntu@<ip>`
3. Run initial updates:
   ```bash
   sudo apt update && sudo apt full-upgrade -y && sudo reboot
   ```
4. Set a static IP or reserve the DHCP lease in your router (the Pi must stay at the same IP the MCP server is configured to reach)

### 1.2 Initial hardening (do early, before exposing services)

```bash
# SSH: key-only, no password, no root
sudo sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo sed -i 's/PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
sudo systemctl reload ssh

# UFW firewall — allow only what is needed
sudo apt install -y ufw
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp      # villa73-web (HTTP, internal LAN only)
sudo ufw enable

# Unattended security updates
sudo apt install -y unattended-upgrades
sudo dpkg-reconfigure --priority=low unattended-upgrades
```

Ports 6001, 6002, 5011, 3010, 3011, 3014, 3016, 8086, 9090, 1883 are **not** opened in UFW — Docker manages them and they are LAN-internal only. If you need external HTTPS access to the dashboard, add a reverse proxy with TLS in a later phase.

### 1.3 Install Docker

```bash
sudo apt install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  -o /etc/apt/keyrings/docker.asc
echo "deb [arch=arm64 signed-by=/etc/apt/keyrings/docker.asc] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list
sudo apt update && sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo usermod -aG docker ubuntu
```

Verify: `docker run --rm --platform linux/arm64 hello-world`

### 1.4 Restore databases

**InfluxDB:**
```bash
# Start InfluxDB with a fresh volume
docker run -d --name influxdb-restore \
  -v influxdb-data:/var/lib/influxdb \
  -v $(pwd)/influxdb-backup:/backup \
  influxdb:1.8.10
docker exec influxdb-restore influxd restore -portable /backup
docker stop influxdb-restore && docker rm influxdb-restore
```

**MariaDB (arm64 image):**
```bash
# Start MariaDB arm64 with empty volume
docker run -d --name mariadb-restore \
  -e MYSQL_ROOT_PASSWORD=<password> \
  -v mariadb-data:/var/lib/mysql \
  mariadb:10.11
# Wait for startup, then restore
docker exec -i mariadb-restore mysql -u root -p<password> < mariadb-backup-YYYYMMDD.sql
docker stop mariadb-restore && docker rm mariadb-restore
```

Verify: spot-check a `SELECT COUNT(*)` from `cabok_db.bookings`.

### 1.5 Deploy homeapp73-docker stack

Update `homeapp73-docker/docker-compose.yml`:
- Replace `linuxserver/mariadb:arm32v7-version-10.5.12-r0` → `mariadb:10.11`
- Upgrade `koenkk/zigbee2mqtt:1.18.1` → `koenkk/zigbee2mqtt:latest` (or pin to latest stable)
- Upgrade `grafana/grafana:8.2.5` → `grafana/grafana:11.x`
- Upgrade `prom/prometheus:v2.31.1` → `prom/prometheus:v2.53`
- Remove explicit `arm32v7` platform pins where present
- Point volume mounts at restored data (`influxdb-data`, `mariadb-data`)
- Re-attach `zigbee2mqtt` to `/dev/ttyACM0` — verify the USB coordinator is recognized (`ls /dev/ttyACM*`)

```bash
cd homeapp73-docker
docker compose up -d
```

### 1.6 Deploy villa73 stack

Rebuild Go binaries for arm64 (handled by Dockerfiles with `FROM golang:... as builder` and `GOARCH=arm64`):

```bash
cd villa73
docker compose build
docker compose up -d
```

Copy `.env` files to `backend/.env` and `backend/integrations/*/. env` from the backup taken in Phase 0.

### 1.7 Verify TLS connections

Ubuntu 24.04 ships with up-to-date CA certificates. Confirm:
```bash
curl -v https://web-api.tp.entsoe.eu/api 2>&1 | grep "SSL connection"
curl -v https://api.sunrise-sunset.org 2>&1 | grep "SSL connection"
```

If you see `SSL connection using TLSv1.3` the TLS issue from Buster is resolved. The ENTSO-E spot price endpoint should now respond correctly to `villa73-api`.

### 1.8 Smoke test

```bash
# Weather
curl http://localhost:6001/api/weathernow | jq .

# Spot prices (today)
curl "http://localhost:6001/api/electricity/prices?start=$(date -u +%Y-%m-%dT00:00:00Z)&end=$(date -u +%Y-%m-%dT23:59:59Z)&timeFormat=Europe/Helsinki" | jq .

# Calendar
curl http://localhost:6001/api/events | jq .

# Indoor climate (old stack)
curl http://localhost:5011/api/indoor/dev_upstairs | jq .

# Solar
curl http://localhost:3016/solar/current | jq .

# Dashboard
curl -s -o /dev/null -w "%{http_code}" http://localhost:80
```

**Milestone 1 is complete when all of the above return 200 and the dashboard renders correctly in a browser.**

---

## Phase 2 — Hardening

### 2.1 Additional SSH hardening

```bash
# /etc/ssh/sshd_config additions
MaxAuthTries 3
LoginGraceTime 20
AllowUsers ubuntu
ClientAliveInterval 300
ClientAliveCountMax 2
```

### 2.2 Fail2ban

```bash
sudo apt install -y fail2ban
# /etc/fail2ban/jail.local
[sshd]
enabled = true
maxretry = 5
bantime = 1h
```

### 2.3 Docker daemon hardening

`/etc/docker/daemon.json`:
```json
{
  "live-restore": true,
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "icc": false,
  "no-new-privileges": true
}
```

`"icc": false` disables inter-container communication by default — containers can only talk via explicit `networks` entries in compose files (which they already use).

### 2.4 Automatic restarts and log rotation

Docker's `restart: unless-stopped` policy on all containers is already in place. Log rotation is handled by the daemon config above.

Consider a weekly cron on the Pi that runs InfluxDB and MariaDB backups to a mounted USB drive or remote location:
```bash
# /etc/cron.weekly/backup-databases
#!/bin/bash
docker exec influxdb influxd backup -portable /backup/influxdb/$(date +%Y%m%d)
docker exec mariadb mysqldump -u root -p"${ROOT_PASSWORD}" --all-databases > /backup/mariadb/$(date +%Y%m%d).sql
```

---

## Phase 3 — Extended MCP Access for LLM Operations

Extend `tools/mcp-pi/main.go` with new constrained tools. No general shell access — each tool does exactly one named operation.

### Proposed new MCP tools

| Tool | What it does | Why needed |
|---|---|---|
| `influxdb_query` | Runs a read-only Flux/InfluxQL query against `homedb` | Check sensor data freshness, debug gaps |
| `mariadb_query` | Runs a read-only SQL SELECT against `cabok_db` | Verify cabin booking counts, last-updated timestamps |
| `docker_ps` | Returns `docker ps --format json` for both compose projects | Quick health overview |
| `container_restart` | Restarts a named container from an allowlist | Recovery without full SSH access |
| `influxdb_backup` | Triggers the backup cron and returns status | On-demand backup before changes |

**Security model:**
- `influxdb_query` and `mariadb_query` must connect via read-only database users (create these during Phase 1.4/1.5)
- `container_restart` must validate the container name against a hardcoded allowlist (same pattern as the current `target: dir1|dir2` constraint)
- No tool exposes arbitrary shell execution

### Read-only database users (create during Phase 1)

```sql
-- InfluxDB (HTTP API, no auth by default — bind to localhost only via UFW or 
-- create a read-only InfluxDB user if AUTH_ENABLED=true)

-- MariaDB
CREATE USER 'mcp_reader'@'%' IDENTIFIED BY '<strong-random-password>';
GRANT SELECT ON cabok_db.* TO 'mcp_reader'@'%';
FLUSH PRIVILEGES;
```

---

## Future / Nice-to-Have (not in Milestone 1)

- **HTTPS on the dashboard**: nginx reverse proxy with Let's Encrypt (needs a domain or internal CA)
- **Connect real indoor temperature to villa73-api**: remove the hardcoded stub; call climateapi or query InfluxDB directly from the Go API
- **PostgreSQL**: bring up the DB container and start persisting spot prices locally
- **Automated nightly backup to off-Pi storage**: USB drive or rsync to NAS
- **Grafana provisioning-as-code**: commit dashboard JSON to the repo so it survives a reinstall
- **Prometheus alert rules**: alertmanager → push notification when a container is down
- **Rootless Docker**: reduces attack surface but adds complexity; evaluate after baseline is stable

---

## Open Questions

1. **Static IP**: Assign a static IP in the router's DHCP reservation or in Ubuntu's Netplan? Router reservation is simpler and requires no config file changes on reinstall.
2. **USB coordinator**: Confirm `/dev/ttyACM0` maps to the same device on Ubuntu. If the udev rule differs, create a persistent symlink: `SUBSYSTEM=="tty", ATTRS{idVendor}=="...", SYMLINK+="zigbee-coordinator"`.
3. **Home Assistant**: It's been running for 4 years outside either compose stack. Decide whether to migrate it into a compose file or continue running it standalone. Its data directory will need a backup before reinstall.
4. **Backup destination for Phase 2 cron**: Where should backups go? USB drive, NAS, or object storage?
