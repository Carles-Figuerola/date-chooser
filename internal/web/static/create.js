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
