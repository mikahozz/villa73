# Architecture Overview

Home dashboard running on a Raspberry Pi 4 (armv7l, Raspbian Buster). The system is split across two Docker Compose stacks that share a Docker network.

---

## Infrastructure

| Property | Value |
|---|---|
| Hardware | Raspberry Pi 4 |
| Architecture | armv7l (32-bit ARM) |
| OS | Raspbian Buster |
| Orchestration | Docker Compose (two stacks) |
| Local registry | `registry:2` on port 5101 |

All images are built for `linux/arm/v7`. Node.js is pinned to `22.22.0` because Node 24 images do not publish `arm/v7` manifests.

---

## Docker Compose Stacks

### Stack 1 — villa73 (`/villa73/docker-compose.yml`) — newer

Connected to the old stack via the external network `homeapp73-docker_default`.

| Container | Image | Port(s) | Role |
|---|---|---|---|
| villa73-web-1 | villa73-web | 80→3000 | React frontend (Nginx) |
| villa73-api-1 | villa73-api | 6001 | Go HTTP API |
| villa73-scheduler-1 | villa73-scheduler | 6002 (health) | Go light scheduler |

### Stack 2 — homeapp73-docker (`/homeapp73-docker/docker-compose.yml`) — older

| Container | Image | Port(s) | Role |
|---|---|---|---|
| climateapi | climateapi | 5011 | Python Flask — indoor climate API |
| electricity | electricity | 3016 | Go — solar power API |
| cabinbookings | cabinbookings | 3011 | Node.js/Express — cabin booking API |
| cabinbookings-refresh | cabinbookings-refresh | — | Node.js — scrapes booking site into MariaDB; run on-demand (`restart: "no"`), not long-running |
| nordpoolrust | nordpoolrust | 3014 | Rust/Actix-web — Nordpool price API (legacy) |
| influxdb | influxdb:1.8.10 | 8086 | InfluxDB time-series database |
| mariadb | linuxserver/mariadb arm32v7 | 3306 (internal) | MariaDB — cabin bookings store |
| mosquitto | mosquitto | 1883 | MQTT broker |
| mqttclient | mqttclient | — | Python MQTT subscriber → InfluxDB |
| homeapp73-docker-sofar-1 | sofar | — | Python — Sofar solar inverter reader |
| zigbee2mqtt | koenkk/zigbee2mqtt:1.18.1 | host network | Zigbee USB bridge → MQTT |
| grafana | grafana/grafana:8.2.5 | 3010 | Grafana dashboards |
| prometheus | prom/prometheus:v3.5.1 | 9090 | Metrics scraping — defined in compose but **commented out**, not currently running |

### Standalone (not in either compose file)

| Container | Image | Notes |
|---|---|---|
| homeassistant | home-assistant/home-assistant:stable | Running for 4+ years, unmanaged by these compose files |
| registry | registry:2 | Local Docker image registry (port 5101) |

---

## Software Architecture

### Frontend — villa73-web

**Tech:** React 18, TypeScript, Vite, Tailwind CSS, served by Nginx.

Single-page app displayed on an always-on tablet. Key components:

- `WeatherNow` / `Forecast` — current conditions and forecast from FMI
- `Indoor` — indoor temperature/humidity (currently returns hardcoded stub data from the Go API)
- `ElectricityPrice` — spot prices from ENTSO-E via villa73-api
- `FamilyCalendar` — upcoming events via CalDAV
- `CabinBookings` — cabin booking calendar from MariaDB
- `Solar` — real-time solar production from InfluxDB
- `BgWrapper` — changes background image based on season/time of day

The Nginx container joins both the `villa73_default` and `homeapp73-docker_default` networks so the frontend can reach services in the old stack directly when needed.

### villa73-api (Go, port 6001)

Go 1.22 HTTP server (`net/http`). Structured logging via `zerolog`. Supports a `--mock` flag that substitutes all integrations with static data (used in local development).

**Endpoints:**

| Endpoint | Integration | Notes |
|---|---|---|
| `GET /api/weathernow` | FMI station `101004` (observations) | Finnish Meteorological Institute open data |
| `GET /api/weatherfore` | FMI `Tapanila,Helsinki` (forecast) | |
| `GET /api/indoor/dev_upstairs` | — | Returns hardcoded stub data (22.5°C, 27.4% RH). Real Zigbee sensor data flows through the old climateapi, not yet wired into the Go API |
| `GET /api/electricity/prices?start=&end=&timeFormat=` | ENTSO-E Transparency Platform | Returns hourly spot prices; supports UTC, local, or IANA timezone |
| `GET /api/events` | CalDAV (iCloud or compatible) | 7-day family calendar window; handles recurrence rules and exceptions |
| `GET /api/sun?start=&end=` | Bundled JSON dataset | Helsinki sunrise/sunset times from a precomputed file |

**Key packages:**
- `integrations/fmi` — parses FMI XML observation/forecast responses
- `integrations/spot` — calls ENTSO-E API, converts MWh prices to cents/kWh with Finnish VAT
- `integrations/cal` — CalDAV client using `go-webdav`; handles RRULE, EXDATE, RECURRENCE-ID
- `integrations/sun` — reads `sun_helsinki.json` bundled data
- `integrations/shelly` — Shelly Gen2 plug control (used by scheduler, not the API)
- `config` — reads `.env` via `godotenv`

**PostgreSQL (defined, not currently running):** `backend/db/` contains a Dockerfile and SQL migrations for a `measurements` table (sensor readings, JSONB value column) and `sync_entries` table (sync state tracking for HOURLY/DAILY/WEEKLY/MONTHLY jobs). These are scaffolded for a future local caching layer.

### villa73-scheduler (Go, port 6002)

Runs a time-based daily schedule loop (ticks every minute). Controls a **Shelly Gen2 outdoor smart plug** via its HTTP RPC API (`/rpc/Switch.Set`, `/rpc/Switch.GetStatus`).

**Current schedules (Helsinki timezone):**

| Name | Category | Trigger | Action |
|---|---|---|---|
| Night lights ON | night_lights | Today's sunset | Shelly → ON |
| Night lights OFF | night_lights | 23:00 | Shelly → OFF |
| Morning lights ON | night_lights | 06:45 | Shelly → ON |
| Morning lights OFF | night_lights | Today's sunrise | Shelly → OFF |

**Category logic:** Within a category, the *last* schedule whose trigger time has been reached today wins. This implements override logic: if sunrise is before 06:45 the "sunrise OFF" schedule is inserted later and suppresses the "06:45 ON" (lights never come on that day).

The sunrise/sunset trigger functions re-evaluate each tick so they stay accurate as the season changes.

### climateapi (Python Flask, port 5011)

Reads the most recent indoor climate reading from InfluxDB (`homedb` database, `indoorclimate` measurement) and returns it as JSON. Used by the old frontend; not yet called from the new villa73-api.

### Sensor pipeline (Zigbee → InfluxDB)

```
Zigbee sensors (temperature/humidity)
    ↓ (RF 2.4 GHz)
zigbee2mqtt (USB dongle /dev/ttyACM0, host network mode)
    ↓ MQTT topics: zigbee2mqtt/dev_<name>
mosquitto (broker, port 1883)
    ↓ subscribe "#"
mqttclient (Python, paho-mqtt)
    ↓ parses JSON payload, writes line protocol
InfluxDB (homedb, measurement: indoorclimate, tag: location=dev_<name>)
    ↓ query: SELECT LAST(temperature),* FROM indoorclimate WHERE location=?
climateapi → villa73-api (not yet connected) → villa73-web
```

Non-Zigbee sensors (legacy Shelly sensors) also publish temperature/humidity/battery to MQTT; `mqttclient` batches all three values before writing.

### Solar inverter pipeline (Sofar → InfluxDB)

```
Sofar solar inverter (LSW3 Wi-Fi logger)
    ↓ Modbus/TCP or UDP
sofar (Python, SofarSensor.py)
    ↓ writes OutputActivePowerW
InfluxDB (homedb, measurement: electricity)
    ↓ Flux query: last record within 5 days
electricity (Go, port 3016) → villa73-web (Solar component)
```

### Cabin bookings pipeline

```
External booking website
    ↓ HTTP scraping (cabinbookings-refresh, Node.js, run on-demand)
MariaDB (cabok_db.bookings, cabok_db.runs)
    ↓ SQL query: bookings within ±365 days
cabinbookings (Node.js/Express, port 3011) → villa73-web (CabinBookings component)
```

### Electricity spot prices

Two parallel implementations exist:

| Service | Tech | Source | Status |
|---|---|---|---|
| villa73-api `/api/electricity/prices` | Go | ENTSO-E Transparency Platform | Active (new stack) |
| nordpoolrust | Rust/Actix-web (port 3014) | Nordpool Group API | Legacy (old stack) |

The Go integration handles timezone conversion and Finnish VAT; prices are returned in cents/kWh.

---

## Network Topology

```
Internet
  │
  ├─ FMI open data API         → villa73-api
  ├─ ENTSO-E API               → villa73-api
  ├─ CalDAV / iCloud           → villa73-api
  └─ Nordpool API              → nordpoolrust (legacy)

Raspberry Pi LAN (192.168.10.217)
  │
  ├─ :80   villa73-web (Nginx)
  │         ├─ /api/*  → villa73-api :6001
  │         └─ static assets
  │
  ├─ :6001 villa73-api
  ├─ :6002 villa73-scheduler
  ├─ :5011 climateapi
  ├─ :3016 electricity (solar)
  ├─ :3011 cabinbookings
  ├─ :3014 nordpoolrust (legacy)
  ├─ :1883 mosquitto (MQTT)
  ├─ :8086 InfluxDB
  ├─ :3306 MariaDB (internal only)
  ├─ :3010 Grafana
  ├─ :9090 Prometheus (commented out in compose, not currently running)
  └─ :5101 Local Docker registry

Docker networks:
  villa73_default           ← villa73 stack
  homeapp73-docker_default  ← old stack  (villa73-web also joins this)
```

---

## Developer Tooling

### MCP Pi Tool (`tools/mcp-pi/`)

A Go MCP server that runs locally (on the developer's laptop) and tunnels commands to the Raspberry Pi over SSH. Prevents arbitrary remote calls — only allows `docker_compose_logs` and `compose_service_api_request` against the two pre-configured Compose directories.

```sh
make run-pi-mcp          # start the MCP server
./scripts/mcp-pi-tool.sh logs --target dir1 --service scheduler --tail 100
./scripts/mcp-pi-tool.sh api  --target dir1 --service api --container-port 6001 --path /api/weathernow
```

SSH target and identity files are configured in the root `.env`.

---

## Known Gaps / In Progress

- **Indoor temperature** in villa73-api returns hardcoded stub data (22.5°C). The real sensor data path (Zigbee → InfluxDB → climateapi) exists in the old stack but is not yet wired into the new Go API.
- **PostgreSQL** schema is defined in `backend/db/` but the container is not currently running. It is scaffolded for future caching of spot prices and sensor measurements.
- **nordpoolrust** is superseded by the ENTSO-E integration in villa73-api but continues to run.
- **Grafana** is running but its dashboards are not version-controlled in the current repo state. **Prometheus** is defined in `homeapp73-docker/docker-compose.yml` (pinned to `v3.5.1`) but commented out — not currently running, and its scrape targets aren't version-controlled either.
