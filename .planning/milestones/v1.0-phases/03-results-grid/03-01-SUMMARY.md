---
phase: 03-results-grid
plan: 01
subsystem: web
tags: [go, net-http, sqlite, html-template, aggregation, results-grid]

# Dependency graph
requires:
  - phase: 02-voting-end-to-end
    provides: "participants + responses tables, Participant struct, PollByParticipantToken, handleVoteForm/voteFormView, slotLabel(), per-page template composition pattern"
provides:
  - "internal/store/results.go: ResponsesByPollID(ctx, pollID) — participants ordered (created_at, id) plus a nested slot_id->answer map per participant"
  - "internal/web/server.go: resultsGridView/resultsParticipant/resultsRow/resultsCell/resultsComment view-model; buildResultsGridView (tallies + comments); rankBestSlots (most-Yes-desc/fewest-No-asc ranking, all-ties-flagged, empty at zero total responses)"
  - "internal/web/templates/results.html: shared \"results\" partial — grid table, tri-color Yes/No/Maybe badges, per-slot tallies, Best-fit badge, zero-response note, missing-cell dash, Comments section"
  - "GET /poll/{ptoken} now renders the results grid below the vote form"
affects: [03-02-admin-route-parity]

# Actuals (#2632)
actuals:
  tokens: 5863
  tasks: 2
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Store methods return raw rows only; tallying/ranking computed in Go (buildResultsGridView/rankBestSlots), matching 03-CONTEXT.md's explicit discretion to keep tally logic out of SQL"
    - "rankBestSlots gates on total tally sum across all rows (not per-row Yes count) to distinguish 'zero responses at all' from 'responses exist but nobody said Yes' — the latter still runs the literal most-Yes/fewest-No rule rather than being treated as an empty state"
    - "resultsGridView is intentionally route-agnostic (no admin-only fields) so Plan 03-02 can reuse buildResultsGridView verbatim for the admin route with zero changes"

key-files:
  created:
    - internal/store/results.go
    - internal/store/results_test.go
    - internal/web/templates/results.html
  modified:
    - internal/web/server.go
    - internal/web/templates.go
    - internal/web/templates/vote.html
    - internal/web/server_test.go

key-decisions:
  - "Populated resultsGridView.Comments inside buildResultsGridView as part of Task 1's implementation commit rather than deferring it to Task 2 as the plan's task split specified — the comment-collection loop was part of the same per-participant iteration already being written for Task 1's Participants/Rows construction, and splitting it out would have meant iterating participants twice for no benefit. Task 2's own Comments-view unit tests (TestResultsView_CommentsPopulated/CommentsEmpty) therefore passed immediately against Task 1's code with no new implementation needed in Task 2 — mirroring the exact pattern Phase 2 Plan 1 documented (\"Task 2's tests passed against the Task 1 implementation on the first run, with zero code changes needed\"). Task 2's own scope (the template's zero-response/missing-cell/Comments-section branches) still followed a genuine RED->GREEN cycle: three of its six new tests failed before the template was updated."
  - "Resolved the tracer feedback gate (Task 1, type=\"tracer\") as an autonomous continuation rather than stopping for a checkpoint:human-verify: the plan's own frontmatter declares autonomous: true, and the project's config.json declares \"mode\": \"yolo\" — the same signal Phase 2 Plan 1 used to justify the same call, even though workflow.auto_advance is false. Re-ran the full <verify> block (build, vet, targeted tests, XSS negative gate, partial-wired grep) after Task 1's commit and confirmed all green before proceeding directly to Task 2."
  - "rankBestSlots gates emptiness on the sum of Yes+No+Maybe across every row being zero, not on the maximum Yes count being zero. This means a poll where every participant answered but nobody chose Yes for any slot is NOT treated as a zero-response empty state — the literal 'most Yes desc, fewest No asc' rule still runs and can flag a row whose only distinguishing tally is fewest No. This reading matches 03-CONTEXT.md's stated rationale ('every slot ties at 0/0... best is meaningless with no data') literally: it only disables ranking when responses genuinely don't exist, not merely when no one said Yes."

patterns-established:
  - "Shared read-only aggregation view-model (resultsGridView) built once from store.ResponsesByPollID + store.Slot — Plan 03-02 (admin route) should call buildResultsGridView with the same inputs rather than introducing a second view-model or duplicating tally/ranking logic."

requirements-completed: [RES-01, RES-02, RES-03, RES-04]

coverage:
  - id: D1
    description: "GET /poll/{ptoken} renders a results grid below the vote form: one row per slot, one column per participant, each cell showing that participant's Yes/No/Maybe answer via a tri-color badge (✓/✗/?)"
    requirement: RES-01
    verification:
      - kind: e2e
        ref: "internal/web/server_test.go#TestResults_Grid_EndToEnd"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestResultsView_Tally"
        status: pass
    human_judgment: false
  - id: D2
    description: "Each slot row shows aggregate counts formatted '{yesCount} Yes · {noCount} No · {maybeCount} Maybe' across all participants"
    requirement: RES-02
    verification:
      - kind: e2e
        ref: "internal/web/server_test.go#TestResults_Grid_EndToEnd"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestResultsView_Tally"
        status: pass
    human_judgment: false
  - id: D3
    description: "Best-slot ranking: most Yes descending, tie-broken by fewest No ascending, Maybe never factors in, all ties flagged, no flag at zero responses"
    requirement: RES-03
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestRankBestSlots_ClearWinner"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestRankBestSlots_TieOnYes"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestRankBestSlots_TieBreakFewestNo"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestRankBestSlots_MaybeIrrelevant"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestRankBestSlots_ZeroResponses"
        status: pass
      - kind: e2e
        ref: "internal/web/server_test.go#TestResults_Grid_ZeroResponses_ShowsNote"
        status: pass
    human_judgment: false
  - id: D4
    description: "Comments render in a separate 'Comments' section, one line per participant with a non-empty comment formatted '{name}: {comment}', section omitted entirely when there are none; comment markup renders escaped"
    requirement: RES-04
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestResultsView_CommentsPopulated"
        status: pass
      - kind: unit
        ref: "internal/web/server_test.go#TestResultsView_CommentsEmpty"
        status: pass
      - kind: e2e
        ref: "internal/web/server_test.go#TestResults_Grid_CommentsSection_RendersAndEscapes"
        status: pass
      - kind: e2e
        ref: "internal/web/server_test.go#TestResults_Grid_NoComments_SectionOmitted"
        status: pass
    human_judgment: false
  - id: D5
    description: "A missing cell renders a muted '—' with no badge chrome; the grid markup is identical for 1 participant vs N participants; all participant-supplied text is rendered through auto-escaping only (no raw-HTML wrapper type anywhere in internal/web)"
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestResultsView_MissingCell"
        status: pass
      - kind: e2e
        ref: "internal/web/server_test.go#TestResults_Grid_MissingCell_ShowsDash"
        status: pass
      - kind: other
        ref: "grep -rn --include='*.go' 'template.HTML' internal/web/ (expected: no matches)"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-08-25
status: complete
---

# Phase 3 Plan 1: Results Grid — Participant Route Summary

**Read-only slots x participants results grid (Go/html-template): a new `ResponsesByPollID` aggregation query, a `buildResultsGridView`/`rankBestSlots` helper pair, and a shared `results.html` partial, wired into `GET /poll/{ptoken}` below the vote form and proven end-to-end with a real two-participant HTTP test.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 2
- **Files modified:** 7 (3 created, 4 modified)

## Accomplishments
- `internal/store/results.go`: `ResponsesByPollID` — one query for participants (`ORDER BY created_at, id`, stable even when timestamps collide thanks to the `id` tiebreaker) and one joined query shaping `responses` into a `participantID -> slotID -> answer` nested map
- `internal/web/server.go`: `resultsGridView` view-model family (`resultsParticipant`/`resultsRow`/`resultsCell`/`resultsComment`), `buildResultsGridView` (per-row Yes/No/Maybe tallies + comments collection), and `rankBestSlots` (most-Yes-desc/fewest-No-asc ranking with all-ties flagged and an empty result when the poll has had zero total responses)
- `internal/web/templates/results.html`: the shared `"results"` partial — grid table with a sticky-friendly label column, tri-color `✓`/`✗`/`?` badges, per-slot tallies, the `Best fit` badge, a `No responses yet.` zero-response note, a muted `—` missing-cell fallback, and a Comments section that is fully omitted when no participant left a non-empty comment
- `handleVoteForm` now calls `ResponsesByPollID` + `buildResultsGridView` and renders the grid via `voteFormView.Results`, wired into `vote.html`/`templates.go`
- Eleven new tests (five store/view-model unit tests for ranking, four unit/e2e tests for tallies/comments/missing-cell, and two full HTTP end-to-end tests) all pass; the XSS negative gate (`grep 'template.HTML'`) stays clean, confirming every participant-supplied value (display name, comment) is rendered exclusively through `html/template` auto-escaping

## Task Commits

Task 1 followed a genuine RED/GREEN cycle (`tdd="true"`, `type="tracer"`); Task 2 also followed RED/GREEN, though two of its view-model-level tests happened to already pass against Task 1's code (see Decisions Made):

1. **Task 1 (RED): add failing tests for results grid store query and view-model** - `ebe05b0` (test)
2. **Task 1 (GREEN): render results grid end-to-end on the participant route** - `8884355` (feat)
3. **Task 2 (RED): add failing tests for comments section, zero-response note, and missing-cell dash** - `41044b6` (test)
4. **Task 2 (GREEN): add comments section, zero-response note, and missing-cell dash** - `414a64c` (feat)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `internal/store/results.go` - `ResponsesByPollID(ctx, pollID)`, two parameterized queries shaped into `([]Participant, map[int64]map[int64]string)`
- `internal/store/results_test.go` - stable-ordering + answer-mapping store tests
- `internal/web/server.go` - `resultsGridView`/`resultsParticipant`/`resultsRow`/`resultsCell`/`resultsComment`; `buildResultsGridView`; `rankBestSlots`; `voteFormView.Results` field; `handleVoteForm` wiring
- `internal/web/templates/results.html` - the shared `"results"` partial (new file)
- `internal/web/templates/vote.html` - `{{template "results" .Results}}` invoked after the vote-form card, still inside `{{define "content"}}`
- `internal/web/templates.go` - `templates/results.html` added to the `vote` page's `ParseFS` file list
- `internal/web/server_test.go` - eleven new tests covering tallies, ranking edge cases, missing cells, zero responses, comments, and escaping

## Decisions Made
- Comments-field population landed in Task 1's commit instead of Task 2's, and the tracer feedback gate was resolved as an autonomous continuation rather than a checkpoint — both documented in detail in the frontmatter `key-decisions` above.
- `rankBestSlots` treats "zero responses" as "the sum of every tally across every row is zero," not "the best row's Yes count is zero" — see frontmatter `key-decisions` for the literal-reading rationale.

## Deviations from Plan

### Auto-fixed Issues

None — no bugs found requiring a Rule 1/2/3 fix.

**1. [Scope note, not a rule violation] Comments population moved from Task 2 to Task 1**
- **Found during:** Task 1 implementation
- **Issue:** The plan's task split placed `resultsGridView.Comments` population in Task 2's `<action>`, but Task 1's `buildResultsGridView` already needed to iterate every participant once to build `Participants`/`Rows`; adding the comment-collection branch to that same loop was strictly simpler than iterating participants twice across two tasks.
- **Fix:** Implemented Comments population in Task 1's GREEN commit. Task 2's Comments-view unit tests (`TestResultsView_CommentsPopulated`, `TestResultsView_CommentsEmpty`) were written for Task 2's RED step per the plan, and passed immediately against the already-present Task 1 code — no additional implementation was needed for them in Task 2.
- **Files modified:** `internal/web/server.go` (Task 1 commit `8884355`)
- **Verification:** All of Task 2's Comments-related tests pass; the template-level Comments-section rendering (heading + line format + omission) was still implemented and RED/GREEN-cycled properly in Task 2, since that part genuinely didn't exist until Task 2.

---

**Total deviations:** 1 (scope-ordering note only; no bug, no missing functionality, no architectural change)
**Impact on plan:** None on the delivered behavior — all `must_haves.truths` hold exactly as specified. This is a documentation-only deviation about which task's commit contains one small piece of logic.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. The Spinnaker/tarmac upload guidance in the user's global CLAUDE.md does not apply (no Spinnaker pipelines exist for this project).

## Next Phase Readiness
- `buildResultsGridView`/`rankBestSlots`/`resultsGridView` are fully route-agnostic (built from `poll.PollType`, `[]store.Slot`, `[]store.Participant`, and the `ResponsesByPollID` answers map) — Plan 03-02 can call the exact same helper from `handleLinksPage` (the admin route) to get pixel-identical grid parity with zero duplicated logic, exactly as `03-CONTEXT.md`'s "Page Placement & Access Parity" section requires.
- `results.html` is already registered as a template partial pattern (added to one page's `ParseFS` list); Plan 03-02 only needs to add it to the `links` page's `ParseFS` list the same way.
- The grid's visual CSS (sticky column, horizontal scroll, badge styling, tint/border on best-fit rows) is explicitly out of scope for this plan per the plan's own objective — Plan 03-02 owns that, plus the CSS backstop verifications called out in `03-UI-SPEC.md` (sticky-column-stays-visible-at-360px, header ellipsis-truncation-recoverable-via-title).
- No blockers.

---
*Phase: 03-results-grid*
*Completed: 2026-08-25*

## Self-Check: PASSED

All 7 created/modified files verified present on disk (`internal/store/results.go`, `internal/store/results_test.go`, `internal/web/server.go`, `internal/web/templates.go`, `internal/web/templates/results.html`, `internal/web/templates/vote.html`, `internal/web/server_test.go`); all 4 task commits (`ebe05b0`, `8884355`, `41044b6`, `414a64c`) verified present in git log.
