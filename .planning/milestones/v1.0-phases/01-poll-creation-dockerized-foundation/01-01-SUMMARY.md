---
phase: 01-poll-creation-dockerized-foundation
plan: 01
subsystem: infra
tags: [go, net-http, sqlite, modernc-sqlite, html-template, docker, distroless, crypto-rand]

# Dependency graph
requires: []
provides:
  - "github.com/cfiguerola/date-chooser Go module scaffold (cmd/server + internal/{store,token,web})"
  - "SQLite persistence layer: idempotent embedded schema.sql (polls, slots), Store.Open/Close/Ping, CreatePoll, PollByTokens"
  - "crypto/rand-based 128-bit URL-safe token generator (internal/token)"
  - "net/http ServeMux routes: GET /{$}, POST /polls, GET /poll/{ptoken}/admin/{atoken}, GET /healthz, GET /static/"
  - "Server-rendered layout+create+links HTML templates embedded via go:embed, minimal CSS per UI-SPEC tokens"
  - "Multi-stage Dockerfile (golang:1.25 build -> distroless nonroot final) with PORT/DB_PATH env-var config and a /data volume that survives container replacement"
affects: [01-02, 01-03, phase-2-voting, phase-3-results-grid, phase-4-admin-management]

# Actuals (#2632)
actuals:
  tokens: 8217
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: [modernc.org/sqlite (pure-Go SQLite driver), gcr.io/distroless/static-debian12]
  patterns:
    - "Embedded idempotent schema.sql applied on every Store.Open (no migration framework for v1)"
    - "Two independent crypto/rand token.New() calls per poll (participant + admin), never one derived from the other"
    - "Post/Redirect/Get: POST /polls -> 303 -> GET /poll/{pt}/admin/{at}"
    - "Per-page html/template composition: layout.html defines {{define \"layout\"}} wrapping {{template \"content\" .}}; each page template.ParseFS's (layout.html, page.html) into its own *template.Template so identically-named \"content\" blocks never collide across pages"
    - "Go 1.22+ ServeMux method+wildcard routing instead of a third-party router"
    - "Multi-stage Docker build: CGO_ENABLED=0 static binary -> distroless nonroot final stage, /data pre-chowned to uid 65532 in the build stage for fresh-volume writability"

key-files:
  created:
    - go.mod
    - go.sum
    - cmd/server/main.go
    - internal/store/store.go
    - internal/store/schema.sql
    - internal/store/poll.go
    - internal/token/token.go
    - internal/token/token_test.go
    - internal/web/server.go
    - internal/web/server_test.go
    - internal/web/templates.go
    - internal/web/templates/layout.html
    - internal/web/templates/create.html
    - internal/web/templates/links.html
    - internal/web/static/style.css
    - Dockerfile
    - .dockerignore
  modified: []

key-decisions:
  - "go.mod go directive bumped from the plan's specified 1.23 to 1.25 (and Dockerfile build stage from golang:1.23 to golang:1.25) because modernc.org/sqlite's current transitive dependency graph requires go >= 1.25 to compile — a Rule 3 blocking-issue fix, not an architectural change; the mandated pure-Go driver and its version pin (go.sum checksums) are unchanged."
  - "Token unit tests renamed to TestTokenNew_* (from TestNew_*) so the plan's verify filter (-run 'Token|CreatePoll|EndToEnd|Health') actually selects them, rather than passing vacuously via 'no tests to run'."
  - "Per-page template composition (layout.html + page.html parsed as two separate *template.Template sets) rather than one glob-parsed set, because both create.html and links.html define a template named \"content\"; parsing all *.html together into one set would let whichever file parses last silently win for both pages."
  - "Docker persistence verify script's PT capture required extracting the URL path from curl's %{redirect_url} (which resolves a relative Location header into a full absolute URL) before reusing it against the second container's port — a verify-script fix, not an application bug; documented here since it's not encoded in any committed file."

patterns-established:
  - "internal/{store,token,web} package layout: store owns persistence + schema, token owns credential generation, web owns HTTP/templates — reused unmodified by later phases per SKELETON.md's subsequent-slice plan."
  - "Embedded schema.sql with CREATE TABLE IF NOT EXISTS is the only migration mechanism for v1; Phase 2 adds participants/responses the same way."

requirements-completed: [POLL-04, OPS-01, OPS-02, OPS-03, OPS-04]

coverage:
  - id: D1
    description: "Organizer submits create-poll form (title + one slot) and is 303-redirected to a links page showing both the participant and admin URLs"
    requirement: POLL-04
    verification:
      - kind: e2e
        ref: "internal/web/server_test.go#TestEndToEnd_CreatePollAndFollowLinks"
        status: pass
    human_judgment: false
  - id: D2
    description: "participant_token and admin_token are generated via crypto/rand, are 128-bit base64url, differ from each other, and admin is not derived from participant"
    requirement: POLL-04
    verification:
      - kind: unit
        ref: "internal/token/token_test.go#TestTokenNew_NonEmpty"
        status: pass
      - kind: unit
        ref: "internal/token/token_test.go#TestTokenNew_DiffersAcrossCalls"
        status: pass
      - kind: unit
        ref: "internal/token/token_test.go#TestTokenNew_DecodedLengthIs16Bytes"
        status: pass
    human_judgment: false
  - id: D3
    description: "App builds into a single Docker image, serves HTTP, and reads PORT/DB_PATH from environment variables"
    requirement: OPS-01
    verification:
      - kind: manual_procedural
        ref: "docker build -t datechooser:test . (build succeeded, two stages, distroless nonroot final)"
        status: pass
    human_judgment: false
  - id: D4
    description: "SQLite data persists on a mounted /data volume across a full container replacement, and the app starts cleanly on a fresh empty volume"
    requirement: "OPS-02, OPS-03"
    verification:
      - kind: manual_procedural
        ref: "docker run against dc_test_vol: healthz 200 on fresh volume, poll created, container replaced, poll readable from the new container (PERSIST_OK)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Config is env-var only (PORT default 8080, DB_PATH default /data/datechooser.db), no hardcoded values"
    requirement: OPS-04
    verification:
      - kind: unit
        ref: "cmd/server/main.go os.Getenv(\"PORT\")/os.Getenv(\"DB_PATH\") with fallback, exercised implicitly by the Docker persistence check"
        status: pass
    human_judgment: false

duration: 11min
completed: 2026-08-24
status: complete
---

# Phase 1 Plan 1: Walking Skeleton Summary

**End-to-end poll creation (Go/net-http/modernc.org-sqlite) producing two independent crypto/rand tokens, shipped as a single distroless Docker image with a persistent /data volume.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-08-24T19:42:55Z
- **Completed:** 2026-08-24T19:53:57Z
- **Tasks:** 2
- **Files modified:** 17 (all created)

## Accomplishments
- Full happy-path flow proven end-to-end under `go test`: POST /polls form submission → SQLite write (polls + slots in one transaction) → two independent crypto/rand tokens → 303 redirect → rendered links page showing both URLs
- SQLite persistence layer with idempotent embedded `schema.sql`, pure-Go `modernc.org/sqlite` driver (no CGO), `Store.Open/Close/Ping`, `CreatePoll`, `PollByTokens`
- `crypto/rand`-only 128-bit URL-safe token generator with a negative gate proving `math/rand` is never imported
- Multi-stage Dockerfile producing a distroless nonroot image; verified a poll created in one container is still readable from a second container after full container replacement on the same named volume, and the app starts cleanly on a brand-new empty volume

## Task Commits

Each task was committed atomically (Task 1 followed the TDD RED/GREEN cycle since it carries `tdd="true"`):

1. **Task 1 (RED): add failing tests** - `8268161` (test)
2. **Task 1 (GREEN): implement create-poll end-to-end flow** - `6f6b1fa` (feat)
3. **Task 2: package as a single Docker image** - `eec777e` (feat)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `go.mod`, `go.sum` - module `github.com/cfiguerola/date-chooser`, go 1.25; `modernc.org/sqlite` as the sole third-party runtime dependency
- `cmd/server/main.go` - entrypoint reading `PORT`/`DB_PATH` env vars with documented fallbacks
- `internal/store/store.go` - `Store`, `Open` (applies embedded `schema.sql`, `PRAGMA foreign_keys=ON`), `Close`, `Ping`, `DB()`
- `internal/store/schema.sql` - idempotent `polls` + `slots` tables (`CREATE TABLE IF NOT EXISTS`)
- `internal/store/poll.go` - `Poll`, `Slot`, `NewPollInput`, `NewSlotInput`, `CreatePoll` (single transaction, parameterized), `PollByTokens` (`ErrNotFound` on miss)
- `internal/token/token.go` + `token_test.go` - `New()` (crypto/rand, 16 bytes, base64.RawURLEncoding)
- `internal/web/server.go` + `server_test.go` - routes, handlers (create form, create-poll POST, links page, healthz, static), httptest end-to-end + healthz tests
- `internal/web/templates.go` - `//go:embed` for templates and static assets; per-page `*template.Template` composition
- `internal/web/templates/{layout,create,links}.html` - minimal-but-real HTML per UI-SPEC tokens
- `internal/web/static/style.css` - background/surface/accent/destructive colors, spacing, 44px tap targets, `word-break: break-all` link boxes
- `Dockerfile` - `golang:1.25` build → `gcr.io/distroless/static-debian12:nonroot` final, `/data` pre-chowned to uid 65532, `VOLUME ["/data"]`, `USER 65532:65532`
- `.dockerignore` - excludes `.git`, `.planning`, `.claude`, local `*.db` files

## Decisions Made
- Bumped `go.mod`'s go directive (and the Dockerfile's `golang` build-stage tag) from the plan's specified 1.23 to 1.25, because `modernc.org/sqlite`'s current transitive dependencies require go >= 1.25. The mandated driver itself, its `go.sum`-pinned version, and every other architectural decision in SKELETON.md are unchanged — this is a toolchain-version correction, not a scope or dependency change.
- Renamed token test functions to `TestTokenNew_*` so the plan's own verify command (`-run 'Token|CreatePoll|EndToEnd|Health'`) selects them explicitly instead of matching zero tests.
- Composed `layout.html` + each page template as two separate `*template.Template` sets (rather than one `ParseFS("templates/*.html")` glob) so `create.html`'s and `links.html`'s identically-named `{{define "content"}}` blocks don't silently overwrite each other.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] go.mod / Dockerfile Go version bumped 1.23 → 1.25**
- **Found during:** Task 1, `go get modernc.org/sqlite@latest`
- **Issue:** The plan's `go.mod (... go 1.23)` and Dockerfile `FROM golang:1.23` could not build with the mandated `modernc.org/sqlite` driver — its transitive dependency graph (`modernc.org/libc` etc.) requires go >= 1.25. This blocked `go build ./...` entirely.
- **Fix:** Set `go.mod`'s go directive to `1.25` and the Dockerfile build stage to `FROM golang:1.25`. Confirmed this is a pure-Go toolchain-version requirement, not a driver substitution (no CGO driver introduced; `mattn/go-sqlite3` negative gate still passes).
- **Files modified:** go.mod, Dockerfile
- **Verification:** `go build ./...` exits 0; `docker build` succeeds on `golang:1.25`; `grep -n 'sqlite' go.mod` still shows only `modernc.org/sqlite`.
- **Committed in:** 6f6b1fa (Task 1), eec777e (Task 2)

**2. [Rule 3 - Blocking] Token test names didn't match the plan's verify `-run` filter**
- **Found during:** Task 1, running the plan's exact verify command
- **Issue:** `token_test.go`'s tests were originally named `TestNew_*`, which does not contain the substring "Token" that the plan's verify regex (`-run 'Token|CreatePoll|EndToEnd|Health'`) selects on — the command would report `[no tests to run]` for that package (a silent, vacuous pass rather than a real one).
- **Fix:** Renamed to `TestTokenNew_NonEmpty`, `TestTokenNew_DiffersAcrossCalls`, `TestTokenNew_DecodedLengthIs16Bytes`.
- **Files modified:** internal/token/token_test.go
- **Verification:** `go test ./internal/token/ ./internal/web/ -run 'Token|CreatePoll|EndToEnd|Health' -count=1 -v` now actually executes and passes all three token tests plus the two web tests.
- **Committed in:** 8268161 (RED), 6f6b1fa (GREEN, no further renames needed)

---

**Total deviations:** 2 auto-fixed (both Rule 3 - blocking, both toolchain/verification-tooling corrections)
**Impact on plan:** No architectural or scope change. Every decision in SKELETON.md's Architectural Decisions table (language, routing, templating, data layer, schema, tokens, URL shape, post-create flow, config, deployment, directory layout) was implemented exactly as specified.

## Issues Encountered
- `docker run --rm datechooser:test /bin/sh -c 'echo shellcheck'` (an ad hoc check for shell absence in the distroless final stage) did not fail fast as expected: because the image's `ENTRYPOINT` is `["/datechooser"]`, the `/bin/sh -c ...` arguments were passed to the app as ignored positional args rather than exec'd, so the server started in the foreground and the check appeared to hang. Not a defect in the shipped image — confirmed via the Dockerfile's `FROM`/`VOLUME`/`USER` lines directly instead, and via the fact that the app never shells out anywhere in its own code path. The stray container was removed.

## User Setup Required

None - no external service configuration required. `tarmac`/Spinnaker upload guidance in the user's global CLAUDE.md does not apply to this phase (no Spinnaker pipelines exist yet for this project).

## Next Phase Readiness
- The Walking Skeleton is fully demonstrable in Docker: create a poll, get two links, restart the container, data survives.
- Plans 02 (full creation form) and 03 (styled links page) can expand the `create.html`/`links.html` templates and `POST /polls` validation horizontally without touching the store, token, or routing layers established here.
- No blockers. One item worth a later look: `go.mod`'s go directive is now 1.25, one minor version ahead of the phase plan's original 1.23 assumption — noted here for anyone auditing SKELETON.md against the shipped code.

---
*Phase: 01-poll-creation-dockerized-foundation*
*Completed: 2026-08-24*

## Self-Check: PASSED

All 17 created files verified present on disk; all 3 task commits (`8268161`, `6f6b1fa`, `eec777e`) verified present in git log.
