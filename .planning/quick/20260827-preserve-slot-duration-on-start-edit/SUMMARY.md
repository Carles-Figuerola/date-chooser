---
slug: preserve-slot-duration-on-start-edit
status: complete
date: 2026-08-27
---

Fixed `internal/web/static/create.js`: the start-time `change` listener now
tracks each row's previous start value (captured at wire time, so a
Copy-cloned row starts from the cloned value). When the end time is already
populated, changing the start time shifts the end time to preserve the
previous duration (e.g. 8:00–10:00, start changed to 3:00, end becomes
5:00) instead of leaving it stale. The existing SLOT-03 empty-end-time
auto-fill (+1h) is unchanged.

Added `TestSlotUX_PreserveDurationOnStartEdit` in
`internal/web/create_ux_test.go` (source-assertion style, matching the
existing tests in that file). All tests pass, `go vet`/`gofmt` clean.
