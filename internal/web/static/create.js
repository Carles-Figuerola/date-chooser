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

  // DEFAULT_HOUR is where an opened time dropdown scrolls to by default —
  // most polls propose slots during working hours, so 9am is a better
  // starting point than midnight.
  var DEFAULT_HOUR = 9;

  function pad2(n) {
    return (n < 10 ? "0" : "") + n;
  }

  function parseTimeValue(v) {
    var m = /^(\d{2}):(\d{2})$/.exec(v || "");
    if (!m) {
      return null;
    }
    return { h: parseInt(m[1], 10), m: parseInt(m[2], 10) };
  }

  function formatTimeValue(h, m) {
    return pad2(h) + ":" + pad2(m);
  }

  function closeAllTimeDropdowns() {
    Array.prototype.slice.call(document.querySelectorAll("[data-time-dropdown]")).forEach(function (list) {
      list.hidden = true;
    });
  }

  // wireTimeField adds the hour-dropdown and +/-15-minute stepper
  // affordances on top of the native <input type="time">, which remains
  // the real, always-functional form field (typing or its native picker
  // still work).
  function wireTimeField(field) {
    var input = field.querySelector("[data-time-input]");
    var list = field.querySelector("[data-time-dropdown]");
    if (!input || !list) {
      return;
    }

    function buildOptions() {
      if (list.childElementCount) {
        return;
      }
      for (var h = 0; h < 24; h++) {
        (function (hour) {
          var li = document.createElement("li");
          var opt = document.createElement("button");
          opt.type = "button";
          opt.className = "time-dropdown-option";
          opt.setAttribute("role", "option");
          opt.setAttribute("data-hour", String(hour));
          opt.textContent = formatTimeValue(hour, 0);
          opt.addEventListener("click", function () {
            input.value = formatTimeValue(hour, 0);
            // Dispatch a real "change" event: setting .value programmatically
            // does not fire one on its own, and the start-time auto-fill
            // listener (SLOT-03) needs a genuine change event to run when a
            // slot's start time is picked via this dropdown (not just via
            // typing or the native picker, which fire "change" natively).
            input.dispatchEvent(new Event("change", { bubbles: true }));
            markSelected();
            list.hidden = true;
          });
          li.appendChild(opt);
          list.appendChild(li);
        })(h);
      }
    }

    function markSelected() {
      var current = parseTimeValue(input.value);
      Array.prototype.slice.call(list.querySelectorAll(".time-dropdown-option")).forEach(function (opt) {
        var h = parseInt(opt.getAttribute("data-hour"), 10);
        opt.setAttribute("aria-selected", current && current.h === h ? "true" : "false");
      });
    }

    function scrollToDefaultHour() {
      var target = list.querySelector('[data-hour="' + DEFAULT_HOUR + '"]');
      if (!target) {
        return;
      }
      var top = target.offsetTop;
      list.scrollTop = top < 0 ? 0 : top;
    }

    function openDropdown() {
      closeAllTimeDropdowns();
      buildOptions();
      markSelected();
      list.hidden = false;
      var current = parseTimeValue(input.value);
      if (current) {
        var selectedEl = list.querySelector('[data-hour="' + current.h + '"]');
        if (selectedEl) {
          list.scrollTop = selectedEl.offsetTop;
          return;
        }
      }
      scrollToDefaultHour();
    }

    // Clicking anywhere on the time input opens the hour dropdown (SLOT-02)
    // — this is now the only way to open it; the separate ▾ toggle button
    // was removed as redundant once click-anywhere existed. stopPropagation
    // is required so the document-level closeAllTimeDropdowns click listener
    // below doesn't instantly re-close it. The native time picker still
    // works — no preventDefault here, overlapping with it is acceptable.
    input.addEventListener("click", function (evt) {
      evt.stopPropagation();
      if (list.hidden) {
        openDropdown();
      } else {
        closeAllTimeDropdowns();
      }
    });

    Array.prototype.slice.call(field.querySelectorAll("[data-time-step]")).forEach(function (stepBtn) {
      stepBtn.addEventListener("click", function () {
        var delta = parseInt(stepBtn.getAttribute("data-time-step"), 10);
        var current = parseTimeValue(input.value) || { h: DEFAULT_HOUR, m: 0 };
        var totalMinutes = current.h * 60 + current.m + delta;
        totalMinutes = ((totalMinutes % 1440) + 1440) % 1440; // wrap within a day
        input.value = formatTimeValue(Math.floor(totalMinutes / 60), totalMinutes % 60);
      });
    });
  }

  document.addEventListener("click", closeAllTimeDropdowns);
  document.addEventListener("keydown", function (evt) {
    if (evt.key === "Escape") {
      closeAllTimeDropdowns();
    }
  });

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
    Array.prototype.slice.call(row.querySelectorAll("[data-time-field]")).forEach(wireTimeField);

    // Clicking anywhere on a date input opens its native date picker
    // (feature-detected showPicker(), same "click anywhere" affordance as
    // the time fields). Falls back to normal click-to-focus behavior on
    // browsers without showPicker().
    Array.prototype.slice.call(row.querySelectorAll("[data-date-input]")).forEach(function (dateInput) {
      dateInput.addEventListener("click", function () {
        if (typeof dateInput.showPicker === "function") {
          try {
            dateInput.showPicker();
          } catch (e) {
            // Ignore — e.g. showPicker() can throw if not called from a
            // user gesture in some browsers; falls back to normal focus.
          }
        }
      });
    });

    // SLOT-03: picking a start time on a specific-time slot auto-fills an
    // empty end time with start+1h. The empty-only guard means a manually
    // set (non-empty) end time is never overwritten, and once auto-filled
    // once, a later start-time change won't clobber it either.
    var startInput = row.querySelector('[name="slot_start_time"]');
    var endInput = row.querySelector('[name="slot_end_time"]');
    if (startInput && endInput) {
      startInput.addEventListener("change", function () {
        if (endInput.value) {
          return;
        }
        var start = parseTimeValue(startInput.value);
        if (!start) {
          return;
        }
        var total = ((start.h * 60 + start.m + 60) % 1440 + 1440) % 1440;
        endInput.value = formatTimeValue(Math.floor(total / 60), total % 60);
      });

      // Start-time +/-15 steppers also shift the end time by the same
      // amount, preserving the gap between them (only once an end time
      // exists — an empty end time is handled by the auto-fill above, not
      // by shifting nothing). End-time steppers are untouched: they only
      // ever affect the end time itself (wired generically in wireTimeField).
      var startField = startInput.closest("[data-time-field]");
      if (startField) {
        Array.prototype.slice.call(startField.querySelectorAll("[data-time-step]")).forEach(function (stepBtn) {
          stepBtn.addEventListener("click", function () {
            if (!endInput.value) {
              return;
            }
            var delta = parseInt(stepBtn.getAttribute("data-time-step"), 10);
            var current = parseTimeValue(endInput.value);
            if (!current) {
              return;
            }
            var total = ((current.h * 60 + current.m + delta) % 1440 + 1440) % 1440;
            endInput.value = formatTimeValue(Math.floor(total / 60), total % 60);
          });
        });
      }
    }

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
