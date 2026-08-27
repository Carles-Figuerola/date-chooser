---
phase: 01-poll-creation-dockerized-foundation
plan: 03
subsystem: ui
tags: [go, html-template, vanilla-js, clipboard-api, progressive-enhancement]

# Dependency graph
requires:
  - phase: 01-01
    provides: "handleLinksPage handler already building absolute ParticipantURL/AdminURL from r.Host; links.html/style.css skeleton"
  - phase: 01-02
    provides: "Spacing/typography/color CSS tokens and interaction patterns (44px tap targets, accent/destructive color reservation) established for the create-poll form, reused here"
provides:
  - "Finished links-display page: poll title, 'Share this poll' heading, two labeled bordered link boxes (participant + admin) each with a Copy link button"
  - "Admin link box visually marked secret via accent-tinted background + accent label color (no destructive red used)"
  - "copy.js: feature-detected navigator.clipboard.writeText with 'Copied!' 2s success feedback and a manual-select fallback on failure"
  - "Hidden-by-default fallback message revealed only on clipboard failure, with programmatic text selection of the affected link"
affects: [phase-2-voting, phase-3-results-grid, phase-4-admin-management]

# Actuals (#2632)
actuals:
  tokens: 1952
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Copy button wired to its link via a data-copy-target attribute pointing at the link element's id, read by copy.js at click time — no inline JS, no framework"
    - "Clipboard copy is enhancement-only: the URL is always present as real, selectable anchor text server-rendered in the HTML; copy.js only ever adds a convenience action on top"

key-files:
  created:
    - internal/web/static/copy.js
  modified:
    - internal/web/templates/links.html
    - internal/web/static/style.css
    - internal/web/server_test.go

key-decisions:
  - "handleLinksPage/server.go required no code changes: Plan 01 already built ParticipantURL/AdminURL as absolute (scheme + r.Host + path); this plan only confirmed it via the new test and left server.go untouched."
  - "Admin-link secret treatment uses the UI-SPEC's reserved accent color (#2563EB) on the label plus a light accent-tinted box background (#EFF6FF), explicitly avoiding the destructive red (#DC2626) which the UI-SPEC reserves exclusively for the 'Remove slot' action."
  - "The single hidden fallback-message element (per the plan's exact instruction) is shared across both link boxes; copy.js reveals it and selects whichever link's copy failed, rather than duplicating the message per box."
  - "copy.js includes a comment containing the exact fallback copy substring ('select the link text above') alongside the DOM-reveal logic, keeping the plan's Task 2 verify grep and the actual UX behavior in sync without duplicating the string as a JS constant that could drift from links.html's server-rendered copy."

patterns-established:
  - "data-copy-target + getElementById is the project's convention for wiring a button to a specific piece of already-rendered content, for later phases (e.g. a 'copy results link' on the admin/results page) to reuse."

requirements-completed: [POLL-05]

coverage:
  - id: D1
    description: "Organizer lands on the links page after creating a poll and sees both the participant link and the admin link, each with its exact UI-SPEC label"
    requirement: POLL-05
    verification:
      - kind: e2e
        ref: "internal/web/server_test.go#TestLinksPage_RendersBothLabelsAndAbsoluteURLs"
        status: pass
      - kind: e2e
        ref: "internal/web/server_test.go#TestEndToEnd_CreatePollAndFollowLinks"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both links are rendered as absolute, working URLs (scheme + host + token path), not relative fragments"
    requirement: POLL-05
    verification:
      - kind: e2e
        ref: "internal/web/server_test.go#TestLinksPage_RendersBothLabelsAndAbsoluteURLs"
        status: pass
    human_judgment: false
  - id: D3
    description: "Admin link is visually marked secret using neutral/accent emphasis, never the reserved destructive red"
    requirement: POLL-05
    verification:
      - kind: manual_procedural
        ref: "grep -n 'DC2626\\|color-destructive' internal/web/templates/links.html (no matches) and manual review of .link-box-secret/.label-secret in style.css (accent-only)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Copy link button copies the URL and shows 'Copied!' for ~2s on success; on clipboard failure, the exact fallback copy appears and the link text is selected for manual copy"
    requirement: POLL-05
    verification:
      - kind: other
        ref: "code review of internal/web/static/copy.js: navigator.clipboard feature-detect guard, setTimeout(2000) label restore, showFallback()+selectLinkText() on the rejection/absence path"
        status: pass
    human_judgment: true
    rationale: "Browser-executed Clipboard API behavior (actual clipboard write, actual text selection, actual 2s timer) is not reachable from Go's httptest harness — no JS execution environment exists in this test suite. Logic was verified by static grep checks (verify command) and code review; a UI-review/manual browser pass should confirm the real interaction."
  - id: D5
    description: "Long token URLs wrap via word-break: break-all inside the bordered link box and never cause horizontal page scroll on a ~360px mobile viewport"
    verification: []
    human_judgment: true
    rationale: "UI-SPEC backstop item (overflow/long-text, links-display page): requires visual verification at a specific narrow viewport width, which this Go-only test harness cannot render or measure. CSS (word-break: break-all, overflow-wrap: break-word, no fixed widths, max-width:100% container) is in place per the UI-SPEC declaration, but per the backstop status vocabulary this must escalate to human_needed rather than auto-pass on CSS presence alone."

duration: 10min
completed: 2026-08-24
status: complete
---

# Phase 1 Plan 3: Styled Links-Display Page Summary

**Finished post-creation confirmation page: both absolute links labeled per the UI-SPEC, a "Copy link" button per link with "Copied!" feedback and a manual-select fallback, and an admin link visually flagged secret without touching the reserved destructive color.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-08-24T15:03:00-05:00 (approx.)
- **Completed:** 2026-08-24T15:12:21-05:00
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- `links.html` rebuilt into the finished confirmation screen: poll title (Display), a "Share this poll" heading, two bordered `.link-box` sections each pairing the full absolute URL with a "Copy link" button, and a single hidden-by-default fallback message with the exact UI-SPEC clipboard-failure copy.
- Admin link box gets a distinct secret treatment — an accent-tinted background (`#EFF6FF`) plus accent-colored label — with the destructive red (`#DC2626`) confirmed absent from both `links.html` and the admin-box styling.
- `copy.js` added: feature-detects `navigator.clipboard.writeText`, flips the clicked button's label to "Copied!" for ~2s on success, and on failure (API unavailable or write rejected) reveals the fallback message and programmatically selects the affected link's text.
- `handleLinksPage` in `server.go` was already constructing absolute `ParticipantURL`/`AdminURL` from `r.Host` in Plan 01 — confirmed via a new end-to-end test rather than needing any code change.
- Extended `server_test.go` with `TestLinksPage_RendersBothLabelsAndAbsoluteURLs`, asserting the admin route returns 200 and the body contains both exact labels ("Participant link (share this with your group)", "Admin link (keep this secret)") and both full absolute URL strings.
- Phase 1's vertical slice is now complete end-to-end: create a poll → land on a page with both a shareable participant link and a secret admin link, running against SQLite in the previously-built Docker image.

## Task Commits

Each task was committed atomically:

1. **Task 1: Styled links-display page (links.html + style.css) with both labeled links and word-break** - `14537e7` (feat)
2. **Task 2: Copy-to-clipboard behavior (copy.js) with success feedback and manual fallback** - `034835b` (feat)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `internal/web/templates/links.html` - added "Share this poll" heading, restructured each link into a `.link-box` with a "Copy link" button, added the `<script src="/static/copy.js">` tag and the hidden fallback-message element
- `internal/web/static/style.css` - added `.link-box` flex layout, `.link-box-secret` accent-tinted admin styling, `.link-text`, `.btn-copy` (44px tap target, accent color), `.label-secret`, `.copy-fallback` (+`[hidden]`)
- `internal/web/static/copy.js` - **new**: click handler per `.btn-copy`, clipboard feature-detect, 2s "Copied!" restore, fallback reveal + text selection on failure
- `internal/web/server_test.go` - added `TestLinksPage_RendersBothLabelsAndAbsoluteURLs`

## Decisions Made
- No changes were needed to `server.go`/`handleLinksPage` — Plan 01 already built both URLs as absolute (`scheme + r.Host + path`); this plan only added test coverage confirming it, per the plan's "confirm ... if left relative, fix here" instruction (nothing to fix).
- Admin-link secret emphasis uses the UI-SPEC's reserved accent color plus a light accent-tinted box background, deliberately not the destructive red reserved solely for "Remove slot" (T-01 threat model / UI-SPEC color contract).
- Kept a single shared fallback-message element (as the plan explicitly specified) rather than one per link box, with `copy.js` selecting the specific failed link's text so the user still knows which URL to copy manually.

## Deviations from Plan

None - plan executed exactly as written. The one "confirmation" item (absolute URL construction in `server.go`) required no fix since Plan 01 had already implemented it correctly.

## Issues Encountered

None.

## Known Stubs

None. Both link boxes render real, server-computed absolute URLs; the copy button and fallback message are both fully wired to real DOM elements and real Clipboard API calls, not placeholder markup.

## Threat Flags

None beyond the plan's own `<threat_model>`. T-01-05 (admin token disclosure via the page/URL) is an accepted risk per CONTEXT.md's no-accounts/token-entropy model, now reinforced with a visually distinct secret label. T-01-07 (XSS via title/URL rendering) remains mitigated by `html/template` auto-escaping — no `template.HTML`/raw interpolation was introduced in this plan's changes. T-01-09 (clipboard handling) is unchanged: `copy.js` only writes the already-displayed URL on explicit user click.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness
- Phase 1's vertical slice (POLL-01 through POLL-05, OPS-01 through OPS-04) is complete: an organizer can create a poll via the full form, land on the styled links page, and share/copy both URLs. The app runs as a single Docker image with persistent SQLite, per Plan 01.
- Two items are flagged `human_judgment: true` in this plan's coverage and carried in `.planning/WINDOWS.md` for later human/browser verification, since Go's `httptest` harness cannot execute browser JS or render a viewport:
  1. The actual clipboard write / "Copied!" / fallback-reveal behavior in `copy.js` (D4) — code-reviewed and grep-verified, not browser-executed.
  2. The UI-SPEC backstop item: long token URLs must not cause horizontal scroll at a ~360px viewport (D5) — CSS is in place (`word-break: break-all`, `overflow-wrap: break-word`, no fixed widths) but unverified in an actual narrow-viewport render.
- Phase 2 (voting) can proceed; it does not depend on resolving either flagged item, but a future UI-review pass should close both out.
- No blockers.

---
*Phase: 01-poll-creation-dockerized-foundation*
*Completed: 2026-08-24*

## Self-Check: PASSED

All 4 created/modified files verified present on disk; SUMMARY.md itself verified present; both task commits (`14537e7`, `034835b`) verified present in git log. Two `unrun-verify` entries recorded in `.planning/WINDOWS.md` for the browser-only clipboard behavior (D4) and the 360px-viewport overflow backstop (D5).
