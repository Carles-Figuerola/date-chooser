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
  var setAllYesBtn = document.getElementById("set-all-yes-btn");
  var submitted = false;

  // "Set all to Yes" only asks for confirmation when there's something a
  // careless click could actually clobber: any unanswered slot, or slots
  // that don't already all share one identical answer (yes/no/maybe alike
  // — a uniform poll is a safe overwrite regardless of which value it's
  // uniform on). Already-all-Yes (or all-No, or all-Maybe) skips the
  // confirm since applying "Yes" to a uniform set changes nothing risky
  // beyond what's already visible.
  if (setAllYesBtn) {
    setAllYesBtn.addEventListener("click", function () {
      var groups = Array.prototype.slice.call(form.querySelectorAll(".pill-group"));
      var values = groups.map(function (group) {
        var checked = group.querySelector('input[type="radio"]:checked');
        return checked ? checked.value : null;
      });
      var allSameAnswer =
        values.length > 0 &&
        values[0] !== null &&
        values.every(function (v) {
          return v === values[0];
        });

      if (!allSameAnswer) {
        var confirmed = window.confirm("Set every slot to Yes? This will overwrite your current selections.");
        if (!confirmed) {
          return;
        }
      }

      groups.forEach(function (group) {
        var yesInput = group.querySelector('input[type="radio"][value="yes"]');
        if (yesInput) {
          yesInput.checked = true;
        }
      });
    });
  }

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
