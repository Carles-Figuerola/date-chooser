// slot-time-fields.js — shared slot date/time-field affordances used by both
// create.js and edit.js: the custom hour-dropdown + ±15-minute steppers on
// top of native <input type="time">, click-anywhere-opens-picker on date
// inputs, and the start-time change handler that auto-fills an empty end
// time (+1h) or, if the end time is already populated, shifts it to
// preserve the existing duration. Kept in one file so the two forms cannot
// drift out of sync the way they did before this file existed.
window.DateChooserSlotFields = (function () {
  "use strict";

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
            // does not fire one on its own, and the start-time handler below
            // needs a genuine change event to run when a slot's start time
            // is picked via this dropdown (not just via typing or the
            // native picker, which fire "change" natively).
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
    // — this is the only way to open it; there is no separate ▾ toggle
    // button. stopPropagation is required so the document-level
    // closeAllTimeDropdowns click listener below doesn't instantly re-close
    // it. The native time picker still works — no preventDefault here,
    // overlapping with it is acceptable.
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

  // wireRow wires every [data-time-field] and [data-date-input] within the
  // given row, plus the start/end duration-preserving change handler. Safe
  // to call once per row, however many rows the page has (create.js's
  // Copy button and edit.js's Add-slot button both add fresh rows that
  // need this wiring too).
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

    // Picking a start time on a specific-time slot auto-fills an empty end
    // time with start+1h. When the end time is already populated (e.g.
    // right after Copy clones both start and end), changing the start time
    // instead shifts the end time to preserve the existing duration —
    // computed against the *previous* start value, captured here at wire
    // time so a Copy-cloned row's original duration is used correctly.
    var startInput = row.querySelector('[name="slot_start_time"]');
    var endInput = row.querySelector('[name="slot_end_time"]');
    if (startInput && endInput) {
      var previousStartValue = startInput.value;
      startInput.addEventListener("change", function () {
        var start = parseTimeValue(startInput.value);
        if (!start) {
          previousStartValue = startInput.value;
          return;
        }
        if (!endInput.value) {
          var total = ((start.h * 60 + start.m + 60) % 1440 + 1440) % 1440;
          endInput.value = formatTimeValue(Math.floor(total / 60), total % 60);
          previousStartValue = startInput.value;
          return;
        }
        var previousStart = parseTimeValue(previousStartValue);
        var end = parseTimeValue(endInput.value);
        if (previousStart && end) {
          var duration = (((end.h * 60 + end.m) - (previousStart.h * 60 + previousStart.m)) % 1440 + 1440) % 1440;
          var newTotal = ((start.h * 60 + start.m + duration) % 1440 + 1440) % 1440;
          endInput.value = formatTimeValue(Math.floor(newTotal / 60), newTotal % 60);
        }
        previousStartValue = startInput.value;
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
  }

  // slotsToText renders a plain-text export: one line per slot, "date"
  // for an all-day slot or "date,start,end" for a specific-time slot.
  // slots is an array of {date, start, end} (start/end only used in
  // date_time mode).
  function slotsToText(mode, slots) {
    var lines = slots.map(function (s) {
      if (mode === "date_time") {
        return s.date + "," + s.start + "," + s.end;
      }
      return s.date;
    });
    return lines.join("\n") + "\n";
  }

  // parseSlotsText parses that same format back into an array of
  // {date, start, end}. Blank lines are silently ignored (not counted as
  // skipped — matches the server's own blank-row tolerance). A line that
  // doesn't match the current mode's expected shape is skipped and
  // counted, never partially applied.
  function parseSlotsText(mode, text) {
    var dateTimeLine = /^(\d{4}-\d{2}-\d{2}),(\d{2}:\d{2}),(\d{2}:\d{2})$/;
    var allDayLine = /^(\d{4}-\d{2}-\d{2})$/;
    var slots = [];
    var skipped = 0;
    text.split(/\r\n|\r|\n/).forEach(function (rawLine) {
      var line = rawLine.trim();
      if (!line) {
        return;
      }
      if (mode === "date_time") {
        var m = dateTimeLine.exec(line);
        if (!m) {
          skipped++;
          return;
        }
        slots.push({ date: m[1], start: m[2], end: m[3] });
      } else {
        var m2 = allDayLine.exec(line);
        if (!m2) {
          skipped++;
          return;
        }
        slots.push({ date: m2[1] });
      }
    });
    return { slots: slots, skipped: skipped };
  }

  return {
    parseTimeValue: parseTimeValue,
    formatTimeValue: formatTimeValue,
    wireRow: wireRow,
    slotsToText: slotsToText,
    parseSlotsText: parseSlotsText
  };
})();
