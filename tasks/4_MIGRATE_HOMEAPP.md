# homeapp73-docker → villa73 Migration

Overall tracking document for migrating all services from `homeapp73-docker` into this monorepo. Each service has a dedicated task file linked below. This file tracks status, dependencies, and milestones.

## Goals

- Consolidate all services into the villa73 monorepo
- Replace Python, Node.js, Rust, and .NET services with Go
- Replace InfluxDB and MariaDB with PostgreSQL
- Keep infrastructure that has no good replacement: Mosquitto, Zigbee2MQTT
- Decommission `homeapp73-docker` entirely when migration is complete

## Phases

| Phase | Focus | Status |
|-------|-------|--------|
| 1 | Database consolidation (InfluxDB + MariaDB → PostgreSQL) | Not started |
| 2 | Core API services (electricity, climateapi) | Not started |
| 3 | IoT pipeline (mqttclient, sofar) | Not started |
| 4 | Application services (cabinbookings, cabinbookings-refresh) | Not started |
| 5 | Infrastructure review (Grafana, Prometheus, Airflow) | Not started |
| 6 | Decommission homeapp73-docker | Not started |

## Services

### Databases / Infrastructure

| Service | Source | Target | Task | Status |
|---------|--------|--------|------|--------|
| InfluxDB 1.8 | time-series storage | PostgreSQL | [tasks/migration/INFLUXDB.md](tasks/migration/INFLUXDB.md) | Not started |
| MariaDB 10.11 | relational storage | PostgreSQL | [tasks/migration/MARIADB.md](tasks/migration/MARIADB.md) | Not started |
| Mosquitto | MQTT broker | Keep as-is | — | N/A |
| Zigbee2MQTT | Zigbee bridge | Keep as-is | — | N/A |
| Grafana | Dashboards | TBD — keep or drop | [tasks/migration/GRAFANA.md](tasks/migration/GRAFANA.md) | Not started |
| Prometheus | Metrics | TBD — keep or drop | [tasks/migration/PROMETHEUS.md](tasks/migration/PROMETHEUS.md) | Not started |

### Backend Services

| Service | Source lang | Target | Task | Status |
|---------|-------------|--------|------|--------|
| electricity | Go | Go, backend/integrations/electricity | [tasks/migration/ELECTRICITY.md](tasks/migration/ELECTRICITY.md) | Not started |
| climateapi | Python (Flask) | Go, backend/integrations/climate | [tasks/migration/CLIMATEAPI.md](tasks/migration/CLIMATEAPI.md) | Not started |
| mqttclient | Python | Go, writes to PostgreSQL | [tasks/migration/MQTTCLIENT.md](tasks/migration/MQTTCLIENT.md) | Not started |
| sofar_lsw3 | Python (ModBus) | Go, reads inverter via ModBus | [tasks/migration/SOFAR.md](tasks/migration/SOFAR.md) | Not started |
| cabinbookings | Node.js (Express) | Go, backed by PostgreSQL | [tasks/migration/CABINBOOKINGS.md](tasks/migration/CABINBOOKINGS.md) | Not started |
| cabinbookings-refresh | Node.js (Chromium) | Go or keep as scraper sidecar | [tasks/migration/CABINBOOKINGS_REFRESH.md](tasks/migration/CABINBOOKINGS_REFRESH.md) | Not started |
| nordpoolrust | Rust (disabled) | Already replaced by integrations/spot | — | Done |
| airflow | Python (disabled) | Replaced by cmd/scheduler | — | Done |
| weatherapi | .NET / C# | Assess if still needed | [tasks/migration/WEATHERAPI.md](tasks/migration/WEATHERAPI.md) | Not started |

### Frontend

| Service | Source | Target | Task | Status |
|---------|--------|--------|------|--------|
| homeclient | React CRA + Nginx | frontend/ (Vite + Nginx) | — | Done |

## Dependencies

```
InfluxDB migration ──► electricity, climateapi, mqttclient, sofar
MariaDB migration  ──► cabinbookings
mqttclient         ──► sofar (sofar publishes to MQTT, mqttclient consumes it)
```

Phase 1 must complete before Phases 2–4 can fully close out — services cannot be decommissioned from homeapp73-docker until their data store has moved.

## Decommission Checklist

Before shutting down `homeapp73-docker`:

- [ ] All PostgreSQL schemas seeded and historical data migrated
- [ ] All active services running in villa73 and verified
- [ ] Grafana/Prometheus decision made (keep in old stack, migrate, or drop)
- [ ] villa73 `old_stack` external network reference removed from docker-compose.yml
- [ ] DNS / reverse proxy updated
- [ ] homeapp73-docker archived or deleted
