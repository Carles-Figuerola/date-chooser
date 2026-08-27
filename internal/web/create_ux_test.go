// create_ux_test.go — mechanical source assertions for Phase 5 (v1.1) slot
// picker UX improvements: SLOT-01 (consistent control height), SLOT-02
// (click-anywhere-opens-dropdown), SLOT-03 (auto-fill end time), and SLOT-04
// (per-row Copy button). These are browser-only behaviors, so — following
// the TestAdminJS_ConfirmCopyAndDoubleSubmitGuard pattern in server_test.go
// — the strongest mechanical check available here is asserting the shipped
// CSS/JS/HTML source carries the required wiring.
package web

import (
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestSlotUX_TimeFieldHeight proves the .time-field input[type="time"] rule
// carries min-height: 44px, matching the sibling ▾/±15 buttons (SLOT-01).
func TestSlotUX_TimeFieldHeight(t *testing.T) {
	data, err := os.ReadFile("static/style.css")
	if err != nil {
		t.Fatalf("reading static/style.css: %v", err)
	}
	content := string(data)

	re := regexp.MustCompile(`\.time-field input\[type="time"\]\s*\{([^}]*)\}`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		t.Fatalf("expected a .time-field input[type=\"time\"] rule in style.css, got none")
	}
	if !strings.Contains(m[1], "min-height: 44px") {
		t.Fatalf("expected .time-field input[type=\"time\"] rule to contain min-height: 44px, got: %s", m[1])
	}
}

// TestSlotUX_ClickAnywhereOpensDropdown proves slot-time-fields.js (shared
// by create.js and edit.js) wires a click listener on the time input itself
// that opens the same dropdown the ▾ toggle opens, and that the handler
// stops propagation so the document-level closeAllTimeDropdowns listener
// doesn't instantly re-close it (SLOT-02).
func TestSlotUX_ClickAnywhereOpensDropdown(t *testing.T) {
	data, err := os.ReadFile("static/slot-time-fields.js")
	if err != nil {
		t.Fatalf("reading static/slot-time-fields.js: %v", err)
	}
	content := string(data)

	idx := strings.Index(content, `input.addEventListener("click"`)
	if idx == -1 {
		t.Fatalf("expected create.js to bind a click listener to the time input, got no match")
	}

	handlerRegion := content[idx:]
	if len(handlerRegion) > 300 {
		handlerRegion = handlerRegion[:300]
	}
	if !strings.Contains(handlerRegion, "stopPropagation") {
		t.Fatalf("expected the input click handler to call evt.stopPropagation(), got: %s", handlerRegion)
	}
}

// TestSlotUX_AutoFillEndTime proves wireRow attaches a change listener to
// the start-time input that fills an empty end time with start+1h and never
// clobbers a non-empty end time (SLOT-03).
func TestSlotUX_AutoFillEndTime(t *testing.T) {
	data, err := os.ReadFile("static/slot-time-fields.js")
	if err != nil {
		t.Fatalf("reading static/slot-time-fields.js: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "slot_start_time") || !strings.Contains(content, `addEventListener("change"`) {
		t.Fatalf("expected slot-time-fields.js to bind a change listener referencing slot_start_time, got no match")
	}

	startIdx := strings.Index(content, `startInput.addEventListener("change"`)
	if startIdx == -1 {
		t.Fatalf("expected a startInput.addEventListener(\"change\", ...) listener in slot-time-fields.js")
	}
	handlerRegion := content[startIdx:]
	if len(handlerRegion) > 1200 {
		handlerRegion = handlerRegion[:1200]
	}

	valueGuardIdx := strings.Index(handlerRegion, "endInput.value")
	assignIdx := strings.Index(handlerRegion, "endInput.value =")
	if valueGuardIdx == -1 || assignIdx == -1 || valueGuardIdx == assignIdx {
		t.Fatalf("expected an empty-guard check on endInput.value before the assignment, got: %s", handlerRegion)
	}
	if valueGuardIdx >= assignIdx {
		t.Fatalf("expected the endInput.value guard to appear before the assignment, guard at %d assign at %d", valueGuardIdx, assignIdx)
	}

	if !strings.Contains(handlerRegion, "+ 60") || !strings.Contains(handlerRegion, "1440") {
		t.Fatalf("expected the +1h wrap math (\"+ 60\" and \"1440\") in the handler, got: %s", handlerRegion)
	}
}

// TestSlotUX_PreserveDurationOnStartEdit proves that when the end time is
// already populated (e.g. right after Copy), changing the start time shifts
// the end time by the previous duration instead of leaving it stale.
func TestSlotUX_PreserveDurationOnStartEdit(t *testing.T) {
	data, err := os.ReadFile("static/slot-time-fields.js")
	if err != nil {
		t.Fatalf("reading static/slot-time-fields.js: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "previousStartValue") {
		t.Fatalf("expected slot-time-fields.js to track a previousStartValue for the start-time change handler, got no match")
	}

	startIdx := strings.Index(content, `startInput.addEventListener("change"`)
	if startIdx == -1 {
		t.Fatalf("expected a startInput.addEventListener(\"change\", ...) listener in slot-time-fields.js")
	}
	handlerRegion := content[startIdx:]
	if len(handlerRegion) > 1600 {
		handlerRegion = handlerRegion[:1600]
	}

	if !strings.Contains(handlerRegion, "parseTimeValue(previousStartValue)") {
		t.Fatalf("expected the handler to parse the previous start value to compute the existing duration, got: %s", handlerRegion)
	}
	durationIdx := strings.Index(handlerRegion, "var duration")
	assignIdx := strings.LastIndex(handlerRegion, "endInput.value =")
	if durationIdx == -1 || assignIdx == -1 || durationIdx >= assignIdx {
		t.Fatalf("expected a duration computed from the previous start/end before the final endInput.value assignment, got: %s", handlerRegion)
	}
}

// TestSlotUX_EditPageSharesTimeFieldWiring proves edit.html loads the same
// slot-time-fields.js module as create.html and carries the same
// data-time-field/data-date-input markup, so the click-anywhere dropdown,
// ±15 steppers, and start/end duration-preserving behavior aren't limited to
// the create-poll page (a real gap: Phase 5/v1.1 only touched create.html).
func TestSlotUX_EditPageSharesTimeFieldWiring(t *testing.T) {
	html, err := os.ReadFile("templates/edit.html")
	if err != nil {
		t.Fatalf("reading templates/edit.html: %v", err)
	}
	htmlContent := string(html)

	if !strings.Contains(htmlContent, `src="/static/slot-time-fields.js"`) {
		t.Fatalf("expected edit.html to load /static/slot-time-fields.js, got no match")
	}
	if !strings.Contains(htmlContent, "data-time-field") {
		t.Fatalf("expected edit.html to carry data-time-field markup, got no match")
	}
	if !strings.Contains(htmlContent, "data-date-input") {
		t.Fatalf("expected edit.html to carry data-date-input markup, got no match")
	}
	if strings.Count(htmlContent, "data-time-field") < 4 {
		t.Fatalf("expected data-time-field to appear at least 4 times (server-rendered row + 2 templates, 2 fields each), got %d", strings.Count(htmlContent, "data-time-field"))
	}

	js, err := os.ReadFile("static/edit.js")
	if err != nil {
		t.Fatalf("reading static/edit.js: %v", err)
	}
	if !strings.Contains(string(js), "window.DateChooserSlotFields.wireRow(row)") {
		t.Fatalf("expected edit.js's wireRow to call window.DateChooserSlotFields.wireRow(row), got no match")
	}
}

// TestSlotUX_CopyButton_Markup proves create.html carries a data-copy-slot
// "Copy" button in the server-rendered row loop and in both row templates
// (SLOT-04).
func TestSlotUX_CopyButton_Markup(t *testing.T) {
	data, err := os.ReadFile("templates/create.html")
	if err != nil {
		t.Fatalf("reading templates/create.html: %v", err)
	}
	content := string(data)

	count := strings.Count(content, "data-copy-slot")
	if count < 3 {
		t.Fatalf("expected data-copy-slot to appear at least 3 times (server-rendered row + 2 templates), got %d", count)
	}
	if !strings.Contains(content, ">Copy<") {
		t.Fatalf("expected a Copy button labeled \"Copy\" in create.html, got no match")
	}
}

// TestSlotUX_CopyButton_JS proves wireRow wires the copy button using the
// existing row-template + wireRow + afterRowChange machinery, appending to
// the end of the list rather than inserting after the original (SLOT-04).
func TestSlotUX_CopyButton_JS(t *testing.T) {
	data, err := os.ReadFile("static/create.js")
	if err != nil {
		t.Fatalf("reading static/create.js: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "data-copy-slot") {
		t.Fatalf("expected create.js to reference data-copy-slot, got no match")
	}
	if !strings.Contains(content, "rowTemplateFor") {
		t.Fatalf("expected the copy handler to reuse rowTemplateFor, got no match")
	}
	if !strings.Contains(content, "appendChild") {
		t.Fatalf("expected the copy handler to append the new row (not insertBefore), got no match")
	}
	if !strings.Contains(content, "afterRowChange") {
		t.Fatalf("expected the copy handler to call afterRowChange(), got no match")
	}
}

// TestSlotUX_CopyButton_RenderedForm proves the initial server-rendered
// create-poll page includes the Copy button on its first slot row.
func TestSlotUX_CopyButton_RenderedForm(t *testing.T) {
	ts, _ := newTestServer(t)
	resp, err := noRedirectClient().Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / error: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "data-copy-slot") {
		t.Fatalf("expected the server-rendered create-poll page to contain data-copy-slot, got: %s", body)
	}
}
