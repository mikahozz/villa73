.PHONY: \
	compose-config compose-ps compose-up compose-up-web compose-down compose-logs \
	check-api-proxy check-api-direct check-legacy-cabin check-legacy-electricity check-legacy-indoor \
	check-all

COMPOSE := docker compose
WEB_BASE := http://localhost:3000
API_BASE := http://localhost:6001

# Stack lifecycle
compose-config:
	$(COMPOSE) config

compose-ps:
	$(COMPOSE) ps

compose-up:
	$(COMPOSE) up -d --build

compose-up-web:
	$(COMPOSE) up -d --build web

compose-down:
	$(COMPOSE) down

compose-logs:
	$(COMPOSE) logs --no-color --tail=120 web api

# Proxy checks (web -> api)
check-api-proxy:
	curl -sS -D - "$(WEB_BASE)/api/weathernow" -o /tmp/villa73_web_api.json | sed -n '1,20p'
	head -c 220 /tmp/villa73_web_api.json; echo

check-api-direct:
	curl -sS -D - "$(API_BASE)/api/weathernow" -o /tmp/villa73_api_direct.json | sed -n '1,20p'
	head -c 220 /tmp/villa73_api_direct.json; echo

# Legacy bridge checks (may return 502 when legacy service is offline)
check-legacy-cabin:
	curl -sS -D - "$(WEB_BASE)/api/cabinbookings/days" -o /tmp/villa73_legacy_cabin.txt | sed -n '1,20p'

check-legacy-electricity:
	curl -sS -D - "$(WEB_BASE)/api/electricity/current" -o /tmp/villa73_legacy_electricity.txt | sed -n '1,20p'

check-legacy-indoor:
	curl -sS -D - "$(WEB_BASE)/api/indoor/dev_upstairs" -o /tmp/villa73_legacy_indoor.txt | sed -n '1,20p'

check-all: compose-ps check-api-proxy check-api-direct
