---
phase: 04-admin-poll-management
plan: 02
subsystem: web
tags: [go, html-template, sqlite, server-rendered-forms, tdd]

# Dependency graph
requires:
  - phase: 02-voting
    provides: participants/responses schema with ON DELETE CASCADE on responses.participant_id, exercised live for the first time by this plan
  - phase: 03-results-grid
    provides: buildResultsGridView/resultsGridView and the shared "results" template partial, extended (not duplicated) by this plan's admin variant
  - phase: 04-admin-poll-management
    provides: "Plan 04-01's admin-token-gated route pattern (PollByTokens, confirm_* server-side gate shape) reused for the delete-response route"
provides:
  - "internal/store/admin.go: DeleteParticipant(ctx, pollID, participantID) — scoped cascade delete of one participant's response"
  - "POST /poll/{ptoken}/admin/{atoken}/responses/{participantID}/delete route + handleDeleteResponse"
  - "resultsGridView.IsAdmin/Title/ParticipantToken/AdminToken and resultsParticipant.ID — the admin-variant view-model extension shared, not duplicated, with the participant route"
  - "Admin-only 'Remove' response-deletion control on the results grid, native confirm()-gated, with a client-side double-submit guard"
affects: [04-03-delete-poll]

# Actuals (#2632)
actuals:
  tokens: 7500
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared view-model with an admin toggle: buildResultsGridView takes an isAdmin flag rather than a second admin-only build function, so the participant route's render path is provably unchanged by construction (byte-identical output) instead of by convention"
    - "Store-level authorization scoping: DELETE ... WHERE id = ? AND poll_id = ? makes cross-poll tampering structurally impossible rather than relying on an application-layer check"
    - "Native confirm() + disable-on-confirm as the double-submit backstop for a single-shot destructive POST, matching Plan 04-01's client+server dual-gate shape but without a server-side confirmation flag (no destructive-with-warning branch needed here — deleting one response is a single atomic action, not a batch with mixed severity)"

key-files:
  created:
    - internal/web/static/admin.js
  modified:
    - internal/store/admin.go
    - internal/store/admin_test.go
    - internal/web/server.go
    - internal/web/server_test.go
    - internal/web/templates/results.html
    - internal/web/templates/links.html
    - internal/web/static/style.css

key-decisions:
  - "resultsGridView/resultsParticipant were extended in place (IsAdmin/Title/ParticipantToken/AdminToken, ID) rather than creating a parallel admin-only view-model, so the participant route's call site changes only by passing isAdmin=false — its rendered output is unchanged by construction, not by a second code path that could drift"
  - "The participant's store ID (not the cookie_token) is rendered into the delete-form action URL — it's the participant row's own primary key, safe to expose, and is what DeleteParticipant is scoped by; CookieToken remains fully unexposed (T-04-06)"
  - "No server-side confirm_delete_response flag was added (unlike Plan 04-01's slot-removal gate) — deleting one participant's response is always a single, atomic, all-or-nothing action with no 'how much data' variance to warn about differently; the native confirm() dialog is the full safety net, backed by the store's safe no-op on a missing id"

requirements-completed: [ADM-03]

coverage:
  - id: D1
    description: "DeleteParticipant removes exactly one participant and cascades to delete only that participant's response rows, scoped to the owning poll, proven with exact before/after row counts"
    requirement: "ADM-03"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_DeleteParticipant_CascadesOnlyThatParticipant"
        status: pass
    human_judgment: false
  - id: D2
    description: "A participant id belonging to a different poll can never be deleted through another poll's admin token (cross-poll scoping)"
    requirement: "ADM-03"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_DeleteParticipant_ScopedByPoll"
        status: pass
    human_judgment: false
  - id: D3
    description: "Deleting a non-existent participant id is a safe no-op (no error, no row changes) — backs the double-submit guard"
    requirement: "ADM-03"
    verification:
      - kind: unit
        ref: "internal/store/admin_test.go#TestAdmin_DeleteParticipant_NonExistent_NoError"
        status: pass
    human_judgment: false
  - id: D4
    description: "POST to the delete-response route removes exactly that participant from the admin page's results grid and recomputes tallies from current DB state; other participants are untouched"
    requirement: "ADM-03"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestDeleteResponse_EndToEnd"
        status: pass
    human_judgment: false
  - id: D5
    description: "A mismatched ptoken/atoken pair on the delete route 404s and deletes nothing"
    requirement: "ADM-03"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestDeleteResponse_WrongTokenPair_404"
        status: pass
    human_judgment: false
  - id: D6
    description: "The 'Remove' control renders only on the admin route and only when the poll has at least one response; the participant route's results view is unchanged from Phase 3"
    requirement: "ADM-03"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestResults_AdminRow_OnlyOnAdminAndWhenResponses"
        status: pass
    human_judgment: false
  - id: D7
    description: "admin.js gates the delete form's submission on window.confirm using the exact 04-UI-SPEC.md copy shape, and disables the submit button on confirm"
    requirement: "ADM-03"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestAdminJS_ConfirmCopyAndDoubleSubmitGuard"
        status: pass
    human_judgment: false
  - id: D8
    description: "A double-click or double-confirm on Remove cannot issue two deletes in a real browser (not merely idempotent in theory)"
    verification: []
    human_judgment: true
    rationale: "Backstop item per the plan's must_haves (verification: backstop) and 04-UI-SPEC.md's loading-state consideration — requires a human to actually double-click in a browser and observe the button disable; the store-level no-op-on-missing-id and admin.js's disable-on-confirm are both mechanically proven, but a genuine race (e.g. a slow network) is not reproducible in a Go HTTP test."

duration: ~20min
completed: 2026-08-25
status: complete
---

# Phase 4 Plan 2: Admin Response Deletion (ADM-03) Summary

**Admin-only per-participant "Remove" control on the results grid, backed by a scoped cascade-delete store method and a native confirm()-gated POST route, with the participant route's grid rendering byte-identical to Phase 3.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-25T21:30:13Z
- **Tasks:** 2 (TDD store method + auto route/UI)
- **Files modified:** 8 (1 created, 7 modified)

## Accomplishments

- `DeleteParticipant` removes one participant and, via the existing `ON DELETE CASCADE` on `responses.participant_id`, exactly that participant's response rows — proven with exact before/after row counts across participants, slots, and a cross-poll scoping test, plus a no-op-on-missing-id test that backs the double-submit guard.
- New `POST /poll/{ptoken}/admin/{atoken}/responses/{participantID}/delete` route resolves the poll via both tokens (404 on mismatch) before deleting anything, then redirects back to the admin page so Phase 3's existing "always recompute from current DB state" design handles tallies/best-fit with zero new logic.
- The results grid gained a shared, not duplicated, admin variant: `resultsGridView`/`resultsParticipant` were extended with `IsAdmin`/`Title`/`ParticipantToken`/`AdminToken`/`ID`, and `buildResultsGridView` takes an `isAdmin` flag — the participant route's call site simply passes `false`, so its render path is unchanged by construction rather than by a second, divergence-prone code path.
- `admin.js` (new) intercepts the delete form's submit, shows the exact 04-UI-SPEC.md confirm() copy built from the template's own `data-participant-name`/`data-poll-title` attributes, and disables the submit button on confirm so a double-click cannot fire two submits — verified by asserting the shipped JS source contains both the confirm() call and the disable-on-confirm guard.

## Task Commits

Each task was committed atomically:

1. **Task 1: DeleteParticipant store method** - TDD, two commits:
   - `61b0355` (test — RED: failing tests for DeleteParticipant, compile-fail since the method didn't exist yet)
   - `07c1fcd` (feat — GREEN: scoped `DELETE ... WHERE id = ? AND poll_id = ?` implementation)
2. **Task 2: Delete-response route + admin-only Remove control + confirm() JS + CSS** - `b2d26c8` (feat)

## Files Created/Modified

- `internal/store/admin.go` - `DeleteParticipant(ctx, pollID, participantID) error`
- `internal/store/admin_test.go` - cascade/scoping/no-op invariant tests for `DeleteParticipant`
- `internal/web/server.go` - `handleDeleteResponse`, route registration, `resultsGridView`/`resultsParticipant` extension, `buildResultsGridView` signature change (both call sites updated)
- `internal/web/server_test.go` - end-to-end delete test, mismatched-token 404 test, admin-row-only-when-admin-and-responses test, admin.js content-assertion test; updated the four pre-existing direct `buildResultsGridView` unit tests and `TestResults_AdminRouteParity`'s no-leak assertion for the new signature/behavior
- `internal/web/templates/results.html` - admin-controls `<tr class="results-admin-row">` inside `<thead>`, gated on `IsAdmin && HasResponses`
- `internal/web/templates/links.html` - `<script src="/static/admin.js" defer>` added after the results partial
- `internal/web/static/admin.js` - confirm()-gate + disable-on-confirm double-submit guard for `.results-admin-remove` forms
- `internal/web/static/style.css` - `.btn-remove-response` (compact `.btn-remove` variant), `.results-admin-row`/`.results-admin-remove` layout, zero new hex values

## Decisions Made

- Extended the existing `resultsGridView`/`resultsParticipant` view-model in place (adding `IsAdmin`, `Title`, `ParticipantToken`, `AdminToken`, `ID`) rather than building a separate admin-only view-model — `buildResultsGridView` now takes an `isAdmin bool` and the participant route passes `false`, so its output stays provably unchanged rather than merely "not intentionally changed."
- Rendered the participant's store `ID` (never `CookieToken`) into the delete-form's action URL — it's the primary key `DeleteParticipant` is scoped by, and per T-04-06 it carries no more sensitivity than the display name already shown in the same column.
- Did not add a server-side `confirm_delete_response` flag (unlike Plan 04-01's slot-removal gate): deleting one participant's response is a single atomic action with no varying "how much would be lost" to communicate differently server-side — the native `confirm()` dialog plus the store's safe no-op on a missing/repeated id is the complete safety net.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated `buildResultsGridView`'s pre-existing direct unit tests for the new signature**
- **Found during:** Task 2 build/test pass
- **Issue:** Adding `isAdmin`/`title`/`ptoken`/`atoken` parameters to `buildResultsGridView` (required by the plan's action for Task 2) broke compilation of five pre-existing Phase 3 unit tests (`TestResultsView_CommentsPopulated`, `TestResultsView_CommentsEmpty`, `TestResultsView_Tally`, `TestResultsView_MissingCell`, `TestResultsView_ZeroParticipants_HasResponsesFalse`) that call the function directly with the old 4-argument signature.
- **Fix:** Updated each call site to pass `false, "", "", ""` for the new admin-variant parameters — no behavioral change to what those tests assert, since none of them exercise the admin path.
- **Files modified:** `internal/web/server_test.go`
- **Verification:** `go build ./...` and the full `internal/web` suite pass.
- **Committed in:** `b2d26c8` (Task 2 commit)

**2. [Rule 1 - Bug] Rescoped `TestResults_AdminRouteParity`'s admin-token no-leak assertion**
- **Found during:** Task 2 final verification pass (`go test ./...`)
- **Issue:** The Phase 3 test asserted the admin token never appears anywhere inside the results-grid markup on the admin route. This plan's design (per its own threat model, T-04-01) intentionally embeds the admin token in each "Remove" control's delete-form action URL, which lives inside the grid's `<thead>` — so the old blanket assertion now fails against the *intended* new behavior, not a bug in it.
- **Fix:** Rescoped the assertion to check that the admin token appears nowhere in the grid *before* the `results-admin-row` (i.e., never attached to participant-data cells), while still passing on its legitimate appearance inside the admin-controls row itself. The underlying T-03-02/T-04-06 guarantee — no participant `CookieToken` is ever exposed — remains structurally true and untouched (it's guaranteed by `resultsParticipant` only ever carrying `ID`/`DisplayName`).
- **Files modified:** `internal/web/server_test.go`
- **Verification:** `TestResults_AdminRouteParity` passes; full suite green.
- **Committed in:** `b2d26c8` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both test-only updates required by the plan's own intentional signature/behavior change; no production behavior beyond what the plan specified).
**Impact on plan:** No scope creep — both are the mechanical fallout of Task 2's planned `buildResultsGridView` signature change and the planned admin-token-in-URL design, not new functionality.

## Issues Encountered

None beyond the two deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `DeleteParticipant`'s scoped-delete pattern (`DELETE ... WHERE id = ? AND poll_id = ?`, safe no-op on a missing row) and `handleDeleteResponse`'s `PollByTokens`-then-mutate-then-redirect shape are the direct template for Plan 04-03's whole-poll delete (same admin-token authorization gate, same "confirm() in the browser, no-op-safe in the store" defense-in-depth shape).
- `resultsGridView.IsAdmin`/`Title`/`ParticipantToken`/`AdminToken` are already wired through both routes; Plan 04-03's danger-zone delete-poll control lives on the separate edit page (per 04-CONTEXT.md), so no further results-grid view-model changes are anticipated for it.
- One backstop item (D8, double-click/double-confirm cannot issue two deletes) remains for human browser verification — mechanically backed by `admin.js`'s disable-on-confirm guard and the store's no-op-on-repeat, but not independently re-verified against a real race in a browser for this plan.

---
*Phase: 04-admin-poll-management*
*Completed: 2026-08-25*

## Self-Check: PASSED

All 8 created/modified source files and the SUMMARY.md itself verified present on disk; all 3 commit hashes (`61b0355`, `07c1fcd`, `b2d26c8`) verified present in git log. `go build ./...`, `go vet ./internal/...`, and `go test ./...` all pass as of the final commit. One WINDOWS.md ledger entry (id 11, unrun-verify) recorded for the browser-only double-submit backstop item (D8).
