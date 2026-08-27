// admin.js — progressive-enhancement guard for the admin-only "Remove"
// response-deletion control on the results grid (Plan 04-02, ADM-03). The
// `.results-admin-remove` form is fully submittable server-side without
// this script (a plain POST to the delete route works); this file adds two
// things on top:
//
//   1. A native confirm() dialog before the delete proceeds, using the
//      exact copy from 04-UI-SPEC.md's Copywriting Contract, built from
//      data-participant-name/data-poll-title attributes the template
//      writes onto the form (never invented client-side).
//   2. A disable-on-confirm guard on the submit button, so a genuine
//      double-click (or a double-confirm somehow) cannot fire two delete
//      submits — the backstop required by 04-UI-SPEC.md's "Remove" /
//      "Delete poll" interactive-control considerations.
//
// The confirm() dialog is a UX safety net only, not the authorization
// control (that's possession of the admin token in the URL); a no-op
// double-delete is also safe server-side (store.DeleteParticipant is a
// no-op on an already-deleted or non-existent id).
(function () {
  "use strict";

  document.addEventListener("submit", function (event) {
    var form = event.target;
    if (!form.classList || !form.classList.contains("results-admin-remove")) {
      return;
    }

    var name = form.getAttribute("data-participant-name") || "";
    var title = form.getAttribute("data-poll-title") || "";
    var confirmed = window.confirm(
      "Remove " + name + "'s response to \"" + title + "\"? This cannot be undone."
    );
    if (!confirmed) {
      event.preventDefault();
      return;
    }

    var button = form.querySelector("button[type=submit]");
    if (button) {
      button.disabled = true;
    }
  });
})();
