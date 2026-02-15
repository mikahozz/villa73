# Agent Instructions

## Command policy
- Prefer root `Makefile` targets over ad-hoc shell commands for local dev checks and Docker workflows.
- When a matching target exists, use `make <target>` from repo root.

## Preferred targets
- Stack lifecycle:
  - `make compose-config`
  - `make compose-ps`
  - `make compose-up`
  - `make compose-up-web`
  - `make compose-down`
  - `make compose-logs`
- API/proxy checks:
  - `make check-api-proxy`
  - `make check-api-direct`
  - `make check-all`
- Legacy bridge checks:
  - `make check-legacy-cabin`
  - `make check-legacy-electricity`
  - `make check-legacy-indoor`

## Notes
- Legacy bridge checks may return `502` when legacy services are offline; this is expected and should not block nginx startup.
- Add new recurring verification commands to the root `Makefile` and then use them via `make`.
