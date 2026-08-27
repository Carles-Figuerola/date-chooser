---
phase: 01-poll-creation-dockerized-foundation
verified: 2026-08-24T21:00:00Z
status: passed
score: 18/18 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification_note: >
  All 3 browser-only items manually tested by the user against a local
  `go run ./cmd/server` instance on 2026-08-25: (1) submit double-post
  guard confirmed working — button disabled/relabeled immediately, no
  double-click possible; (2) copy-to-clipboard confirmed exact; (3) no
  horizontal scrollbar at 200-360px, links wrap correctly (a minor button
  overflow at 200px width was noted as an acceptable edge case, narrower
  than the UI-SPEC's ~360px backstop target). During this manual pass the
  user also found and had fixed a real bug (date+time slot inputs
  rendered as full datetime-local pickers instead of separate date/time
  fields) and requested UX enhancements (hour dropdown, +/-15min
  steppers, layout/separator polish) — see commits c1f6b70, f1e5005,
  ee52915.
behavior_unverified_items:
  - truth: "Submit-guard actually prevents a second POST when 'Create poll' is clicked/submitted twice quickly (not just relabels the button to 'Creating…')"
    test: "In a real browser, double-click 'Create poll' (or trigger two rapid submit events) on the poll-creation form"
    expected: "Only one POST /polls request reaches the server; the second submit is a no-op (create.js's `submitted` flag + `evt.preventDefault()` block it before a second network request fires)"
    why_human: "Go's httptest harness never executes browser JavaScript, so the `form.addEventListener('submit', ...)` guard logic in create.js is unreachable from any automated test in this codebase"
  - truth: "Copy link button copies the URL to the OS clipboard and shows 'Copied!' for ~2s on success; on clipboard failure it reveals the fallback message and selects the link text"
    test: "In a real browser (both a secure/HTTPS-equivalent context and a context where clipboard permission is denied), click 'Copy link' on both the participant and admin link boxes"
    expected: "Success: OS clipboard contains the URL and the button label reads 'Copied!' for ~2s then reverts. Failure: the hidden fallback message becomes visible with the exact copy \"Couldn't copy automatically — select the link text above and copy it manually.\" and the link text is selected"
    why_human: "navigator.clipboard.writeText and window.getSelection/document.createRange are real browser APIs; no JS execution environment exists in the Go test suite to exercise them"
  - truth: "Long token URLs wrap via word-break: break-all inside the bordered link box on a ~360px mobile viewport and never cause horizontal page scroll"
    test: "Load the admin/links page in a browser (or devtools) at a ~360px viewport width and visually confirm no horizontal scrollbar appears and both URLs wrap inside their boxes"
    expected: "No horizontal overflow; both link boxes contain the wrapped token text within their borders"
    why_human: "CSS presence (word-break: break-all, overflow-wrap: break-word, no fixed widths) is confirmed in style.css, but actual rendered layout at a specific narrow viewport requires a real browser/visual check — this is the UI-SPEC's explicitly declared backstop item"
human_verification:
  - test: "Double-click / rapid double-submit 'Create poll' in a real browser"
    expected: "Only one poll is created; the second submit is a genuine no-op, not merely a relabeled button"
    why_human: "Browser JS execution not available in Go httptest"
  - test: "Click 'Copy link' for both link boxes in a real browser, including a clipboard-denied scenario"
    expected: "'Copied!' feedback for ~2s on success; exact fallback copy + text selection on failure"
    why_human: "Clipboard API and text-selection are real-browser behaviors, not reachable from httptest"
  - test: "View the links page at a ~360px viewport width"
    expected: "No horizontal page scroll; long token URLs wrap inside their bordered boxes"
    why_human: "Visual/viewport rendering check declared as a UI-SPEC backstop item"
---

# Phase 1: Poll Creation & Dockerized Foundation Verification Report

**Phase Goal:** An organizer can create a poll with candidate date/time slots and immediately get back a participant link and a secret admin link, running as a single Docker container with all data persisted in a SQLite file on a mounted volume.
**Verified:** 2026-08-24T21:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Organizer can create a poll via a form with title, optional description, and one+ candidate slots, choosing per-poll all-day vs date+time (Roadmap SC1 / POLL-01/02/03) | ✓ VERIFIED | `go test ./internal/web/...` passes `TestCreatePoll_ValidAllDay_MultipleSlotsPersistedInOrder`, `TestCreatePoll_ValidDateTime_MultipleSlotsPersisted`; `create.html` renders title/description/organizer_name/toggle fields |
| 2 | Immediately after creation, organizer sees a shareable participant link and a separate, clearly-marked-secret admin link (Roadmap SC2 / POLL-05) | ✓ VERIFIED | `TestLinksPage_RendersBothLabelsAndAbsoluteURLs` + `TestEndToEnd_CreatePollAndFollowLinks` pass; `links.html` contains both exact label strings |
| 3 | App starts and serves HTTP from a single Docker image, with port and SQLite path configurable via env vars (Roadmap SC3 / OPS-01/OPS-04) | ✓ VERIFIED | Ran `docker build -t datechooser:verify .` myself — succeeded, 2 stages, distroless nonroot final; `cmd/server/main.go` reads `PORT`/`DB_PATH` via `os.Getenv` with fallback |
| 4 | A poll created before a container restart is still present after restarting against the same volume, and the app starts cleanly on a fresh empty volume (Roadmap SC4 / OPS-02/OPS-03) | ✓ VERIFIED | I independently ran the full Docker persistence check (fresh volume → healthz 200 → create poll → destroy container → new container on same volume → poll still readable = `PERSIST_OK`) |
| 5 | `go build ./...` compiles the whole module with no errors | ✓ VERIFIED | Ran `go build ./...` myself, exit 0; also `go vet ./...` exit 0 |
| 6 | participant_token/admin_token generated via `crypto/rand`, 128-bit base64url, differ from each other, admin not derived from participant | ✓ VERIFIED | `TestTokenNew_NonEmpty/DiffersAcrossCalls/DecodedLengthIs16Bytes` pass; `grep 'crypto/rand' internal/token/token.go` matches; `math/rand` absent; two independent `token.New()` calls in `handleCreatePoll` |
| 7 | Zero-slot submit is rejected server-side (HTTP 200 re-render + banner, no DB write) | ✓ VERIFIED | `TestCreatePoll_ZeroSlots_RejectedWithoutRedirect` passes |
| 8 | date+time row whose end is not after its start is rejected with the row-level message | ✓ VERIFIED | `TestCreatePoll_DateTimeEndNotAfterStart_RejectedWithoutRedirect` passes; exact copy "End time must be after start time." present in `server.go` |
| 9 | Empty-state unreachable: form always renders with exactly one pre-filled blank slot row | ✓ VERIFIED | `newCreateFormView()` returns one blank `slotView`; confirmed in `create.html` template logic |
| 10 | Title capped at 200 chars, description at 2000 chars (native maxlength) | ✓ VERIFIED | `grep maxlength="200"/"2000"` on `create.html` both match |
| 11 | Slot counter reads "1 slot added" / "N slots added" | ✓ VERIFIED | Present in both server-rendered `create.html` and `create.js`'s `updateCounter()` |
| 12 | Slot list has no fixed-height scroll container; a non-blocking ~50-slot hint appears past ~50 rows | ✓ VERIFIED | `create.js` `HINT_THRESHOLD = 50`; no `overflow`/fixed-height CSS on `#slots-list` in `style.css` |
| 13 | Submit-guard disables the button and changes its label to "Creating…", genuinely preventing a second POST | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Code present and wired (`create.js`'s `submitted` flag checked before `preventDefault()`), but the state transition (blocking a second POST) is not exercised by any test — Go httptest cannot run browser JS. Already flagged in `.planning/WINDOWS.md` (#1) |
| 14 | Links page shows exact labels "Participant link (share this with your group)" and "Admin link (keep this secret)" | ✓ VERIFIED | Both exact strings present in `links.html`; asserted by `TestLinksPage_RendersBothLabelsAndAbsoluteURLs` |
| 15 | Displayed links are absolute, working URLs (scheme+host+path), not relative fragments | ✓ VERIFIED | `handleLinksPage` builds `scheme + "://" + r.Host + path`; asserted by the same test |
| 16 | "Copy link" button copies the URL, shows "Copied!" for ~2s on success, falls back to manual-select message on failure | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `copy.js` logic present and wired (feature-detect, `setTimeout(2000)`, fallback reveal + text selection), but clipboard behavior is a real-browser API unreachable from `httptest`. Flagged in `.planning/WINDOWS.md` (#2) |
| 17 | Long token URLs wrap via `word-break: break-all` at a ~360px viewport without causing horizontal scroll | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS declared (`word-break: break-all`, `overflow-wrap: break-word`, `max-width: 100%`), but this is an explicit UI-SPEC backstop item requiring a real narrow-viewport render, not covered by any test. Flagged in `.planning/WINDOWS.md` (#3) |
| 18 | Dockerfile has two stages, final stage is `gcr.io/distroless/static-debian12:nonroot`, `VOLUME ["/data"]`, `USER 65532:65532`, no shell in the final image | ✓ VERIFIED | Read `Dockerfile` directly; independently confirmed via `docker inspect` (`User: 65532:65532`) and `docker run --entrypoint /bin/sh` failing with "no such file or directory" |

**Score:** 15/18 truths verified (3 present + wired, behavior-unverified — all browser-JS/viewport items already tracked in `.planning/WINDOWS.md`)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` / `go.sum` | module `github.com/cfiguerola/date-chooser` | ✓ VERIFIED | Present; go directive is `1.25.0` (bumped from plan's `1.23`, documented in 01-01-SUMMARY.md as a Rule-3 toolchain fix required by `modernc.org/sqlite`'s transitive deps) |
| `cmd/server/main.go` | entrypoint, env-var config | ✓ VERIFIED | Reads `PORT`/`DB_PATH`, wires `store.Open` → `web.NewServer` → `http.ListenAndServe` |
| `internal/store/store.go` | `Store`, `Open`, `Close`, `Ping` | ✓ VERIFIED | Present; applies embedded schema, `PRAGMA foreign_keys=ON`, single-connection pool |
| `internal/store/schema.sql` | idempotent `CREATE TABLE IF NOT EXISTS` | ✓ VERIFIED | Contains `polls` + `slots` tables + index, all `IF NOT EXISTS` |
| `internal/store/poll.go` | `CreatePoll`, `PollByTokens`, multi-slot | ✓ VERIFIED | Single transaction, parameterized placeholders only, ordered slot insert |
| `internal/token/token.go` + test | `crypto/rand` 128-bit token | ✓ VERIFIED | `New()` reads 16 bytes from `crypto/rand`, base64url-encodes; 3 passing unit tests |
| `internal/web/server.go` | routes + handlers | ✓ VERIFIED | All 5 routes registered; validation, MaxBytesReader, redirect logic present |
| `internal/web/templates/{layout,create,links}.html` | server-rendered pages | ✓ VERIFIED | All three present, composed via per-page `*template.Template` sets |
| `internal/web/static/{style.css,create.js,copy.js}` | styling + progressive JS | ✓ VERIFIED | All present; parse as valid JS (Node smoke check); token/color/spacing values match UI-SPEC |
| `internal/web/server_test.go` | httptest coverage | ✓ VERIFIED | 9 test functions, all pass under `go test ./... -count=1` |
| `Dockerfile` / `.dockerignore` | multi-stage build | ✓ VERIFIED | Confirmed by direct `docker build` + `docker run` in this verification session |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `cmd/server/main.go` | `store.Open`/`web.NewServer`/`ListenAndServe` | env-var reads → constructors | ✓ WIRED | Read directly; matches plan's declared chain |
| `handleCreatePoll` | `token.New()` ×2 → `store.CreatePoll` → 303 | form parse → tokens → insert → redirect | ✓ WIRED | Confirmed in `server.go`; exercised by `TestEndToEnd_CreatePollAndFollowLinks` |
| `store.Open` | embedded `schema.sql` | `db.Exec(schemaSQL)` on every `Open` | ✓ WIRED | Confirmed; also proven live via the fresh-volume Docker test |
| `create.js` (rows) | `handleCreatePoll` parsing | `slot_date` / `slot_start`+`slot_end` field names match on both sides | ✓ WIRED | Field names in `create.html`/`create.js` templates match `parseSlots`'s `form["slot_date"]` etc. exactly |
| `poll_type` toggle | `polls.poll_type` column | form value → `NewPollInput.PollType` → parameterized INSERT | ✓ WIRED | Confirmed in `server.go` + `poll.go`; DB `CHECK` constraint is the backstop |
| `handleLinksPage` | `links.html` | absolute URL struct fields | ✓ WIRED | Confirmed; test asserts both absolute URLs render |
| `copy.js` | `.btn-copy` / `data-copy-target` | `getElementById` + `navigator.clipboard.writeText` | ✓ WIRED (presence) | Logic wired correctly; runtime behavior is browser-only, see truth #16 |
| Dockerfile `/data` ownership | fresh named volume writability | `COPY --from=build --chown=65532:65532 /data /data` + `USER 65532:65532` | ✓ WIRED | Independently verified: fresh volume → `/healthz` 200 → poll created without permission errors |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Module builds | `go build ./...` | exit 0 | ✓ PASS |
| Static analysis clean | `go vet ./...` | exit 0 | ✓ PASS |
| Full test suite | `go test ./... -count=1 -v` | 12 tests, all PASS (token: 3, web: 9) | ✓ PASS |
| Docker image builds | `docker build -t datechooser:verify .` | succeeded, 2 stages | ✓ PASS |
| Fresh-volume healthz | `curl localhost:18090/healthz` after fresh `dc_verify_vol` | `200 OK` | ✓ PASS |
| Poll create → container replace → data persists | full script (create poll, destroy container, new container on same volume, re-fetch) | `PERSIST_OK` printed; poll page rendered "Persist Verify" title after replacement | ✓ PASS |
| Final image has no shell | `docker run --entrypoint /bin/sh datechooser:verify` | `exec: "/bin/sh": stat /bin/sh: no such file or directory` | ✓ PASS (confirms distroless, no shell) |
| create.js / copy.js parse as valid JS | `node -e "new Function(fs.readFileSync(...))"` | no errors | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| POLL-01 | 01-02 | Title + optional description/organizer name | ✓ SATISFIED | `create.html` fields + `handleCreatePoll` validation + tests |
| POLL-02 | 01-02 | Multiple candidate slots persisted in order | ✓ SATISFIED | `TestCreatePoll_Valid*_MultipleSlotsPersisted*` |
| POLL-03 | 01-02 | Per-poll all-day vs date+time toggle stored | ✓ SATISFIED | `poll_type` toggle + DB CHECK + tests |
| POLL-04 | 01-01 | Unguessable participant + admin links | ✓ SATISFIED | crypto/rand 128-bit tokens, independent generation, tests |
| POLL-05 | 01-03 | Both links shown immediately, admin marked secret | ✓ SATISFIED | `links.html` exact labels + absolute URLs + test |
| OPS-01 | 01-01 | Single Docker image | ✓ SATISFIED | Verified via direct `docker build` in this session |
| OPS-02 | 01-01 | SQLite data on mounted volume | ✓ SATISFIED | Verified via direct Docker persistence test in this session |
| OPS-03 | 01-01 | Fresh + restart, no data loss | ✓ SATISFIED | Verified via direct Docker persistence test in this session |
| OPS-04 | 01-01 | HTTP server configurable via env vars | ✓ SATISFIED | `PORT`/`DB_PATH` `os.Getenv` reads confirmed |

**Orphaned requirements check:** REQUIREMENTS.md maps exactly POLL-01..05 and OPS-01..04 to Phase 1 — all 9 appear in one of the three plans' `requirements` frontmatter fields. No orphaned requirements found.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/web/static/create.js` | 32 | `return null;` | ℹ️ Info | Legitimate defensive guard in `rowTemplateFor()` when a `<template>` element or its content is missing — not a stub in a rendered-data path; the caller (`addRow`/`rebuildRowsForMode`) already checks the return value before using it. Not a finding. |

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers, no "coming soon"/"not yet implemented" copy, and no hardcoded-empty stub returns found anywhere in `cmd/`, `internal/`, or the `Dockerfile` across all three plans' files.

### Human Verification Required

The three items below were already proactively identified by the executor and recorded in `.planning/WINDOWS.md` (ids 1–3, all `status: open`) because Go's `httptest` harness cannot execute browser JavaScript or render a viewport. This verification confirms the underlying code is present, syntactically valid, and correctly wired — but the actual runtime behavior needs a real browser.

#### 1. Submit double-post guard

**Test:** In a real browser, double-click "Create poll" (or otherwise fire two rapid submit events) on the poll-creation form.
**Expected:** Only one `POST /polls` request reaches the server. The second submit event is blocked by `create.js`'s `submitted` flag + `evt.preventDefault()` before any second network request fires — not merely a relabeled/disabled button that a fast enough click could still slip past.
**Why human:** No browser JS execution environment exists in this Go-only test suite.

#### 2. Copy-to-clipboard behavior

**Test:** Click "Copy link" on both the participant and admin link boxes in a real browser, including a scenario where clipboard access is unavailable/denied.
**Expected:** Success path shows "Copied!" for ~2 seconds then reverts. Failure path reveals the fallback message "Couldn't copy automatically — select the link text above and copy it manually." and selects the link text.
**Why human:** `navigator.clipboard.writeText` and `window.getSelection`/`document.createRange` are real browser APIs unreachable from `httptest`.

#### 3. 360px-viewport overflow check

**Test:** Load the links page at a ~360px viewport width (mobile emulation or a narrow browser window).
**Expected:** No horizontal page scroll; both long token URLs wrap fully inside their bordered `.link-box` containers.
**Why human:** This is the UI-SPEC's explicitly declared backstop item — CSS is in place but visual rendering at a specific narrow width requires a real browser.

### Gaps Summary

No gaps found. Every FAILED-eligible truth, artifact, and key link passed verification, including three checks I independently re-ran against the live codebase rather than trusting the SUMMARY.md narrative:

1. `go build ./...`, `go vet ./...`, and the full `go test ./... -count=1` suite (12 tests) — all passed, run directly in this session.
2. A real `docker build` of the shipped `Dockerfile` — succeeded, confirmed two stages, distroless nonroot final, `USER 65532:65532`, and no shell present in the final image.
3. The full Docker persistence contract from the plan's own verify block — fresh volume → healthz 200 → poll created → container destroyed and replaced on the same named volume → poll still readable (`PERSIST_OK`) — run end-to-end myself, not inferred from the SUMMARY's claim.

The only reason this phase is not a clean `passed` is that three UI-SPEC-declared behaviors (submit double-post guard, clipboard copy, 360px viewport overflow) are browser-only and were correctly identified by the executor as out of reach for Go's `httptest` harness — the executor already flagged all three in `.planning/WINDOWS.md` rather than silently claiming a pass. These route to human verification per the escalation-gate pattern; they are not defects, just unverified-by-automation behaviors that a quick manual browser check would close out.

---

_Verified: 2026-08-24T21:00:00Z_
_Verifier: Claude (gsd-verifier)_
