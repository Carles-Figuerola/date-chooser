# Phase 3: Results Grid - Context

**Gathered:** 2026-08-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Anyone holding the participant OR admin link sees a grid of slots (rows) by participants (columns), each cell showing that participant's Yes/No/Maybe, plus aggregate per-slot tallies, a visual highlight on the best-availability slot(s), and participant comments listed below the grid. No editing/deleting yet (Phase 4). The grid renders on the same page as the voting form for the participant link, and identically for the admin link.

</domain>

<decisions>
## Implementation Decisions

### Results Grid Layout
- On narrow screens, the grid scrolls horizontally with the slot-label column pinned/sticky on the left, so the row context never scrolls out of view
- Each cell renders a small colored badge reusing Phase 2's tri-color tokens (green `#16A34A` for Yes, red `#DC2626` for No, gray `#64748B` for Maybe) — not colored dots alone, not full-word labels (too wide for a dense grid)
- A missing cell (shouldn't normally occur — Phase 2 requires every slot answered before submit) renders as "—" rather than blank, as an honest data fallback, not a designed empty state
- Column headers show just the participant's display name; comments are never crammed into headers

### Best-Slot Highlighting Rule
- Ranking: sort by most Yes (descending), tie-break by fewest No (ascending) — Maybe does not factor into the ranking, matching the phase goal's literal "most Yes, fewest No"
- If multiple slots tie for the top rank, highlight ALL of them — never arbitrarily pick one
- No highlighting at all when the poll has zero responses (every slot ties at 0/0; "best" is meaningless with no data) — show the grid with slot rows and a "No responses yet." note instead
- Highlighted row(s) get BOTH a background tint (reusing the existing light accent tint `#EFF6FF` + accent border already used for admin-link boxes) AND a text badge ("Best fit") — color alone is not accessible for colorblind users

### Comments Display
- Comments render in a separate "Comments" section below the grid, one line per participant who left a non-empty comment: "**{name}**: {comment}"
- If nobody left a comment, the entire section is omitted — no empty "Comments" heading shown
- Full comment text is shown, no truncation (already capped at 500 chars from Phase 2, short enough not to need it)

### Page Placement & Access Parity
- The results grid renders on the exact same route/template as the voting form (`GET /poll/{ptoken}`) — vote form on top, results grid below — not a separate results sub-route
- The admin link (`GET /poll/{ptoken}/admin/{atoken}`) shows the identical grid (same data, same rendering) — no admin-only controls yet; those arrive in Phase 4 (ADM-02/03/04)

### Claude's Discretion
- Exact HTML table vs. CSS grid implementation choice for the results table, as long as the sticky-first-column + horizontal-scroll behavior works on mobile
- Internal handler/query structure for computing tallies and the best-slot ranking
- Whether tallies are computed in SQL or in Go after fetching rows — pick whichever is simplest given the existing `internal/store` patterns

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/store` (Phase 1/2) — `participants`/`responses`/`slots`/`polls` tables already exist; this phase is read-only aggregation over them, no new tables needed
- `internal/web` (Phase 1/2) — `html/template` server-rendered pages; `vote.html` is the template this phase extends (results grid renders on the same page)
- Tri-color pill tokens from `02-UI-SPEC.md` / `style.css` (`--color-yes`, `--color-destructive`, `--color-text-muted`) — reuse directly for grid-cell badges
- The existing admin-link-box accent tint (`#EFF6FF` + accent border) from Phase 1's `links.html` — reuse for the best-slot row highlight

### Established Patterns
- Server-rendered pages with no new JS dependency unless needed for a specific interaction (this phase likely needs none beyond what Phase 2 already added)
- `internal/store` methods that fetch and shape data for template consumption (see `PollByParticipantToken`, `ParticipantByCookie` from Phase 2)

### Integration Points
- `GET /poll/{ptoken}` (Phase 2's vote-form route) gains a results-grid section
- `GET /poll/{ptoken}/admin/{atoken}` (Phase 1's links route) gains the same results-grid section
- New store query/queries: fetch all participants + their responses for a poll, shaped for grid rendering plus tally computation

</code_context>

<specifics>
## Specific Ideas

None beyond the reference-product framing already captured in PROJECT.md (Doodle/Rallly-style results grid).

</specifics>

<deferred>
## Deferred Ideas

- Admin-only visual distinction (badge/header) on the admin view of the grid — deferred; admin view is pixel-identical to participant view for now
- Editing/deleting responses or the poll from this page — deferred to Phase 4 (ADM-02/03/04)

</deferred>
