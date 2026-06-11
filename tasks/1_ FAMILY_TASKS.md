# Family Tasks Feature

## Status Legend

- `TODO`: Not started
- `IN_PROGRESS`: Actively being implemented
- `BLOCKED`: Waiting for dependency/decision
- `DONE`: Completed

## Goal

Implement and maintain a family task workflow where iCloud Reminders tasks are shown in the dashboard as horizontally swipeable cards above the family calendar, can be completed with optional completer selection, can be deleted with in-app confirmation, and are refreshed/polled based on task availability.

## Scope

- In scope:
  - Backend integration and API for family tasks
  - Frontend task card UX above calendar
  - Parsing size labels (`1-5`) and assignee names from task notes
  - Completion flow to close tasks in iCloud
  - Delete flow to remove tasks in iCloud
  - Polling/refresh behavior
  - Mock data path for local frontend development
  - Unit and component tests for new code
- Out of scope:
  - Adding tasks via Reminders app (already handled externally)

## Functional Requirements (Normalized)

- Source of truth is iCloud task list.
- Task size is derived from numeric labels `1..5`.
- Assignee detection is based on note text containing whole-word, case-insensitive names:
  - `Elias`, `Elise`, `Ella`, `Iskä`, `Äiti`
- Whole-word matching examples:
  - Match: `ella, jotain lisätietoa`
  - Match: `tehtävä: iskä`
  - No match: `Istuimella`
- Frontend card behavior:
  - Render above family calendar.
  - Show cards in oldest-first order in a horizontal carousel-style track.
  - Show only the active card in view by default (no pre-peek).
  - Reveal next card only when swipe/scroll starts.
  - Snap quickly while swiping, and allow at most one-card movement per gesture.
  - Show colored task icon.
  - Show delete `X` action in top-right corner.
  - Ask confirmation before delete using an in-app dialog.
  - Show circle paging indicators above the cards when more than one task is available.
  - Render task `size` as `💪` repeated `1..5` next to title.
- Inline action behavior:
  - Show assignee/completer inline control only when task has assignee.
  - If task has no assignee and user taps `Complete task`, open in-app dialog with optional completer select.
  - `Complete task` action marks task complete in iCloud.
  - Completed tasks stay visible in the card list and render as a compact one-row summary:
    - task title
    - `Completed by <name>`
  - Delete action removes task in backend and removes it from app list.
- Empty state:
  - If no tasks, render nothing above calendar.
  - If no tasks, poll every 15 minutes for new tasks.

## Architecture Decisions

1. iCloud integration placement:
   - Extend `backend/integrations/cal` to include reminder/task capability.
   - Reuse the existing iCloud token/session model from calendar integration.
2. API surface:
   - Add backend endpoints for:
     - List family tasks (sorted oldest-first)
     - Complete task with selected completer
     - Delete task by `taskId`
3. Frontend isolation:
   - Add a dedicated `FamilyTasksCard` feature component mounted above `FamilyCalendar`.
   - Keep card interaction state local to the feature module.
4. Dev ergonomics:
   - Add mock backend responses and frontend mock mode path for quick local iteration without live iCloud.

## Implementation Tasks and Status

| ID | Area | Task | Status | Owner | Notes |
|---|---|---|---|---|---|
| FT-001 | Backend | Map current `backend/integrations/cal` token/session usage and extension points for reminders | DONE | Codex | Implemented via shared caldav path in `integrations/cal/tasks.go` |
| FT-002 | Backend | Implement reminders/task client in `integrations/cal` using shared token | DONE | Codex | Added `GetFamilyTasks` and `CompleteFamilyTask` |
| FT-003 | Backend | Define task domain model (`id`, `title`, `note`, `size`, `assignee`, `createdAt`, `labels`) | DONE | Codex | Added in `backend/familytasks/tasks.go` |
| FT-004 | Backend | Implement assignee parser with whole-word, case-insensitive matching incl. umlauts | DONE | Codex | Regex parser with tests |
| FT-005 | Backend | Add endpoint: list family tasks oldest-first | DONE | Codex | Added `/api/family/tasks` |
| FT-006 | Backend | Add endpoint: complete task (taskId + completer) and close in iCloud | DONE | Codex | Added `/api/family/tasks/complete` |
| FT-006B | Backend | Add endpoint: delete task (taskId) and remove from iCloud | DONE | Codex | Added `/api/family/tasks/delete` |
| FT-007 | Backend | Add mock task provider (toggle/config) for local frontend development | DONE | Codex | Added in-memory mock provider and Vite mock responses |
| FT-008 | Frontend | Add data hooks/service for fetch + polling strategy (1h refresh w/tasks, 15m when empty) | DONE | Codex | Added TanStack Query hook with interval strategy |
| FT-009 | Frontend | Implement `FamilyTasksCard` container above family calendar | DONE | Codex | Component integrated in `Home.tsx` |
| FT-010 | Frontend | Implement carousel-style swipe between tasks | DONE | Codex | Added snap carousel, directional parity fixes, and one-card-per-gesture lock |
| FT-011 | Frontend | Style task card with icon, assignee control, and delete action | DONE | Codex | Added icon/card styling, gradient border, and title-adjacent size emoji |
| FT-012 | Frontend | Implement inline completer dropdown and complete action | DONE | Codex | Inline completer shown only for tasks with assignee |
| FT-013 | Frontend | Render completed cards in compact one-row format | DONE | Codex | Completed task cards show title + `Completed by` text |
| FT-013B | Frontend | Add delete confirmation and backend delete integration | DONE | Codex | In-app confirm dialog + delete mutation to backend |
| FT-013C | Frontend | Add top circle paging indicators for task carousel | DONE | Codex | Added clickable carousel indicators |
| FT-013D | Frontend | Support no-assignee completion with optional completer in dialog | DONE | Codex | Added completion dialog flow with optional completer |
| FT-013E | Frontend | Harden complete/delete cache updates against backend/mock id mismatch | DONE | Codex | Mutations now update by clicked `taskId` from variables |
| FT-014 | Frontend | Integrate feature into `FamilyCalendar` layout | DONE | Codex | Rendered above family calendar panel |
| FT-015 | Tests | Backend unit tests: assignee parser edge cases and label-size parsing | DONE | Codex | Added tests in `backend/familytasks/tasks_test.go` |
| FT-016 | Tests | Backend handler tests: list + complete/delete endpoints and failure paths | DONE | Codex | Added tests in `backend/familytasksapi/handlers_test.go` |
| FT-017 | Tests | Frontend component tests: card render, complete flow, delete flow | DONE | Codex | Updated `FamilyTasksCard.test.tsx` |
| FT-018 | Tests | Frontend polling tests: 1h vs 15m behavior | DONE | Codex | Added hook interval tests |
| FT-019 | QA | End-to-end check with mock mode and live integration mode | IN_PROGRESS | Codex | Automated tests passed; live iCloud runtime validation pending |
| FT-020 | DevEx | Add/extend root `Makefile` test/check target(s) for recurring verification | DONE | Codex | Added `make check-family-tasks` |

## Testing Strategy

- Backend unit tests:
  - Assignee matching boundaries and case-insensitivity.
  - Finnish character handling (`Iskä`, `Äiti`).
  - Label parsing and invalid label fallback.
- Backend integration/handler tests:
  - Task listing sorted by oldest.
  - Complete task success/failure and iCloud upstream failures.
  - Delete task success/validation failure.
- Frontend component tests:
  - Visibility when tasks exist vs no tasks.
  - Card content (icon, size emoji, assignee/no-assignee variants).
  - Inline completer dropdown behavior when assignee exists.
  - Completion dialog behavior when assignee is missing.
  - Post-completion compact `Completed by` state.
  - Delete confirmation dialog + mutation trigger.
  - Carousel indicator rendering for multi-card task lists.
- Frontend timer/polling tests:
  - 1 hour refresh when task list non-empty.
  - 15 minute polling when empty.
  - Timer cleanup on unmount.

## Risks and Mitigations

- iCloud reminders API differences vs calendar API:
  - Mitigation: isolate adapter interface and keep calendar token reuse only at auth/session layer.
- Unicode word boundary handling for Finnish names:
  - Mitigation: explicit regex/unit tests with representative strings.
- Carousel swipe UX regressions on small screens:
  - Mitigation: component tests + manual mobile viewport verification with snap + peek behavior.
- Polling/timer leaks:
  - Mitigation: centralize timer logic in one hook and test unmount cleanup.

## Open Decisions

- No blocking open decisions currently.

## Status Updates Log

Use this section during implementation to track progress with timestamps.

| Date (YYYY-MM-DD) | Update | Related IDs |
|---|---|---|
| 2026-02-28 | Plan created and normalized from feature request. | FT-001..FT-020 |
| 2026-02-28 | Implemented backend + frontend family tasks feature, tests, and Makefile check target. | FT-001..FT-018, FT-020 |
| 2026-02-28 | Live iCloud integration QA still pending in running environment. | FT-019 |
| 2026-03-01 | Updated implementation: native horizontal card scrolling, inline completion compact state, and backend-backed task delete with confirmation. | FT-006B, FT-010..FT-013B, FT-016..FT-017 |
| 2026-03-01 | Refined task browsing to carousel-style swipe with top paging indicators, no default peek, and active-slide height adaptation for compact completed cards. | FT-010, FT-013, FT-013C |
| 2026-03-01 | Added one-card-per-gesture swipe lock, task size emoji rendering, no-assignee completion dialog, and in-app delete dialog parity. | FT-010, FT-011, FT-012, FT-013B, FT-013D |
| 2026-03-01 | Hardened completion/delete updates to use clicked `taskId` and fixed dev-mode delete response path. | FT-013E, FT-017 |
