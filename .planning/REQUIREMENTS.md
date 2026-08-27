# Requirements: Date Chooser — Milestone v1.2

**Defined:** 2026-08-27
**Core Value:** Make it faster to populate/clone polls, and give the organizer a way to recover poll links without losing them.

## v1.2 Requirements

### Slot Import/Export

- [ ] **IMPORT-01**: An "Export slots" control on the create/edit page downloads the current slot list as a `.txt` file, one line per slot: `YYYY-MM-DD,HH:MM,HH:MM` for specific-time polls, `YYYY-MM-DD` for all-day polls
- [ ] **IMPORT-02**: An "Import slots" control on the create/edit page reads a `.txt` file in that same format (client-side) and replaces the current slot rows in the form — the organizer reviews the populated form and submits normally; no server round-trip for the import step itself
- [ ] **IMPORT-03**: Lines that don't parse as a valid slot are skipped with a visible count (e.g. "2 of 8 lines could not be read and were skipped") rather than blocking the whole import or failing silently

### Instance Admin Page

- [ ] **ADMIN-01**: A new `settings` table stores a single instance-admin secret (crypto/rand token), auto-generated on first server startup if not already present
- [ ] **ADMIN-02**: A `/admin/login` page accepts the secret via a form; on success it sets a session-only cookie (no `Max-Age`/`Expires` — cleared when the browser session ends) authorizing access to `/admin`
- [ ] **ADMIN-03**: `/admin` (unreachable without a valid session cookie) lists every poll: title, created date, participant link, admin link — read-only, no delete/edit actions from this page
- [ ] **ADMIN-04**: README documents the exact `sqlite3` command to retrieve the secret directly from the database file

## Out of Scope

| Feature | Reason |
|---------|--------|
| Deleting/editing polls from the instance-admin page | Explicitly scoped read-only — recovering a lost link is the goal; destructive/edit actions already exist via each poll's own admin link |
| Rate-limiting or lockout on `/admin/login` | The secret is a long random token (same threat model as the existing per-poll admin token, which also has no rate limiting) — brute force is infeasible regardless |
| Rotating/resetting the secret through the web UI | Explicitly requested to be sqlite3-only; keeps the attack surface for the secret itself at zero HTTP exposure |
| Live-synced textbox for import (types-as-you-go) | User chose file-based import/export over a live-editable textbox — simpler, matches this app's low-JS-complexity pattern |
| Merging imported slots with existing ones | User chose replace-whole-list semantics — simpler mental model, avoids accidental duplicates |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| IMPORT-01 | Phase 6 | Pending |
| IMPORT-02 | Phase 6 | Pending |
| IMPORT-03 | Phase 6 | Pending |
| ADMIN-01 | Phase 7 | Pending |
| ADMIN-02 | Phase 7 | Pending |
| ADMIN-03 | Phase 7 | Pending |
| ADMIN-04 | Phase 7 | Pending |

**Coverage:**

- v1.2 requirements: 7 total
- Mapped to phases: 7
- Unmapped: 0 ✓

---
*Requirements defined: 2026-08-27*
*Last updated: 2026-08-27*
