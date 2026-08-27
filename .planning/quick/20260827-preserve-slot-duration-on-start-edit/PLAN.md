---
slug: preserve-slot-duration-on-start-edit
date: 2026-08-27
---

Bug fix: editing a slot row's start time (after Copy, or on any row with both
start and end already filled) leaves the end time stale instead of shifting
it to preserve the original duration.

## Root cause

`internal/web/static/create.js`'s start-time `change` listener (SLOT-03) only
acts when `endInput.value` is empty (auto-fill start+1h). When the end time
is already populated — e.g. right after using Copy, which clones both start
and end — changing the start time hits the early `return` and the end time
is left untouched.

## Fix

Track each row's previous start-time value (captured at `wireRow` time, so
Copy-cloned rows correctly start from the cloned start value). On a start-time
`change`:
- If the end time is empty: keep existing SLOT-03 auto-fill (+1h).
- Else: compute the duration between the *previous* start and current end,
  then set the new end to `newStart + duration` (mod 1440), preserving the
  gap. Update the tracked previous value after handling either branch.

This covers all three ways a start time changes (typing, native picker,
hour-dropdown selection) since all three already dispatch a `change` event
on the start input. The existing ±15 stepper cross-shift logic (which
already preserves duration by shifting the end by the same delta) is
untouched.

## Files

- `internal/web/static/create.js`
- `internal/web/create_ux_test.go` (add a source-assertion test covering the
  duration-preserving branch, matching existing test style in that file)
