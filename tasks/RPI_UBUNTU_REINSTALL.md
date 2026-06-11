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
- All required Docker images have arm64 variants

### MCP server: configure before the upgrade

The existing `tools/mcp-pi/` MCP server runs on the developer's laptop and SSHes into the Pi. It already provides pre-approved, constrained access. Set up SSH credentials for the new Pi before reinstalling so LLM tooling can monitor and assist the restoration process from the start. Extend the MCP server with database tools as Phase 3.

---

## Compatibility: arm32v7 → arm64

Most images support multi-arch including arm64. Two require explicit attention:

| Container           | Current image                                    | Action needed                                             |
| ------------------- | ------------------------------------------------ | --------------------------------------------------------- |
| mariadb             | `linuxserver/mariadb:arm32v7-version-10.5.12-r0` | Replace with `mariadb:10.11` (official, multi-arch)       |
| zigbee2mqtt         | `koenkk/zigbee2mqtt:1.18.1`                      | Upgrade to current release (arm64 support added in ~1.20) |
| grafana             | `grafana/grafana:8.2.5`                          | Upgrade to 10.x or 11.x (arm64 supported)                 |
| prometheus          | `prom/prometheus:v2.31.1`                        | Upgrade to v2.50+ (arm64 supported)                       |
| villa73 Go binaries | built for arm/v7                                 | Rebuild with `GOARCH=arm64`                               |
| Node.js frontend    | pinned `22.22.0` arm/v7                          | Use official Node 22 arm64 image                          |

All other images (InfluxDB 1.8.10, mosquitto, influxdb client libs, Python services) have arm64 variants. Home Assistant publishes its own arm64 image.

**Risk:** The MariaDB data directory is 32-bit format. A `mysqldump` (logical backup) is required — a raw volume copy will not work across architectures.

---

## Variables used in laptop commands

Commands run **from the laptop** below reference `PI_USERNAME` and `PI_HOST`. Export them in your laptop shell so the commands can be copy-pasted as-is. The Pi keeps the same IP across the reinstall (see 1.1 step 4), but the username changes — `PI_USERNAME` is whatever you set in the Imager (0.4); update the export once you reach that point:

```bash
export PI_USERNAME=<pi-ssh-username>
export PI_HOST=<pi-ip-or-hostname>
```

---

## Phase 0 — Preparation (do NOW, before any reinstall)

Goal: everything ready so reinstall is fast and reversible. No downtime yet.

### 0.1 Document current configuration

- [x] Record all container env vars (redact secrets) — especially MariaDB credentials and InfluxDB database name (`homedb`)
- [x] Record `zigbee2mqtt` coordinator model and Zigbee network config (`zigbee2mqtt/configuration.yaml`, `devices.yaml`) — see `local-notes.md`
- [x] Copy all docker.env / .env files to a secure location

### 0.2 Database backups

- [x] **InfluxDB** (portable backup, stores in binary format, includes all databases):

```bash
# On the Pi
docker exec influxdb influxd backup -portable /var/lib/influxdb/backup
# Then copy out of the volume
docker cp influxdb:/var/lib/influxdb/backup ./dbbackup-influxdb-$(date +%Y%m%d)
# And copy to laptop
scp -r ${PI_USERNAME}@${PI_HOST}:~/dbbackup-influxdb-$(date +%Y%m%d) .
```

- [x] **MariaDB** (logical dump — required for cross-architecture restore):

```bash
docker exec mariadb mysqldump \
  -u root -p"${MYSQL_ROOT_PASSWORD}" \
  --single-transaction --routines --triggers \
  cabok_db \
  > ./dbbackup-mariadb-$(date +%Y%m%d).sql
# And copy to laptop
scp -r ${PI_USERNAME}@${PI_HOST}:~/dbbackup-mariadb-$(date +%Y%m%d) .
```

- [x] **Verify restores before proceeding.** Test the MariaDB dump and Influx backup by restoring into a temporary container on the developer machine.

### [x] 0.3 Prepare SSH access for new Pi

- Generate a dedicated Ed25519 SSH key pair for MCP/LLM access (separate from your personal key):

```bash
ssh-keygen -t ed25519 -C "mcp-pi-access" -f ~/.ssh/mcp_pi_ed25519
```

Keep `~/.ssh/mcp_pi_ed25519.pub` — it will be added to `authorized_keys` during Ubuntu setup. Update `.env` in the villa73 repo to point at this new key once the Pi is reinstalled.

### [x] Prepare Raspberry PI 4 for Ubuntu 26.04

- Run `rpi-eeprom-update` to check if date is later than 2022-11-25
- If not, run `sudo rpi-eeprom-update -a`, restart and check again

### [x] 0.4 Download and Flash Ubuntu for Raspberry Pi

- Download **Raspberry Pi Imager** from `raspberrypi.com/software` (not Balena Etcher — it has no customization options). Install and open it.
- In Imager: choose OS → Other general-purpose OS → Ubuntu → Ubuntu Server 26.04 LTS (64-bit).
- Before writing, click the gear icon (or Ctrl+Shift+X) and configure:
  - Hostname: `somehostname`
  - User: `${PI_USERNAME}` (the value you exported above)
  - Enable SSH → "Allow public-key authentication only" → paste contents of your public key(s) in `~/.ssh`
  - Set a temporary password too (useful fallback if key auth fails)
  - Timezone: `Europe/Helsinki`
- Write to the SD card.

### [x] 0.5 Fallback: SSH setup failed — debug via direct console

If the Pi doesn't appear on the network or SSH connections are refused/denied, connect a monitor and keyboard directly to the Pi and log in at the console with the username/password set during imaging (or the default `ubuntu`/`ubuntu`, which forces a password change on first login).

Once logged in at the console, work through these checks in order:

**1. Confirm cloud-init finished applying your customization**

```bash
cloud-init status --long
```

If it shows `status: running` or `status: error`, wait a minute and re-check, or read `/var/log/cloud-init.log` / `/var/log/cloud-init-output.log` for the failure. A failed cloud-init run is the most common reason customization (hostname, SSH key, user) silently didn't apply.

**2. Confirm the network has an IP address**

```bash
ip a
hostname -I
```

If there's no IP on `eth0`/`end0`, the cable/DHCP is the problem, not SSH. Re-check the cable and your router's DHCP leases.

**3. Confirm the SSH service is installed, enabled, and running**

```bash
systemctl status ssh
sudo systemctl enable ssh --now
```

Ubuntu Server normally ships with `openssh-server` preinstalled and enabled; if it's missing:

```bash
sudo apt update && sudo apt install -y openssh-server
sudo systemctl enable ssh --now
```

**4. Confirm your public key actually landed in authorized_keys**

```bash
cat ~/.ssh/authorized_keys
```

If it's empty or missing, add it manually. On your laptop, print the key:

```bash
cat ~/.ssh/mcp_pi_ed25519.pub
```

Copy that single line, then on the Pi:

```bash
mkdir -p ~/.ssh && chmod 700 ~/.ssh
echo "<paste the public key line here>" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

**5. Confirm sshd_config allows the auth method you're using**

```bash
sudo sshd -T | grep -iE "passwordauthentication|pubkeyauthentication|permitrootlogin"
```

`pubkeyauthentication` should be `yes`. If you're temporarily relying on a password, `passwordauthentication` must also be `yes` (Ubuntu's default cloud-init can disable it when a key is supplied). Edit `/etc/ssh/sshd_config` if needed and reload:

```bash
sudo systemctl reload ssh
```

**6. Confirm the firewall isn't blocking port 22**

```bash
sudo ufw status
```

On a fresh Ubuntu Server image UFW is inactive by default, so this is unlikely — but worth ruling out before you go further.

**7. Re-test from your laptop**

```bash
ssh -i ~/.ssh/mcp_pi_ed25519 -v ${PI_USERNAME}@${PI_HOST}
```

The `-v` flag shows exactly which keys are offered and why the server rejects them — this is the fastest way to pinpoint a remaining mismatch (wrong username, wrong key, key not in authorized_keys, or auth method disabled).

Once SSH access is confirmed working from the laptop, continue with Phase 1.

---

## Phase 1 — Milestone 1: Ubuntu Up and Solution Running

### 1.1 [x] Install Ubuntu

1. Boot from the freshly flashed SD card
2. SSH in with your public key: `ssh ${PI_USERNAME}@${PI_HOST}`
3. Run initial updates:
   ```bash
   sudo apt update && sudo apt full-upgrade -y && sudo reboot
   ```
4. Set a static IP or reserve the DHCP lease in your router (the Pi must stay at the same IP the MCP server is configured to reach)

### 1.2 [x] Initial hardening (do early, before exposing services)

```bash
# SSH: key-only, no password, no root
sudo sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo sed -i 's/PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
sudo systemctl reload ssh

# UFW firewall — for non-Docker traffic (SSH, host-level services)
sudo apt install -y ufw
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw enable
```

Automatic/unattended upgrades are intentionally **not** configured — package updates can land
mid-session and break a running container or service without warning, and on a "minimal touch,
LLM-operated" box it's preferable to apply updates deliberately (e.g. `sudo apt update && sudo
apt upgrade` during a planned maintenance window) so any breakage is noticed and tied to a known
change rather than discovered later as a mystery outage.

**Important — Docker bypasses UFW.** Docker inserts its own rules into the `DOCKER`/`DOCKER-USER`
iptables chains, which are evaluated _before_ UFW's filter chain. Any port published with
`ports:` in a compose file (e.g. `"6001:6001"`) is reachable from the whole LAN regardless of
`ufw allow`/`deny` — the `ufw allow ssh` above only matters for host-level services, not
container ports. This trips up a lot of people; see
[github.com/chaifeng/ufw-docker](https://github.com/chaifeng/ufw-docker) for background.

Of the published container ports, only **80** (villa73-web, viewed from the tablet) and
optionally **3010** (Grafana, if viewed from a LAN browser) need to be reachable from other
devices on the LAN. Everything else (6001, 6002, 5011, 3011, 3014, 3016, 8086, 9090, 1883) is
only ever called by other containers on the same Docker network and should not be exposed at
all. The correct fix is to bind those container ports to localhost in the compose files, so
Docker never publishes them to the LAN interface in the first place:

```yaml
# before (reachable from the whole LAN, bypassing UFW)
ports:
  - "6001:6001"

# after (only reachable from the Pi itself)
ports:
  - "127.0.0.1:6001:6001"
```

Apply `127.0.0.1:` binding to every `ports:` entry except `villa73-web` (port 80) and, if
needed, `grafana` (port 3010). Containers that talk to each other over the shared Docker
network (e.g. villa73-web → villa73-api, mqttclient → influxdb) keep working unchanged —
inter-container traffic never goes through the published host port. If you'd rather keep UFW as
the single source of truth for firewall rules instead of editing every compose file, install
[`ufw-docker`](https://github.com/chaifeng/ufw-docker), which rewrites Docker's iptables rules
to respect UFW.

If you need external HTTPS access to the dashboard from outside the LAN, add a reverse proxy
with TLS in a later phase rather than exposing container ports directly.

### 1.3 [x] Install Docker

```bash
sudo apt install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  -o /etc/apt/keyrings/docker.asc
echo "deb [arch=arm64 signed-by=/etc/apt/keyrings/docker.asc] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list
sudo apt update && sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo usermod -aG docker $USER   # adds the currently logged-in user to the docker group
```

Verify: `docker run --rm --platform linux/arm64 hello-world`

### 1.4 [x] Restore databases

**Copy backup files from laptop to the Pi** (reverse of the `scp` in 0.2 — run from the laptop, into a working directory on the Pi, e.g. `~/restore`):

```bash
ssh ${PI_USERNAME}@${PI_HOST} "mkdir -p ~/restore"
scp -r ./dbbackup-influxdb-20260606 ${PI_USERNAME}@${PI_HOST}:~/restore/influxdb-backup
scp ./dbbackup-mariadb-20260606.sql ${PI_USERNAME}@${PI_HOST}:~/restore/mariadb-backup-20260606.sql
```

Then `ssh` into the Pi and `cd ~/restore` before running the restore commands below — they expect `./influxdb-backup/` and `./mariadb-backup-20260606.sql` relative to the current directory.

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
export MYSQL_ROOT_PASSWORD=<password>

# Start MariaDB arm64 with empty volume
docker run -d --name mariadb-restore \
  -e MYSQL_ROOT_PASSWORD \
  -v mariadb-data:/var/lib/mysql \
  mariadb:10.11

# Wait for the *second* "ready for connections" — the official entrypoint
# bootstraps the mysql system DB, restarts mysqld, then accepts connections
until [ "$(docker logs mariadb-restore 2>&1 | grep -c 'ready for connections')" -ge 2 ]; do
  sleep 2
done

docker exec -i mariadb-restore mysql -u root -p$MYSQL_ROOT_PASSWORD \
  -e "CREATE DATABASE IF NOT EXISTS cabok_db"
docker exec -i mariadb-restore mysql -u root -p$MYSQL_ROOT_PASSWORD cabok_db < mariadb-backup-YYYYMMDD.sql
docker stop mariadb-restore && docker rm mariadb-restore
```

Verify: spot-check a `SELECT COUNT(*)` from `cabok_db.bookings`.

### 1.5 Deploy homeapp73-docker stack

Update `homeapp73-docker/docker-compose.yml`:

- Replace `linuxserver/mariadb:arm32v7-version-10.5.12-r0` → `mariadb:10.11`
- Upgrade `koenkk/zigbee2mqtt:1.18.1` → `koenkk/zigbee2mqtt:2.11.0`
- Upgrade `grafana/grafana:8.2.5` → `grafana/grafana:11.6.15`
- Remove explicit `arm32v7` platform pins where present
- Point volume mounts at restored data (`influxdb-data`, `mariadb-data`)
- Re-attach `zigbee2mqtt` to `/dev/ttyACM0` — verify the USB coordinator is recognized (`ls /dev/ttyACM*`)

Copy env files from the backup machine before starting:

```bash
scp /path/to/backup/homeapp73-docker/.env pi@villa73.local:~/homeapp73-docker/.env
scp /path/to/backup/homeapp73-docker/mariadb/docker.env pi@villa73.local:~/homeapp73-docker/mariadb/docker.env
```

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

| Tool                | What it does                                                | Why needed                                           |
| ------------------- | ----------------------------------------------------------- | ---------------------------------------------------- |
| `influxdb_query`    | Runs a read-only Flux/InfluxQL query against `homedb`       | Check sensor data freshness, debug gaps              |
| `mariadb_query`     | Runs a read-only SQL SELECT against `cabok_db`              | Verify cabin booking counts, last-updated timestamps |
| `docker_ps`         | Returns `docker ps --format json` for both compose projects | Quick health overview                                |
| `container_restart` | Restarts a named container from an allowlist                | Recovery without full SSH access                     |
| `influxdb_backup`   | Triggers the backup cron and returns status                 | On-demand backup before changes                      |

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
