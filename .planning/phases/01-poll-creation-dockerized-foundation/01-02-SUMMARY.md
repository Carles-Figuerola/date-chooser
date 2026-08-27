---
phase: 01-poll-creation-dockerized-foundation
plan: 02
subsystem: web
tags: [go, net-http, html-template, progressive-enhancement, server-side-validation]

# Dependency graph
requires: ["01-01"]
provides:
  - "Full server-rendered poll-creation form (create.html) per 01-UI-SPEC.md: title, optional description, optional organizer name, per-poll all-day/date+time toggle, repeatable slot rows, live counter, ~50-slot hint"
  - "create.js progressive-enhancement layer: add/remove rows, singular/plural counter, toggle-driven mode switching, disable-last-row-Remove, ~50-slot hint reveal, submit double-post guard"
  - "Hardened POST /polls: title/poll_type/slot server-side validation, 200 re-render (not redirect) on failure with exact UI-SPEC banner/row copy and preserved values, no DB write on failure"
  - "Form field contract: title, description, organizer_name, poll_type (all_day|date_time); slot_date (all_day) or index-aligned slot_start/slot_end (date_time)"
affects: ["01-03", phase-2-voting, phase-3-results-grid, phase-4-admin-management]

# Actuals (#2632)
actuals:
  tokens: 7867
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "createFormView/slotView structs shared by both the fresh GET render and the validation-failure re-render, so submitted values and per-field/per-row errors round-trip through the same template with html/template auto-escaping"
    - "Server-side slot parsing keyed off poll_type: parseAllDaySlots collects ordered non-empty slot_date values; parseDateTimeSlots walks index-aligned slot_start/slot_end pairs, treating an incomplete or non-increasing pair as the same row-level error"
    - "<template> elements (slot-row-template-all_day / slot-row-template-date_time) hold the JS-cloned row markup so the one visible, real (non-<template>) row keeps the form submittable with JS disabled"
    - "Submit-guard uses a JS closure flag checked before disabling the button, so a second submit event is prevented (not merely re-styled)"

key-files:
  created:
    - internal/web/static/create.js
  modified:
    - internal/web/templates/create.html
    - internal/web/static/style.css
    - internal/web/server.go
    - internal/web/server_test.go

key-decisions:
  - "internal/store/poll.go required no changes: Plan 01's CreatePoll already persists description, organizer_name, poll_type, and the full ordered multi-slot list (including nullable ends_at) in one transaction with parameterized SQL, so Task 2's persistence requirements were already satisfied — only the HTTP-layer validation and slot parsing needed to be built."
  - "poll_type is treated as server-authoritative: an empty value defaults to all_day, but any non-empty value outside {all_day, date_time} is rejected as a validation error (banner shown, no DB write) rather than silently coerced, satisfying threat T-01-08 alongside the existing DB CHECK constraint."
  - "A date_time row with only one of start/end filled (the UI-SPEC 'partial' case) is treated as the same validation failure as end-not-after-start, reusing the single declared row-error copy 'End time must be after start time.' rather than inventing an undeclared 'incomplete row' message."
  - "Oversized-body handling distinguishes the MaxBytesReader trip (413, matched via the ParseForm error text containing 'too large') from other ParseForm failures (400 with the banner copy), so the DoS cap and generic malformed-form handling don't share one ambiguous code path."
  - "Toggling poll_type client-side rebuilds all slot rows from the matching <template> (clearing typed values) rather than attempting in-place field conversion, since the toggle is a per-poll, pre-submit decision with no spec requirement to preserve values across a mode switch."

patterns-established:
  - "Validation-failure re-render is HTTP 200 (never a redirect), reuses the exact same createFormView the GET handler renders, and echoes submitted values through html/template's auto-escaping — the pattern later phases (voting form, admin edit) can follow for their own re-render-on-error flows."

requirements-completed: [POLL-01, POLL-02, POLL-03]

coverage:
  - id: D1
    description: "Organizer submits title + optional description/organizer name + multiple all_day slots; all slots persisted in submitted order, poll_type stored"
    requirement: POLL-01, POLL-02, POLL-03
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestCreatePoll_ValidAllDay_MultipleSlotsPersistedInOrder"
        status: pass
    human_judgment: false
  - id: D2
    description: "Organizer submits date_time poll with two index-aligned start/end slot pairs; both starts_at and ends_at persisted in order"
    requirement: POLL-02, POLL-03
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestCreatePoll_ValidDateTime_MultipleSlotsPersisted"
        status: pass
    human_judgment: false
  - id: D3
    description: "Zero-slot submit is rejected server-side with a 200 re-render carrying the exact banner copy and no DB write"
    requirement: POLL-02
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestCreatePoll_ZeroSlots_RejectedWithoutRedirect"
        status: pass
    human_judgment: false
  - id: D4
    description: "Empty title submit is rejected server-side with a 200 re-render and no DB write"
    requirement: POLL-01
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestCreatePoll_EmptyTitle_RejectedWithoutRedirect"
        status: pass
    human_judgment: false
  - id: D5
    description: "date_time row with end not strictly after start is rejected with the exact row-level copy, no DB write"
    requirement: POLL-03
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestCreatePoll_DateTimeEndNotAfterStart_RejectedWithoutRedirect"
        status: pass
    human_judgment: false
  - id: D6
    description: "A request body exceeding the ~1 MiB MaxBytesReader cap is handled without panic (413) and writes nothing to the DB"
    requirement: "OPS (T-01-04 threat mitigation)"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestCreatePoll_OversizedBody_RejectedWithoutPanic"
        status: pass
    human_judgment: false
  - id: D7
    description: "Full create-poll form renders per UI-SPEC (fields, toggle, add/remove rows, counter, hint, submit-guard) and degrades to a submittable form with JS disabled"
    requirement: POLL-01, POLL-02, POLL-03
    verification:
      - kind: manual_procedural
        ref: "go build ./... plus grep checks for maxlength=200/2000, 'Add another slot', 'Create poll', name=\"poll_type\", and 'Creating' in create.js — all pass; manual review confirmed the one server-rendered slot row exists outside any <template> element"
        status: pass
    human_judgment: false
  - id: D8
    description: "Re-rendered user-submitted values are auto-escaped (XSS mitigation, T-01-07)"
    requirement: "Security (T-01-07 threat mitigation)"
    verification:
      - kind: manual_procedural
        ref: "Ad hoc scratch test posted title '<script>alert(1)</script>' and confirmed the re-rendered value attribute contains the html/template-escaped '&lt;script&gt;alert(1)&lt;/script&gt;'; scratch test file removed, not committed"
        status: pass
    human_judgment: false
  - id: D9
    description: "Submit-guard prevents a second POST (not just a visual button change) — UI-SPEC backstop item"
    requirement: "UI-SPEC loading/backstop"
    verification:
      - kind: unit
        ref: "None — browser-executed JS behavior, not reachable from Go httptest; code review of create.js's submitted flag + preventDefault() confirms the logic blocks a second submit event before it fires"
        status: insufficient_spec
    human_judgment: true

duration: 6min
completed: 2026-08-24
status: complete
---

# Phase 1 Plan 2: Full Poll-Creation Form Summary

**Full create-poll form (title/description/organizer/toggle/multi-slot rows) per the UI-SPEC, backed by a hardened POST /polls handler that validates title, poll_type, and per-row date+time ordering server-side and re-renders (200, not a redirect) with preserved values and exact UI-SPEC error copy on any failure.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-24T20:00:42Z
- **Completed:** 2026-08-24T20:06:00Z
- **Tasks:** 2
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments
- `create.html` expanded from the Plan 01 skeleton's single date field into the full UI-SPEC form: title/description/organizer fields, a segmented all-day/date+time toggle, repeatable slot rows (one always pre-rendered, satisfying the "empty state unreachable" requirement), a singular/plural counter, a non-blocking ~50-slot hint, and validation banner/inline-error placeholders — all while remaining fully submittable with JavaScript disabled (the real row lives outside any `<template>`).
- `create.js` adds add/remove-row behavior, toggle-driven row-mode rebuilding, disables/hides "Remove" on the last remaining row so the client can't reach zero slots, reveals the 50+ hint, and blocks a second form submission via a closure flag checked before `preventDefault()` — not just a button re-style.
- `handleCreatePoll` now validates title (required, trimmed), `poll_type` (must be `all_day`/`date_time`, server-authoritative), and slots by mode — all_day collects ordered `slot_date` values, date_time walks index-aligned `slot_start`/`slot_end` pairs and rejects incomplete or non-increasing ranges. Any failure re-renders `create.html` at HTTP 200 (never a redirect) with the exact UI-SPEC banner ("Check the highlighted fields and try again.") and row copy ("End time must be after start time."), submitted values preserved, and writes nothing to the DB.
- `internal/store/poll.go` needed no code changes — Plan 01's `CreatePoll` already persisted `description`, `organizer_name`, `poll_type`, and the full ordered slot list (with nullable `ends_at`) inside one parameterized transaction.
- Extended `server_test.go` with 6 new test functions covering the full `<behavior>` contract (valid all_day multi-slot, valid date_time multi-slot, zero-slot rejection, empty-title rejection, end-not-after-start rejection, oversized-body handling) plus updated the pre-existing end-to-end test to the new `slot_date` field name.

## Task Commits

Task 2 followed the TDD RED/GREEN cycle since it carries `tdd="true"`:

1. **Task 1: full poll-creation form UI** - `67f1aac` (feat)
2. **Task 2 (RED): add failing validation/multi-slot tests** - `5d2fbec` (test)
3. **Task 2 (GREEN): harden handleCreatePoll with server-side validation** - `bcc43ac` (feat)

## TDD Gate Compliance

Task 2's RED gate (`5d2fbec`) and GREEN gate (`bcc43ac`) are both present in git log, in order. No REFACTOR commit was needed — the GREEN implementation required no follow-up cleanup.

## Files Created/Modified
- `internal/web/static/create.js` - **new**: add/remove slot rows, singular/plural counter, toggle-driven mode switching, disable-last-row-Remove, ~50-slot hint, submit double-post guard
- `internal/web/templates/create.html` - full form per UI-SPEC: fields, toggle, slot rows + `<template>`-backed row prototypes, counter, hint, banner/inline-error placeholders
- `internal/web/static/style.css` - spacing tokens (`--space-xs..2xl`), toggle/segmented-control, slot-row, banner-error, field-error, hint, btn-secondary/btn-remove styles, all interactive controls at 44px min height
- `internal/web/server.go` - `createFormView`/`slotView` types, `parseSlots`/`parseAllDaySlots`/`parseDateTimeSlots`/`isValidDateTimeRange` helpers, hardened `handleCreatePoll`, `renderCreateForm` shared by GET and re-render paths
- `internal/web/server_test.go` - 6 new test functions plus a field-name fix to the existing end-to-end test

## Decisions Made
- `internal/store/poll.go` required no changes at all — Plan 01's `CreatePoll` already satisfied every persistence requirement in this plan's `<behavior>` block (ordered multi-slot insert, `poll_type`, nullable `ends_at`), so all of Task 2's work landed in the HTTP layer.
- `poll_type` validation is server-authoritative: empty defaults to `all_day`, but any other invalid value triggers a validation error rather than silent coercion (threat T-01-08), backstopped by the existing DB `CHECK` constraint.
- A date_time row with only one of start/end filled reuses the single UI-SPEC row-error copy ("End time must be after start time.") rather than inventing an undeclared "incomplete row" message, since the UI-SPEC declares no separate copy for that case.
- Oversized-body handling distinguishes a `MaxBytesReader` trip (413, detected via the ParseForm error text containing "too large") from other `ParseForm` failures (400 with the banner copy), keeping the DoS cap and general malformed-form handling on separate, unambiguous paths.
- Client-side toggle switching rebuilds all slot rows from the matching `<template>` (clearing any typed values) rather than converting fields in place — a per-poll, pre-submit toggle has no spec requirement to preserve values across a mode change, and this keeps the JS simple.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - missing critical functionality] `handleCreateForm` needed a default view struct wired up**
- **Found during:** Task 2, GREEN implementation
- **Issue:** Task 1's rebuilt `create.html` expects a `createFormView` struct (`.Title`, `.PollType`, `.Slots`, etc.), but `handleCreateForm` (the plain `GET /` handler) still passed `nil` to `ExecuteTemplate` as inherited from the Plan 01 skeleton. This wasn't caught by Task 1's own verify command (which only runs `go build` + greps, never executes the template), and no existing test exercises `GET /` directly, so it would have first surfaced as a runtime template-execution error on the first real page load.
- **Fix:** Added `newCreateFormView()` (returns `PollType: "all_day"`, one blank `slotView`) and a shared `renderCreateForm` helper used by both `handleCreateForm` and the validation-failure re-render path in `handleCreatePoll`.
- **Files modified:** internal/web/server.go
- **Verification:** Manual scratch test (`go run` against a temp DB) confirmed `GET /` renders the full form with one visible slot row and no template-execution error; removed the scratch file before committing.
- **Committed in:** bcc43ac (Task 2 GREEN)

**2. [Rule 1 - bug] Existing end-to-end test used the superseded `slot` field name**
- **Found during:** Task 2, RED phase
- **Issue:** `TestEndToEnd_CreatePollAndFollowLinks` (from Plan 01) posted a single field named `slot`, which is the pre-Plan-02 skeleton contract. Plan 02's `<artifacts_this_phase_produces>` explicitly supersedes this with `slot_date` (all_day) / `slot_start`+`slot_end` (date_time), so the old test would fail against the new contract regardless of any other change.
- **Fix:** Updated the test to `form.Set("slot_date", "2026-09-01")`.
- **Files modified:** internal/web/server_test.go
- **Verification:** Test passes under the new contract; included in the RED commit (fails against old server.go) and confirmed passing in the GREEN commit.
- **Committed in:** 5d2fbec (RED), verified passing after bcc43ac (GREEN)

---

**Total deviations:** 2 auto-fixed (Rule 2 missing-functionality, Rule 1 bug-fix on a pre-existing test); no architectural changes, no Rule 4 escalations.
**Impact on plan:** No scope or architectural change. Every field, validation rule, and copy string in `<artifacts_this_phase_produces>` and the UI-SPEC Copywriting Contract was implemented as specified.

## Issues Encountered
None beyond the two auto-fixed deviations above.

## Threat Flags

None beyond the plan's own `<threat_model>` — no new network endpoints, auth paths, or schema changes were introduced; T-01-03 (parameterized SQL), T-01-04 (MaxBytesReader), T-01-07 (auto-escaping), and T-01-08 (poll_type allow-list) are all verified per the coverage table above.

## Known Stubs

None. Every field and control declared in `<artifacts_this_phase_produces>` is wired to real server-side parsing/validation/persistence; no hardcoded empty values or placeholder copy remain in the shipped paths.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness
- The create-poll form is feature-complete for Phase 1's scope (POLL-01/02/03); Plan 03 can now style the links-display page using the same spacing/typography/color tokens established here.
- One item deferred to human/UI verification: the submit-double-post guard (UI-SPEC backstop item, D9 above) is implemented in `create.js` but not reachable from Go's `httptest` — no browser JS execution in this test harness. Flagged `human_judgment: true` in coverage; a future UI-review pass or manual browser check should confirm it in practice.
- No blockers.

---
*Phase: 01-poll-creation-dockerized-foundation*
*Completed: 2026-08-24*

## Self-Check: PASSED

All 5 created/modified source files verified present on disk; all 3 task commits (`67f1aac`, `5d2fbec`, `bcc43ac`) verified present in git log. One `unrun-verify` entry recorded in `.planning/WINDOWS.md` for the browser-only submit-guard behavior (D9).
