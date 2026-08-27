# Phase 2: Voting End-to-End - Context

**Gathered:** 2026-08-25
**Status:** Ready for planning

<domain>
## Phase Boundary

A participant opens the participant link (no login), enters a display name, answers Yes/No/Maybe for every slot in the poll, and optionally attaches one short comment to their overall response. They can revisit the same link later and see their previous response pre-filled, then resubmit a changed version. Works on mobile and desktop. Results display (the tally grid) is Phase 3 — this phase only needs to collect and persist responses correctly, including edits.

</domain>

<decisions>
## Implementation Decisions

### Participant Identity & Revisit
- On first submit, set an HTTP-only cookie scoped to `/poll/{participant_token}` identifying this participant (random token) — no login or name-based matching
- On revisit, if the cookie matches an existing response for this poll, pre-fill the form from it; if not, treat as a brand-new participant
- A submission with no matching cookie ALWAYS creates a new response row, even if the display name matches an existing participant — never silently overwrite someone else's vote by name collision alone
- Cookie lifetime: ~1 year, so "come back later and change your vote" actually works across browser restarts, not just within a session
- No cookie / different device = no recovery flow in v1 (accepted limitation) — the participant just submits as a new response; duplicate-looking names in the results grid are an acceptable v1 trade-off

### Voting Form Layout
- Slots render as a vertical list of rows (one per slot: date/time label + answer control) — not a table; matches the mobile-first, single-column pattern from Phase 1's forms
- Each slot's Yes/No/Maybe choice is a segmented pill-button group (44px min tap target), not native radio buttons
- Every slot requires an explicit answer before submit — no blank/implicit-No; keeps Phase 3's tally data complete and unambiguous
- Display-name field is at the top of the form, above the slot list (identity/context first, then choices — same order as the poll-creation form)

### Response Data Model & Persistence
- Normalized schema: `participants` (id, poll_id, display_name, comment, cookie_token, created_at, updated_at) + `responses` (participant_id, slot_id, answer) — extends Phase 1's already-locked normalized shape (polls/slots/participants/responses)
- Resubmitting/editing an existing response UPDATEs the participant + response rows in place — no edit history kept in v1
- `answer` is a string enum (`"yes"` / `"no"` / `"maybe"`) with a DB CHECK constraint, matching Phase 1's `poll_type` enum pattern
- `responses.slot_id` uses `ON DELETE CASCADE` back to `slots` — cheap forward-compatibility for Phase 4's admin editing; slot editing itself is out of scope for this phase

### Validation & Confirmation Copy
- Blank display name is rejected with an inline field error: "Your name is required."
- Display name: max 100 chars (matches `organizer_name`'s limit from Phase 1). Comment: max 500 chars (VOTE-04 frames it as a "short comment")
- On successful submit, redirect back (303) to the same voting page, now showing a confirmation banner: "Thanks! Your response has been saved." above the still-editable form
- An unknown/invalid poll link shows a plain 404 page: "This poll link isn't valid. Double-check the link you were given."

### Claude's Discretion
- Exact styling of the pill-button group and confirmation banner, consistent with 01-UI-SPEC.md's established design tokens (spacing scale, color usage, typography)
- Internal handler/package structure for the new voting routes
- Whether the cookie token is a bare random string or reuses the existing `internal/token` package's generator (likely the latter, for consistency)

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/token` package (Phase 1) — `token.New()` crypto/rand token generator; reuse for the participant cookie token
- `internal/store` package (Phase 1) — SQLite access patterns, embedded `schema.sql` with idempotent `CREATE TABLE IF NOT EXISTS`; extend with `participants`/`responses` tables
- `internal/web` package (Phase 1) — `html/template` server-rendered pages, `create.html`/`links.html` as reference patterns for form + confirmation-banner layout; `create.js` as the reference pattern for progressive-enhancement JS (this phase's pill-button group JS should follow the same style)
- Design tokens from `01-UI-SPEC.md` — spacing scale, color usage (60/30/10 split), typography scale, 44px tap-target rule

### Established Patterns
- Server-rendered forms with inline field errors + a banner-error summary (see `create.html`'s `#form-banner` pattern)
- POST → validate → either re-render with errors (200) or redirect (303) on success
- Poll type / enum values stored as validated strings with a DB CHECK constraint

### Integration Points
- New routes needed: `GET /poll/{ptoken}` (voting form) and `POST /poll/{ptoken}/responses` (or similar) — separate from the existing `GET /poll/{ptoken}/admin/{atoken}` links page
- `internal/store/schema.sql` gains `participants` and `responses` table definitions

</code_context>

<specifics>
## Specific Ideas

None beyond the reference-product framing already captured in PROJECT.md (Doodle/Rallly-style voting).

</specifics>

<deferred>
## Deferred Ideas

- Recovery flow for a participant who lost their cookie/switched devices (e.g. a resend-my-edit-link feature) — deferred, no email infra and not requested
- Edit history / audit trail of past responses — deferred, no one asked for it
- Response cleanup when an admin edits/removes slots — deferred to Phase 4 (admin poll management), which will need to handle it when it adds slot editing

</deferred>
