---
phase: 02-voting-end-to-end
plan: 02
subsystem: api
tags: [go, net-http, html-template, sqlite, form-validation]

# Dependency graph
requires:
  - phase: 02-voting-end-to-end (02-01)
    provides: "GET /poll/{ptoken} and POST /poll/{ptoken}/responses routes, voteFormView/voteSlotView, renderVoteForm, participant cookie handling, vote.html"
provides:
  - "Server-side validation in handleSubmitResponse: display name required (<=100 runes), comment <=500 runes, every slot answered with exactly yes/no/maybe — each failure re-renders vote.html at HTTP 200 with the exact UI-SPEC copy and all submitted values preserved, with zero DB writes"
  - "voteFormView.NameError/BannerError and voteSlotView.Error fields, plus matching inline/banner error placeholders in vote.html"
  - "renderNotFound helper and internal/web/templates/notfound.html: branded invalid-link 404 page (no form, no nav), wired into handleVoteForm and handleSubmitResponse's ErrNotFound branches"
affects: [02-03, phase-3-results-grid, phase-4-admin-management]

# Actuals (#2632)
actuals:
  tokens: 4100
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Validate-then-conditionally-write pattern extended from Phase 1's handleCreatePoll to handleSubmitResponse: build the full re-render view from submitted values first, run all checks without early-return, then either re-render at 200 (view has every submitted value, nothing written) or fall through to the existing SaveResponse/303 path"
    - "notfound.html follows the same per-page template composition as create/links/vote (layout.html + page content, parsed as its own *template.Template) rather than being folded into an existing page"

key-files:
  created:
    - internal/web/templates/notfound.html
  modified:
    - internal/web/server.go
    - internal/web/templates/vote.html
    - internal/web/templates.go
    - internal/web/server_test.go

key-decisions:
  - "Task 1 (tdd=\"true\") followed a genuine RED/GREEN split: the five validation tests were added and run first, confirming four failed for the expected reason (400/303 instead of a 200 re-render with the right copy) while the pre-existing e2e test and the plan's own regression-guard test (TestVote_AllValid_StillPersists) passed, before implementing the validation logic and vote.html placeholders in a separate GREEN commit."
  - "TestVote_AllValid_StillPersists passing during RED was expected and not a fail-fast trigger: the plan explicitly frames it as a guard against over-eager validation, not a new-behavior test — its pre-existing pass is the desired signal that the guard already holds."
  - "All three validation checks (name required, length caps, per-slot answer) are evaluated in a single non-early-returning pass so every applicable inline/row error is set before deciding whether to re-render, matching the create-poll form's existing all-at-once validation shape."
  - "notfound.html is rendered with a nil template data value since its content defines no fields; layout.html only references '.' by passing it through to {{template \"content\" .}}, which is safe with nil since content itself never dereferences a field."

patterns-established:
  - "Any future page needing a branded error/empty state should follow notfound.html's shape: a plain card, layout.html + its own content define, added as a new pageTemplates field — not folded into an existing template."

requirements-completed: [VOTE-01, VOTE-02, VOTE-03]

coverage:
  - id: D1
    description: "A blank display name is rejected with the inline error 'Your name is required.' plus the banner 'Check the highlighted fields and try again.', the form re-renders at HTTP 200 with submitted values preserved, and nothing is written to the database"
    requirement: VOTE-01
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestVote_BlankName_RejectedNoWrite"
        status: pass
    human_judgment: false
  - id: D2
    description: "A slot left with no answer selected is rejected with the inline row error 'Choose Yes, No, or Maybe for this slot.' surfaced on submit, other slots' answers preserved, and no DB write occurs"
    requirement: VOTE-01
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestVote_MissingSlotAnswer_RejectedNoWrite"
        status: pass
    human_judgment: false
  - id: D3
    description: "A display name longer than 100 characters or a comment longer than 500 characters is rejected server-side (defense beyond the native maxlength) with the banner copy and no DB write"
    requirement: VOTE-02
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestVote_NameTooLong_RejectedNoWrite"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestVote_CommentTooLong_RejectedNoWrite"
        status: pass
    human_judgment: false
  - id: D4
    description: "A fully valid submission is unaffected by the new validation: it still 303-redirects and persists exactly one participants row (regression guard against over-eager validation)"
    requirement: VOTE-02
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestVote_AllValid_StillPersists"
        status: pass
    human_judgment: false
  - id: D5
    description: "Opening a poll link whose participant token matches no poll returns HTTP 404 with the branded page copy 'This poll link isn't valid. Double-check the link you were given.' and no voting form"
    requirement: VOTE-03
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestNotFound_InvalidPollLink"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-08-25
status: complete
---

# Phase 2 Plan 2: Vote Validation & Invalid-Link 404 Summary

**Full server-side validation of the vote-submission form (blank name, missing slot answers, name/comment length caps) with exact UI-SPEC error copy and a 200 re-render preserving submitted values, plus a branded 404 page replacing the tracer's plain `http.NotFound` stub.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-25T10:58:00-05:00 (approx.)
- **Completed:** 2026-08-25T11:01:28-05:00
- **Tasks:** 2
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments
- `handleSubmitResponse` now validates display name (required, <=100 runes), comment (<=500 runes), and every slot's answer (must be exactly yes/no/maybe) BEFORE any DB write; any failure re-renders `vote.html` at HTTP 200 with all submitted values preserved and the exact UI-SPEC copy, and a fully valid submission is unaffected (still 303s and persists)
- `voteFormView` gained `NameError`/`BannerError`, `voteSlotView` gained `Error`; `vote.html` renders them inline under the name field, as a page-level banner (mirroring `create.html`'s `#form-banner`), and inline under the offending slot row
- New `internal/web/templates/notfound.html` — a plain branded 404 page (no form, no nav) with the exact invalid-link copy — parsed by `templates.go` the same per-page way as `create`/`links`/`vote`
- `renderNotFound` helper (sets HTTP 404, executes the notfound template) now backs both `handleVoteForm` and `handleSubmitResponse`'s `ErrNotFound` branches, replacing the tracer's bare `http.NotFound` stub (the separate 404 in `handleLinksPage` was left untouched — out of this plan's scope)
- Six new/extended tests, all passing: five vote-validation tests plus `TestNotFound_InvalidPollLink`
- Negative gate confirmed: no `template.HTML` or raw HTML interpolation anywhere in `internal/web` — re-rendered name/comment values round-trip through `html/template` auto-escaping (T-02-04)

## Task Commits

Task 1 followed the TDD RED/GREEN cycle (`tdd="true"`):

1. **Task 1 (RED): add failing tests for vote validation and 200 re-render** - `10f791e` (test)
2. **Task 1 (GREEN): validate vote submission and re-render at 200 with preserved values** - `14fd260` (feat)
3. **Task 2: render branded 404 page for invalid poll links** - `ecbe18d` (feat)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `internal/web/server.go` - `nameRequiredCopy`, `slotAnswerRequiredCopy`, `maxDisplayNameRunes`/`maxCommentRunes` constants; `voteFormView.NameError`/`BannerError`, `voteSlotView.Error`; validate-then-conditionally-write logic in `handleSubmitResponse`; `renderNotFound` helper; both `ErrNotFound` branches in `handleVoteForm`/`handleSubmitResponse` now call it
- `internal/web/templates/vote.html` - added `#form-banner`, inline `{{.NameError}}`, and per-slot `{{.Error}}` placeholders
- `internal/web/templates/notfound.html` - new branded 404 page (plain card, exact invalid-link copy, no form/nav)
- `internal/web/templates.go` - added `notfound` field to `pageTemplates`, parsed the same per-page way as `vote`
- `internal/web/server_test.go` - `TestVote_BlankName_RejectedNoWrite`, `TestVote_MissingSlotAnswer_RejectedNoWrite`, `TestVote_NameTooLong_RejectedNoWrite`, `TestVote_CommentTooLong_RejectedNoWrite`, `TestVote_AllValid_StillPersists`, `TestNotFound_InvalidPollLink`, plus a shared `createVotePollWithSlots` test helper

## Decisions Made
- Followed a genuine RED/GREEN split for Task 1's `tdd="true"` gate: added the five tests first, ran them, confirmed four failed for the expected reason (wrong status code / missing copy) while the pre-existing e2e test and the plan's own regression-guard test passed, then implemented the validation logic and template changes in a separate commit.
- `TestVote_AllValid_StillPersists` passing before any implementation change was expected, not a fail-fast trigger — the plan itself frames this test as a guard against over-eager validation rather than a new-behavior assertion.
- Evaluated all three validation checks (name required, length caps, per-slot answer) in one non-early-returning pass so every applicable error is populated on the view before deciding to re-render, matching `handleCreatePoll`'s existing all-at-once validation shape rather than short-circuiting on the first failure.
- Rendered `notfound.html` with `nil` template data since its "content" define references no fields — verified this executes without a runtime error.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' acceptance criteria were met on the first implementation attempt; no bugs were found requiring additional fix cycles.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. The Spinnaker/tarmac upload guidance in the user's global CLAUDE.md does not apply (no Spinnaker pipelines exist for this project).

## Next Phase Readiness
- Vote submission is now fully validated end-to-end: blank name, missing slot answers, and over-length name/comment are all rejected with the exact UI-SPEC copy, a 200 re-render, preserved values, and zero DB writes; a fully valid submission still persists and redirects.
- An unknown/invalid participant link now shows the branded 404 page instead of the tracer's plain stub, for both GET and POST.
- Plan 02-03 can now layer the pill-button-group styling and progressive-enhancement JS onto the existing native-radio `vote.html` without touching the validation logic or the 404 page established here.
- No blockers.

## Known Stubs

One plan-sanctioned interim state remains, explicitly scoped to the next plan in this phase:

| Stub | File | Reason / Resolving plan |
|------|------|--------------------------|
| Yes/No/Maybe controls render as native HTML radio inputs, not the 44px-tap-target segmented pill-button group from `02-UI-SPEC.md` | `internal/web/templates/vote.html` | Deferred to Plan 02-03 per 02-01's own scoping; unaffected by this plan's validation/error-copy changes |

---
*Phase: 02-voting-end-to-end*
*Completed: 2026-08-25*

## Self-Check: PASSED

All 6 created/modified files verified present on disk (`internal/web/server.go`, `internal/web/templates/vote.html`, `internal/web/templates/notfound.html`, `internal/web/templates.go`, `internal/web/server_test.go`, this SUMMARY.md); all 3 task commits (`10f791e`, `14fd260`, `ecbe18d`) verified present in git log.
