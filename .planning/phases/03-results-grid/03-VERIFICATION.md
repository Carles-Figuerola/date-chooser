---
phase: 03-results-grid
verified: 2026-08-25T19:15:00Z
status: passed
score: 11/11 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification_note: >
  Both browser-only items manually confirmed by the user on 2026-08-25 against
  a local `go run ./cmd/server` instance with a multi-participant test poll:
  (1) sticky slot-label column correctly stays pinned while the grid scrolls
  horizontally at ~360px; (2) a long participant name ("Alexandria Katherine
  Montgomery-Whitfield") correctly truncates with an ellipsis in the column
  header. Note: two duplicate "Alexandria..." columns appeared during testing
  because each curl-based test submission was a separate request with no
  shared cookie — confirmed via sqlite (distinct cookie_token per row) to be
  the intended anti-overwrite-by-name-collision behavior from Phase 2's
  CONTEXT.md, not a defect. The truncation check ran successfully on both
  duplicate columns.
behavior_unverified_items:
  - truth: "On a narrow (~360px) viewport the grid scrolls horizontally with the slot-label column pinned via position:sticky/left:0 so it stays visible while the participant columns scroll"
    test: "Open GET /poll/{ptoken} (or the admin route) in a real browser at a ~360px viewport with 3+ participants; scroll the grid horizontally"
    expected: "The slot-label column (with the tally/Best-fit text) remains visible and pinned to the left edge while participant columns scroll underneath/past it; no transparent seam shows through"
    why_human: "CSS `position: sticky` layout behavior during scroll is a browser paint/layout fact — Go httptest never renders CSS or executes scroll. The rule's presence (`internal/web/static/style.css:517` `position: sticky; left: 0;`) is confirmed by grep, but that only proves the declaration exists, not that it holds visually. Already tracked as WINDOWS.md ledger entry #9 (open, unrun-verify)."
  - truth: "A participant display name longer than the column-header width truncates with an ellipsis at max-width 140px and the full name remains recoverable via the native title attribute"
    test: "Submit a response with a display name longer than ~20-25 characters; open the results grid in a real browser and observe the column header"
    expected: "The long name is visually truncated with an ellipsis inside the 140px-wide header cell; hovering/long-pressing the truncated header reveals the full name via the native `title` tooltip"
    why_human: "CSS text-overflow/ellipsis rendering and native browser tooltip behavior are facts a layout/paint engine produces — Go httptest cannot render CSS truncation or trigger a hover tooltip. The `title` attribute and the `max-width:140px; overflow:hidden; text-overflow:ellipsis` rule are both present in code (`internal/web/templates/results.html:10`, `internal/web/static/style.css:498-507`), but presence is not the same as verified visual truncation. Already tracked as WINDOWS.md ledger entry #10 (open, unrun-verify)."
---

# Phase 3: Results Grid Verification Report

**Phase Goal:** Anyone holding the participant or admin link can see, at a glance, who is available for which slot and which slot(s) work best for the group.
**Verified:** 2026-08-25T19:15:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

**Note on MVP-mode goal format:** ROADMAP.md's `**Goal:**` line for Phase 3 is a narrative paraphrase, not literally in `As a ... I want to ... so that ...` form (`gsd-tools query user-story.validate` on that exact string returns `valid: false`). However, PLAN 03-01's `<objective>` derives and states a valid User Story ("As a person holding a Date Chooser poll link, I want to see, below the voting form, a grid of who answered Yes/No/Maybe for each slot with per-slot tallies and the best slot(s) highlighted, plus everyone's comments, so that I can tell at a glance which slot works best for the group." — confirmed `valid: true`), which was the operative goal used throughout planning and execution. This report follows the same precedent set by `02-VERIFICATION.md` and proceeds with goal-backward verification against the ROADMAP Success Criteria plus the PLAN-derived story, rather than refusing on a technicality.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | RES-01: `GET /poll/{ptoken}` renders a results grid — one row per slot, one column per participant, each cell showing Yes/No/Maybe as a tri-color badge | ✓ VERIFIED | `internal/web/server.go:555-606` (`handleVoteForm` calls `ResponsesByPollID`→`buildResultsGridView`→`view.Results`); `internal/web/templates/results.html:5-39` (table markup); `TestResults_Grid_EndToEnd` passes (names, ✓ glyph present) |
| 2 | RES-02: Each slot row shows aggregate counts formatted `"{yes} Yes · {no} No · {maybe} Maybe"` | ✓ VERIFIED | `internal/web/templates/results.html:21`; `TestResults_Grid_EndToEnd` asserts exact string `"2 Yes · 0 No · 0 Maybe"`; `TestResultsView_Tally` unit test |
| 3 | RES-03: Best slot(s) ranked most-Yes-desc/fewest-No-asc, Maybe excluded, all ties flagged, no flag at zero responses | ✓ VERIFIED | `rankBestSlots` (`internal/web/server.go:493-519`) traced by hand against the tie-break case (A=2yes/1no, B=2yes/0no → only B); `TestRankBestSlots_ClearWinner/TieOnYes/TieBreakFewestNo/MaybeIrrelevant/ZeroResponses` all pass |
| 4 | RES-04: Comments render in a separate "Comments" section, one line per non-empty comment as `"{name}: {comment}"`, section omitted entirely when none exist | ✓ VERIFIED | `internal/web/templates/results.html:43-50`; `TestResults_Grid_CommentsSection_RendersAndEscapes` and `TestResults_Grid_NoComments_SectionOmitted` pass |
| 5 | ADM-01: `GET /poll/{ptoken}/admin/{atoken}` renders the identical grid (same partial, same view-model) below the share-links card, no admin-only leak | ✓ VERIFIED | `internal/web/server.go:325-371` (`handleLinksPage` calls the same `ResponsesByPollID`/`buildResultsGridView` chain); `internal/web/templates/links.html:26`; `TestResults_AdminRouteParity` passes including the admin-token-not-in-grid-markup assertion |
| 6 | A missing cell renders a muted "—" with no badge chrome, not a blank cell | ✓ VERIFIED | `internal/web/templates/results.html:32`; `TestResults_Grid_MissingCell_ShowsDash`, `TestResultsView_MissingCell` pass |
| 7 | Grid markup is structurally identical for 1 participant vs. N participants (no branching on count beyond the zero-response empty state) | ✓ VERIFIED | `buildResultsGridView`/`results.html` use a plain `{{range .Participants}}`/`{{range .Cells}}` loop with no count-based branch; code inspection confirms no special-casing |
| 8 | Participant-supplied display names and comments are rendered exclusively via `html/template` auto-escaping — no raw-HTML wrapper anywhere in `internal/web` (T-03-01) | ✓ VERIFIED | `grep -rn --include='*.go' 'template.HTML' internal/web/` returns zero matches; `TestResults_Grid_CommentsSection_RendersAndEscapes` confirms `<b>x</b>` renders as literal `&lt;b&gt;x&lt;/b&gt;` |
| 9 | Grid CSS (badges, best-fit tint) reuses only existing Phase 1/2 color tokens — zero new hex values | ✓ VERIFIED | `grep -oE '#[0-9A-Fa-f]{6}' internal/web/static/style.css \| sort -u` → exactly the 9 established tokens + pre-existing `#FEF2F2` (confirmed via `git log -p` to predate this phase, added in commit `a89c782`, Phase 2); this task added zero net-new hex values |
| 10 | 🧪 backstop: sticky slot-label column stays visible while participant columns scroll at ~360px | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS rule present (`style.css:516-522`) but no browser renders it in this harness — see Human Verification below and WINDOWS.md #9 |
| 11 | 🧪 backstop: participant header name truncates with ellipsis at 140px, full name recoverable via `title` | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS + `title` attribute present (`results.html:10`, `style.css:498-507`) but no browser renders/hover-tests it in this harness — see Human Verification below and WINDOWS.md #10 |

**Score:** 9/11 truths verified (2 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/store/results.go` | `ResponsesByPollID(ctx, pollID)` — participants ordered + nested answer map | ✓ VERIFIED | Exists, substantive (real parameterized SQL, two queries), wired into both handlers |
| `internal/store/results_test.go` | Store ordering + answer-mapping tests | ✓ VERIFIED | 2 tests, both pass, real assertions on ordering and per-participant answer maps |
| `internal/web/templates/results.html` | Shared `"results"` partial: grid, tallies, badges, best-fit, zero-state, comments | ✓ VERIFIED | All branches present and match UI-SPEC copy/structure |
| `buildResultsGridView` + `rankBestSlots` (`internal/web/server.go`) | View-model + ranking helpers | ✓ VERIFIED | Present, substantive, logic hand-traced against tie-break case and confirmed via 5 unit tests |
| `internal/web/static/style.css` results-grid block | Sticky column, tri-color badges, best-fit tint/border | ✓ VERIFIED | All selectors present (`results-label-col`, `results-badge-*`, `results-row-best`), reusing only `var(--color-*)` tokens or the pre-approved `#EFF6FF` literal |
| `internal/web/templates/links.html` results invocation | `{{template "results" .Results}}` on admin route | ✓ VERIFIED | Present at line 26, inside `{{define "content"}}`, after the share-links card |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `handleVoteForm` | `store.ResponsesByPollID` → `buildResultsGridView` → `results.html` | `voteFormView.Results` field, `{{template "results" .Results}}` in `vote.html:52` | ✓ WIRED | Confirmed by code read + `TestResults_Grid_EndToEnd` |
| `handleLinksPage` | `store.ResponsesByPollID` → `buildResultsGridView` → `results.html` | anonymous struct `.Results` field, `{{template "results" .Results}}` in `links.html:26` | ✓ WIRED | Confirmed by code read + `TestResults_AdminRouteParity` |
| `templates.go` `vote`/`links` `ParseFS` lists | `templates/results.html` | file-list registration | ✓ WIRED | Both lines 34 and 38 include `templates/results.html` |
| `results.html` cell `Answer` values | badge glyph CSS classes | `{{if eq .Answer "yes"}}` etc. → `results-badge-yes/-no/-maybe` / `results-missing` | ✓ WIRED | 1:1 mapping confirmed in template; classes match CSS selectors present in `style.css` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `resultsGridView.Rows[].Cells[].Answer` | `answers[p.ID][sl.ID]` | `SELECT ... FROM responses r JOIN participants p ...` (`results.go:39-44`) | Yes — real SQLite query, not a mock/static return | ✓ FLOWING |
| `resultsGridView.Participants[].DisplayName` | `p.DisplayName` | `SELECT ... FROM participants WHERE poll_id = ?` | Yes | ✓ FLOWING |
| `resultsRow.BestFit` | `rankBestSlots(view.Rows)` | Computed from real tallies, not hardcoded | Yes | ✓ FLOWING |
| `resultsGridView.Comments` | non-empty `p.Comment` per participant | Same participants query | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full package build | `go build ./...` | exit 0 | ✓ PASS |
| Static analysis | `go vet ./...` | exit 0, no findings | ✓ PASS |
| Targeted results/links tests | `go test ./internal/store/ ./internal/web/ -run 'Results\|Links' -v` | 14 tests, all PASS | ✓ PASS |
| Ranking unit tests | `go test ./internal/web/ -run 'TestRankBestSlots' -v` | 5 tests, all PASS | ✓ PASS |
| Full workspace test suite (run once) | `go test ./... -count=1` | all packages `ok` | ✓ PASS |
| XSS negative gate | `grep -rn --include='*.go' 'template.HTML' internal/web/` | no matches | ✓ PASS |
| No-new-hex gate (corrected allow-list) | `grep -oE '#[0-9A-Fa-f]{6}' internal/web/static/style.css \| sort -u` | 10 values = 9 allow-listed + pre-existing `#FEF2F2` | ✓ PASS |

### Probe Execution

Not applicable — this phase has no `scripts/*/tests/probe-*.sh` conventional probes and none are declared in the PLAN/SUMMARY files. Step 7c: SKIPPED (no probes declared or found).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|--------------|--------|----------|
| RES-01 | 03-01 | Results grid: slots × participants, per-cell Yes/No/Maybe | ✓ SATISFIED | `handleVoteForm` wiring + `TestResults_Grid_EndToEnd` |
| RES-02 | 03-01 | Aggregate Yes/No/Maybe counts per slot | ✓ SATISFIED | Exact-format tally assertions in tests |
| RES-03 | 03-01 | Best-availability slot(s) highlighted | ✓ SATISFIED | `rankBestSlots` + 5 passing ranking unit tests |
| RES-04 | 03-01 | Comments visible alongside responses | ✓ SATISFIED | Comments section tests (populated/empty/escaped) |
| ADM-01 | 03-02 | Admin link shows the same results grid | ✓ SATISFIED | `TestResults_AdminRouteParity` |

No orphaned requirements: `REQUIREMENTS.md`'s Phase 3 mapping (RES-01..04, ADM-01) exactly matches the union of `requirements:` fields declared across 03-01-PLAN.md and 03-02-PLAN.md — every ID is accounted for, none left unclaimed.

**Documentation note (non-blocking):** `.planning/REQUIREMENTS.md` still shows RES-01..04 and ADM-01 as unchecked `[ ]` / "Pending" in its traceability table, even though the functional requirement is satisfied in the codebase (verified above). This appears to be a housekeeping step normally performed at ship/milestone-completion time (Phase 1/2's POLL-*/VOTE-* rows were flipped to `[x]`/"Complete" only after their respective ship flow ran) rather than a code gap. Flagged for the record, not treated as a verification gap.

### Anti-Patterns Found

None. Scanned all phase-modified files (`internal/store/results.go`, `internal/store/results_test.go`, `internal/web/server.go`, `internal/web/templates/results.html`, `internal/web/templates/vote.html`, `internal/web/templates/links.html`, `internal/web/templates.go`, `internal/web/static/style.css`, `internal/web/server_test.go`) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER|coming soon|not yet implemented`. The only matches were legitimate HTML `placeholder="..."` form attributes pre-existing from Phase 2's `vote.html` — not stub markers.

### Human Verification Required

### 1. Sticky slot-label column at narrow viewport

**Test:** Open `GET /poll/{ptoken}` (or the admin route) with 3+ participants in a real browser at a ~360px viewport width; scroll the grid horizontally.
**Expected:** The slot-label column (name, tally, Best-fit badge) stays pinned to the left edge and remains fully visible while participant columns scroll underneath/past it, with no visible transparent seam.
**Why human:** `position: sticky` is a real-browser layout/paint behavior that Go's `httptest` cannot render or scroll. Already tracked as WINDOWS.md ledger entry #9 (open).

### 2. Participant header name truncation + title recoverability

**Test:** Submit a response with a long display name (20-25+ characters); view the results grid column header in a real browser.
**Expected:** The name visually truncates with an ellipsis inside the 140px header cell; hovering (desktop) or long-pressing (mobile) reveals the full name via the native `title` tooltip.
**Why human:** CSS `text-overflow: ellipsis` rendering and native tooltip behavior require a real browser paint/interaction engine. Already tracked as WINDOWS.md ledger entry #10 (open).

### Gaps Summary

No gaps found. All 5 ROADMAP Success Criteria and all 5 phase requirements (RES-01..04, ADM-01) are satisfied in the codebase with passing automated tests (`go build`, `go vet`, and the full `go test ./... -count=1` all green). The only open items are the two CSS/browser-rendering backstops explicitly declared as `verification: backstop` in PLAN 03-02's frontmatter — these were never claimed to be automatable and are already correctly logged in `.planning/WINDOWS.md` (#9, #10) awaiting a real-browser UI-review pass, consistent with the same pattern used and accepted in Phase 1 and Phase 2.

---

*Verified: 2026-08-25T19:15:00Z*
*Verifier: Claude (gsd-verifier)*
