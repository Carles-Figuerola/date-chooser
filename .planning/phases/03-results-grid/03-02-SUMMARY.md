---
phase: 03-results-grid
plan: 02
subsystem: web
tags: [go, net-http, html-template, css, results-grid, admin-route]

# Dependency graph
requires:
  - phase: 03-results-grid (Plan 01)
    provides: "resultsGridView/buildResultsGridView/rankBestSlots view-model family, results.html shared partial, GET /poll/{ptoken} grid rendering"
provides:
  - "GET /poll/{ptoken}/admin/{atoken} (handleLinksPage) now renders the identical results grid below the share-links card, satisfying ADM-01 access parity"
  - "internal/web/static/style.css: .results-section/.results-scroll/.results-table CSS block — sticky slot-label column, tri-color .results-badge-yes/-no/-maybe, .results-row-best accent tint + border, .results-badge-best text badge — all reusing existing tokens"
affects: [04-admin-controls]

# Actuals (#2632)
actuals:
  tokens: 1979
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Admin route reuses the exact same buildResultsGridView/ResponsesByPollID chain as the participant route with zero duplicated logic, per 03-01-SUMMARY's stated readiness note"
    - "CSS class names were taken verbatim from the already-shipped results.html markup (e.g. results-badge-best, not the plan's own results-best-badge typo) rather than from the plan's prose, since the real DOM is the source of truth for selectors"

key-files:
  created:
    - .planning/phases/03-results-grid/deferred-items.md
  modified:
    - internal/web/server.go
    - internal/web/templates.go
    - internal/web/templates/links.html
    - internal/web/static/style.css
    - internal/web/server_test.go
    - .planning/WINDOWS.md

key-decisions:
  - "Used the CSS class names as they actually exist in results.html (results-badge-best, results-label-col, results-tally, results-missing) rather than the plan's prose names (results-best-badge) — the plan's own read_first instruction for Task 2 said to match the real markup, and the real markup is authoritative over the plan's paraphrase."
  - "The admin-token no-leak test scopes its assertion to the results-section markup substring, not the full response body — the admin token legitimately appears earlier in the page inside the pre-existing admin-link box (that's the whole point of the links page), so a whole-body assertion would be a false failure. T-03-02's actual concern (no admin-only data inside the grid) is what the test checks."

patterns-established:
  - "Grid CSS is fully additive/append-only at the bottom of style.css; no existing selector's declarations were changed, so Phase 1/2 visual regressions are not a risk from this plan."

requirements-completed: [ADM-01]

coverage:
  - id: D1
    description: "GET /poll/{ptoken}/admin/{atoken} renders the identical results grid (same data, same partial, same view-model) as the participant route, appended below the share-links card, with no admin-only data leaking into the grid"
    requirement: ADM-01
    verification:
      - kind: e2e
        ref: "internal/web/server_test.go#TestResults_AdminRouteParity"
        status: pass
      - kind: other
        ref: "grep -rn --include='*.go' 'template.HTML' internal/web/ (expected: no matches)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Results grid CSS: tri-color badges, sticky slot-label column with horizontal scroll, best-fit tint/border/badge, all reusing existing tokens (zero new hex/size/weight)"
    verification:
      - kind: other
        ref: "grep -Eq 'position:\\s*sticky' && grep -q results-badge-yes && grep -q results-row-best internal/web/static/style.css"
        status: pass
    human_judgment: false
  - id: D3
    description: "Browser-only render backstops: sticky column stays visible while scrolling at ~360px viewport; header-name ellipsis truncation triggers with the full name recoverable via the title attribute"
    verification: []
    human_judgment: true
    rationale: "CSS-render facts Go httptest cannot observe — recorded as WINDOWS.md unrun-verify entries #9/#10 for a later real-browser UI-review pass, per the plan's own explicit instruction and Phase 1/2 precedent."

duration: 15min
completed: 2026-08-25
status: complete
---

# Phase 3 Plan 2: Results Grid — Admin Route Parity & Visual Form Summary

**Admin route (`GET /poll/{ptoken}/admin/{atoken}`) now renders the exact same results grid as the participant route (ADM-01), and the grid gained its full CSS — tri-color badges, sticky slot-label column with horizontal scroll, and the best-fit accent-tint highlight — all built from Phase 1/2's existing color/spacing/type tokens with zero new values.**

## Performance

- **Duration:** ~15 min
- **Tasks:** 2
- **Files modified:** 6 (5 modified, 1 created)

## Accomplishments
- `handleLinksPage` (`internal/web/server.go`) now calls `s.store.ResponsesByPollID` + `buildResultsGridView` using the slots already returned by `PollByTokens` (no second slot query), and passes the resulting `resultsGridView` as `.Results` on the links page's view struct
- `templates.go` registers `templates/results.html` for the `links` page's `ParseFS` list; `links.html` invokes `{{template "results" .Results}}` immediately after the share-links card
- `TestResults_AdminRouteParity` proves the admin body carries the same participant names, the `✓` badge glyph, the exact tally line, and the `Best fit` badge as the participant route, and that the admin token does not leak into the results-grid markup itself (T-03-02)
- `internal/web/static/style.css` gained a full grid CSS block: 960px `.results-section` wrapper, `.results-scroll` horizontal-overflow container, `.results-table` with a `position: sticky; left: 0` slot-label column and `max-width: 140px` ellipsis-truncated participant headers, tri-color `.results-badge-yes/-no/-maybe` (28px min, 6px radius, 14px/600 white text), muted `—` missing-cell style, `.results-row-best` accent-tint (`#EFF6FF`) rows with a `border-left: 3px solid var(--color-accent)` on the sticky column, and `.results-badge-best` text badge — no existing selector's declarations were changed, and every hex value added (`#EFF6FF`) was already established in Phase 1/2
- Two browser-only render backstops (sticky column visibility at ~360px; header ellipsis truncation + title recoverability) recorded as WINDOWS.md entries #9 and #10 for a later real-browser UI-review pass

## Task Commits

Both tasks followed a straightforward implementation-then-verify flow (no TDD gate on this plan — `type: execute`, not `type: tdd`):

1. **Task 1: Admin-route results-grid parity (ADM-01)** - `b70bb81` (feat)
2. **Task 2: Results-grid CSS — badges, sticky column, best-fit highlight, responsive scroll** - `7fc1f29` (feat)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `internal/web/server.go` - `handleLinksPage` gains the `ResponsesByPollID`/`buildResultsGridView` chain and a `Results resultsGridView` field on its view struct
- `internal/web/templates.go` - `templates/results.html` added to the `links` page's `ParseFS` file list
- `internal/web/templates/links.html` - `{{template "results" .Results}}` invoked after the share-links card, inside `{{define "content"}}`
- `internal/web/static/style.css` - new results-grid CSS block (`.results-section`, `.results-scroll`, `.results-table`, `.results-label-col`, `.results-badge*`, `.results-row-best`, `.results-badge-best`, `.results-tally*`, `.results-missing`)
- `internal/web/server_test.go` - `TestResults_AdminRouteParity` (new)
- `.planning/WINDOWS.md` - two new `unrun-verify` entries (#9, #10) for the CSS render backstops
- `.planning/phases/03-results-grid/deferred-items.md` - new file logging an out-of-scope gap in the plan's own no-new-hex verify grep (see Deviations)

## Decisions Made
- CSS class names matched to the real markup already shipped in `results.html` (`results-badge-best`, not the plan prose's `results-best-badge`) rather than introducing a second, unused class name — see frontmatter `key-decisions`.
- The admin no-leak test scopes its token-absence assertion to the results-section substring of the body, not the whole page, because the admin token legitimately appears earlier in the pre-existing admin-link box — see frontmatter `key-decisions`.

## Deviations from Plan

### Auto-fixed Issues

None — no bugs found requiring a Rule 1/2/3 fix.

**1. [Scope boundary — logged, not fixed] Plan's no-new-hex verify gate is missing a pre-existing color from its allow-list**
- **Found during:** Task 2 verification
- **Issue:** The plan's `<verify>` command `grep -oE '#[0-9A-Fa-f]{6}' internal/web/static/style.css | grep -viE '#F8FAFC|#FFFFFF|#2563EB|#DC2626|#16A34A|#1E293B|#64748B|#E2E8F0|#EFF6FF' | ...` reports a non-zero result because `style.css` already contained `#FEF2F2` (`.banner-error`'s background) from Phase 1/2 — confirmed via `git log -p` that this predates this plan entirely. The gate's allow-list simply never included it.
- **Fix:** Not fixed — this is a pre-existing gap in the plan's own verify command, out of this task's scope per the executor's scope-boundary rule (only auto-fix issues the current task's changes directly caused). Logged in `.planning/phases/03-results-grid/deferred-items.md` with the full `sort -u` hex listing proving zero net-new hex values were introduced by this task; the only hex value this task added (`#EFF6FF`) was already on the intended allow-list and already used elsewhere in the file (`.banner-success`, `.link-box-secret`).
- **Files modified:** None (documentation only — `deferred-items.md`).
- **Verification:** Manually confirmed the file's full hex value set is exactly the 9 allow-listed values plus the one pre-existing `#FEF2F2`; this task's diff introduces zero new distinct hex values.

---

**Total deviations:** 1 (scope-boundary log, not a fix; no bug, no missing functionality, no architectural change)
**Impact on plan:** None on delivered behavior — the actual `must_haves.truths` intent ("no new hex values introduced") holds exactly; only the literal grep command in the plan's own verify block has a documented, pre-existing blind spot.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. The Spinnaker/tarmac upload guidance in the user's global CLAUDE.md does not apply (no Spinnaker pipelines exist for this project).

## Next Phase Readiness
- Phase 3 (Results Grid) is now fully complete: both the participant route (Plan 03-01) and the admin route (this plan) render the identical, fully-styled results grid, satisfying RES-01..04 and ADM-01.
- Two browser-only CSS/render backstops remain open in WINDOWS.md (#9 sticky column at 360px, #10 header ellipsis truncation) — carried forward for a later real-browser UI-review pass, consistent with Phase 1/2 precedent; they do not block this plan's completion.
- Phase 4 (admin controls — edit/delete) can build directly on `handleLinksPage`'s existing structure; no admin-only grid fields exist yet by design (CONTEXT.md's deferred admin-only visual distinction), so Phase 4 is free to add them without needing to touch Plan 03-01's shared view-model.
- No blockers.

---
*Phase: 03-results-grid*
*Completed: 2026-08-25*

## Self-Check: PASSED

All 8 created/modified files verified present on disk (`internal/web/server.go`, `internal/web/templates.go`, `internal/web/templates/links.html`, `internal/web/static/style.css`, `internal/web/server_test.go`, `.planning/WINDOWS.md`, `.planning/phases/03-results-grid/deferred-items.md`, `.planning/phases/03-results-grid/03-02-SUMMARY.md`); both task commits (`b70bb81`, `7fc1f29`) verified present in git log.
