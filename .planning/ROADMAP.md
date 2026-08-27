# Roadmap: Date Chooser

## Milestones

- ✅ **v1.0** — Date Chooser MVP (poll creation, voting, results grid, admin management) — shipped 2026-08-25. See `.planning/milestones/v1.0-ROADMAP.md`.

## Current Milestone: v1.1 — Poll-Creation Slot UX Improvements

Make creating a multi-slot poll faster and less error-prone: fix the time-picker's visual sizing, make the whole time field clickable (not just the icon), auto-fill end-time from start-time, add a per-slot "Copy" button, and reject exact-duplicate slots on submit.

- [x] **Phase 5: Slot Picker UX Improvements** — Fixes and additions to the poll-creation form's slot inputs (Not started) (completed 2026-08-27)

## Phase Details

### Phase 5: Slot Picker UX Improvements

**Goal**: An organizer creating a poll can pick times more easily (consistent-height controls, click-anywhere popup, auto-filled end time), duplicate a slot with one click instead of re-entering it, and gets a clear error instead of a silently-accepted duplicate slot.
**Depends on**: Phase 1 (extends the existing poll-creation form)
**Requirements**: SLOT-01, SLOT-02, SLOT-03, SLOT-04, SLOT-05
**Success Criteria** (what must be TRUE):

  1. The time-picker input, its dropdown-toggle button, and its +/-15 buttons are all the same visual height
  2. Clicking anywhere on a time input (not just the clock icon) opens the hour-dropdown popup
  3. Picking a start time on a specific-time slot auto-fills that row's end time to 1 hour later; the organizer can still edit it afterward
  4. A "Copy" button on each slot row appends a new row pre-filled with that row's current values
  5. Submitting a poll containing two exact-duplicate slots (same date for all-day; same date+start+end for specific-time) is rejected with an inline error identifying the duplicate, not silently saved or partially saved

**Plans:** 2/2 plans complete

Plans:

- [x] 05-01-PLAN.md — Client-side slot-picker UX: time-field height (SLOT-01), click-anywhere popup (SLOT-02), auto-fill end time (SLOT-03), per-row Copy button (SLOT-04)
- [x] 05-02-PLAN.md — Server-side exact-duplicate-slot rejection (SLOT-05)

**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 5. Slot Picker UX Improvements | 2/2 | Complete    | 2026-08-27 |
