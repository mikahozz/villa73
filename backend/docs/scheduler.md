# Scheduler

A time-based daily action scheduler that runs as a Docker container. It evaluates registered schedules once per minute and executes their actions when the trigger time is reached.

## Architecture

```
backend/cmd/scheduler/
├── main.go            – registered schedules (what runs and when)
├── daily_scheduler.go – scheduler engine (evaluation loop, category logic)
└── shell_action.go    – helper for running shell/docker commands
```

## How it works

### Evaluation loop

The scheduler ticks once per minute. On each tick it calls `evaluate(now)` over every registered schedule. A schedule fires when all of the following are true:

1. Its trigger time falls on today's calendar date
2. `now` has reached or passed that trigger time
3. Its filters pass (if any are set)
4. It has not already triggered today
5. Its action is not still running from a previous tick

### Categories

Schedules that share a `Category` string participate in a winner-takes-all selection within that category:

- **All** time-eligible candidates in the category are collected first (even those that already triggered today).
- The **last candidate in insertion order** wins.
- Only the winner is executed (and only if it has not already triggered today and is not running).

This makes it easy to express override logic. Example: "morning lights ON at 06:45" followed by "morning lights OFF at sunrise" share a category. When sunrise is before 06:45, OFF is the last triggerable candidate and wins — lights never turn on. When sunrise is after 06:45, ON wins instead.

Schedules without a category are independent: they run as soon as their trigger time is reached and do not interact with each other.

### Retry on failure

If an action returns an error, `LastTriggered` is not stamped, so the scheduler retries the action on the next minute tick. This continues until the action succeeds or midnight resets the day.

`runShellCommand` always returns `nil` regardless of the command's exit code (see [Shell and Docker commands](#shell-and-docker-commands)), so shell-triggered actions are never retried.

## Registered schedules

All schedules use the `Europe/Helsinki` timezone.

| Name | Trigger | Category | Action |
|------|---------|----------|--------|
| Night lights ON at sunset | Sunset (dynamic) | `night_lights` | Shelly ON |
| Night lights OFF at 23:00 | 23:00 | `night_lights` | Shelly OFF |
| Morning lights ON at 6:45 | 06:45 | `night_lights` | Shelly ON |
| Morning lights OFF at sunrise | Sunrise (dynamic) | `night_lights` | Shelly OFF |
| Cabin bookings refresh at 06:30 | 06:30 | — | `docker compose run --rm cabinbookings-refresh` in `/homeapp73-docker` |

### Night lights category logic

The four `night_lights` schedules express two daily patterns:

- **Evening:** OFF at 23:00 wins over ON at sunset once 23:00 is past.
- **Morning:** either ON at 06:45 or OFF at sunrise wins, depending on which is later in insertion order and whose trigger time has been reached.

## Shell and Docker commands

`runShellCommand(ctx, dir, name, args...)` runs an arbitrary executable in the given directory and logs stdout and stderr. It always returns `nil` — if the command exits non-zero the error is logged as a warning but the scheduler marks the action as done and does not retry it. This is intentional: commands like `docker compose run` can partially succeed (scrape data, write a file) and still exit non-zero (e.g. when a downstream database is unreachable).

Shell features like pipes and redirects are not available directly; pass `sh` as the command and use `-c` to get a full shell:

```go
// Simple command with arguments
runShellCommand(ctx, "/some/dir", "docker", "compose", "run", "--rm", "my-service")

// Shell pipeline
runShellCommand(ctx, "/some/dir", "sh", "-c", "cat file.txt | grep pattern > result.txt")
```

## Deployment

The scheduler runs as the `villa73-scheduler` Docker container defined in the root `docker-compose.yml`.

### Docker socket access

The container needs access to the host Docker daemon to run `docker compose` commands against other stacks. The socket and an adjacent stack directory are mounted at build time:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
  - ../homeapp73-docker:/homeapp73-docker
```

The `docker-cli` and `docker-cli-compose` packages are installed in the runner image (`alpine:3.20`) so `docker compose` is available inside the container.

### Local development / testing

Run the scheduler binary directly from `backend/`:

```bash
CGO_ENABLED=0 go run ./cmd/scheduler
```

To test a new shell-command schedule without waiting until its scheduled time, temporarily set the trigger to a few minutes from now:

```go
testTrigger := time.Now().In(zone).Add(5 * time.Minute)
Trigger: Trigger{
    Time: func() time.Time {
        now := time.Now().In(zone)
        return time.Date(now.Year(), now.Month(), now.Day(),
            testTrigger.Hour(), testTrigger.Minute(), 0, 0, zone)
    },
},
```

Change the trigger back to the production time once the test run confirms the command works.

## Adding a new schedule

1. Open `backend/cmd/scheduler/main.go`.
2. Call `scheduler.AddSchedule` with a `*DailySchedule` before `scheduler.Start()`.

Minimal example — fixed time, no category:

```go
scheduler.AddSchedule(&DailySchedule{
    Name: "My daily task at 08:00",
    Trigger: Trigger{
        Time: func() time.Time {
            now := time.Now().In(zone)
            return time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, zone)
        },
    },
    Action: func(ctx context.Context) error {
        return runShellCommand(ctx, "/some/dir", "docker", "compose", "run", "--rm", "my-service")
    },
})
```

For a schedule that should be superseded by a later one in the same category, set `Category` to a shared string and rely on insertion order to determine the winner.

## Tests

```bash
make test-scheduler   # from repo root
```

Unit tests live in `daily_scheduler_test.go` (scheduler evaluation logic) and `shell_action_test.go` (command execution). Integration tests using a fake clock are in `daily_scheduler_integ_test.go`.
