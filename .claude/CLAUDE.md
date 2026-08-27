<!-- GSD:project-start source:PROJECT.md -->

## Project

**Date Chooser**

A self-hosted, account-free meeting scheduler (a Doodle/Rallly-style poll tool). An organizer creates a poll with a set of candidate dates/times, shares a link with a group, and each invitee marks Yes/No/Maybe (with an optional comment) per slot. The organizer gets a secret admin link to view results and manage the poll. Runs as a single Docker container backed by SQLite on a mounted volume.

**Core Value:** An organizer can create a poll of date/time options and get back, without any signup, a clear grid of who's available when — so they can pick the best slot.

### Constraints

- **Tech stack**: Backend in Go (decided over Python — see Key Decisions) — Why: user is comfortable with both; Go produces a single static binary and a minimal Docker image, which fits the "one container + one volume" deployment model cleanly.
- **Database**: SQLite only — Why: explicitly requested by user; avoids running/operating a separate DB service.
- **Auth model**: No user accounts. Access control is via unguessable link tokens (participant link, admin link) — Why: explicitly requested; user is fine with treating the admin link as a secret to store safely.
- **Deployment**: Must run as a Docker container with the SQLite file on a mounted volume — Why: explicitly requested deployment shape.
- **Interface**: Must be usable from both desktop browsers and mobile — Why: participants will respond from whatever device they have.

<!-- GSD:project-end -->

<!-- GSD:stack-start source:STACK.md -->

## Technology Stack

Technology stack not yet documented. Will populate after codebase mapping or first phase.
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:

- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
