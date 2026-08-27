# Roadmap: Date Chooser

## Overview

Date Chooser ships as four end-to-end vertical slices, each independently demoable in the target Docker+SQLite deployment. Phase 1 gets an organizer from "nothing" to a running container that can create a poll and hand back a participant link and a secret admin link, with data surviving a restart. Phase 2 lets a participant open that link with no account, vote, and come back later to change their mind. Phase 3 turns those votes into the results grid everyone (participant or admin) actually came for, including the best-slot highlight. Phase 4 closes the loop with organizer-side editing and deletion once there's a real poll with real responses to manage. Each phase builds directly on data created by the previous one — there is no "build all the backend, then all the frontend" layer here.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Poll Creation & Dockerized Foundation** - Organizer creates a poll and gets participant/admin links, running in a Docker container with persistent SQLite storage (completed 2026-08-25)
- [x] **Phase 2: Voting End-to-End** - Participant opens the link with no account, votes Yes/No/Maybe with a comment, and can revise later (completed 2026-08-25)
- [x] **Phase 3: Results Grid** - Participants and admin see the slots-by-participants grid with tallies, best-slot highlight, and comments (completed 2026-08-25)
- [ ] **Phase 4: Admin Poll Management** - Organizer edits poll details and deletes responses or the whole poll via the secret admin link

## Phase Details

### Phase 1: Poll Creation & Dockerized Foundation

**Goal**: An organizer can create a poll with candidate date/time slots and immediately get back a participant link and a secret admin link, running as a single Docker container with all data persisted in a SQLite file on a mounted volume.
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: POLL-01, POLL-02, POLL-03, POLL-04, POLL-05, OPS-01, OPS-02, OPS-03, OPS-04
**Success Criteria** (what must be TRUE):

  1. Organizer can create a poll via a form with a title, optional description, and one or more candidate slots, choosing per-poll whether slots are all-day dates or specific date+time ranges
  2. Immediately after creation, the organizer sees a shareable participant link and a separate admin link that is clearly marked as secret
  3. The application starts and serves HTTP traffic from a single Docker image, with listen port and SQLite file path configurable via environment variables
  4. A poll created before a container restart is still present after restarting the container against the same mounted volume, and the app also starts cleanly against a fresh empty volume

**Plans:** 3/3 plans complete

- [x] 01-01-PLAN.md — Walking Skeleton: end-to-end poll-creation tracer (Go + net/http + modernc.org/sqlite + crypto/rand tokens) packaged as a single Docker image with persistent /data volume
- [x] 01-02-PLAN.md — Full poll-creation form (title/description/organizer, all-day vs date+time toggle, add/remove slot rows) with server-side validation and multi-slot persistence
- [x] 01-03-PLAN.md — Links-display page showing participant + secret admin links with copy-to-clipboard

**UI hint**: yes

### Phase 2: Voting End-to-End

**Goal**: A participant can open the shared poll link with no account, respond Yes/No/Maybe (with an optional comment) to each slot, and return later to change their response — on both mobile and desktop.
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: VOTE-01, VOTE-02, VOTE-03, VOTE-04, VOTE-05, VOTE-06
**Success Criteria** (what must be TRUE):

  1. Participant can open the participant link directly, with no login or signup step, and see the poll's slots
  2. Participant enters a display name and selects Yes, No, or Maybe for each slot before submitting
  3. Participant can attach an optional short comment to their overall response
  4. Participant can revisit the same participant link later, see their previous response pre-filled, and resubmit a changed response
  5. The voting page renders and works correctly on both narrow (mobile) and wide (desktop) screens

**Plans:** 3/3 plans complete

- [x] 02-01-PLAN.md — Tracer: end-to-end voting slice (participants/responses schema + store + GET/POST routes + HttpOnly cookie), happy path proven with revisit/resubmit
- [x] 02-02-PLAN.md — Server-side validation (name required, every slot answered, length caps) with exact error copy, plus the branded invalid-link 404 page
- [x] 02-03-PLAN.md — Voting UI: tri-color Yes/No/Maybe pill groups, confirmation banner, responsive layout (VOTE-06), and the vote.js submit guard

**UI hint**: yes

### Phase 3: Results Grid

**Goal**: Anyone holding the participant or admin link can see, at a glance, who is available for which slot and which slot(s) work best for the group.
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: RES-01, RES-02, RES-03, RES-04, ADM-01
**Success Criteria** (what must be TRUE):

  1. The poll page shows a grid of slots (rows) by participants (columns), with each cell showing that participant's Yes/No/Maybe
  2. Each slot displays aggregate Yes/No/Maybe counts across all respondents
  3. The slot(s) with the best overall availability (most Yes, fewest No) are visually highlighted in the grid
  4. Participant comments are shown alongside their responses in the results view
  5. Opening the poll via the admin link shows the same results grid as the participant link

**Plans:** 2/2 plans complete

- [x] 03-01-PLAN.md — Results-grid tracer: read-only store aggregation + tally/best-slot ranking + comments/zero-state rendering on the participant route (RES-01..04)
- [x] 03-02-PLAN.md — Admin-route grid parity (ADM-01) + grid CSS (tri-color badges, sticky slot-label column, best-fit highlight)

**UI hint**: yes

### Phase 4: Admin Poll Management

**Goal**: The organizer, via the secret admin link, can correct or clean up a poll after it has gone live — editing its details or removing responses or itself — with no account needed.
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: ADM-02, ADM-03, ADM-04
**Success Criteria** (what must be TRUE):

  1. Admin can edit the poll's title, description, and slot list after creation, and those changes are reflected for participants
  2. Admin can delete a single participant's response, removing it from the results grid
  3. Admin can delete the entire poll, after which the participant and admin links no longer serve poll data

**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Poll Creation & Dockerized Foundation | 3/3 | Complete    | 2026-08-25 |
| 2. Voting End-to-End | 3/3 | Complete    | 2026-08-25 |
| 3. Results Grid | 2/2 | Complete    | 2026-08-25 |
| 4. Admin Poll Management | 0/TBD | Not started | - |
