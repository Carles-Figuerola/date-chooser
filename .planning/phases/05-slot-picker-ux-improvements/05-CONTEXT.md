# Phase 5: Slot Picker UX Improvements - Context

**Gathered:** 2026-08-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Improvements to the poll-creation form's slot-input UX only (`create.html`/`create.js`/`style.css`/`server.go`'s slot parsing). No changes to the voting, results, or admin-edit forms unless a shared component/CSS class naturally carries the fix (e.g. `.time-field`/`.btn-time-step` styling is shared, but the fix should not be scoped to touch edit.html's behavior beyond incidental CSS reuse).

</domain>

<decisions>
## Implementation Decisions

### Time-picker sizing (SLOT-01)
- The `<input type="time">`, the `▾` dropdown-toggle button, and the `−15`/`+15` stepper buttons inside `.time-field` must render at the same visual height. Root cause to check first: the native time input's default height vs. the buttons' explicit `min-height: 44px` — likely just needs `min-height: 44px` (or equivalent) applied to the input, or `box-sizing`/line-height inconsistency. Fix in `internal/web/static/style.css`.

### Click-anywhere popup (SLOT-02)
- Clicking anywhere on the `<input type="time">` itself (not just the separate `▾` button) should open the same hour-dropdown popup that the `▾` button already opens. Implementation: add a `click` listener on the time `<input>` in `create.js`'s `wireTimeField` that triggers the same open logic as the toggle button (careful not to double-fire if the click also opens the native browser time picker — acceptable overlap, both can be visible). Do not remove the native picker's own icon-click behavior.

### Auto-fill end time (SLOT-03)
- On a `change` event on a slot's start-time input (specific-time mode only), if the end-time input for that same row is currently empty OR still holds its previous auto-filled value, set it to start + 1 hour (wrapping past midnight is out of scope — CONTEXT note: if start is late enough that +1h crosses midnight, just let the raw HH:MM wrap per `formatTimeValue`'s existing modulo logic already used for the ±15 steppers; do not block or warn). If the organizer has already manually edited the end time away from the auto-filled value, do not overwrite it on a later start-time change (avoid clobbering an intentional edit) — simplest correct approach: only auto-fill when the end-time input is currently empty at the moment start changes.

### Copy slot button (SLOT-04)
- Each slot row (both `all_day` and `date_time` modes) gets a "Copy" button next to "Remove slot". Clicking it clones that row's current field values (date, and for date_time mode also start/end time) into a brand-new row appended to the end of the slot list — not inserted right after the original. Reuse the existing row-template + `wireRow` machinery from `create.js`; the new row needs the copied values written into its inputs after cloning (row templates start blank). Copying does not remove or alter the original row. The new row participates normally in the existing remove-visibility/counter/hint logic.

### Duplicate-slot rejection (SLOT-05)
- Exact-match only, per user decision: two `all_day` slots are duplicates if they have the same date; two `date_time` slots are duplicates if they have the same date AND start time AND end time. Comparing across modes doesn't apply (a poll is one mode at a time, per Phase 1's already-locked poll_type-is-one-mode-per-poll decision).
- Validation happens server-side (client-side is optional/discretionary, not required) in `handleCreatePoll`'s existing validation block — same pattern as existing `hasError`/`bannerErrorCopy` flow. On finding a duplicate, mark BOTH duplicate rows' `slotView.Error` with a clear message (e.g. "This slot is the same as another one below.") and set the banner error, re-rendering at 200 with all field values preserved (identical pattern to existing row-level errors). Do not silently dedupe or silently save.
- If more than one duplicate pair/group exists, flag all of them (not just the first pair found).

### Claude's Discretion
- Exact wording of the duplicate-slot error message, consistent with existing UI-SPEC copy conventions (Phase 1's `rowErrorCopy` pattern: "End time must be after start time.")
- Whether to add matching client-side (JS) duplicate detection as a nice-to-have; server-side is the required, authoritative check
- Icon/label choice for the "Copy" button (e.g. "Copy" text label, consistent with existing text-only button convention — no icon library per Phase 1's locked decision)

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/web/templates/create.html` — the two `<template id="slot-row-template-*">` blocks and the server-rendered row loop; `.time-field` markup with `data-time-field`/`data-time-input`/`data-time-dropdown-toggle`/`data-time-step` attributes
- `internal/web/static/create.js` — `wireRow`, `wireTimeField`, `rowTemplateFor`, `afterRowChange` — all the row-management machinery this phase extends
- `internal/web/static/style.css` — `.time-field`, `.btn-time-step`, `.btn-time-dropdown-toggle`, `.slot-row` — existing tokens/classes to fix rather than replace
- `internal/web/server.go` — `parseSlots`/`parseAllDaySlots`/`parseDateTimeSlots`, `slotView` struct, `hasError`/`bannerErrorCopy` validation pattern in `handleCreatePoll`

### Established Patterns
- Row-level errors render via `slotView.Error` + `{{if $slot.Error}}<p class="field-error">{{$slot.Error}}</p>{{end}}` in `create.html`
- All validation is server-authoritative; JS is progressive enhancement only

### Integration Points
- No new routes, no schema changes. All changes are within `create.html`, `create.js`, `style.css`, and `handleCreatePoll`/`parseSlots` in `server.go`.

</code_context>

<specifics>
## Specific Ideas

None beyond what's captured above.

</specifics>

<deferred>
## Deferred Ideas

- Overlap-based (not just exact-match) duplicate detection — explicitly rejected by the user for this milestone
- Applying the same Copy/duplicate-detection UX to the admin edit form (Phase 4's `edit.html`) — out of scope, poll-creation form only

</deferred>
