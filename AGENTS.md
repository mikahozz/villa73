# Agent Instructions

## Command policy

- Prefer root `Makefile` targets over ad-hoc shell commands for local dev checks and Docker workflows.
- When a matching target exists, use `make <target>` from repo root.

## Committing

- Never commit confidential information like user info, passwords or environment details like IP address or similar things. This is a public repo.

- When creating automatic commit messages, summarize the key change and its purpose in one line. Then write a more thorough change description on a separate paragraph below.

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
- Platform parity checks:
  - `make web-build-arm64`
- Legacy bridge checks:
  - `make check-legacy-cabin`
  - `make check-legacy-electricity`
  - `make check-legacy-indoor`

## Notes

- Legacy bridge checks may return `502` when legacy services are offline; this is expected and should not block nginx startup.
- Add new recurring verification commands to the root `Makefile` and then use them via `make`.
- `CGO_ENABLED=0` is baked into all `backend/Makefile` test targets because this project has no C dependencies. It also avoids a dyld crash on macOS 15+ with Go 1.22. Never set it manually in ad-hoc commands; use the Make targets instead.

## Backend tests

Always use these targets from the repo root. Never run `go test` directly.

- `make test-backend` — full backend test suite
- `make test-scheduler` — scheduler package only (verbose)
