// create.js — progressive-enhancement behavior for the poll-creation form.
// The form is fully submittable server-side without this script; this file
// only adds add/remove-row affordances, the live counter, the all-day vs
// date+time toggle, the soft ~50-slot hint, and the submit double-click
// guard described in 01-UI-SPEC.md.
(function () {
  "use strict";

  var form = document.getElementById("create-poll-form");
  if (!form) {
    return;
  }

  var slotsList = document.getElementById("slots-list");
  var addBtn = document.getElementById("add-slot-btn");
  var exportBtn = document.getElementById("export-slots-btn");
  var importBtn = document.getElementById("import-slots-btn");
  var importFile = document.getElementById("import-slots-file");
  var importMessage = document.getElementById("import-slots-message");
  var counter = document.getElementById("slot-counter");
  var hint = document.getElementById("slot-hint");
  var submitBtn = document.getElementById("submit-btn");
  var toggleGroup = document.getElementById("poll-type-toggle");

  var HINT_THRESHOLD = 50;
  var submitted = false;

  function currentMode() {
    var checked = toggleGroup && toggleGroup.querySelector('input[name="poll_type"]:checked');
    return checked ? checked.value : "all_day";
  }

  function rowTemplateFor(mode) {
    var tpl = document.getElementById("slot-row-template-" + mode);
    if (!tpl || !tpl.content || !tpl.content.firstElementChild) {
      return null;
    }
    return tpl.content.firstElementChild.cloneNode(true);
  }

  function rows() {
    return Array.prototype.slice.call(slotsList.querySelectorAll("[data-slot-row]"));
  }

  function rowValues(row) {
    var get = function (name) {
      var input = row.querySelector('[name="' + name + '"]');
      return input ? input.value : "";
    };
    return { date: get("slot_date"), start: get("slot_start_time"), end: get("slot_end_time") };
  }

  function wireRow(row) {
    window.DateChooserSlotFields.wireRow(row);

    // SLOT-04: Copy clones the current mode's blank row template, writes
    // this row's live date/start/end values into it, wires it (so its own
    // copy/remove/time-field/auto-fill behavior works), and appends it to
    // the END of the list — never inserted right after the original, and
    // the original row is never modified.
    var copyBtn = row.querySelector("[data-copy-slot]");
    if (copyBtn) {
      copyBtn.addEventListener("click", function () {
        var newRow = rowTemplateFor(currentMode());
        if (!newRow) {
          return;
        }
        ["slot_date", "slot_start_time", "slot_end_time"].forEach(function (name) {
          var src = row.querySelector('[name="' + name + '"]');
          var dst = newRow.querySelector('[name="' + name + '"]');
          if (src && dst) {
            dst.value = src.value;
          }
        });
        wireRow(newRow);
        slotsList.appendChild(newRow);
        afterRowChange();
      });
    }

    var removeBtn = row.querySelector("[data-remove-slot]");
    if (!removeBtn) {
      return;
    }
    removeBtn.addEventListener("click", function () {
      row.parentNode && row.parentNode.removeChild(row);
      afterRowChange();
    });
  }

  function updateCounter() {
    if (!counter) {
      return;
    }
    var n = rows().length;
    counter.textContent = n === 1 ? "1 slot added" : n + " slots added";
  }

  function updateRemoveVisibility() {
    var all = rows();
    var onlyOneLeft = all.length <= 1;
    all.forEach(function (row) {
      var removeBtn = row.querySelector("[data-remove-slot]");
      if (!removeBtn) {
        return;
      }
      removeBtn.hidden = onlyOneLeft;
      removeBtn.disabled = onlyOneLeft;
    });
  }

  function updateHint() {
    if (!hint) {
      return;
    }
    hint.hidden = rows().length <= HINT_THRESHOLD;
  }

  function afterRowChange() {
    updateCounter();
    updateRemoveVisibility();
    updateHint();
  }

  function addRow() {
    var row = rowTemplateFor(currentMode());
    if (!row) {
      return;
    }
    wireRow(row);
    slotsList.appendChild(row);
    afterRowChange();
  }

  function rebuildRowsForMode(mode) {
    var count = rows().length || 1;
    slotsList.innerHTML = "";
    for (var i = 0; i < count; i++) {
      var row = rowTemplateFor(mode);
      if (!row) {
        continue;
      }
      wireRow(row);
      slotsList.appendChild(row);
    }
    slotsList.setAttribute("data-mode", mode);
    afterRowChange();
  }

  // Wire up whatever rows the server already rendered.
  rows().forEach(wireRow);
  afterRowChange();

  if (addBtn) {
    addBtn.addEventListener("click", addRow);
  }

  if (toggleGroup) {
    toggleGroup.addEventListener("change", function (evt) {
      if (evt.target && evt.target.name === "poll_type") {
        var options = toggleGroup.querySelectorAll(".toggle-option");
        options.forEach(function (opt) {
          var input = opt.querySelector('input[name="poll_type"]');
          if (input && input.checked) {
            opt.classList.add("toggle-active");
          } else {
            opt.classList.remove("toggle-active");
          }
        });
        rebuildRowsForMode(evt.target.value);
      }
    });
  }

  // IMPORT-01: Export dumps every current row as plain text (one line per
  // slot) and triggers a browser download — no server round-trip.
  if (exportBtn) {
    exportBtn.addEventListener("click", function () {
      var mode = currentMode();
      var slots = rows().map(rowValues);
      var text = window.DateChooserSlotFields.slotsToText(mode, slots);
      var blob = new Blob([text], { type: "text/plain" });
      var url = URL.createObjectURL(blob);
      var a = document.createElement("a");
      a.href = url;
      a.download = "slots.txt";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    });
  }

  // IMPORT-02/03: Import reads the chosen file, parses it against the
  // current mode, and REPLACES every current row with the parsed result —
  // never a partial/merged apply. Unparsable lines are skipped and counted
  // rather than blocking the whole import.
  if (importBtn && importFile) {
    importBtn.addEventListener("click", function () {
      importFile.click();
    });
    importFile.addEventListener("change", function () {
      var file = importFile.files && importFile.files[0];
      if (!file) {
        return;
      }
      var reader = new FileReader();
      reader.onload = function () {
        var mode = currentMode();
        var result = window.DateChooserSlotFields.parseSlotsText(mode, String(reader.result));
        slotsList.innerHTML = "";
        result.slots.forEach(function (s) {
          var row = rowTemplateFor(mode);
          if (!row) {
            return;
          }
          ["slot_date", "slot_start_time", "slot_end_time"].forEach(function (name) {
            var input = row.querySelector('[name="' + name + '"]');
            if (!input) {
              return;
            }
            if (name === "slot_date") {
              input.value = s.date || "";
            } else if (name === "slot_start_time") {
              input.value = s.start || "";
            } else if (name === "slot_end_time") {
              input.value = s.end || "";
            }
          });
          wireRow(row);
          slotsList.appendChild(row);
        });
        if (rows().length === 0) {
          addRow();
        }
        afterRowChange();
        if (importMessage) {
          if (result.skipped > 0) {
            var total = result.slots.length + result.skipped;
            importMessage.textContent = result.skipped + " of " + total + " line(s) could not be read and were skipped.";
            importMessage.hidden = false;
          } else {
            importMessage.hidden = true;
          }
        }
        importFile.value = "";
      };
      reader.readAsText(file);
    });
  }

  // Submit guard: disable the button and swap its label so a second click
  // (or a double form submission) is a genuine no-op, not just a re-style.
  form.addEventListener("submit", function (evt) {
    if (submitted) {
      evt.preventDefault();
      return;
    }
    submitted = true;
    if (submitBtn) {
      submitBtn.disabled = true;
      submitBtn.textContent = "Creating…";
    }
  });
})();
