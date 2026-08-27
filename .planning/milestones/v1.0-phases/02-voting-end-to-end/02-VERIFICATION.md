---
phase: 02-voting-end-to-end
verified: 2026-08-25T16:30:00Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification_note: >
  Both browser-only items manually confirmed by the user on 2026-08-25 against
  a local `go run ./cmd/server` instance: tri-color pills (green/red/gray),
  44px tap targets, and no horizontal overflow at mobile width all render
  correctly; revisit/pre-fill works; the submit-guard correctly blocks a
  second click. Reported: "all correct".
behavior_unverified_items:
  - truth: "The voting page renders and works correctly on both narrow (~360px mobile) and wide (desktop) screens — pill tri-color states, 44px tap targets, no horizontal overflow"
    test: "Open GET /poll/{ptoken} in a real browser at a ~360px viewport and at a wide desktop viewport; select each of Yes/No/Maybe on a slot"
    expected: "No horizontal overflow at 360px; each pill segment is tappable at >=44px; the selected pill shows green (#16A34A) for Yes, red (#DC2626) for No, gray (#64748B) for Maybe; unselected pills show no color"
    why_human: "CSS layout, color rendering, and tap-target geometry are facts a browser layout/paint engine produces — Go's httptest never renders CSS, so grep/wiring checks can confirm the rules exist but not that they render correctly. Already tracked as WINDOWS.md ledger entry #7 (open, unrun-verify)."
  - truth: "On submit the Save button disables and its label changes to 'Saving…', genuinely blocking a second POST (not merely a visual change)"
    test: "In a browser, click Save once, then rapidly click it again (or submit again) before the redirect completes; then disable JavaScript and confirm the form still submits and persists"
    expected: "The first click disables the button and shows 'Saving…'; the second click/submit is a real no-op (preventDefault fires, no second POST reaches the server); with JS disabled the native form submit still works"
    why_human: "vote.js only runs in a real browser JS engine — Go's httptest never executes client-side script, so the closure-flag guard's blocking behavior cannot be exercised by any Go test. The 02-03 plan explicitly declared this must-have with verification: backstop for this reason. Already tracked as WINDOWS.md ledger entry #8 (open, unrun-verify)."
---

# Phase 2: Voting End-to-End Verification Report

**Phase Goal:** A participant can open the shared poll link with no account, respond Yes/No/Maybe (with an optional comment) to each slot, and return later to change their response — on both mobile and desktop.
**Verified:** 2026-08-25T16:30:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

**Note on MVP-mode goal format:** ROADMAP.md's `**Goal:**` line for Phase 2 is a narrative paraphrase, not literally in `As a ... I want to ... so that ...` form (`gsd-tools query user-story.validate` on that exact string returns `valid: false`). However, all three PLAN.md files consistently derive and use a valid User Story ("As a poll participant who was sent a link, I want to open it with no account, mark Yes/No/Maybe (with an optional comment) for every proposed slot, and come back later to change my answer, so that the organizer can see my availability and I can keep it up to date — on phone or desktop." — confirmed `valid: true`). That derived story was the operative goal used throughout planning and execution, so this report proceeds with MVP-mode User Flow Coverage against it rather than refusing verification on a technicality.

## User Flow Coverage

User story: «As a poll participant who was sent a link, I want to open it with no account, mark Yes/No/Maybe (with an optional comment) for every proposed slot, and come back later to change my answer, so that the organizer can see my availability and I can keep it up to date — on phone or desktop.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Open the link | `GET /poll/{ptoken}` renders the poll title, name field, per-slot Yes/No/Maybe, comment — no login/cookie/account required | `internal/web/server.go:421` `handleVoteForm` (no auth middleware on the route); `TestVote_EndToEnd_SubmitAndRevisit` GET assertion | ✓ |
| Mark Yes/No/Maybe + optional comment | Every slot requires an answer; comment has no `required` attribute and is capped at 500 runes server-side | `internal/web/templates/vote.html:44` (textarea has no `required`); `internal/web/server.go:521-545` (per-slot enum validation); `TestVote_MissingSlotAnswer_RejectedNoWrite`, `TestVote_CommentTooLong_RejectedNoWrite` | ✓ |
| Submit | Valid submission persists 1 participants row + 1 responses row/slot, sets HttpOnly `dc_participant` cookie, 303s to `?saved=1` | `internal/store/participant.go:127` `SaveResponse`; `TestVote_EndToEnd_SubmitAndRevisit` DB-count + Set-Cookie assertions | ✓ |
| Return later, change answer | Cookie-matched revisit pre-fills name/comment/answers; resubmit with same cookie updates in place, no duplicate participant row | `internal/store/participant.go:75` `ParticipantByCookie`; `TestVote_EndToEnd_SubmitAndRevisit` revisit+resubmit assertions; `TestParticipant_SameCookieUpdatesInPlace`, `TestParticipant_NewCookieCreatesNewParticipant` | ✓ |
| Outcome — usable on phone and desktop | Segmented pill groups (tri-color, 44px tap target), single-column fluid layout, no fixed-height scroll container | `internal/web/static/style.css:250-304` (`.pill-group`/`.pill` rules, colors match spec exactly); real-browser rendering at 360px/desktop not executed | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED |

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Participant can open the participant link directly, with no login or signup step, and see the poll's slots | ✓ VERIFIED | `GET /poll/{ptoken}` has no auth check; renders title/name/slots/comment. `TestVote_EndToEnd_SubmitAndRevisit` |
| 2 | Participant enters a display name and selects Yes, No, or Maybe for each slot before submitting | ✓ VERIFIED | `handleSubmitResponse` rejects blank name (`nameRequiredCopy`) and any slot without exactly yes/no/maybe (`slotAnswerRequiredCopy`), zero DB write on failure. `TestVote_BlankName_RejectedNoWrite`, `TestVote_MissingSlotAnswer_RejectedNoWrite` |
| 3 | Participant can attach an optional short comment to their overall response | ✓ VERIFIED | `comment` textarea has no `required`; server caps at 500 runes (`maxCommentRunes`), rejects over-length with no write. `TestVote_CommentTooLong_RejectedNoWrite` |
| 4 | Participant can revisit the same participant link later, see their previous response pre-filled, and resubmit a changed response | ✓ VERIFIED | `ParticipantByCookie` prefill + `SaveResponse` upsert keyed on `(poll_id, cookie_token)`. `TestVote_EndToEnd_SubmitAndRevisit` (revisit pre-check + resubmit), `TestParticipant_SameCookieUpdatesInPlace`, `TestParticipant_ByCookiePrefillMapping` |
| 5 | The voting page renders and works correctly on both narrow (mobile) and wide (desktop) screens | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS present and matches UI-SPEC exactly (colors, 44px min-height, flex-wrap single column, no fixed-height scroll container) but not exercised in a real browser. WINDOWS.md #7 (open) |
| 6 | On submit the Save button disables and label changes to "Saving…", genuinely blocking a second POST | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `vote.js` closure-flag guard present and grep-verified (`Saving`, `preventDefault`) but JS execution unreachable from Go httptest; plan explicitly flagged `verification: backstop`. WINDOWS.md #8 (open) |

**Score:** 4/6 truths verified (2 present, behavior-unverified)

### Additional Verified Invariants (not roadmap SCs, but plan must-haves)

| Invariant | Status | Evidence |
|-----------|--------|----------|
| Anti-overwrite rule (no matching cookie always creates new participant, even on display-name collision) | ✓ VERIFIED | `TestParticipant_NewCookieCreatesNewParticipant` passes |
| `answer` DB CHECK enforced end-to-end through `SaveResponse` | ✓ VERIFIED | `TestResponse_AnswerEnumRejected` passes |
| Unknown participant token → branded 404, not naked stub | ✓ VERIFIED | `TestNotFound_InvalidPollLink`; `internal/web/templates/notfound.html` contains exact copy, no `<form>` |
| Confirmation banner shows after successful submit (`?saved=1`) and stays hidden otherwise | ✓ VERIFIED | Live spot-check test run against a fresh `httptest` server (added and removed during this verification): GET with `?saved=1` shows unhidden "Thanks! Your response has been saved."; GET without it shows the banner `hidden` |
| No raw/unescaped HTML interpolation (XSS gate, T-02-04) | ✓ VERIFIED | `grep -rn 'template.HTML' internal/web/` → no matches |
| Cookie token from `crypto/rand` only (T-02-01) | ✓ VERIFIED | `grep -rn 'math/rand' internal/web/` → no matches; `token.New()` used at all 3 call sites in `server.go` |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/store/schema.sql` | `participants` + `responses` tables, idempotent, `answer` CHECK, unique indexes | ✓ VERIFIED | Exact columns/constraints/indexes present (lines 29-50), matches plan spec verbatim |
| `internal/store/participant.go` | `PollByParticipantToken`, `ParticipantByCookie`, `SaveResponse` on `*Store` | ✓ VERIFIED | All 3 exported, parameterized SQL throughout, transactional upsert with delete-then-insert responses |
| `internal/web/templates/vote.html` | Full voting form: name, per-slot pill group, comment, banner, errors, vote.js tag | ✓ VERIFIED | 53 lines, all bindings present (`.NameError`, `.BannerError`, per-slot `.Error`, `.Selected`), `maxlength` on both fields |
| `internal/web/templates/notfound.html` | Plain branded 404, no form/nav | ✓ VERIFIED | Exact copy "This poll link isn't valid. Double-check the link you were given.", no `<form>` |
| `internal/web/server.go` | `handleVoteForm`, `handleSubmitResponse`, `renderNotFound`, cookie set/read, validation | ✓ VERIFIED | 603 lines; all handlers present, `dc_participant` cookie HttpOnly/SameSite=Lax/1yr/Secure-on-https |
| `internal/web/static/vote.js` | Submit-guard IIFE (disable + "Saving…", closure-flag no-op) | ✓ VERIFIED (code) / see behavior-unverified #6 for runtime proof | 31 lines, mirrors `create.js` pattern, no pill-selection logic |
| `internal/web/static/style.css` | `--color-yes` token, `.pill`/`.pill-group`, tri-color `:checked`, `.banner-success` | ✓ VERIFIED | All rules present with exact hex values matching UI-SPEC |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `internal/web/server.go` | `internal/store/participant.go` | `handleVoteForm`/`handleSubmitResponse` call `PollByParticipantToken`, `ParticipantByCookie`, `SaveResponse` | ✓ WIRED | All three calls present and exercised by passing tests |
| `internal/web/server.go` | `internal/token/token.go` | `token.New()` generates the cookie value | ✓ WIRED | 3 call sites; `math/rand` gate clean |
| `internal/store/participant.go` | `internal/store/schema.sql` | `INSERT INTO participants` / `INSERT INTO responses` | ✓ WIRED | Confirmed in `SaveResponse` |
| `internal/web/templates/vote.html` | `internal/web/static/style.css` | `.pill`/`.pill-yes`/`.pill-no`/`.pill-maybe` classes | ✓ WIRED | Markup and CSS selectors match exactly |
| `internal/web/templates/vote.html` | `internal/web/static/vote.js` | `<script src="/static/vote.js" defer>` | ✓ WIRED | Tag present; form/button ids (`vote-form`/`submit-btn`) match script's lookups |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./...` | `go build ./...` | exit 0 | ✓ PASS |
| `go vet ./...` | `go vet ./...` | exit 0 | ✓ PASS |
| Full test suite | `go test ./... -count=1` | all packages `ok` | ✓ PASS |
| All 3 claimed named tests exist and pass (not just "no tests to run") | `go test ./internal/web/ ./internal/store/ -list '.*'` then targeted `-run` | all 16 web tests + 5 store tests present and passing | ✓ PASS |
| Confirmation banner actually renders on `?saved=1` and stays hidden otherwise | Ad-hoc httptest spot-check (added, run, removed) hitting a live `httptest.Server` | banner unhidden with exact copy on `?saved=1`; hidden without it | ✓ PASS |
| XSS negative gate (no `template.HTML`) | `grep -rn 'template.HTML' internal/web/` | no matches | ✓ PASS |
| Crypto-rand negative gate (no `math/rand`) | `grep -rn 'math/rand' internal/web/` | no matches | ✓ PASS |
| Pill color rendering, 44px tap target, 360px no-overflow | requires real browser layout engine | not run | ? SKIP — routed to human verification |
| vote.js submit-guard blocking a second POST, JS-disabled fallback | requires real browser JS engine | not run | ? SKIP — routed to human verification |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|-------------|--------|----------|
| VOTE-01 | 02-01, 02-02 | Participant can open the participant link without logging in or creating an account | ✓ SATISFIED | `handleVoteForm`, no auth; e2e test |
| VOTE-02 | 02-01, 02-02 | Participant enters a display name before voting (no password) | ✓ SATISFIED | `display_name` field, required, `nameRequiredCopy` validation |
| VOTE-03 | 02-01, 02-02, 02-03 | Participant can select Yes, No, or Maybe for each slot | ✓ SATISFIED | Per-slot answer enum, pill markup, validation |
| VOTE-04 | 02-01, 02-03 | Participant can add an optional short comment | ✓ SATISFIED | Comment field optional, 500-char cap |
| VOTE-05 | 02-01, 02-03 | Participant can submit their response and later revisit the link to change it | ✓ SATISFIED | Cookie upsert, revisit prefill, resubmit-in-place, all test-backed |
| VOTE-06 | 02-03 | The voting UI is usable on both mobile and desktop screen sizes | ? NEEDS HUMAN | CSS correct per spec; real-browser render unverified (WINDOWS #7) |

No orphaned requirements: REQUIREMENTS.md maps VOTE-01 through VOTE-06 to Phase 2; the union of `requirements:` across 02-01/02-02/02-03 PLAN frontmatter covers all six IDs.

### Anti-Patterns Found

None. Scanned all files created/modified across the three plans (`schema.sql`, `participant.go`, `participant_test.go`, `server.go`, `vote.html`, `notfound.html`, `templates.go`, `vote.js`, `style.css`, `server_test.go`) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` and stub-shaped patterns (`return null`, hardcoded empty data flowing to render, console.log-only handlers). The only "placeholder" hits are legitimate HTML `placeholder=` attributes on the name/comment inputs, not stub markers.

### Broken-Windows Ledger Cross-Check

`.planning/WINDOWS.md` entries for phase 02:

| id | kind | status | Independently confirmed? |
|----|------|--------|---------------------------|
| 4 | stub (naked 404) | fixed | ✓ Confirmed — `renderNotFound` + branded copy present |
| 5 | stub (generic 400 validation) | fixed | ✓ Confirmed — per-field inline errors present, exact copy matches |
| 6 | stub (native radios, not pill group) | fixed | ✓ Confirmed — pill markup + tri-color CSS present |
| 7 | unrun-verify (pill color/tap-target/360px) | open | Correctly routed to human verification in this report |
| 8 | unrun-verify (JS submit guard + no-JS fallback) | open | Correctly routed to human verification in this report |

Phase 01's open ledger entries (#1, #2, #3) are pre-existing, out of this phase's scope, and do not affect Phase 2's status.

## Human Verification Required

### 1. Voting page responsive/visual rendering (mobile + desktop)

**Test:** Open a real poll's `GET /poll/{ptoken}` in a browser at a ~360px viewport width and at a wide desktop width. Select Yes, then No, then Maybe on a slot.
**Expected:** No horizontal overflow at 360px; each pill segment is at least 44px tall and tappable; the selected pill turns green (#16A34A) for Yes, red (#DC2626) for No, gray (#64748B) for Maybe with white text; unselected pills show neutral surface/border only; the slot list grows with the page (no inner scrollbar).
**Why human:** Go's `httptest` never renders CSS or lays out a page — only a real browser can confirm color, tap-target size, and overflow behavior. Tracked as WINDOWS.md #7.

### 2. Submit-guard behavior and JS-disabled fallback

**Test:** In a browser, click "Save my response" once, then immediately click it again (or attempt a second submit) before the page navigates away. Then disable JavaScript entirely and submit the form again.
**Expected:** First click disables the button and changes its label to "Saving…"; the second click/submit produces no second network request (a real no-op, not just a re-style). With JavaScript disabled, the form still submits via native browser behavior and the response persists.
**Why human:** `vote.js`'s closure-flag guard only executes in a real JS engine; Go's `httptest` never runs client-side script. The plan explicitly declared this must-have `verification: backstop` for this reason. Tracked as WINDOWS.md #8.

## Gaps Summary

No gaps. All roadmap Success Criteria and plan must-haves are either fully verified against the codebase (build/vet/test all pass, all claimed tests exist by name and pass, schema/store/handler/template/CSS/JS artifacts are substantive and wired, all negative security gates clean, zero anti-pattern markers) or correctly and explicitly routed to human verification for the two browser-only behaviors (responsive/visual CSS rendering and the JS submit-guard) that no Go test can observe. Both open items were already self-identified by the executing plans and recorded in `.planning/WINDOWS.md` (#7, #8) — this verification independently confirms they are the correct and complete set of remaining unverifiable-by-automation items, with no additional gaps found.

---

*Verified: 2026-08-25T16:30:00Z*
*Verifier: Claude (gsd-verifier)*
