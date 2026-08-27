---
phase: 05-slot-picker-ux-improvements
plan: 02
subsystem: poll-creation-duplicate-slot-rejection
tags: [backend, go, validation, tdd]
dependency-graph:
  requires: []
  provides:
    - "markDuplicateSlots(pollType, views) — server-authoritative exact-duplicate-slot detector reused by handleCreatePoll"
    - "duplicateSlotCopy row-level error constant, rendered via the existing slotView.Error / create.html {{if $slot.Error}} path"
  affects:
    - "internal/web/server.go (handleCreatePoll validation flow)"
tech-stack:
  added: []
  patterns:
    - "TDD RED/GREEN per task: create_dup_test.go asserts behavior via real HTTP POST /polls + DB row-count checks, following the existing TestCreatePoll_* pattern in server_test.go"
    - "NUL-separated (\\x00) composite map keys to prevent cross-field string-concatenation collisions when hashing multi-field duplicate keys"
key-files:
  created:
    - "internal/web/create_dup_test.go"
  modified:
    - "internal/web/server.go"
decisions:
  - "duplicateSlotCopy wording is position-neutral (\"This slot is a duplicate of another slot.\") rather than the CONTEXT.md example (\"...same as another one below.\") because every member of a duplicate group is flagged, not just the second occurrence — \"below\" would read wrong on the last flagged row."
metrics:
  duration: "~10 min"
  completed: 2026-08-27
status: complete
actuals:
  tokens: 2900
  tasks: 2
  commits: 3
---

# Phase 5 Plan 02: Slot Picker UX Improvements (Server-Side Duplicate Rejection) Summary

Server-authoritative exact-duplicate-slot rejection (SLOT-05): submitting two or more slots that are exact duplicates (same date for all_day; same date+start+end for date_time) is now rejected at HTTP 200 with an inline error on every duplicate row, values preserved, no poll saved — implemented as a single pure helper wired into the existing `hasError` validation gate in `handleCreatePoll`.

## What Was Built

- **SLOT-05 (server.go):** Added `duplicateSlotCopy` constant next to the existing `bannerErrorCopy`/`rowErrorCopy` convention, and a new pure function `markDuplicateSlots(pollType string, views []slotView) bool`. It builds a per-row duplicate key — `Date` alone for `all_day`, `Date+"\x00"+StartTime+"\x00"+EndTime` for `date_time` — skipping any row missing a field the match rule depends on (incomplete rows can't be duplicates). Every row whose key collides with another row's gets `Error = duplicateSlotCopy` set, not just the second occurrence, so 3+-way groups and multiple independent duplicate pairs are all flagged in one pass. Wired into `handleCreatePoll` immediately after `parseSlots`, before the existing `len(slots)==0`/`hasError` gate — duplicate rows are still appended to `slots` upstream by `parseSlots`, so setting `hasError = true` is what blocks the save; the function never mutates or dedupes `slots` itself. No template change was needed — `create.html`'s existing `{{if $slot.Error}}` row-error rendering already carries the new message.

## New Test File

`internal/web/create_dup_test.go` (package `web`) — six behavioral HTTP tests, following the existing `TestCreatePoll_*` pattern in `server_test.go` (`newTestServer`, `noRedirectClient`, direct SQLite row-count assertions):

- `TestCreatePoll_AllDayDuplicate_Rejected` — two identical all_day dates → 200, duplicate copy in body, no poll row.
- `TestCreatePoll_DateTimeDuplicate_Rejected` — two identical date_time slots → 200, duplicate copy in body, no poll row.
- `TestCreatePoll_ThreeWayDuplicate_AllFlagged` — three identical all_day dates → duplicate copy appears exactly 3 times (all rows flagged, not just one pair).
- `TestCreatePoll_TwoIndependentDuplicatePairs_AllFlagged` — two distinct date_time duplicate pairs in one submission → duplicate copy appears exactly 4 times (both pairs flagged, proving the scan doesn't stop at the first group).
- `TestCreatePoll_DateTimeSameDateDifferentStart_NotDuplicate` — same date, same end time, different start time → 303 redirect, 2 slots persisted (proves the match rule is date+start+end, not date-only).
- `TestCreatePoll_AllDayDistinctDates_NotDuplicate` — two different dates → 303 redirect, 2 slots persisted (guards against false positives).

## TDD Gate Compliance

Task 1 (tracer, tdd="true"): RED commit `4cdd45b` (both `TestCreatePoll_AllDayDuplicate_Rejected` and `TestCreatePoll_DateTimeDuplicate_Rejected` confirmed failing against the un-wired server — 303 returned, duplicates silently saved), GREEN commit `3f63bca` (both pass after adding `duplicateSlotCopy`/`markDuplicateSlots` and wiring it in). Tracer feedback gate: re-ran the full `internal/web` package suite post-commit (not just the two targeted tests) to confirm no regression in the existing valid multi-slot create paths — passed; proceeded to Task 2 per autonomous-run protocol.

Task 2 (auto, tdd="true"): all four edge-case tests were written and run against Task 1's already-committed implementation (no server.go change was expected or needed per the plan — the algorithm already handles 3+-way groups and multiple independent pairs by construction). All four passed on first run; committed as `1b7e6ae`. This is the same "tests prove existing behavior" variation on strict RED/GREEN documented in 05-01's summary — the write-test/run/confirm-intended-behavior cadence was followed, but RED (failing before implementation) doesn't apply since the implementation predates these specific tests by design.

## Deviations from Plan

None — plan executed exactly as written. The `markDuplicateSlots` signature, key-building rule, wiring point, and constant naming all match the plan's `<action>` text verbatim.

## Verification

- `go build ./...` — succeeds.
- `go vet ./internal/web/` — clean.
- `go test ./internal/web/ -run 'TestCreatePoll_.*Duplicate|TestCreatePoll_.*NotDuplicate|TestCreatePoll_.*Flagged' -count=1` — all 6 new tests pass.
- `go test ./internal/web/ -count=1` — full package suite green, including the pre-existing valid all_day/date_time multi-slot create tests (no over-rejection).
- `go test ./... -count=1` — full repo suite (store, token, web, cmd/server) green, no regressions.

## Known Stubs

None — this plan wires real, working server-side validation end-to-end; no placeholder data or unwired components were introduced.

## Threat Flags

None beyond what the plan's own threat model already covers (T-05-02 mitigated via NUL-separated composite keys; T-05-03 accepted, bounded by the existing `maxFormBytes` cap and a single O(n) scan).

## Self-Check: PASSED

- FOUND: internal/web/create_dup_test.go
- FOUND: internal/web/server.go (`duplicateSlotCopy` constant, `markDuplicateSlots` function, wiring line in `handleCreatePoll`)
- FOUND commit 4cdd45b (test: RED)
- FOUND commit 3f63bca (feat: GREEN)
- FOUND commit 1b7e6ae (test: edge coverage)
