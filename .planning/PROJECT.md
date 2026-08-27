# Date Chooser

## What This Is

A self-hosted, account-free meeting scheduler (a Doodle/Rallly-style poll tool). An organizer creates a poll with a set of candidate dates/times, shares a link with a group, and each invitee marks Yes/No/Maybe (with an optional comment) per slot. The organizer gets a secret admin link to view results and manage the poll. Runs as a single Docker container backed by SQLite on a mounted volume.

## Core Value

An organizer can create a poll of date/time options and get back, without any signup, a clear grid of who's available when — so they can pick the best slot.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Organizer can create a poll with a name/description and a set of candidate date/time slots
- [ ] Organizer receives a shareable participant link and a separate secret admin link
- [ ] Participant can open the poll link (no login) and respond Yes/No/Maybe per slot, with an optional short comment
- [ ] Poll page shows a results grid (slots x participants) with per-slot Yes/No/Maybe tallies
- [ ] Admin can view all responses and see which slot(s) have the best overall availability
- [ ] Admin can edit or delete the poll
- [ ] Works well on both desktop and mobile browsers
- [ ] App runs from a single Docker image, storing all data in a SQLite file on a mounted volume

### Out of Scope

- User accounts / authentication — security model is unguessable links (participant link + secret admin link) instead; simplest option matching an internal/small-group tool
- OAuth / social login — no accounts at all in v1
- Email notifications / reminders — no email infrastructure in v1; can be added later
- Timezone auto-detection / multi-timezone display — start with a single timezone (server or poll-selected) to keep v1 simple; revisit if needed
- Real-time collaborative editing of responses — simple submit/re-submit is enough
- Calendar integrations (Google/Outlook sync) — out of scope for v1, self-contained poll tool only

## Context

- Reference products: TallyCal, Doodle, Rallly (github.com/lukevella/rallly) — all "propose dates, collect availability, view results" tools. This project is a lightweight, self-hosted clone of that pattern.
- No existing codebase — greenfield project.
- Intended deployment: single Docker container, SQLite file on a volume mount. No external services, no managed DB.

## Constraints

- **Tech stack**: Backend in Go (decided over Python — see Key Decisions) — Why: user is comfortable with both; Go produces a single static binary and a minimal Docker image, which fits the "one container + one volume" deployment model cleanly.
- **Database**: SQLite only — Why: explicitly requested by user; avoids running/operating a separate DB service.
- **Auth model**: No user accounts. Access control is via unguessable link tokens (participant link, admin link) — Why: explicitly requested; user is fine with treating the admin link as a secret to store safely.
- **Deployment**: Must run as a Docker container with the SQLite file on a mounted volume — Why: explicitly requested deployment shape.
- **Interface**: Must be usable from both desktop browsers and mobile — Why: participants will respond from whatever device they have.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go over Python for backend | User offered either; Go gives a single static binary, trivial minimal-footprint Docker image, and simple SQLite driver story (no separate runtime/venv to containerize) — best fit for the "single container + volume" deployment goal | — Pending |
| No accounts, link-based access (participant + admin tokens) | Explicitly requested; matches reference tools' common "shareable link" pattern for small/ad-hoc groups | — Pending |
| SQLite for storage | Explicitly requested; sufficient for expected scale (small groups, low write volume) | — Pending |
| Server-rendered web UI (not a SPA) over a Go backend | Simplest way to get a fast, mobile-friendly UI with minimal JS, fits a single-binary Go app well | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-08-24 after initialization*
