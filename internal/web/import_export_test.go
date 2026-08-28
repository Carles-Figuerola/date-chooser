// import_export_test.go — mechanical source assertions for Phase 6 (v1.2)
// slot import/export (IMPORT-01..03). Following create_ux_test.go's pattern,
// these check the shipped JS/HTML source carries the required wiring.
package web

import (
	"os"
	"strings"
	"testing"
)

// TestImportExport_SharedParseAndFormatFunctions proves slot-time-fields.js
// exposes slotsToText/parseSlotsText and that parsing skips (and counts,
// rather than silently drops) unparsable lines while leaving blank lines
// uncounted (IMPORT-01, IMPORT-03).
func TestImportExport_SharedParseAndFormatFunctions(t *testing.T) {
	data, err := os.ReadFile("static/slot-time-fields.js")
	if err != nil {
		t.Fatalf("reading static/slot-time-fields.js: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "function slotsToText(") {
		t.Fatalf("expected a slotsToText function in slot-time-fields.js, got no match")
	}
	if !strings.Contains(content, "function parseSlotsText(") {
		t.Fatalf("expected a parseSlotsText function in slot-time-fields.js, got no match")
	}
	if !strings.Contains(content, "skipped") {
		t.Fatalf("expected parseSlotsText to track a skipped-line count, got no match")
	}
	if !strings.Contains(content, "slotsToText: slotsToText") || !strings.Contains(content, "parseSlotsText: parseSlotsText") {
		t.Fatalf("expected both functions exported on the DateChooserSlotFields return object, got no match")
	}
}

// TestImportExport_CreatePageMarkupAndWiring proves create.html carries the
// Export/Import controls and create.js wires them using the shared
// slot-time-fields.js functions, triggering a real download for Export
// (IMPORT-01) and replacing the row list for Import (IMPORT-02).
func TestImportExport_CreatePageMarkupAndWiring(t *testing.T) {
	html, err := os.ReadFile("templates/create.html")
	if err != nil {
		t.Fatalf("reading templates/create.html: %v", err)
	}
	htmlContent := string(html)

	for _, id := range []string{"export-slots-btn", "import-slots-btn", "import-slots-file", "import-slots-message"} {
		if !strings.Contains(htmlContent, `id="`+id+`"`) {
			t.Fatalf("expected create.html to contain id=%q, got no match", id)
		}
	}

	js, err := os.ReadFile("static/create.js")
	if err != nil {
		t.Fatalf("reading static/create.js: %v", err)
	}
	jsContent := string(js)

	if !strings.Contains(jsContent, "DateChooserSlotFields.slotsToText(") {
		t.Fatalf("expected create.js to call DateChooserSlotFields.slotsToText, got no match")
	}
	if !strings.Contains(jsContent, "DateChooserSlotFields.parseSlotsText(") {
		t.Fatalf("expected create.js to call DateChooserSlotFields.parseSlotsText, got no match")
	}
	if !strings.Contains(jsContent, "createObjectURL") {
		t.Fatalf("expected create.js's export handler to trigger a real file download via createObjectURL, got no match")
	}
	if !strings.Contains(jsContent, `slotsList.innerHTML = ""`) {
		t.Fatalf("expected create.js's import handler to replace the current row list, got no match")
	}
}

// TestImportExport_EditPageHasNoImportExportControls proves Import/Export
// is scoped to the create page only, per explicit user direction — edit.html
// must not carry these controls, and edit.js must not reference the shared
// slotsToText/parseSlotsText functions.
func TestImportExport_EditPageHasNoImportExportControls(t *testing.T) {
	html, err := os.ReadFile("templates/edit.html")
	if err != nil {
		t.Fatalf("reading templates/edit.html: %v", err)
	}
	htmlContent := string(html)

	for _, id := range []string{"export-slots-btn", "import-slots-btn", "import-slots-file", "import-slots-message"} {
		if strings.Contains(htmlContent, `id="`+id+`"`) {
			t.Fatalf("expected edit.html to NOT contain id=%q (Import/Export is create-page only), but found it", id)
		}
	}

	js, err := os.ReadFile("static/edit.js")
	if err != nil {
		t.Fatalf("reading static/edit.js: %v", err)
	}
	jsContent := string(js)

	if strings.Contains(jsContent, "DateChooserSlotFields.slotsToText(") || strings.Contains(jsContent, "DateChooserSlotFields.parseSlotsText(") {
		t.Fatalf("expected edit.js to NOT reference slotsToText/parseSlotsText (Import/Export is create-page only), but found a reference")
	}
}
