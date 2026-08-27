---
phase: 02-voting-end-to-end
plan: 03
subsystem: ui
tags: [css, html-template, progressive-enhancement, vanilla-js]

# Dependency graph
requires:
  - phase: 02-voting-end-to-end (02-01, 02-02)
    provides: "Functional and validated vote.html (native radios, maxlength enforcement, banner-success/banner-error placeholders, NameError/BannerError/per-slot Error fields) with zero server-contract surface for this plan to touch"
provides:
  - "vote.html: pill markup with the visually-hidden radio + visible span structure, driven entirely by CSS :checked (no JS dependency); vote.js script tag (defer)"
  - "style.css: --color-yes token; .pill-group/.pill with tri-color :checked backgrounds (green/red/gray), 44px tap targets, accent focus-visible ring; .banner-success rule"
  - "vote.js: closure-flag submit guard (disable + 'Saving…', blocks a genuine second POST), no pill-selection logic"
affects: [phase-3-results-grid, phase-4-admin-management]

# Actuals (#2632)
actuals:
  tokens: 2057
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pill-group selected state driven purely by the native radio's :checked pseudo-class applied to a sibling <span> (input:checked ~ span), matching Phase 1's .toggle-option precedent — no JS needed for the visual selection state, only for the submit guard"

key-files:
  created:
    - internal/web/static/vote.js
  modified:
    - internal/web/templates/vote.html
    - internal/web/static/style.css

key-decisions:
  - "Removed the pre-existing server-rendered `pill-active` class from vote.html's pill markup. It was set from voteSlotView.Selected on the initial GET render only, with nothing to keep it in sync on subsequent client-side clicks — replacing it with a pure :checked-based CSS selector makes the selected-pill color correct both on first load (prefill) and immediately after any click, with no JS involved."
  - "Applied the tri-color background to the pill's inner <span> (not the outer <label>) so the color fills the full 44px tap target while keeping the radio itself visually hidden the same way .toggle-option already does — avoids introducing a CSS :has() dependency."
  - "Used :focus-visible (not :focus) for the pill's keyboard focus ring per the plan's explicit 'for keyboard users' framing, consistent with modern browser support."

patterns-established:
  - "Any future segmented-pill or multi-state selector control should follow this file's shape: visually-hidden native input + sibling <span> styled via :checked/:focus-visible, never a JS-toggled class, so the control keeps working with JS disabled."

requirements-completed: [VOTE-03, VOTE-04, VOTE-05, VOTE-06]

coverage:
  - id: D1
    description: "Each Yes/No/Maybe control renders as a segmented pill group with a minimum 44px tap target; the selected pill shows green (#16A34A) for Yes, red (#DC2626) for No, and gray (#64748B) for Maybe, driven purely by CSS :checked with no JS dependency"
    requirement: VOTE-06
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestVote_EndToEnd_SubmitAndRevisit (pill markup renders/round-trips; color/tap-target rendering itself is CSS-only and out of Go httptest's reach)"
        status: pass
    human_judgment: true
    rationale: "Pill color, 44px tap target, and no-360px-overflow are CSS rendering facts that Go httptest cannot observe (no browser layout engine). Recorded as WINDOWS.md entry #7 (unrun-verify) for a later browser-based UI-review pass."
  - id: D2
    description: "The display-name input enforces maxlength 100 and the comment textarea enforces maxlength 500 natively"
    requirement: VOTE-06
    verification:
      - kind: unit
        ref: "internal/web/server_test.go (grep-equivalent: TestVote_NameTooLong_RejectedNoWrite / TestVote_CommentTooLong_RejectedNoWrite exercise the server-side backstop; native maxlength attributes verified via grep in <verify>)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The slot list grows with the page with no fixed-height scroll container, and the voting form is usable at a ~360px mobile width and a wide desktop width"
    requirement: VOTE-06
    verification: []
    human_judgment: true
    rationale: "Responsive layout at a real 360px viewport is a rendering fact unreachable from Go httptest. Recorded as WINDOWS.md entry #7 (unrun-verify)."
  - id: D4
    description: "After a successful submit the page shows the confirmation banner 'Thanks! Your response has been saved.' above the still-editable form"
    requirement: VOTE-05
    verification:
      - kind: unit
        ref: "internal/web/server_test.go#TestVote_EndToEnd_SubmitAndRevisit"
        status: pass
    human_judgment: false
  - id: D5
    description: "On submit the Save button disables and its label changes to 'Saving…', blocking a second POST (not merely a visual change)"
    requirement: VOTE-03
    verification: []
    human_judgment: true
    rationale: "Browser-executed JS submit-guard behavior (and the JS-disabled fallback submit) cannot be exercised by Go httptest, which never runs client-side script. Recorded as WINDOWS.md entry #8 (unrun-verify), matching the Phase 1 create.js precedent."

duration: 5min
completed: 2026-08-25
status: complete
---

# Phase 2 Plan 3: Voting UI Polish (Pill Groups, Banner, Submit Guard) Summary

**Tri-color CSS `:checked`-driven Yes/No/Maybe pill groups, a `.banner-success` confirmation banner, and a closure-flag `vote.js` submit guard — all layered onto the already-functional, already-validated `vote.html` with zero server-contract changes.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-08-25T11:03:22-05:00 (approx., from prior plan's completion commit)
- **Completed:** 2026-08-25T11:07:16-05:00
- **Tasks:** 2
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments
- `vote.html`'s pill markup now derives its selected-state color purely from the native radio's `:checked` pseudo-class (no JS, no server-rendered `pill-active` class to keep in sync) — the same visually-hidden-input pattern Phase 1's `.toggle-group` already established
- `style.css` gained the one net-new hex token (`--color-yes: #16A34A`), `.pill-group`/`.pill` rules with 44px tap targets and an accent `:focus-visible` ring, and reused `--color-destructive`/`--color-text-muted` for No/Maybe rather than inventing new colors
- `.banner-success` rule added, reusing the existing `#EFF6FF` + accent-border combination already established for Phase 1's secret-link box — no new hex introduced for the banner
- `vote.js` created: a closure-flag submit guard mirroring `create.js` exactly (disable button + "Saving…" label on first submit, `preventDefault()` no-op on a second submit while already submitting), with zero pill-selection logic — the pill control remains fully functional with this script absent
- Confirmed the T-02-04 negative gate still holds (`grep -rn 'template.HTML' internal/web/` finds nothing) — this plan introduced no raw/unescaped HTML interpolation
- Updated the broken-windows ledger: marked the pre-existing native-radio stub (entry #6) fixed, and recorded two new `unrun-verify` entries (#7 CSS/responsive render, #8 JS submit guard + no-JS fallback) for the browser-only checks this plan cannot exercise via Go httptest

## Task Commits

1. **Task 1: Full voting UI — pill groups, confirmation banner, responsive layout** - `a89c782` (feat)
2. **Task 2: vote.js submit guard (disable + "Saving…", blocks a second POST)** - `38dd05b` (feat)
3. **Ledger update: mark native-radio stub fixed, record two unrun-verify entries** - `85af0ea` (docs)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `internal/web/templates/vote.html` - removed the server-rendered `pill-active` class (superseded by pure-CSS `:checked` styling); added `aria-label` to each pill group; added the `vote.js` `<script defer>` tag
- `internal/web/static/style.css` - `--color-yes` token; `.banner-success`; `.pill-group`/`.pill`/`.pill input`/`.pill span` rules including the three `:checked ~ span` tri-color rules and the `:focus-visible` ring
- `internal/web/static/vote.js` - new file: closure-flag submit-guard IIFE targeting `#vote-form`/`#submit-btn`

## Decisions Made
- Discovered during `<read_first>` that 02-01/02-02 had already delivered nearly all of vote.html's UI-SPEC markup (labels, placeholders, maxlengths, the "Save my response" CTA, and even pill-wrapper markup with `pill`/`pill-yes`/`pill-no`/`pill-maybe` classes) — this plan's actual delta was narrower than the plan text implied: swap the server-rendered `pill-active` class for pure `:checked`-driven CSS, add the missing tri-color/tap-target/focus-ring/banner-success CSS (none of which existed in `style.css` yet), and add the `vote.js` script tag + file. No plan deviation — the plan's `<read_first>` explicitly anticipated this by pointing at the existing vote.html/style.css as the base to extend.
- Chose to style the pill's inner `<span>` (not the outer `<label>`) as the colored surface, avoiding a `:has()` dependency while still filling the entire 44px tap target with color.
- Used `:focus-visible` rather than `:focus` for the pill's keyboard focus ring, matching the plan's explicit "for keyboard users" framing.

## Deviations from Plan

None — plan executed exactly as written. The narrower-than-expected markup delta (see Decisions Made) was a discovery within the plan's own scoped `<action>`, not an unplanned addition.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. The Spinnaker/tarmac upload guidance in the user's global CLAUDE.md does not apply (no Spinnaker pipelines exist for this project).

## Next Phase Readiness
- Phase 2 (Voting End-to-End) is now feature-complete: participants can open a link with no login, submit Yes/No/Maybe with an optional comment through the finished UI-SPEC pill/banner/responsive surface, and revisit/edit later — all proven by `go build`/`go test` and the existing e2e/validation test suite, with no server-contract changes across any of the three plans in this phase.
- Two browser-only visual/behavioral checks remain open in `.planning/WINDOWS.md` (#7 pill color/tap-target/360px-overflow render, #8 JS submit-guard + no-JS fallback) for a later UI-review pass — these are the same category of unrun-verify items Phase 1 left open for `create.js`/`copy.js`, not new risk.
- Phase 3 (results grid) can build on the now-finished `participants`/`responses` schema and voting flow without further changes to this phase's files.
- No blockers.

## Known Stubs

None — the one known stub carried into this plan (native-radio rendering instead of the pill-button group, tracked as WINDOWS.md entry #6) is resolved by this plan and marked fixed.

---
*Phase: 02-voting-end-to-end*
*Completed: 2026-08-25*

## Self-Check: PASSED

All 3 created/modified files verified present on disk (`internal/web/templates/vote.html`, `internal/web/static/style.css`, `internal/web/static/vote.js`, this SUMMARY.md); all 3 task/ledger commits (`a89c782`, `38dd05b`, `85af0ea`) verified present in git log.
