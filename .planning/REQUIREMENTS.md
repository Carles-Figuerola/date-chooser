# Requirements: Date Chooser

**Defined:** 2026-08-24
**Core Value:** An organizer can create a poll of date/time options and get back, without any signup, a clear grid of who's available when — so they can pick the best slot.

## v1 Requirements

### Poll Creation

- [ ] **POLL-01**: Organizer can create a poll with a title and optional description
- [ ] **POLL-02**: Organizer can add a set of candidate date/time slots when creating a poll
- [ ] **POLL-03**: Organizer can choose whether slots are all-day dates or specific date+time ranges
- [ ] **POLL-04**: On poll creation, the app generates a unique participant link (shareable) and a separate, unguessable admin link
- [ ] **POLL-05**: Organizer sees both links immediately after creating the poll, with the admin link clearly marked as secret

### Voting

- [ ] **VOTE-01**: Participant can open the participant link without logging in or creating an account
- [ ] **VOTE-02**: Participant enters a display name before voting (no password)
- [ ] **VOTE-03**: Participant can select Yes, No, or Maybe for each slot in the poll
- [ ] **VOTE-04**: Participant can add an optional short comment attached to their overall response
- [ ] **VOTE-05**: Participant can submit their response and later revisit the link to change it
- [ ] **VOTE-06**: The voting UI is usable on both mobile and desktop screen sizes

### Results

- [ ] **RES-01**: Poll page shows a grid of slots (rows) x participants (columns) with each cell showing that participant's Yes/No/Maybe for that slot
- [ ] **RES-02**: Each slot shows aggregate counts of Yes/No/Maybe across all participants
- [ ] **RES-03**: The slot(s) with the best overall availability (most Yes, fewest No) are visually highlighted
- [ ] **RES-04**: Participant comments are visible alongside their responses in the results view

### Admin

- [ ] **ADM-01**: Admin (via admin link) can view the same results grid as participants
- [ ] **ADM-02**: Admin can edit poll details (title, description, slots) after creation
- [ ] **ADM-03**: Admin can delete a participant's response
- [ ] **ADM-04**: Admin can delete the entire poll

### Platform / Ops

- [ ] **OPS-01**: Application is packaged as a single Docker image
- [ ] **OPS-02**: All application data is persisted in a SQLite database file stored on a mounted volume
- [ ] **OPS-03**: Application starts correctly against an empty volume (fresh install) and against an existing volume (restart) without data loss
- [ ] **OPS-04**: Application runs as an HTTP server configurable via environment variables (at least: listen port, database file path)

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Notifications

- **NOTF-01**: Organizer can receive an email/webhook when a participant responds

### Scheduling Depth

- **SCHED-01**: Timezone-aware slot display (auto-detect participant timezone, convert display)
- **SCHED-02**: Calendar export (.ics) of the finalized slot
- **SCHED-03**: Organizer can "finalize" a poll to lock in the chosen slot and notify participants

## Out of Scope

| Feature | Reason |
|---------|--------|
| User accounts / login | Explicitly excluded — link-based access is the whole point of the tool |
| OAuth / social login | No accounts at all in v1 |
| Email sending / reminders | No email infrastructure in v1 |
| Multi-timezone auto-detection | Adds real complexity; single timezone is enough for v1 |
| Real-time collaborative editing | Simple submit/re-submit is sufficient; no need for live sync |
| Calendar integrations (Google/Outlook) | Self-contained poll tool only, no external service dependencies |
| Multi-tenancy / hosted SaaS concerns (billing, orgs) | This is a self-hosted single-instance tool, not a hosted product |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| POLL-01 | Phase 1 | Pending |
| POLL-02 | Phase 1 | Pending |
| POLL-03 | Phase 1 | Pending |
| POLL-04 | Phase 1 | Pending |
| POLL-05 | Phase 1 | Pending |
| OPS-01 | Phase 1 | Pending |
| OPS-02 | Phase 1 | Pending |
| OPS-03 | Phase 1 | Pending |
| OPS-04 | Phase 1 | Pending |
| VOTE-01 | Phase 2 | Pending |
| VOTE-02 | Phase 2 | Pending |
| VOTE-03 | Phase 2 | Pending |
| VOTE-04 | Phase 2 | Pending |
| VOTE-05 | Phase 2 | Pending |
| VOTE-06 | Phase 2 | Pending |
| RES-01 | Phase 3 | Pending |
| RES-02 | Phase 3 | Pending |
| RES-03 | Phase 3 | Pending |
| RES-04 | Phase 3 | Pending |
| ADM-01 | Phase 3 | Pending |
| ADM-02 | Phase 4 | Pending |
| ADM-03 | Phase 4 | Pending |
| ADM-04 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 23 total (corrected from initial count of 22 — POLL(5) + VOTE(6) + RES(4) + ADM(4) + OPS(4) = 23)
- Mapped to phases: 23
- Unmapped: 0 ✓

---
*Requirements defined: 2026-08-24*
*Last updated: 2026-08-24 after roadmap creation — all v1 requirements mapped to Phases 1-4*
