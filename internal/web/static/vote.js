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
  var submitted = false;

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
