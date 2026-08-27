---
schema_version: 1
open_count: 3
waived_count: 0
fixed_count: 0
total_count: 3
last_updated: 2026-08-24T20:13:40.606Z
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
  }
]
````
