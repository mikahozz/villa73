# SD → HDD Migration Status

## Done

### 1. HDD cleared
- Reformatted `sda1` as FAT32 (label=`boot`)
- Reformatted `sda2` as ext4 (label=`rootfs`, UUID=`7b5c0f13-e971-4318-bb74-c68d3ecb7975`)

### 2. Root filesystem cloned (SD → HDD)
- `rsync -aHAXx` from `/` → `/dev/sda2` (mounted at `/mnt/hdd_root`)
- Excluded pseudo-filesystems: `/dev`, `/proc`, `/sys`, `/run`, `/tmp`, `/mnt`, `/media`
- Empty mount point directories created on sda2

### 3. Boot partition cloned (SD → HDD)
- `rsync -aHAX` from `/boot/firmware/` → `/dev/sda1` (mounted at `/mnt/hdd_boot`)

### 4. Boot config fixed on HDD
- `/mnt/hdd_boot/current/cmdline.txt`: changed `root=LABEL=writable` → `root=LABEL=rootfs`
- `/mnt/hdd_root/etc/fstab`: changed `LABEL=writable` → `LABEL=rootfs` and `LABEL=system-boot` → `LABEL=boot`

### 5. EEPROM boot order updated
- Changed from `0xf41` (SD first) → `0xf14` (USB/HDD first, SD fallback)
- Update staged in `/boot/firmware/pieeprom.upd` + `recovery.bin`
- Applied on first reboot; Pi rebooted automatically into HDD on second boot

### 6. Verified running from HDD ✅
```
NAME        LABEL       MOUNTPOINT
sda1        boot        /boot/firmware
sda2        rootfs      /
mmcblk0p1   system-boot (not mounted)
mmcblk0p2   writable    (not mounted)
```
- `sda2` confirmed at `/`, `sda1` at `/boot/firmware` — fully running from HDD.

### 7. Docker services verified ✅
All services started with `docker compose up -d`. Two issues found and fixed:

**Fix 1 — Grafana permission error**
- Symptom: `GF_PATHS_DATA '/var/lib/grafana' is not writable`
- Cause: `grafana/data` bind-mount directory was owned by `root` after clone
- Fix: `sudo chown -R 472:472 grafana/data`

**Fix 2 — mqttclient Python incompatibility**
- Symptom: `ModuleNotFoundError: No module named 'urllib3.packages.six.moves'`
- Cause: `FROM python:3` pulled Python 3.12+, which broke the `six.moves` virtual module trick used by the old `influxdb==5.3.0` client
- Fix: Pinned `FROM python:3.11` in `mqttclient/Dockerfile`

**Final service status:**

| Container | Status |
|---|---|
| cabinbookings | ✅ Up |
| climateapi | ✅ Up |
| electricity | ✅ Up |
| grafana | ✅ Up |
| influxdb | ✅ Up |
| mariadb | ✅ Up |
| mosquitto | ✅ Up |
| mqttclient | ✅ Up |
| sofar | ✅ Up |
| zigbee2mqtt | ✅ Up |
| cabinbookings-refresh | ⚠️ Exits (by design — `restart: "no"`) — needs `URL`, `USERNAME`, `PASSWORD`, `DBUSER`, `DBPASSWORD` env vars set before running manually |

### 8. Crontab — nothing to migrate
- SD card had no custom cron jobs; only standard Ubuntu system crontab
- HDD crontab is identical (cloned from SD)

---

## Pending

### Remove SD card
The EEPROM boot order is `0xf14` (USB first). Once you're satisfied everything is working, the SD card can be physically removed — the Pi will boot from HDD without it.

### Configure cabinbookings-refresh credentials
Add to `.env` or pass via `docker compose run`:
- `URL` — target booking site URL
- `USERNAME` / `PASSWORD` — login credentials
- `DBUSER` / `DBPASSWORD` — MariaDB credentials

---

## Reference: partition layout

| Device     | Label        | Role              |
|------------|--------------|-------------------|
| mmcblk0p1  | system-boot  | SD boot (old)     |
| mmcblk0p2  | writable     | SD root (old)     |
| sda1       | boot         | HDD boot (active) |
| sda2       | rootfs       | HDD root (active) |
