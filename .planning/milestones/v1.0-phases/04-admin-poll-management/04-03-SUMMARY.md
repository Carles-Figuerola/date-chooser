---
phase: 04-admin-poll-management
plan: 03
subsystem: web
tags: [go, html-template, sqlite, server-rendered-forms, tdd]

# Dependency graph
requires:
  - phase: 04-admin-poll-management
    provides: "Plan 04-01/04-02's admin-token-gated route pattern (PollByTokens, redirect-to-recompute shape) reused for the delete-poll route"
  - phase: 01-poll-creation
    provides: schema.sql's ON DELETE CASCADE on slots.poll_id and participants.poll_id — exercised for the first time from the polls table itself
  - phase: 02-voting
    provides: notfound.html — reused unchanged as the post-delete 404 for both links
provides:
  - "internal/store/admin.go: DeletePoll(ctx, pollID) — single-statement full-cascade delete of a poll and everything in it"
  - "POST /poll/{ptoken}/admin/{atoken}/delete route + handleDeletePoll"
  - "Danger-zone section on the edit page: .btn-danger (the single new component this phase introduces) + native confirm() gate naming the poll title and response count"
affects: []

# Actuals (#2632)
actuals:
  tokens: 6200
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single-statement cascade delete: DELETE FROM polls WHERE id = ? relies entirely on schema.sql's existing ON DELETE CASCADE chain rather than manually deleting child rows — the same 'let the schema own the cascade' pattern Plans 04-01/04-02 already established at the slot/participant level, now exercised from the top of the hierarchy"
    - "Danger-zone response count computed once (via ResponsesByPollID) and threaded through editFormView on both the GET render and the validation-failure re-render, so the confirm() copy always names live data regardless of which code path rendered the page"
    - "Post-delete redirect to a URL that then 404s (no tombstone) — the delete handler does not need to know or care that the target is now invalid; PollByTokens' existing not-found path handles it for free"

key-files:
  created: []
  modified:
    - internal/store/admin.go
    - internal/store/admin_test.go
    - internal/web/server.go
    - internal/web/server_test.go
    - internal/web/templates/edit.html
    - internal/web/static/edit.js
    - internal/web/static/style.css

key-decisions:
  - "ResponseCount on editFormView is the poll's participant count (len(participants) from ResponsesByPollID), matching the app's existing 'one response = one participant's full submission' convention used everywhere else (e.g. results-grid tallies, admin.js's per-participant Remove control) — not a raw responses-table row count, which would double-count one participant's answers across multiple slots"
  - "The delete-poll confirm() gate lives in edit.js, not admin.js, mirroring 04-CONTEXT.md's page-boundary decision that whole-poll deletion belongs on the separate edit page, not the results/admin page where per-response deletion lives"
  - "handleDeletePoll redirects to /poll/{ptoken} (the now-defunct participant link) rather than any other URL — this is deliberate: PollByTokens' existing ErrNotFound path renders the generic 404 for free, so no new 'poll deleted' state/message/route is needed anywhere in the app"

requirements-completed: [ADM-04]

coverage:
  - id: D1
    description: "DeletePoll removes a poll and its entire cascade (slots, participants, responses) with exact zero-count assertions"
    requirement: "ADM-04"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_DeletePoll_CascadesEverything"
        status: pass
    human_judgment: false
  - id: D2
    description: "A co-resident poll in the same database is provably untouched by deleting a different poll"
    requirement: "ADM-04"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_DeletePoll_OtherPollUntouched"
        status: pass
    human_judgment: false
  - id: D3
    description: "Deleting a non-existent poll id is a safe no-op (no error, no row changes)"
    requirement: "ADM-04"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_DeletePoll_NonExistent_NoError"
        status: pass
    human_judgment: false
  - id: D4
    description: "POST to the delete-poll route deletes the poll and redirects to the participant link, which (along with the admin link) now 404s"
    requirement: "ADM-04"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestDeletePoll_EndToEnd"
        status: pass
    human_judgment: false
  - id: D5
    description: "A mismatched ptoken/atoken pair on the delete-poll route 404s and leaves the poll intact; the correct admin link still resolves afterward"
    requirement: "ADM-04"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestDeletePoll_WrongTokenPair_404"
        status: pass
    human_judgment: false
  - id: D6
    description: "The edit page renders the 'Danger zone' section with the exact body copy, a btn-danger 'Delete poll' button posting to the delete route, and the poll title + response count in data-* attributes"
    requirement: "ADM-04"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestDeletePoll_DangerZoneRendered"
        status: pass
    human_judgment: false
  - id: D7
    description: "edit.js gates the delete-poll form's submission on window.confirm using the exact copy shape (title + '{N} response(s)') and disables the submit button on confirm"
    requirement: "ADM-04"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestEditJS_DeletePollConfirmAndDoubleSubmitGuard"
        status: pass
    human_judgment: false
  - id: D8
    description: "A double-click or double-confirm on Delete poll cannot issue two deletes in a real browser (not merely idempotent in theory)"
    verification: []
    human_judgment: true
    rationale: "Backstop item per the plan's must_haves (verification: backstop), mirroring Plan 04-02's identical D8 for the Remove control. The store-level no-op-on-missing-id (DeletePoll) and edit.js's disable-on-confirm guard are both mechanically proven, but a genuine race (e.g. a slow network between confirm() and the disable) is not reproducible in a Go HTTP test. Recorded as WINDOWS.md ledger entry 12."

duration: ~20min
completed: 2026-08-25
status: complete
---

# Phase 4 Plan 3: Whole-Poll Deletion (ADM-04) Summary

**Danger-zone "Delete poll" control on the edit page — a single-statement full-cascade `DeletePoll` store method, a token-gated delete route, and a native confirm() naming the poll and its response count — completing ADM-04 and all 23 v1 requirements.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-25T21:39:19Z
- **Tasks:** 2 (TDD store method + auto route/UI)
- **Files modified:** 7 (0 created, 7 modified)

## Accomplishments

- `DeletePoll` removes a poll with a single `DELETE FROM polls WHERE id = ?`, relying entirely on schema.sql's existing `ON DELETE CASCADE` chain (slots.poll_id, participants.poll_id, responses transitively) — proven with exact zero-count assertions across all four tables, a co-resident poll left provably untouched, and a safe no-op on a stale/missing id.
- New `POST /poll/{ptoken}/admin/{atoken}/delete` route resolves the poll via both tokens (404 on mismatch, never trusting a client-supplied id) before deleting, then redirects to the now-invalid participant link — which, along with the admin link, serves the existing branded 404 with zero new "deleted" state or message.
- The edit page gained a "Danger zone" card — always the last section, separated by `--space-xl`, tinted with the exact `#FEF2F2`/destructive-border pairing `.banner-error` already uses — containing the single new component this phase introduces: `.btn-danger`, a solid destructive-color button (an inversion of the already-declared `#DC2626`, not a new hex value).
- `edit.js` gates the delete-poll form's submit on a native `confirm()` built from the template's own `data-poll-title`/`data-response-count` attributes (never invented client-side), matching 04-UI-SPEC.md's exact copy, and disables the submit button on confirm so a double-click cannot fire two delete submits.

## Task Commits

Each task was committed atomically:

1. **Task 1: DeletePoll store method** - TDD, two commits:
   - `de5e95b` (test — RED: failing tests for `DeletePoll`, compile-fail since the method didn't exist yet)
   - `41733ae` (feat — GREEN: single-statement `DELETE FROM polls WHERE id = ?` implementation)
2. **Task 2: Delete-poll route + danger-zone section + confirm() JS + btn-danger CSS** - `0ec0323` (feat)

## Files Created/Modified

- `internal/store/admin.go` - `DeletePoll(ctx, pollID int64) error`
- `internal/store/admin_test.go` - cascade/scoping/no-op invariant tests for `DeletePoll`
- `internal/web/server.go` - `handleDeletePoll`, route registration, `editFormView.ResponseCount`, `newEditFormView` signature change (both call sites — GET render and validation re-render — updated)
- `internal/web/server_test.go` - end-to-end delete test, mismatched-token 404 test, danger-zone markup test, edit.js content-assertion test
- `internal/web/templates/edit.html` - `.card.danger-zone` section (heading, body copy, delete form with `data-poll-title`/`data-response-count`, `.btn-danger` button)
- `internal/web/static/edit.js` - delete-poll-form confirm()-gate + disable-on-confirm double-submit guard
- `internal/web/static/style.css` - `.danger-zone` (reuses `.banner-error`'s tint/border pair), `.btn-danger` (new solid destructive component), zero new hex values

## Decisions Made

- `ResponseCount` on `editFormView` is the poll's participant count (`len(participants)` from `ResponsesByPollID`), matching the app's existing "one response = one participant's full submission" convention used everywhere else — not a raw `responses` table row count, which would over-count a single participant's answers across multiple slots.
- The delete-poll confirm() gate lives in `edit.js`, not `admin.js` — per 04-CONTEXT.md's page-boundary decision that whole-poll deletion belongs on the edit page, separate from per-response deletion on the results/admin page.
- `handleDeletePoll` redirects to `/poll/{ptoken}` (the participant link, now defunct) rather than any dedicated "deleted" URL — `PollByTokens`' existing not-found path renders the generic 404 for free, so no new state, route, or message was added anywhere in the app, matching 04-CONTEXT.md's explicit "no tombstone" decision.

## Deviations from Plan

None — plan executed exactly as written. Both tasks matched their `<action>` and `<acceptance_criteria>` without requiring a Rule 1-4 deviation.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None.

## Threat Flags

None — every trust boundary this plan touches (`POST /delete`, `handleDeletePoll`) was already anticipated and mitigated in the plan's own threat model (T-04-01, T-04-02, T-04-04); no new, unanticipated surface was introduced.

## Next Phase Readiness

This is the final plan of the final phase of v1 — no further phases are planned. All 23 v1 requirements (as tracked in REQUIREMENTS.md) are now shipped:
- ADM-02 (Plan 04-01), ADM-03 (Plan 04-02), and ADM-04 (this plan) complete Phase 4 and the full v1 requirement set.

One backstop item (D8, double-click/double-confirm on "Delete poll" cannot issue two deletes) remains for human browser verification — mechanically backed by `edit.js`'s disable-on-confirm guard and the store's no-op-on-repeat, but not independently re-verified against a real race in a browser for this plan. Recorded as WINDOWS.md ledger entry 12, alongside the identically-shaped entry 11 from Plan 04-02's "Remove" control and the phase's other unrun-verify browser-only backstops (ledger entries 1-3, 7-10).

---
*Phase: 04-admin-poll-management*
*Completed: 2026-08-25*

## Self-Check: PASSED

All 7 modified source files and this SUMMARY.md verified present on disk; all 3 commit hashes (`de5e95b`, `41733ae`, `0ec0323`) verified present in git log. `go build ./...`, `go vet ./internal/...`, and `go test ./...` all pass as of the final commit. WINDOWS.md ledger entry 12 (unrun-verify) recorded for the browser-only double-submit backstop item (D8).
