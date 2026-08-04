# Raspberry Pi Power Management & Fan Control

## Status: IN_PROGRESS

---

## Goal

Reduce energy consumption and fan noise on the Raspberry Pi 4 (Argon ONE case) running the
villa73 home automation stack. The machine runs 24/7 with many Docker services and has an
active fan that spins unnecessarily at idle temperatures.

**Minimum requirement:** Lower power draw and fan noise all day long without disrupting
running services.

**Ideal (future):** Machine suspends when idle and wakes on network activity.

---

## Hardware

- Raspberry Pi 4 Model B Rev 1.1 (aarch64)
- Argon ONE case with I2C-controlled fan (I2C bus 1, address `0x1a`)
- Boot: SSD via USB (not SD card)
- Hostname: `argon`

---

## Option A — Argon ONE Fan Control

**Status: DONE**

The Argon ONE case fan is controlled via I2C. Without a fan daemon the fan runs at an
uncontrolled speed. Installing a daemon with a custom temperature curve silences the fan
at normal idle temperatures (~41°C).

### Steps

#### A.1 [x] Install dependencies

```bash
sudo apt-get install -y python3-smbus i2c-tools
```

Verify I2C device is present at address `0x1a`:

```bash
i2cdetect -y 1
# Should show "1a" in the grid
```

#### A.2 [x] Create fan daemon

File: `/usr/local/bin/argonone-fan`

```python
#!/usr/bin/env python3
"""Argon ONE fan controller daemon — controls fan via I2C 0x1a on bus 1."""

import sys
import time
import signal
import smbus

TEMP_FILE = "/sys/class/thermal/thermal_zone0/temp"
I2C_BUS = 1
I2C_ADDR = 0x1a
POLL_INTERVAL = 10  # seconds

# Fan curve: (temp_threshold_celsius, fan_speed_percent)
FAN_CURVE = [
    (70, 100),
    (65,  80),
    (60,  50),
    (55,  30),
    (0,    0),
]

def read_temp():
    with open(TEMP_FILE) as f:
        return int(f.read().strip()) / 1000.0

def set_fan(bus, speed):
    bus.write_byte(I2C_ADDR, speed)

def get_target_speed(temp):
    for threshold, speed in FAN_CURVE:
        if temp >= threshold:
            return speed
    return 0

def main():
    bus = smbus.SMBus(I2C_BUS)
    current_speed = -1

    def shutdown(sig, frame):
        set_fan(bus, 0)
        sys.exit(0)

    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)

    while True:
        temp = read_temp()
        target = get_target_speed(temp)
        if target != current_speed:
            set_fan(bus, target)
            current_speed = target
        time.sleep(POLL_INTERVAL)

if __name__ == "__main__":
    main()
```

```bash
sudo chmod +x /usr/local/bin/argonone-fan
```

#### A.3 [x] Create and enable systemd service

File: `/etc/systemd/system/argonone-fan.service`

```ini
[Unit]
Description=Argon ONE Fan Controller
After=multi-user.target

[Service]
Type=simple
ExecStart=/usr/local/bin/argonone-fan
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable argonone-fan
sudo systemctl start argonone-fan
sudo systemctl status argonone-fan
```

### Fan curve

| CPU temp   | Fan speed |
| ---------- | --------- |
| < 55°C     | 0% (off)  |
| 55–60°C    | 30%       |
| 60–65°C    | 50%       |
| 65–70°C    | 80%       |
| ≥ 70°C     | 100%      |

At normal idle (~41°C) the fan is completely silent. The fan turns off on service stop
(SIGTERM handler writes 0 to I2C before exiting).

### Adjusting the fan curve

Edit the `FAN_CURVE` list in `/usr/local/bin/argonone-fan` and restart the service:

```bash
sudo systemctl restart argonone-fan
```

---

## Option B — CPU Governor & Unused Services

**Status: DONE**

### B.1 [x] Switch CPU governor to `schedutil`

`schedutil` adapts frequency to scheduler load signals and performs better than `ondemand`
on modern kernels. Applied to all four cores.

```bash
for f in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
  echo schedutil | sudo tee $f
done
```

Made persistent via systemd oneshot service:

File: `/etc/systemd/system/cpu-governor.service`

```ini
[Unit]
Description=Set CPU scaling governor to schedutil
After=sysinit.target

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'for f in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do echo schedutil > $f; done'
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable cpu-governor
```

### B.2 [x] Disable ModemManager

No modem is attached to this machine. ModemManager was consuming ~13 MB RAM and CPU cycles
probing nonexistent modems.

```bash
sudo systemctl stop ModemManager
sudo systemctl disable ModemManager
```

### B.3 [ ] Evaluate snapd

Only `snapd` itself is installed (no user snaps). Consider removing it to free ~50 MB RAM
and eliminate periodic snap refresh polling:

```bash
snap list                           # verify no user snaps
sudo systemctl stop snapd
sudo systemctl disable snapd
# or full removal:
sudo apt-get purge -y snapd
```

**Note:** `unattended-upgrades` is also running. This installs security updates
automatically which is useful, but can restart services unexpectedly. Leave it enabled
unless it causes issues.

---

## Option C — Nighttime Scheduled Suspend (Future / Not Implemented)

**Prerequisites before attempting:**
- Add a swapfile (hibernate requires swap >= RAM size, i.e. ≥ 4 GB)
- Write graceful Docker stop/start wrapper scripts
- Test RTC wakeup alarm works correctly on this kernel

**Why it's complex:** All Docker home automation services (zigbee2mqtt, MQTT broker,
MariaDB, InfluxDB, climate sensors, electricity) go offline during suspend. They must be
stopped cleanly before suspend and restarted after wake.

**Outline when ready:**

```bash
# 1. Stop non-critical containers first, then critical ones
docker stop villa73-web villa73-api villa73-scheduler grafana
docker stop zigbee2mqtt mqttclient climateapi electricity cabinbookings sofar
docker stop mariadb influxdb mosquitto

# 2. Set RTC wakeup alarm (e.g. 06:00 next morning)
sudo sh -c "echo 0 > /sys/class/rtc/rtc0/wakealarm"
sudo sh -c "echo $(date -d 'tomorrow 06:00' +%s) > /sys/class/rtc/rtc0/wakealarm"

# 3. Suspend
sudo systemctl suspend

# 4. On wake: start containers (via @reboot cron or systemd)
docker start mosquitto mariadb influxdb
docker start zigbee2mqtt mqttclient climateapi electricity cabinbookings sofar
docker start villa73-api villa73-scheduler villa73-web grafana
```

---

## Option D — Idle-Based Suspend + Wake-on-LAN (Future / Not Implemented)

Requires another always-on device on the LAN to send the WoL magic packet. Not suitable
as the only machine on the local network.

---

## Verification

```bash
# Fan daemon running
sudo systemctl status argonone-fan

# CPU governor active on all cores
cat /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Current CPU frequency (should drop at idle)
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq

# Current temperature
cat /sys/class/thermal/thermal_zone0/temp   # divide by 1000 for °C

# ModemManager gone
systemctl is-active ModemManager   # should print "inactive"
```
