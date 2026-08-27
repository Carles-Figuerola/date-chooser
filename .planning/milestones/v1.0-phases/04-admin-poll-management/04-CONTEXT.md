# Phase 4: Admin Poll Management - Context

**Gathered:** 2026-08-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Via the secret admin link, the organizer can edit a live poll's title, description, and slot list; delete a single participant's response; or delete the entire poll (after which both links 404). No new identity/auth model — access is still purely via the admin token. This is the final phase of v1.

</domain>

<decisions>
## Implementation Decisions

### Slot Editing & Existing Responses
- Removing a slot cascades to delete only that slot's response rows (reuses the `ON DELETE CASCADE` already set up in Phase 2's schema) — a participant's answers for other slots and their comment remain intact
- Adding a new slot to a poll that already has participants shows "—" (the existing missing-cell fallback from Phase 3) for those participants in the new column — no forced re-vote, no notification
- `poll_type` (all-day vs. specific-time) is locked after creation — the edit form can change title, description, and the slot list, but never the poll's type
- Because removing a slot with existing responses is real, irreversible data loss (unlike Phase 1's unsaved-draft slot removal), the edit form shows an inline warning with the response count and requires an explicit confirm step before saving such an edit

### Deletion UX
- A single participant's response is deleted via a "Remove" button next to that participant in the results grid (admin view only) — no separate response-management page
- Deleting a response requires a native `confirm()` JS dialog (real, irreversible action)
- Deleting the entire poll requires a native `confirm()` JS dialog with clear copy (poll title + response count) — proportionate friction for a small self-hosted tool; no type-the-title-to-confirm pattern
- After a poll is deleted, both the participant and admin links show the same 404 page already established in Phase 2 for invalid links — no distinct "deleted by organizer" message (avoids extra state/tombstone tracking)

### Admin Edit Form
- Editing happens on a separate route (`GET/POST /poll/{ptoken}/admin/{atoken}/edit`), reusing the poll-creation form's structure (title, description, slot rows with add/remove) — not inline editing on the results/admin page
- Editing never changes the participant or admin link tokens — they are immutable once created
- Slots keep their existing `position` order on edit; values can change in place, new slots are appended, existing slots can be removed — no drag-to-reorder UI
- The poll-type toggle is shown on the edit form but disabled/read-only (not hidden), so the admin can see what type the poll is

### Page Layout & Deletion Semantics
- Response deletion lives on the results/admin page (`GET /poll/{ptoken}/admin/{atoken}`), next to the grid where participants are already listed
- Whole-poll deletion lives in a danger-zone section on the separate edit page, grouped with other poll-level admin actions
- Deletion (of a response or a whole poll) is immediate and permanent — no soft-delete, no undo window
- After deleting a response, the results grid's tallies and best-slot highlight recompute naturally on next render — no special-case logic needed, this falls out of Phase 3's existing "always recompute from current DB state" design

### Claude's Discretion
- Exact styling of the danger-zone section and inline slot-removal warning, consistent with 01/02/03-UI-SPEC.md's established design tokens
- Internal handler/route naming and package structure for edit/delete endpoints
- Whether slot edits are applied as a full replace-the-slot-list operation or a diff (add/remove/update) internally, as long as the cascade-delete and position-preservation behaviors above hold

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/store` (Phases 1-3) — `polls`/`slots`/`participants`/`responses` tables; `ON DELETE CASCADE` on `responses.slot_id` already in place from Phase 2
- `internal/web` — `create.html`/`create.js` (Phase 1) as the direct pattern to reuse for the edit form's slot add/remove UI; `notfound.html` (Phase 2) for the post-deletion 404; `results.html` partial (Phase 3) already renders per-participant rows in the grid, where the "Remove" button for response deletion attaches
- Design tokens from `01/02/03-UI-SPEC.md` — spacing scale, tri-color tokens, destructive color (`#DC2626`) already reserved for destructive actions — this phase is the first to actually use it for real destructive controls (delete buttons)

### Established Patterns
- Server-rendered forms with inline field errors + banner-error summary (Phase 1's `create.html` pattern)
- POST → validate → either re-render with errors (200) or redirect (303) on success
- `handleCreateForm`/`handleCreatePoll` in `server.go` as the direct structural analog for the new edit-poll handlers

### Integration Points
- New routes: `GET/POST /poll/{ptoken}/admin/{atoken}/edit` (edit form), `POST /poll/{ptoken}/admin/{atoken}/responses/{participantID}/delete`, `POST /poll/{ptoken}/admin/{atoken}/delete` (whole poll)
- `internal/store` needs new methods: update poll (title/description), replace/update slot list (with cascade awareness), delete a single participant, delete a poll (cascading to slots/participants/responses)

</code_context>

<specifics>
## Specific Ideas

None beyond the reference-product framing already captured in PROJECT.md (Doodle/Rallly-style admin management).

</specifics>

<deferred>
## Deferred Ideas

- Soft-delete / undo window for deleted responses or polls — deferred, adds complexity (background cleanup) not requested
- Drag-and-drop slot reordering in the edit form — deferred, not requested
- A distinct "this poll was deleted" message (vs. reusing the generic invalid-link 404) — deferred, avoids extra tombstone state

</deferred>
