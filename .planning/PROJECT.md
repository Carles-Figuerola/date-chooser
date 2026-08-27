# Date Chooser

## Current State

**v1.1 shipped 2026-08-27.** Poll-creation slot picker UX improvements (SLOT-01..05) delivered and verified: consistent time-field sizing with narrow-screen wrapping, click-anywhere popups on both date and time inputs, start-time auto-fill/±15-linked end time, per-slot Copy button, and server-side exact-duplicate-slot rejection with accurate error copy. Two rounds of manual testing caught and fixed 3 real bugs (row overflow, auto-fill not firing via the custom dropdown, misleading error copy on a missing date) beyond the original 5 requirements.

**v1.0 shipped 2026-08-25.** All 23 v1 requirements delivered and verified across 4 phases (Poll Creation & Docker, Voting, Results Grid, Admin Management). Full history: [`.planning/milestones/v1.0-ROADMAP.md`](milestones/v1.0-ROADMAP.md) and [`.planning/milestones/v1.0-REQUIREMENTS.md`](milestones/v1.0-REQUIREMENTS.md).

## Next Milestone Goals

No milestone is currently active. Run `/gsd-new-milestone` when ready to scope the next one.

## Deferred (v2 candidates, not in this milestone)

- Notifications (email/webhook on new responses)
- Timezone-aware slot display
- Calendar (.ics) export
- Poll "finalize" step that locks in the chosen slot and notifies participants

## What This Is

A self-hosted, account-free meeting scheduler (a Doodle/Rallly-style poll tool). An organizer creates a poll with a set of candidate dates/times, shares a link with a group, and each invitee marks Yes/No/Maybe (with an optional comment) per slot. The organizer gets a secret admin link to view results and manage the poll. Runs as a single Docker container backed by SQLite on a mounted volume.

## Core Value

An organizer can create a poll of date/time options and get back, without any signup, a clear grid of who's available when — so they can pick the best slot.

## Requirements

### Validated

- ✓ Organizer can create a poll with a name/description and a set of candidate date/time slots — Phase 1
- ✓ Organizer receives a shareable participant link and a separate secret admin link — Phase 1
- ✓ App runs from a single Docker image, storing all data in a SQLite file on a mounted volume — Phase 1
- ✓ Participant can open the poll link (no login) and respond Yes/No/Maybe per slot, with an optional short comment — Phase 2
- ✓ Poll page shows a results grid (slots x participants) with per-slot Yes/No/Maybe tallies — Phase 3
- ✓ Admin can view all responses and see which slot(s) have the best overall availability — Phase 3
- ✓ Admin can edit or delete the poll — Phase 4
- ✓ Works well on both desktop and mobile browsers — Phase 1 + Phase 2 + Phase 3 + Phase 4 (every screen confirmed at mobile width)

- ✓ Time-picker button height matches the +/-15/dropdown buttons visually — Phase 5
- ✓ Clicking anywhere on a date or time input opens its picker/dropdown — Phase 5
- ✓ Selecting a start time auto-fills the end time to +1 hour; ±15 steppers on start also shift end — Phase 5
- ✓ A "Copy" button per slot duplicates that slot's values into a new row — Phase 5
- ✓ Poll submission is rejected if it contains two exact-duplicate slots, with accurate row-level error copy — Phase 5

### Active

(None — v1.1 shipped and validated.)

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
| Go over Python for backend | User offered either; Go gives a single static binary, trivial minimal-footprint Docker image, and simple SQLite driver story (no separate runtime/venv to containerize) — best fit for the "single container + volume" deployment goal | ✓ Good — Phase 1 shipped a working multi-stage distroless image |
| No accounts, link-based access (participant + admin tokens) | Explicitly requested; matches reference tools' common "shareable link" pattern for small/ad-hoc groups | ✓ Good — Phase 1: independent crypto/rand tokens, admin token not derivable from participant token |
| SQLite for storage | Explicitly requested; sufficient for expected scale (small groups, low write volume) | ✓ Good — `modernc.org/sqlite` (pure Go, no CGO), embedded idempotent schema |
| Server-rendered web UI (not a SPA) over a Go backend | Simplest way to get a fast, mobile-friendly UI with minimal JS, fits a single-binary Go app well | ✓ Good — Phase 1: `html/template` + a small progressive-enhancement JS file per page |
| Go 1.25 (not 1.23 as originally planned) | `modernc.org/sqlite`'s transitive deps required go ≥1.25 to compile — discovered during Phase 1 execution | ✓ Good — no other impact |
| Date+time slot inputs: separate `<input type=date>` + two `<input type=time step=900>` (not `datetime-local` x2) | User found the original `datetime-local` x2 UX confusing (calendar icon only picks date, no visible time picker) during manual testing; CONTEXT.md already established start/end share one date, so this only needed a rendering split, not a data-model change | ✓ Good — Phase 1, plus a custom hour-dropdown + ±15min stepper layered on top per follow-up user feedback |
| Participant identity via HttpOnly cookie (not login, not name-matching) | No accounts in scope; a cookie lets "revisit and change your vote" work without introducing any auth; anti-overwrite rule (no cookie match = always a new response) protects against name-collision data loss | ✓ Good — Phase 2, confirmed working (revisit pre-fill + resubmit) in manual testing |
| Yes/No/Maybe pill buttons: tri-color (green/red/gray), not single-accent | UI researcher's draft used one accent color for all three states to stay within Phase 1's 2-color budget; user explicitly asked for the more scannable Doodle/Rallly convention instead | ✓ Good — Phase 2, confirmed correct rendering (colors, tap targets, no overflow at mobile width) |
| Results grid cell glyphs: ✓/✗/? (not Y/N/M letters) | Researcher's draft used single letters to avoid an icon-library dependency; user asked for checkmark/X/question glyphs instead — still plain Unicode text, no new dependency, more scannable | ✓ Good — Phase 3, confirmed correct rendering (sticky column, name truncation) |
| Best-slot ranking: most Yes desc, fewest No as tiebreak; highlight ALL ties, never arbitrarily pick one | Matches the roadmap's literal "most Yes, fewest No" wording; picking only one slot on a tie would be misleadingly precise given the data | ✓ Good — Phase 3 |
| No admin-only visual distinction on the results grid yet (identical to participant view) | Editing/deleting is Phase 4's scope; nothing to distinguish yet — avoids premature UI for controls that don't exist | ✓ Good — Phase 3; Phase 4 added the "Remove" control and edit-page link, gated to the admin route only |
| Removing a slot with existing responses cascades (deletes only that slot's responses); adding a new slot to a live poll shows "—" for existing participants; poll_type is locked after creation; deletion (response or poll) is immediate/permanent with a native `confirm()` gate, no soft-delete | No accounts/auth in scope, so a lightweight `confirm()` dialog is the safety net, not a heavier type-to-confirm pattern; matches the tool's small-group, low-stakes scale | ✓ Good — Phase 4, confirmed working (all 3 double-submit guards + cascade-delete row counts) |
| Destructive-red (`#DC2626`) escalated to real destructive controls, with two visual weights: outline for per-response removal, solid fill for whole-poll deletion | First phase to actually use the destructive token for real data-loss actions (previously only "No" pill semantics); differentiating severity visually reinforces the `confirm()` dialog's warning | ✓ Good — Phase 4 |
| "Edit poll" link added to the admin results page | Manual testing caught a real navigation gap: the `/edit` route worked but nothing in the UI linked to it. No UI-SPEC/plan had specified this entry point — a genuine spec gap, not an execution bug | ✓ Good — Phase 4, fixed and confirmed working |

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
*Last updated: 2026-08-27 after Phase 5 (v1.1 complete)*
