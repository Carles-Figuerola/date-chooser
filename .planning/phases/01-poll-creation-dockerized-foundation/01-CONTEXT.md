# Phase 1: Poll Creation & Dockerized Foundation - Context

**Gathered:** 2026-08-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Organizer can create a poll (title, optional description, candidate date/time slots) via a web form and immediately receives a shareable participant link plus a separate, clearly-marked secret admin link. The app runs as a single Docker image, configured via environment variables, with all data persisted in a SQLite file on a mounted volume — surviving restarts and starting cleanly on a fresh volume. No voting, results display, or admin editing yet (those are Phases 2-4).

</domain>

<decisions>
## Implementation Decisions

### Poll Creation Form
- Organizer adds slots one at a time via date+time picker rows (add/remove rows) — simplest to build, matches Doodle/Rallly UX
- All-day vs specific-time is a per-poll toggle, not per-slot — simpler mental model, one type of slot per poll
- Minimum 1 slot required, no hard maximum (soft UI guidance around ~50 for usability)
- Organizer name field is optional at creation — it's just a label; the admin link is the actual access control

### Link & Token Security
- Participant link: `/poll/{token}` where token is a cryptographically random, URL-safe token (~128 bits of entropy)
- Admin link: `/poll/{participant_token}/admin/{admin_token}` where admin_token is a separate, independently-random secret — NOT derivable from the participant token
- No rate limiting on link access in v1 — token entropy is the only protection; acceptable for a small-group, self-hosted tool
- HTTPS termination is the operator's responsibility (reverse proxy / TLS terminator) — the app itself does not enforce or redirect to HTTPS

### Data Model & Storage
- Normalized relational schema: `polls`, `slots`, `participants`, `responses` tables (responses join participant + slot with a Yes/No/Maybe choice; comment lives on the participant's overall response — see Phase 2 for full response shape)
- Schema managed via an embedded `schema.sql` run with idempotent `CREATE TABLE IF NOT EXISTS` statements on startup — no separate migration framework for v1
- SQLite driver: pure-Go (`modernc.org/sqlite`), not CGO-based `mattn/go-sqlite3` — avoids C toolchain / cross-compilation complexity in the Docker multi-stage build
- Comment is one-per-participant-per-poll (attached to their overall response), not one-per-slot — matches VOTE-04

### Docker & Deployment
- Multi-stage Dockerfile: build stage compiles a static Go binary, final stage is `scratch` or `distroless` for minimal image size
- Configuration via environment variables only: `PORT` (default `8080`), `DB_PATH` (default `/data/datechooser.db`)
- Documented volume mount point: `/data`
- App exposes `GET /healthz` returning `200 OK` for container orchestration health checks

### Claude's Discretion
- Exact router/framework choice (e.g. stdlib `net/http` + `html/template` vs a minimal router library) — pick whatever keeps the binary simple and dependency-light
- Exact HTML/CSS approach for the poll-creation form, as long as it's usable on mobile and desktop (server-rendered pages, minimal JS)
- Internal package structure / file layout

</decisions>

<code_context>
## Existing Code Insights

Greenfield project — no existing codebase, no prior conventions to reuse.

### Reusable Assets
- None yet.

### Established Patterns
- None yet — this phase establishes the initial patterns (Go module layout, SQLite access, server-rendered templates, Docker build).

### Integration Points
- None yet — first phase.

</code_context>

<specifics>
## Specific Ideas

Reference products: TallyCal, Doodle, Rallly (github.com/lukevella/rallly) — the poll-creation flow (title/description + list of candidate slots → shareable link) should feel like those tools' creation forms.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. (Voting, results grid, and admin editing are already scoped to Phases 2-4 per the roadmap.)

</deferred>
