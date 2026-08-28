// vote.js — progressive-enhancement behavior for the voting form.
// The Yes/No/Maybe pill selection is handled entirely by native radio
// inputs styled via CSS :checked (see style.css); this script adds no
// pill-selection logic and the control remains fully functional if this
// script never loads. The only enhancement here is the submit guard,
// mirroring create.js's closure-flag pattern: disable the button and
// swap its label so a second click (or a double form submission) is a
// genuine no-op, not just a re-style.
(function () {
  "use strict";

  var form = document.getElementById("vote-form");
  if (!form) {
    return;
  }

  var submitBtn = document.getElementById("submit-btn");
  var setAllBtns = Array.prototype.slice.call(form.querySelectorAll("[data-set-all]"));
  var submitted = false;

  // A "Set all to X" button only asks for confirmation when there's
  // something a careless click could actually clobber: slots that don't
  // already all share one identical answer (yes/no/maybe alike — a
  // uniform poll is a safe overwrite regardless of which value it's
  // uniform on). Skips the confirm both when the poll is already uniform
  // AND when no slot has been answered yet, since there's nothing to
  // overwrite in either case.
  setAllBtns.forEach(function (btn) {
    btn.addEventListener("click", function () {
      var target = btn.getAttribute("data-set-all");
      var groups = Array.prototype.slice.call(form.querySelectorAll(".pill-group"));
      var values = groups.map(function (group) {
        var checked = group.querySelector('input[type="radio"]:checked');
        return checked ? checked.value : null;
      });
      var hasAnyAnswer = values.some(function (v) {
        return v !== null;
      });
      var allSameAnswer =
        hasAnyAnswer &&
        values.every(function (v) {
          return v === values[0];
        });

      if (hasAnyAnswer && !allSameAnswer) {
        var confirmed = window.confirm(
          "Set every slot to " + target.charAt(0).toUpperCase() + target.slice(1) + "? This will overwrite your current selections."
        );
        if (!confirmed) {
          return;
        }
      }

      groups.forEach(function (group) {
        var input = group.querySelector('input[type="radio"][value="' + target + '"]');
        if (input) {
          input.checked = true;
        }
      });
    });
  });

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
