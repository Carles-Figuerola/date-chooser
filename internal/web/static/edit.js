// edit.js — progressive-enhancement behavior for the poll-edit form. The
// form is fully submittable server-side without this script (a plain
// slot_id/slot_removed/slot_date field set is enough); this file adds:
// add/remove-slot-row behavior branched on each row's response count, the
// aggregate removal warning + required confirm checkbox gate, the submit
// double-post guard (mirroring create.js/edit.js's Task 1 guard), and (Plan
// 04-03) the danger-zone "Delete poll" confirm()-gate.
(function () {
  "use strict";

  // Delete-poll confirm() gate (Plan 04-03, ADM-04) — lives here, not in
  // admin.js, per 04-CONTEXT.md ("editing happens on a separate route");
  // the danger zone is part of the edit page, not the results/admin page.
  // Mirrors admin.js's shape: a native confirm() using the exact
  // 04-UI-SPEC.md copy built from data-poll-title/data-response-count
  // attributes the template writes onto the form (never invented
  // client-side), then a disable-on-confirm guard so a double-click cannot
  // fire two delete submits (T-04-04's double-submit backstop). The
  // confirm() dialog is a UX safety net only, not the authorization
  // control (that's possession of the admin token in the URL); a repeated
  // delete is also safe server-side (store.DeletePoll is a no-op on an
  // already-deleted or non-existent id).
  document.addEventListener("submit", function (event) {
    var deleteForm = event.target;
    if (!deleteForm.classList || !deleteForm.classList.contains("delete-poll-form")) {
      return;
    }

    var title = deleteForm.getAttribute("data-poll-title") || "";
    var count = deleteForm.getAttribute("data-response-count") || "0";
    var confirmed = window.confirm(
      "Delete \"" + title + "\"? This will permanently delete the poll and its " +
        count + " response(s). This cannot be undone."
    );
    if (!confirmed) {
      event.preventDefault();
      return;
    }

    var deleteButton = deleteForm.querySelector("button[type=submit]");
    if (deleteButton) {
      deleteButton.disabled = true;
    }
  });

  var form = document.getElementById("edit-poll-form");
  if (!form) {
    return;
  }

  var slotsList = document.getElementById("slots-list");
  var addBtn = document.getElementById("add-slot-btn");
  var submitBtn = document.getElementById("submit-btn");
  var aggregateWarning = document.getElementById("slot-removal-aggregate-warning");
  var aggregateWarningText = aggregateWarning && aggregateWarning.querySelector("[data-aggregate-warning-text]");
  var confirmCheckbox = document.getElementById("confirm-slot-removal");
  var submitted = false;

  function rows() {
    return Array.prototype.slice.call(slotsList.querySelectorAll("[data-slot-row]"));
  }

  function responseCount(row) {
    return parseInt(row.getAttribute("data-response-count"), 10) || 0;
  }

  function isMarked(row) {
    return row.classList.contains("slot-row-marked");
  }

  // updateSaveGate recomputes the aggregate removal warning, the required
  // confirm checkbox's visibility, and whether "Save changes" may be
  // clicked at all. Called after every mark/unmark/add/remove.
  function updateSaveGate() {
    var markedRows = rows().filter(isMarked);
    var slotCount = markedRows.length;
    var responseTotal = markedRows.reduce(function (sum, row) {
      return sum + responseCount(row);
    }, 0);

    if (slotCount === 0) {
      if (aggregateWarning) {
        aggregateWarning.hidden = true;
      }
      if (confirmCheckbox) {
        confirmCheckbox.checked = false;
      }
      if (submitBtn) {
        submitBtn.disabled = false;
      }
      return;
    }

    if (aggregateWarning) {
      aggregateWarning.hidden = false;
    }
    if (aggregateWarningText) {
      var slotWord = slotCount === 1 ? "slot" : "slot(s)";
      var responseWord = responseTotal === 1 ? "response(s)" : "response(s)";
      aggregateWarningText.textContent =
        "Removing " + slotCount + " " + slotWord + " will permanently delete " +
        responseTotal + " " + responseWord + " tied to those slots.";
    }
    if (submitBtn) {
      submitBtn.disabled = !(confirmCheckbox && confirmCheckbox.checked);
    }
  }

  function markRow(row) {
    row.classList.add("slot-row-marked");
    var idInput = row.querySelector("[data-slot-removed]");
    if (idInput) {
      idInput.value = "1";
    }
    var fields = row.querySelector("[data-slot-row-fields]");
    if (fields) {
      fields.setAttribute("aria-hidden", "true");
    }
    var removeBtn = row.querySelector("[data-remove-slot]");
    var undoBtn = row.querySelector("[data-undo-slot]");
    if (removeBtn) {
      removeBtn.hidden = true;
    }
    if (undoBtn) {
      undoBtn.hidden = false;
    }
    var warning = row.querySelector("[data-slot-warning]");
    if (warning) {
      var count = responseCount(row);
      warning.textContent =
        "This slot has " + count + " response(s). Removing it will permanently delete those responses.";
      warning.hidden = false;
    }
  }

  function unmarkRow(row) {
    row.classList.remove("slot-row-marked");
    var idInput = row.querySelector("[data-slot-removed]");
    if (idInput) {
      idInput.value = "";
    }
    var fields = row.querySelector("[data-slot-row-fields]");
    if (fields) {
      fields.removeAttribute("aria-hidden");
    }
    var removeBtn = row.querySelector("[data-remove-slot]");
    var undoBtn = row.querySelector("[data-undo-slot]");
    if (removeBtn) {
      removeBtn.hidden = false;
    }
    if (undoBtn) {
      undoBtn.hidden = true;
    }
    var warning = row.querySelector("[data-slot-warning]");
    if (warning) {
      warning.hidden = true;
    }
  }

  function updateRemoveVisibility() {
    var all = rows();
    var unmarkedCount = all.filter(function (row) {
      return !isMarked(row);
    }).length;
    var onlyOneLeft = unmarkedCount <= 1;
    all.forEach(function (row) {
      if (isMarked(row)) {
        return;
      }
      var removeBtn = row.querySelector("[data-remove-slot]");
      if (!removeBtn) {
        return;
      }
      removeBtn.hidden = onlyOneLeft;
      removeBtn.disabled = onlyOneLeft;
    });
  }

  function wireRow(row) {
    window.DateChooserSlotFields.wireRow(row);

    // Copy clones this row's current date/start/end values into a new,
    // always-blank-slate row (no slot_id, 0 response count) appended to the
    // end of the list — mirrors create.js's Copy button (SLOT-04). The
    // original row is never modified.
    var copyBtn = row.querySelector("[data-copy-slot]");
    if (copyBtn) {
      copyBtn.addEventListener("click", function () {
        var tpl = document.getElementById("slot-row-template-" + currentMode());
        if (!tpl || !tpl.content || !tpl.content.firstElementChild) {
          return;
        }
        var newRow = tpl.content.firstElementChild.cloneNode(true);
        ["slot_date", "slot_start_time", "slot_end_time"].forEach(function (name) {
          var src = row.querySelector('[name="' + name + '"]');
          var dst = newRow.querySelector('[name="' + name + '"]');
          if (src && dst) {
            dst.value = src.value;
          }
        });
        wireRow(newRow);
        slotsList.appendChild(newRow);
        updateRemoveVisibility();
        updateSaveGate();
      });
    }

    var removeBtn = row.querySelector("[data-remove-slot]");
    var undoBtn = row.querySelector("[data-undo-slot]");

    if (removeBtn) {
      removeBtn.addEventListener("click", function () {
        if (responseCount(row) > 0) {
          // Real, irreversible data loss if saved — mark instead of
          // removing from the DOM, per 04-CONTEXT.md.
          markRow(row);
          updateRemoveVisibility();
          updateSaveGate();
          return;
        }
        // Zero-response row: reversible, remove immediately (unchanged
        // Phase 1 behavior).
        row.parentNode && row.parentNode.removeChild(row);
        updateRemoveVisibility();
        updateSaveGate();
      });
    }

    if (undoBtn) {
      undoBtn.addEventListener("click", function () {
        unmarkRow(row);
        updateRemoveVisibility();
        updateSaveGate();
      });
    }
  }

  function currentMode() {
    var toggleGroup = document.getElementById("poll-type-toggle");
    var checked = toggleGroup && toggleGroup.querySelector('input[name="poll_type"]:checked');
    return checked ? checked.value : "all_day";
  }

  function addRow() {
    var tpl = document.getElementById("slot-row-template-" + currentMode());
    if (!tpl || !tpl.content || !tpl.content.firstElementChild) {
      return;
    }
    var row = tpl.content.firstElementChild.cloneNode(true);
    wireRow(row);
    slotsList.appendChild(row);
    updateRemoveVisibility();
    updateSaveGate();
  }

  // Wire up whatever rows the server already rendered.
  rows().forEach(wireRow);
  updateRemoveVisibility();
  updateSaveGate();

  if (addBtn) {
    addBtn.addEventListener("click", addRow);
  }

  if (confirmCheckbox) {
    confirmCheckbox.addEventListener("change", updateSaveGate);
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
      submitBtn.textContent = "Saving…";
    }
  });
})();
