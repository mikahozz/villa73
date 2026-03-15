.PHONY: build test test-scheduler run

# This project has no C dependencies; CGO_ENABLED=0 keeps builds reproducible
# and also avoids a dyld crash on macOS 15+ with Go 1.22.
CGO_FLAGS := CGO_ENABLED=0

build:
	go build ./cmd/...

test:
	$(CGO_FLAGS) go test ./...

test-scheduler:
	$(CGO_FLAGS) go test ./cmd/scheduler/... -count=1 -v

run-api:
	go run ./cmd/api

run-api-mock:
	go run ./cmd/api -mock

run-sync:
	go run ./cmd/sync
