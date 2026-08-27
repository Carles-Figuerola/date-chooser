---
phase: 04-admin-poll-management
plan: 01
subsystem: web
tags: [go, html-template, sqlite, server-rendered-forms, tdd]

# Dependency graph
requires:
  - phase: 01-poll-creation
    provides: create.html/create.js slot-row structure and the shared server-rendered-form + banner-error validation pattern reused by the edit form
  - phase: 02-voting
    provides: participants/responses schema with ON DELETE CASCADE on responses.slot_id, exercised live for the first time by this plan's slot removal
  - phase: 03-results-grid
    provides: buildResultsGridView/resultsGridView, reused unchanged by the admin route to show the missing-cell "—" after a slot add
provides:
  - "GET/POST /poll/{ptoken}/admin/{atoken}/edit route: admin can edit title, description, organizer name, and the slot list (add/edit-in-place/remove)"
  - "internal/store/admin.go: UpdatePollDetails, UpdatePollSlots (diff-based keep/add/remove), SlotResponseCounts"
  - "Client+server slot-removal safety mechanism: zero-response rows remove immediately, responded rows require an inline warning + checkbox + server-side confirm_slot_removal flag before a destructive save"
affects: [04-02-delete-response, 04-03-delete-poll]

# Actuals (#2632)
actuals:
  tokens: 14121
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Diff-based slot mutation (keep/add/remove) instead of delete-all-reinsert, to keep ON DELETE CASCADE scoped to only the removed slot's responses"
    - "Row-presence-implies-removal: a slot row missing entirely from a form submission (dropped from the DOM client-side) is treated the same as an explicit removal marker, avoiding a separate 'is this row gone' signal"
    - "Client+server dual-gate for irreversible actions: disabled-submit-until-checkbox client-side, confirm_slot_removal flag checked server-side — same defense-in-depth shape as every prior phase's validation"

key-files:
  created:
    - internal/store/admin.go
    - internal/store/admin_test.go
    - internal/web/templates/edit.html
    - internal/web/static/edit.js
  modified:
    - internal/web/server.go
    - internal/web/server_test.go
    - internal/web/templates.go
    - internal/web/static/style.css

key-decisions:
  - "Slot-list editing implemented as a diff (keep/add/removeIDs) rather than full replace, per 04-CONTEXT.md's explicit discretion — required to keep the cascade delete scoped to only the removed slot"
  - "A slot row absent from the submitted form (dropped from DOM for a zero-response row) is treated as an implicit removal, unified with the explicit slot_removed=1 marker used for responded rows that stay visible but marked"
  - "Poll type is rendered but never accepted from the edit form — the handler always uses poll.PollType from the resolved row, never r.FormValue('poll_type')"

patterns-established:
  - "editFormView/editSlotView view-model mirrors createFormView/slotView but adds ID + ResponseCount, which drive both the client warning and the server confirm gate off the same counts"

requirements-completed: [ADM-02]

coverage:
  - id: D1
    description: "Admin can open the edit page and see the poll pre-filled (title, description, slots), with the poll-type toggle visible but disabled"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_TitleDescription_EndToEnd"
        status: pass
    human_judgment: false
  - id: D2
    description: "Editing title/description/organizer persists and is reflected on the participant page; tokens and poll_type never change"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_TitleDescription_EndToEnd"
        status: pass
    human_judgment: false
  - id: D3
    description: "GET/POST to the edit route 404s when ptoken and atoken do not belong to the same poll"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_MismatchedTokens_NotFound"
        status: pass
    human_judgment: false
  - id: D4
    description: "Removing a slot cascades to delete only that slot's response rows; other slots' responses and every participant's comment survive intact"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_UpdatePollSlots_RemoveCascadesOnlyThatSlotsResponses"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_SlotRemovalWithConfirm_Cascades"
        status: pass
    human_judgment: false
  - id: D5
    description: "Adding a slot appends at position = prior max + 1, leaves existing responses untouched, and shows the '—' missing-cell fallback for existing participants in the new column"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_UpdatePollSlots_AddAppendsAtEndKeepsResponses"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_AddSlot_ShowsDashForExistingParticipants"
        status: pass
    human_judgment: false
  - id: D6
    description: "Editing an existing slot's value in place preserves its position and its response rows"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_UpdatePollSlots_EditInPlacePreservesPositionAndResponses"
        status: pass
    human_judgment: false
  - id: D7
    description: "Every slot write is scoped by poll_id, so a slot id from another poll can never be updated or deleted through this poll's edit"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_UpdatePollSlots_ScopedByPoll"
        status: pass
    human_judgment: false
  - id: D8
    description: "A destructive slot removal (>=1 response) is rejected server-side with a 200 re-render and no DB write when confirm_slot_removal is absent; the same request with the flag set cascades and redirects"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_SlotRemovalWithoutConfirm_RejectedNoWrite"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_SlotRemovalWithConfirm_Cascades"
        status: pass
    human_judgment: false
  - id: D9
    description: "Edit-form slot rows carry per-slot response counts driving the inline removal-warning scaffold and the aggregate confirm checkbox, rendered in the served markup"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_SlotRemovalWarningMarkupPresent"
        status: pass
    human_judgment: false
  - id: D10
    description: "Title/description values are HTML-escaped on re-render (no reflected XSS via the edit form)"
    requirement: "ADM-02"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestEdit_ScriptBearingTitle_Escaped"
        status: pass
    human_judgment: false
  - id: D11
    description: "'Save changes' disables and its label changes to a saving state on submit, blocking a genuine second POST (client-side backstop, not merely a visual state change)"
    verification: []
    human_judgment: true
    rationale: "Backstop UI-state item per 04-UI-SPEC.md — requires a human to click twice/observe the button state in a real browser; not mechanically provable from a Go HTTP test."

duration: ~45min
completed: 2026-08-25
status: complete
---

# Phase 4 Plan 1: Admin Poll Editing (ADM-02) Summary

**Admin edit route (`GET/POST /poll/{ptoken}/admin/{atoken}/edit`) with diff-based slot mutation, scoped cascade deletes, and a client+server dual-gated slot-removal confirmation.**

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-08-25T21:19:45Z
- **Tasks:** 3 (tracer + TDD auto + auto)
- **Files modified:** 8 (4 created, 4 modified)

## Accomplishments

- New admin-only edit route lets the organizer change title, description, and organizer name, with changes immediately visible on the participant page; poll type and both tokens are provably immutable.
- Slot-list editing (add/edit-in-place/remove) implemented as a store-level diff (`UpdatePollSlots`) so removing one slot cascades to delete only that slot's responses — proven with exact before/after row counts, not just absence of error.
- A destructive slot removal (one with existing responses) is gated both client-side (mark-not-remove + required checkbox, disabling "Save changes") and server-side (`confirm_slot_removal` flag; rejected with a 200 re-render and zero DB writes otherwise) — matching the plan's one-way-reversibility requirement.
- Adding a slot to a poll with existing participants correctly falls back to Phase 3's "—" missing-cell rendering with no forced re-vote.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end edit of poll title/description** - `c1ad0eb` (feat)
2. **Task 2: Slot-list editing with scoped cascade-delete + confirm gate** - TDD, three commits:
   - `11eb9b4` (test — RED: failing store tests for UpdatePollSlots/SlotResponseCounts)
   - `58d4df1` (feat — GREEN: store-layer implementation)
   - `2fd03c4` (feat — handler-layer slot classification + confirm gate)
3. **Task 3: Edit-form slot UI** - `f0a9678` (feat)

**Plan-level verification (not tied to a single task, added per the plan's overall `<verification>`/threat-model requirements):**
- `f70cde3` (test — auto-escaping proof for a script-bearing title, T-04-05)

## Files Created/Modified

- `internal/store/admin.go` - `UpdatePollDetails`, `UpdatePollSlots` (diff-based), `SlotResponseCounts`, `SlotEdit`
- `internal/store/admin_test.go` - cascade/position/scoping invariant tests for the new store methods
- `internal/web/server.go` - `editFormView`/`editSlotView`, `handleEditForm`, `handleUpdatePoll`, `parseEditSlots`, route registration
- `internal/web/server_test.go` - end-to-end edit tests (title/description, mismatched tokens, slot removal with/without confirm, add-slot dash fallback, warning markup, XSS escaping)
- `internal/web/templates.go` - registers `edit.html` as its own parse set
- `internal/web/templates/edit.html` - edit form: pre-filled fields, disabled poll-type toggle, editable slot rows with hidden `slot_id`/`slot_removed` + `data-response-count`, aggregate removal-warning banner + confirm checkbox, row templates for JS-added slots
- `internal/web/static/edit.js` - submit double-post guard, add/remove-row wiring branched on response count, mark/unmark + Undo, aggregate confirm-gate logic
- `internal/web/static/style.css` - `.slot-row-marked`, `.toggle-disabled`, warning-region spacing (zero new hex values)

## Decisions Made

- Implemented slot editing as a diff (keep/add/removeIDs) rather than delete-all-reinsert, per 04-CONTEXT.md's explicit discretion — a full replace would have cascade-deleted every response on every edit, which is prohibited.
- Unified "row dropped from the DOM" (zero-response quick-remove) and "row present but explicitly marked" (responded row) into a single removeIDs computation: any existing slot ID not found among the submitted, non-removed rows is removed. This keeps the handler's classification logic single-pass and matches the "no silent data loss, no silent extra loss" requirement.
- Poll type is always taken from the resolved poll row (`poll.PollType`), never from the submitted form, even though the disabled radios are still rendered with the poll's own value for display.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated Task 1's end-to-end test to resend the existing slot's hidden fields**
- **Found during:** Task 2 (slot-list editing)
- **Issue:** Task 2 introduced "a slot row missing from the submission is treated as removed." Task 1's `TestEdit_TitleDescription_EndToEnd` posted only title/description/organizer_name (no slot fields), which — under Task 2's new logic — would have caused the poll's only slot to be implicitly (and wrongly) deleted by a plain title edit.
- **Fix:** Updated the test to also submit `slot_id`/`slot_date` matching the poll's existing slot, exactly as the real form's hidden inputs always do. No production behavior changed; the test was made representative of a real form submission.
- **Files modified:** `internal/web/server_test.go`
- **Verification:** `TestEdit_TitleDescription_EndToEnd` passes; full suite green.
- **Committed in:** `2fd03c4` (Task 2 commit)

**2. [Rule 2 - Missing Critical] Added the auto-escaping (XSS) proof test required by the plan's own threat model**
- **Found during:** final plan-level verification pass
- **Issue:** The plan's `<verification>` section and threat T-04-05 explicitly require "a test posts a script-bearing title and asserts the escaped form in the output," but none of the three tasks' acceptance criteria individually called this test out — it would have been silently missing from the delivered test suite.
- **Fix:** Added `TestEdit_ScriptBearingTitle_Escaped`, posting `<script>alert(1)</script>` as the title and asserting the re-rendered page contains only the HTML-escaped form. No production code change was needed — `html/template`'s default auto-escaping already satisfies the mitigation; this is a proof-of-existing-safety test.
- **Files modified:** `internal/web/server_test.go`
- **Verification:** Test passes; full suite green.
- **Committed in:** `f70cde3`

---

**Total deviations:** 2 auto-fixed (1 bug fix to an already-committed test, 1 missing verification coverage added)
**Impact on plan:** Both are test-only changes; no production behavior was altered beyond what the plan already specified. No scope creep.

## Issues Encountered

None beyond the two deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/store/admin.go` and the edit-route pattern (`editFormView`/`editSlotView`, `PollByTokens` gate, `confirm_*` flag pattern) are the direct foundation for Plan 04-02 (delete a single participant's response) and Plan 04-03 (delete the entire poll) — both reuse the same admin-token authorization gate and the same client+server confirm-gate shape.
- `SlotResponseCounts` is reusable as-is if a future admin action needs to know per-slot response counts.
- One backstop item (D11, "Save changes" saving-state on submit) remains for human browser verification — mechanically satisfied by `edit.js`'s submit guard (mirrors the already-verified Phase 1 create.js pattern) but not independently re-verified in a real browser for this plan.

---
*Phase: 04-admin-poll-management*
*Completed: 2026-08-25*

## Self-Check: PASSED

All 8 created/modified source files and the SUMMARY.md itself verified present on disk; all 6 commit hashes (`c1ad0eb`, `11eb9b4`, `58d4df1`, `2fd03c4`, `f0a9678`, `f70cde3`) verified present in git log. `go build ./...`, `go vet ./internal/...`, and `go test ./internal/...` all pass as of the final commit.
