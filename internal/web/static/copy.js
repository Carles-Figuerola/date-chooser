// copy.js - progressive enhancement for the links-display page.
//
// Copying to the clipboard is a convenience only: both links are already
// rendered as plain, selectable text in the server-rendered HTML, so the
// page remains fully usable with JavaScript disabled.
//
// On success, the clicked button's label temporarily changes to "Copied!"
// for ~2 seconds. On failure - the Clipboard API is unavailable (e.g. a
// non-secure context) or the write is rejected (e.g. permission denied) -
// this reveals the pre-rendered fallback message ("Couldn't copy
// automatically - select the link text above and copy it manually.") and
// programmatically selects the link text so the user can copy it manually.
(function () {
  "use strict";

  var COPIED_LABEL = "Copied!";
  var COPIED_DURATION_MS = 2000;

  function showFallback() {
    var fallback = document.querySelector(".copy-fallback");
    if (fallback) {
      fallback.hidden = false;
    }
  }

  function selectLinkText(linkEl) {
    if (!linkEl || !window.getSelection || !document.createRange) {
      return;
    }
    var range = document.createRange();
    range.selectNodeContents(linkEl);
    var selection = window.getSelection();
    selection.removeAllRanges();
    selection.addRange(range);
  }

  function handleFailure(linkEl) {
    showFallback();
    selectLinkText(linkEl);
  }

  function handleSuccess(button) {
    var originalLabel = button.textContent;
    button.textContent = COPIED_LABEL;
    setTimeout(function () {
      button.textContent = originalLabel;
    }, COPIED_DURATION_MS);
  }

  function onCopyClick(event) {
    var button = event.currentTarget;
    var targetId = button.getAttribute("data-copy-target");
    var linkEl = targetId ? document.getElementById(targetId) : null;
    if (!linkEl) {
      return;
    }

    var url = linkEl.textContent.trim();

    // Feature-detect rather than assume: navigator.clipboard is undefined
    // in non-secure contexts and some older browsers.
    if (!navigator.clipboard || !navigator.clipboard.writeText) {
      handleFailure(linkEl);
      return;
    }

    navigator.clipboard.writeText(url).then(
      function () {
        handleSuccess(button);
      },
      function () {
        // Permission denied or the write otherwise rejected.
        handleFailure(linkEl);
      }
    );
  }

  document.addEventListener("DOMContentLoaded", function () {
    var buttons = document.querySelectorAll(".btn-copy");
    for (var i = 0; i < buttons.length; i++) {
      buttons[i].addEventListener("click", onCopyClick);
    }
  });
})();
