# Roadmap: Date Chooser

## Milestones

- ✅ **v1.0** — Date Chooser MVP (poll creation, voting, results grid, admin management) — shipped 2026-08-25. See `.planning/milestones/v1.0-ROADMAP.md`.
- ✅ **v1.1** — Poll-Creation Slot UX Improvements (SLOT-01..05) — shipped 2026-08-27.

## Current Milestone: v1.2 — Slot Import/Export & Instance Admin Page

Two independent features: (1) export/import a poll's slots as a plain-text file so populating or cloning a poll doesn't mean re-entering every date/time by hand, and (2) a site-wide `/admin` page listing every poll (with its links), gated by a secret only retrievable via direct `sqlite3` access to the database file — a fallback for when an organizer loses a poll's individual links.

- [ ] **Phase 6: Slot Import/Export** — Export/import a poll's slots as a `.txt` file (Not started)
- [ ] **Phase 7: Instance Admin Page** — Secret-gated `/admin` page listing all polls (Not started)

## Phase Details

### Phase 6: Slot Import/Export

**Goal**: An organizer can export a poll's current slots to a text file and re-import that same format (on this poll or a new one) to populate the slot list, instead of re-entering every date/time by hand.
**Depends on**: Phase 1 (extends the existing create/edit forms)
**Requirements**: IMPORT-01, IMPORT-02, IMPORT-03
**Success Criteria** (what must be TRUE):

  1. Clicking "Export slots" downloads a `.txt` file with one line per current slot, in the documented format
  2. Choosing a `.txt` file via "Import slots" replaces the form's current slot rows with the parsed contents, ready to review and submit
  3. A file with some unparsable lines still imports the valid lines and shows a visible skipped-line count — it never silently drops lines or blocks the whole import

**Plans:** 0/? plans complete

**UI hint**: yes

### Phase 7: Instance Admin Page

**Goal**: An organizer who loses a poll's individual links can recover them from a single site-wide page, without that page being reachable by anyone who doesn't have direct file access to the server's SQLite database.
**Depends on**: Phase 1 (lists polls created via the existing poll-creation flow)
**Requirements**: ADMIN-01, ADMIN-02, ADMIN-03, ADMIN-04, ADMIN-05
**Success Criteria** (what must be TRUE):

  1. On a fresh database, the instance-admin secret is auto-generated on first server startup and persisted in a `settings` table
  2. `/admin` is unreachable without first submitting the correct secret at `/admin/login`; a valid submission sets a session-only cookie
  3. Once logged in, `/admin` lists every poll's title, created date, participant link, and admin link
  4. The README documents the exact `sqlite3` command to read the secret out of the database file
  5. The `/admin/login` page itself has a collapsed `<details>` disclosure showing that same `sqlite3` command, so it's discoverable without leaving the page

**Plans:** 0/? plans complete

**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 6, 7

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 6. Slot Import/Export | 0/? | Not started | - |
| 7. Instance Admin Page | 0/? | Not started | - |
