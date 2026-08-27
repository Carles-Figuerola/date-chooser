---
schema_version: 1
open_count: 7
waived_count: 0
fixed_count: 3
total_count: 10
last_updated: 2026-08-25T18:42:50.966Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 01 | unrun-verify | internal/web/static/create.js |  | Submit double-post guard (disable button + preventDefault on second submit) implemented but not exercised by Go httptest — no browser JS execution in this harness; needs manual/UI-review confirmation. | open |  | 2026-08-24T20:07:31.146Z |  |
| 2 | 01 | unrun-verify | internal/web/static/copy.js |  | Clipboard copy/'Copied!'/fallback-reveal behavior code-reviewed and grep-verified only — not executed in a real browser (Go httptest cannot run JS). | open |  | 2026-08-24T20:13:40.475Z |  |
| 3 | 01 | unrun-verify | internal/web/static/style.css |  | UI-SPEC backstop: long token URLs must not cause horizontal scroll at a ~360px viewport — CSS in place (word-break: break-all, overflow-wrap) but unverified in an actual narrow-viewport render. | open |  | 2026-08-24T20:13:40.606Z |  |
| 4 | 02 | stub | internal/web/server.go |  | Unknown /poll/{ptoken} returns plain http.NotFound instead of the branded 404 copy - resolved in Plan 02-02 | fixed |  | 2026-08-25T15:55:09.294Z | 2026-08-25T16:02:56.283Z |
| 5 | 02 | stub | internal/web/server.go |  | Invalid vote submission returns generic 400 banner, not per-field inline errors - resolved in Plan 02-02 | fixed |  | 2026-08-25T15:55:09.432Z | 2026-08-25T16:02:56.436Z |
| 6 | 02 | stub | internal/web/templates/vote.html |  | Yes/No/Maybe renders as native radios, not the pill-button group - resolved in Plan 02-03 | fixed |  | 2026-08-25T15:55:09.567Z | 2026-08-25T16:07:01.118Z |
| 7 | 02 | unrun-verify | internal/web/static/style.css |  | Pill tri-color checked states, 44px tap targets, and no horizontal overflow at ~360px viewport - CSS in place but unverified in a real browser (Go httptest cannot render CSS). | open |  | 2026-08-25T16:07:01.266Z |  |
| 8 | 02 | unrun-verify | internal/web/static/vote.js |  | Submit-guard behavior (disable + 'Saving...' label, blocks a second POST) and the JS-disabled fallback submit - implemented and grep-verified only, not executed in a real browser (Go httptest cannot run JS). | open |  | 2026-08-25T16:07:01.398Z |  |
| 9 | 03 | unrun-verify | internal/web/static/style.css |  | Sticky slot-label column (position:sticky; left:0) must stay visible while the participant columns scroll horizontally at a ~360px viewport - CSS in place but unverified in a real browser (Go httptest cannot render CSS). | open |  | 2026-08-25T18:42:50.823Z |  |
| 10 | 03 | unrun-verify | internal/web/templates/results.html |  | Participant column-header ellipsis truncation (max-width:140px) must actually trigger for a long display name, with the full name still recoverable via the native title attribute - implemented per UI-SPEC but unverified in a real browser (Go httptest cannot render CSS truncation). | open |  | 2026-08-25T18:42:50.966Z |  |

````json
[
  {
    "id": 1,
    "kind": "unrun-verify",
    "phase": "01",
    "file": "internal/web/static/create.js",
    "line": null,
    "description": "Submit double-post guard (disable button + preventDefault on second submit) implemented but not exercised by Go httptest — no browser JS execution in this harness; needs manual/UI-review confirmation.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-24T20:07:31.146Z",
    "resolved_at": null
  },
  {
    "id": 2,
    "kind": "unrun-verify",
    "phase": "01",
    "file": "internal/web/static/copy.js",
    "line": null,
    "description": "Clipboard copy/'Copied!'/fallback-reveal behavior code-reviewed and grep-verified only — not executed in a real browser (Go httptest cannot run JS).",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-24T20:13:40.475Z",
    "resolved_at": null
  },
  {
    "id": 3,
    "kind": "unrun-verify",
    "phase": "01",
    "file": "internal/web/static/style.css",
    "line": null,
    "description": "UI-SPEC backstop: long token URLs must not cause horizontal scroll at a ~360px viewport — CSS in place (word-break: break-all, overflow-wrap) but unverified in an actual narrow-viewport render.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-24T20:13:40.606Z",
    "resolved_at": null
  },
  {
    "id": 4,
    "kind": "stub",
    "phase": "02",
    "file": "internal/web/server.go",
    "line": null,
    "description": "Unknown /poll/{ptoken} returns plain http.NotFound instead of the branded 404 copy - resolved in Plan 02-02",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-25T15:55:09.294Z",
    "resolved_at": "2026-08-25T16:02:56.283Z"
  },
  {
    "id": 5,
    "kind": "stub",
    "phase": "02",
    "file": "internal/web/server.go",
    "line": null,
    "description": "Invalid vote submission returns generic 400 banner, not per-field inline errors - resolved in Plan 02-02",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-25T15:55:09.432Z",
    "resolved_at": "2026-08-25T16:02:56.436Z"
  },
  {
    "id": 6,
    "kind": "stub",
    "phase": "02",
    "file": "internal/web/templates/vote.html",
    "line": null,
    "description": "Yes/No/Maybe renders as native radios, not the pill-button group - resolved in Plan 02-03",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-25T15:55:09.567Z",
    "resolved_at": "2026-08-25T16:07:01.118Z"
  },
  {
    "id": 7,
    "kind": "unrun-verify",
    "phase": "02",
    "file": "internal/web/static/style.css",
    "line": null,
    "description": "Pill tri-color checked states, 44px tap targets, and no horizontal overflow at ~360px viewport - CSS in place but unverified in a real browser (Go httptest cannot render CSS).",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-25T16:07:01.266Z",
    "resolved_at": null
  },
  {
    "id": 8,
    "kind": "unrun-verify",
    "phase": "02",
    "file": "internal/web/static/vote.js",
    "line": null,
    "description": "Submit-guard behavior (disable + 'Saving...' label, blocks a second POST) and the JS-disabled fallback submit - implemented and grep-verified only, not executed in a real browser (Go httptest cannot run JS).",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-25T16:07:01.398Z",
    "resolved_at": null
  },
  {
    "id": 9,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "internal/web/static/style.css",
    "line": null,
    "description": "Sticky slot-label column (position:sticky; left:0) must stay visible while the participant columns scroll horizontally at a ~360px viewport - CSS in place but unverified in a real browser (Go httptest cannot render CSS).",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-25T18:42:50.823Z",
    "resolved_at": null
  },
  {
    "id": 10,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "internal/web/templates/results.html",
    "line": null,
    "description": "Participant column-header ellipsis truncation (max-width:140px) must actually trigger for a long display name, with the full name still recoverable via the native title attribute - implemented per UI-SPEC but unverified in a real browser (Go httptest cannot render CSS truncation).",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-25T18:42:50.966Z",
    "resolved_at": null
  }
]
````
