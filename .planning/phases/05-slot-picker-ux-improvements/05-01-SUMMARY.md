---
phase: 05-slot-picker-ux-improvements
plan: 01
subsystem: poll-creation-form-client-ux
tags: [frontend, css, js, progressive-enhancement, tdd]
dependency-graph:
  requires: []
  provides:
    - "Consistent 44px min-height across time-field controls (input, ▾ toggle, ±15 steppers)"
    - "Click-anywhere-on-time-input opens the hour dropdown (shared openDropdown() function)"
    - "wireRow auto-fills an empty end time to start+1h on slot_start_time change"
    - "Per-row Copy button (data-copy-slot) that appends a pre-filled duplicate row"
  affects:
    - "internal/web/static/style.css"
    - "internal/web/static/create.js"
    - "internal/web/templates/create.html"
tech-stack:
  added: []
  patterns:
    - "TDD RED/GREEN per task: create_ux_test.go asserts shipped CSS/JS/HTML source, following the existing TestAdminJS_ConfirmCopyAndDoubleSubmitGuard os.ReadFile + strings.Contains pattern"
    - "Shared open-dropdown inner function reused by both the ▾ toggle and the new time-input click listener, to keep the two entry points in sync"
key-files:
  created:
    - "internal/web/create_ux_test.go"
  modified:
    - "internal/web/static/style.css"
    - "internal/web/static/create.js"
    - "internal/web/templates/create.html"
decisions:
  - "Named the new per-row Copy button's CSS class .btn-copy-slot rather than .btn-copy (as the plan literally specified), because .btn-copy already exists for the unrelated links.html \"Copy link\" button and is wired via copy.js's document-wide .btn-copy click listener — reusing the name would have overridden that button's styling site-wide and risked copy.js misfiring if ever loaded on this page."
metrics:
  duration: "~15 min"
  completed: 2026-08-27
status: complete
actuals:
  tokens: 3190
  tasks: 3
  commits: 4
---

# Phase 5 Plan 01: Slot Picker UX Improvements (Client-Side) Summary

Consistent-height time-field controls, click-anywhere-to-open hour dropdown, auto-filled end time (+1h, no-clobber), and a per-row Copy button for the poll-creation form — implemented client-side (CSS + progressive-enhancement JS) with a new Go test file mechanically asserting each behavior against the shipped source.

## What Was Built

- **SLOT-01 (style.css):** `.time-field input[type="time"]` gained `min-height: 44px; box-sizing: border-box;`, matching the sibling `.btn-time-step` / `.btn-time-dropdown-toggle` buttons, which already had `min-height: 44px`. That mismatch was the root cause of the height inconsistency.
- **SLOT-02 (create.js):** Extracted the ▾ toggle's open logic in `wireTimeField` into a shared `openDropdown()` inner function. Added a new `click` listener on the time `<input>` itself that calls `openDropdown()` when the dropdown is currently closed, with `evt.stopPropagation()` so the document-level `closeAllTimeDropdowns` click listener doesn't instantly re-close it. The toggle button now calls the same `openDropdown()` function, so both entry points stay in sync. The native browser time picker still works — no `preventDefault` was added.
- **SLOT-03 (create.js):** In `wireRow`, added a `change` listener on the row's `slot_start_time` input (looked up alongside `slot_end_time` by `name`; both are `null` on `all_day` rows, so the feature is naturally skipped there). The handler returns immediately if `slot_end_time` is non-empty (the empty-only no-clobber guard), otherwise computes `start + 60` minutes with the same `((x % 1440) + 1440) % 1440` wrap idiom already used by the ±15 stepper, and writes the result via the existing `formatTimeValue` helper.
- **SLOT-04 (create.html + create.js + style.css):** Added a `data-copy-slot` "Copy" button immediately before the "Remove slot" button in the server-rendered slot loop and in both `<template id="slot-row-template-*">` blocks. In `wireRow`, wired the button to clone `rowTemplateFor(currentMode())`, copy `slot_date`/`slot_start_time`/`slot_end_time` values from the source row into the new (blank) template row, call `wireRow(newRow)` so the clone's own Copy/Remove/time-field/auto-fill behavior works, `appendChild` it to the end of `slotsList` (never inserted after the original), and call `afterRowChange()` so the counter/remove-visibility/hint all update. The original row is never modified. Added a new `.btn-copy-slot` CSS rule (see Deviations) modeled on `.btn-remove`'s neutral outline styling, non-destructive (no red).

## New Test File

`internal/web/create_ux_test.go` (package `web`) — six tests, all following the `os.ReadFile` + `strings.Contains`/targeted-window assertion pattern already established by `TestAdminJS_ConfirmCopyAndDoubleSubmitGuard` in `server_test.go`:

- `TestSlotUX_TimeFieldHeight` — asserts the `.time-field input[type="time"]` CSS rule body contains `min-height: 44px`.
- `TestSlotUX_ClickAnywhereOpensDropdown` — asserts create.js binds a `click` listener on the time input and that handler calls `stopPropagation`.
- `TestSlotUX_AutoFillEndTime` — asserts a `startInput.addEventListener("change", ...)` listener exists, that the `endInput.value` guard appears before the assignment, and that the `+ 60` / `1440` wrap math is present.
- `TestSlotUX_CopyButton_Markup` — asserts `data-copy-slot` appears at least 3 times in create.html and the `>Copy<` label is present.
- `TestSlotUX_CopyButton_JS` — asserts create.js's copy handler references `rowTemplateFor`, `appendChild`, and `afterRowChange`.
- `TestSlotUX_CopyButton_RenderedForm` — asserts a live GET `/` response body contains `data-copy-slot`.

## TDD Gate Compliance

Task 1 (tracer, tdd="true"): RED commit `46ac777` (failing `TestSlotUX_TimeFieldHeight` + `TestSlotUX_ClickAnywhereOpensDropdown`, confirmed failing against pre-edit source), GREEN commit `34d0a30` (both pass). Tracer feedback gate: re-ran both named tests end-to-end post-commit — passed; proceeded to expansion tasks per autonomous-run protocol.

Tasks 2 and 3 (auto, tdd="true"): all six tests were authored together in the initial RED commit (single file, single Write call) since the full test surface for the plan was known upfront; each task's implementation was verified RED (targeted test run confirmed failing before the edit) then GREEN (targeted test run confirmed passing after) before committing, matching the RED→GREEN cadence per task even though the RED test text itself landed in the Task 1 commit. This is a variation on the letter of the RED/GREEN commit split (tests for tasks 2–3 weren't in their *own* separate test-only commit) but the substance — write test first, confirm it fails, then implement, confirm it passes — was followed for every task.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Renamed the new Copy button's CSS class from `.btn-copy` to `.btn-copy-slot`**
- **Found during:** Task 3, before implementing the CSS rule.
- **Issue:** The plan's action text says to add "a `.btn-copy` rule modeled on `.btn-remove`". `internal/web/static/style.css` already has a `.btn-copy` rule (solid accent background) used by `internal/web/templates/links.html`'s "Copy link" buttons, and `internal/web/static/copy.js` attaches a clipboard-copy click handler to every `.btn-copy` element on `DOMContentLoaded`. Reusing the class name would have (a) silently overridden the existing links.html button's visual style site-wide via cascade order, and (b) created a latent footgun if `copy.js` were ever loaded on the create-poll page, since it would try to wire a clipboard handler onto a button with no `data-copy-target` attribute.
- **Fix:** Used `.btn-copy-slot` for the new per-row Copy button's CSS class instead. The `data-copy-slot` attribute (which the plan's JS/test acceptance criteria are actually keyed on) is unaffected — all `create.js` wiring and the `TestSlotUX_CopyButton_*` tests use the `data-copy-slot` attribute selector, not the CSS class, so this rename has zero effect on any specified behavior or test.
- **Files modified:** `internal/web/static/style.css`, `internal/web/templates/create.html`.
- **Commit:** `940a1a3`

### Test-authoring bug caught and fixed during Task 1's own RED/GREEN cycle

While implementing SLOT-02, the initial `TestSlotUX_ClickAnywhereOpensDropdown` regex (`function[^}]*\{([\s\S]*?)\}`) matched the nested `if (list.hidden) { ... }` block's braces instead of the outer click-handler function's braces, due to greedy backtracking preferring the rightmost `{`. This produced a false failure against *correct* implementation code. Fixed by replacing the regex with a simple index-window assertion (find `input.addEventListener("click"`, check the next 300 chars for `stopPropagation`), matching the style already used in `TestSlotUX_AutoFillEndTime`. Not logged as a plan deviation since it's a self-contained test-file fix with no behavior-code impact; verified both before (RED, correctly for the right reason) and after (GREEN) the source implementation.

## Verification

- `go build ./...` — succeeds.
- `go test ./internal/web/ -run 'TestSlotUX' -count=1` — all 6 tests pass.
- `go test ./... -count=1` — full repo test suite (store, token, web packages) passes, no regressions.

**Deferred (non-blocking, per plan):** human visual smoke test of the create-poll form (control heights, click-anywhere dropdown, auto-fill, Copy button) — recommended post-merge, not gated here per the plan's own verification note (unattended run, low-risk additive change).

## Known Stubs

None — this plan wires real, working functionality end-to-end; no placeholder data or unwired components were introduced.

## Threat Flags

None beyond what the plan's own threat model already covers (T-05-01, accepted: no new attack surface, server remains authoritative).

## Self-Check: PASSED

- FOUND: internal/web/create_ux_test.go
- FOUND: internal/web/static/style.css (`.time-field input[type="time"]` min-height:44px; `.btn-copy-slot` rule present)
- FOUND: internal/web/static/create.js (`openDropdown`, `slot_start_time` change listener, `data-copy-slot` handler present)
- FOUND: internal/web/templates/create.html (`data-copy-slot` x3, `>Copy<` label present)
- FOUND commit 46ac777 (test: RED)
- FOUND commit 34d0a30 (feat: Task 1 GREEN)
- FOUND commit 7dadcdb (feat: Task 2 GREEN)
- FOUND commit 940a1a3 (feat: Task 3 GREEN)
