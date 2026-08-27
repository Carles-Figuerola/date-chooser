# Requirements: Date Chooser — Milestone v1.1

**Defined:** 2026-08-27
**Core Value:** Make creating a multi-slot poll faster and less error-prone.

## v1.1 Requirements

### Slot Picker UX

- [x] **SLOT-01**: The time-picker input's visual height matches the +/-15 and dropdown-toggle buttons beside it (currently mismatched)
- [x] **SLOT-02**: Clicking anywhere on a time input (not just the small clock icon) opens the hour-dropdown popup
- [x] **SLOT-03**: When the organizer picks a start time on a specific-time slot, the end time auto-fills to exactly 1 hour later (organizer can still edit it manually)
- [x] **SLOT-04**: Each slot row has a "Copy" button that duplicates that slot's current values (date/time or date, depending on mode) into a new row appended to the list
- [x] **SLOT-05**: Submitting a poll with two exact-duplicate slots is rejected server-side with an inline error identifying the duplicate row(s) — no partial/silent save. "Exact duplicate" = same date for all-day slots; same date + start time + end time for specific-time slots

## Out of Scope

| Feature | Reason |
|---------|--------|
| Fuzzy/overlap duplicate detection (partial time overlap) | User explicitly chose exact-match only; overlapping-but-distinct slots are a legitimate use case (e.g. two candidate windows on the same day) |
| Editing an existing live poll's slots to use this same duplicate/copy UX | Out of scope — this milestone targets the poll-creation form only; Phase 4's edit form is a separate template not touched here unless a shared component naturally carries the fix |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SLOT-01 | Phase 5 | Complete |
| SLOT-02 | Phase 5 | Complete |
| SLOT-03 | Phase 5 | Complete |
| SLOT-04 | Phase 5 | Complete |
| SLOT-05 | Phase 5 | Complete |

**Coverage:**

- v1.1 requirements: 5 total
- Mapped to phases: 5
- Unmapped: 0 ✓

---
*Requirements defined: 2026-08-27*
*Last updated: 2026-08-27*
