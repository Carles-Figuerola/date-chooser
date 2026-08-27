# Deferred Items — Phase 3 (Results Grid)

Out-of-scope discoveries logged per the executor's scope-boundary rule (fix only what the
current task's changes directly caused; pre-existing issues are logged here, not fixed).

## 03-02 Task 2: no-new-hex verify gate's allow-list is missing a pre-existing color

- **Found during:** 03-02 Task 2 (`go test` + verify gate execution)
- **What:** The plan's `<verify>` command for the no-new-hex gate greps
  `internal/web/static/style.css` for hex colors and excludes an allow-list that does not
  include `#FEF2F2`. `#FEF2F2` is `.banner-error`'s background, added in Phase 1/2 (confirmed
  via `git log -p` — present before this plan touched the file), not introduced by this plan.
  The literal grep command therefore reports a false failure.
- **Why deferred, not fixed:** This is a gap in the plan's own verify command (an incomplete
  allow-list), not a defect in `internal/web/static/style.css`. Fixing it would mean editing the
  plan's `<verify>` text, which is out of this task's scope (the task's scope is CSS + WINDOWS.md
  entries, not amending PLAN.md verification commands post hoc). No pre-existing selector's
  declarations were changed by this task, and every hex value this task added
  (`#EFF6FF` for the best-fit row/badge, reusing the same value already used by
  `.banner-success`/`.link-box-secret`) is already on the intended allow-list.
- **Verification performed manually:** `grep -oE '#[0-9A-Fa-f]{6}' internal/web/static/style.css | sort -u`
  lists exactly 10 distinct hex values in the file: the 9 on the plan's allow-list plus
  `#FEF2F2` (pre-existing, Phase 1/2, `.banner-error`). Zero net-new hex values were added by
  this task.
- **Recommendation:** If a future phase re-runs a similar no-new-hex gate, extend its allow-list
  to include `#FEF2F2` alongside the other 9 established Phase 1/2 tokens.
