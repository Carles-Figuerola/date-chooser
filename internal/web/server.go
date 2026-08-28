package web

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfiguerola/date-chooser/internal/store"
	"github.com/cfiguerola/date-chooser/internal/token"
)

// maxFormBytes bounds the size of a POST /polls request body to guard
// against unbounded-body denial-of-service (threat T-01-04).
const maxFormBytes = 1 << 20 // 1 MiB

// dateTimeLayout matches the value format of an HTML5 datetime-local input
// (no timezone, no seconds): "2006-01-02T15:04".
const dateTimeLayout = "2006-01-02T15:04"

// bannerErrorCopy is the exact UI-SPEC validation-banner copy shown above
// the create-poll form whenever server-side validation rejects a submission.
const bannerErrorCopy = "Check the highlighted fields and try again."

// rowErrorCopy is the exact UI-SPEC row-level error shown under a date+time
// slot row whose end is not strictly after its start.
const rowErrorCopy = "End time must be after start time."

// dateRequiredCopy is the row-level error shown when a date+time slot row
// has a start and/or end time but no date — distinct from rowErrorCopy so
// the organizer isn't told about a start/end ordering problem that doesn't
// exist yet.
const dateRequiredCopy = "Enter a date for this slot."

// timesRequiredCopy is the row-level error shown when a date+time slot row
// has a date but is missing a start and/or end time.
const timesRequiredCopy = "Enter both a start and end time."

// duplicateSlotCopy is the exact row-level error shown on every row of an
// exact-duplicate group (SLOT-05). Position-neutral wording, since ALL
// members of a duplicate group are flagged, not just the second occurrence.
const duplicateSlotCopy = "This slot is a duplicate of another slot."

// nameRequiredCopy is the exact UI-SPEC inline field error shown under the
// display-name field on the voting form when it is blank.
const nameRequiredCopy = "Your name is required."

// slotAnswerRequiredCopy is the exact UI-SPEC inline row error shown under a
// voting-form slot row left without a Yes/No/Maybe answer at submit time.
const slotAnswerRequiredCopy = "Choose Yes, No, or Maybe for this slot."

// maxDisplayNameRunes and maxCommentRunes are the server-side defense-in-depth
// caps behind the voting form's native maxlength attributes (a client can
// bypass maxlength, so these are enforced again here).
const maxDisplayNameRunes = 100
const maxCommentRunes = 500

// dateOnlyLayout matches the value format of an HTML5 date input used for
// all_day slots: "2006-01-02".
const dateOnlyLayout = "2006-01-02"

// participantCookieName is the HttpOnly cookie identifying a participant
// across visits, scoped to this poll's voting path (threat T-02-01/T-02-02).
const participantCookieName = "dc_participant"

// participantCookieMaxAge is ~1 year, per CONTEXT.md's "come back later and
// change your vote" requirement.
const participantCookieMaxAge = 365 * 24 * 60 * 60

// instanceAdminCookieName is the session-only (no MaxAge/Expires) cookie
// authorizing access to /admin, set after a correct secret is submitted at
// /admin/login (ADMIN-02). Its value is the secret itself — a bearer-token
// pattern, matching how participant/poll-admin tokens already work in this
// app — so no server-side session store is needed.
const instanceAdminCookieName = "dc_instance_admin"

// instanceAdminSessionMaxAge is how long a session created at
// /admin/login remains valid, checked against the admin_sessions row's
// created_at at auth time — not enforced via the cookie's own expiry (the
// cookie carries no MaxAge/Expires; the browser may keep it around longer,
// in which case the request is simply treated as unauthenticated once the
// session has aged out).
const instanceAdminSessionMaxAge = 24 * time.Hour

// instanceAdminLoginErrorCopy is shown on an incorrect secret submission at
// /admin/login. Deliberately generic — it never reveals whether the
// database has a secret at all.
const instanceAdminLoginErrorCopy = "Incorrect secret."

// createFormView is the data passed to create.html for both the initial GET
// render and any validation-failure re-render (HTTP 200, values preserved).
type createFormView struct {
	Title         string
	Description   string
	OrganizerName string
	PollType      string // "all_day" or "date_time"
	Slots         []slotView
	BannerError   string
	TitleError    string
}

// slotView is one rendered slot row: either just a Date (all_day mode) or a
// Date plus StartTime/EndTime (date_time mode — start and end always share
// the same date, per CONTEXT.md), plus an optional row-level error.
type slotView struct {
	Date      string
	StartTime string
	EndTime   string
	Error     string
}

// newCreateFormView returns the default, empty form view rendered on first
// load: one pre-filled blank slot row so the true zero-slot state is never
// shown (per UI-SPEC "empty" state).
func newCreateFormView() createFormView {
	return createFormView{
		PollType: "all_day",
		Slots:    []slotView{{}},
	}
}

// Server holds the dependencies needed to serve Date Chooser's HTTP routes.
type Server struct {
	store               *store.Store
	tmpl                *pageTemplates
	instanceAdminSecret string
	dbPath              string
}

// NewServer constructs a Server backed by the given store, parsing the
// embedded templates once. instanceAdminSecret gates /admin (ADMIN-02) and
// dbPath is shown (path only, never the secret's value) on the /admin/login
// page's "forgotten the secret?" disclosure (ADMIN-05), so the sqlite3
// command shown there matches this exact deployment's database file.
func NewServer(st *store.Store, instanceAdminSecret, dbPath string) (*Server, error) {
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, fmt.Errorf("web: parsing templates: %w", err)
	}
	return &Server{store: st, tmpl: tmpl, instanceAdminSecret: instanceAdminSecret, dbPath: dbPath}, nil
}

// Routes builds the HTTP handler for all Date Chooser routes.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", s.handleCreateForm)
	mux.HandleFunc("POST /polls", s.handleCreatePoll)
	mux.HandleFunc("GET /poll/{ptoken}/admin/{atoken}", s.handleLinksPage)
	mux.HandleFunc("GET /poll/{ptoken}/admin/{atoken}/edit", s.handleEditForm)
	mux.HandleFunc("POST /poll/{ptoken}/admin/{atoken}/edit", s.handleUpdatePoll)
	mux.HandleFunc("POST /poll/{ptoken}/admin/{atoken}/responses/{participantID}/delete", s.handleDeleteResponse)
	mux.HandleFunc("POST /poll/{ptoken}/admin/{atoken}/delete", s.handleDeletePoll)
	mux.HandleFunc("GET /poll/{ptoken}", s.handleVoteForm)
	mux.HandleFunc("POST /poll/{ptoken}/responses", s.handleSubmitResponse)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /admin/login", s.handleInstanceAdminLoginForm)
	mux.HandleFunc("POST /admin/login", s.handleInstanceAdminLogin)
	mux.HandleFunc("GET /admin", s.handleInstanceAdminPage)

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		// staticFS is embedded at build time; this can only fail if the
		// embed directive itself is wrong, which would fail the build.
		panic(fmt.Sprintf("web: static assets misconfigured: %v", err))
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))

	return mux
}

func (s *Server) handleCreateForm(w http.ResponseWriter, r *http.Request) {
	s.renderCreateForm(w, newCreateFormView())
}

// renderCreateForm executes create.html with the given view. It is used for
// both the fresh GET / render and any validation-failure re-render (which
// must be HTTP 200, not a redirect, per the UI-SPEC error contract).
func (s *Server) renderCreateForm(w http.ResponseWriter, view createFormView) {
	if err := s.tmpl.create.ExecuteTemplate(w, "layout", view); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (s *Server) handleCreatePoll(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)

	if err := r.ParseForm(); err != nil {
		if strings.Contains(err.Error(), "too large") {
			http.Error(w, "Request body too large.", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, bannerErrorCopy, http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := r.FormValue("description")
	organizerName := r.FormValue("organizer_name")
	pollType := r.FormValue("poll_type")
	if pollType == "" {
		pollType = "all_day"
	}

	view := createFormView{
		Title:         title,
		Description:   description,
		OrganizerName: organizerName,
		PollType:      pollType,
	}

	hasError := false

	// The server is the authority on poll_type: reject anything outside the
	// allowed set rather than silently coercing it (threat T-01-08). The DB
	// CHECK constraint is the backstop, this is the user-facing gate.
	if pollType != "all_day" && pollType != "date_time" {
		hasError = true
		view.PollType = "all_day"
	}

	if title == "" {
		view.TitleError = "Poll title is required."
		hasError = true
	}

	var slots []store.NewSlotInput
	var rowErr bool
	view.Slots, slots, rowErr = parseSlots(pollType, r.Form)
	if rowErr {
		hasError = true
	}
	if markDuplicateSlots(pollType, view.Slots) {
		hasError = true
	}
	if len(slots) == 0 {
		hasError = true
	}

	if hasError {
		view.BannerError = bannerErrorCopy
		s.renderCreateForm(w, view)
		return
	}

	participantToken, err := token.New()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	adminToken, err := token.New()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	_, err = s.store.CreatePoll(ctx, store.NewPollInput{
		Title:         title,
		Description:   description,
		OrganizerName: organizerName,
		PollType:      pollType,
		Slots:         slots,
	}, participantToken, adminToken)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/poll/"+participantToken+"/admin/"+adminToken, http.StatusSeeOther)
}

// parseSlots reads the submitted slot fields for the given poll_type mode
// and returns: the slotView rows to re-render (always at least one, so the
// zero-row empty state is never re-shown), the valid persisted slots (empty
// if any row is invalid or incomplete), and whether any row-level error was
// found.
func parseSlots(pollType string, form map[string][]string) ([]slotView, []store.NewSlotInput, bool) {
	if pollType == "date_time" {
		return parseDateTimeSlots(form["slot_date"], form["slot_start_time"], form["slot_end_time"])
	}
	return parseAllDaySlots(form["slot_date"])
}

// markDuplicateSlots scans views for exact-duplicate rows per pollType's
// duplicate-match rule (SLOT-05): all_day rows duplicate when they share the
// same Date; date_time rows duplicate when they share the same
// Date+StartTime+EndTime. Incomplete rows (missing a field the match rule
// depends on) are never compared — they cannot be a duplicate.
//
// Every row in a duplicate group gets its Error field set (not just the
// second occurrence), so 3+-way groups and multiple independent duplicate
// pairs are all flagged. Keys are joined with "\x00" (NUL) so date/time text
// can never collide across field boundaries (threat T-05-02). Returns true
// iff at least one duplicate group was found.
func markDuplicateSlots(pollType string, views []slotView) bool {
	seen := make(map[string][]int)
	for i, v := range views {
		var key string
		if pollType == "date_time" {
			if v.Date == "" || v.StartTime == "" || v.EndTime == "" {
				continue
			}
			key = v.Date + "\x00" + v.StartTime + "\x00" + v.EndTime
		} else {
			if v.Date == "" {
				continue
			}
			key = v.Date
		}
		seen[key] = append(seen[key], i)
	}

	found := false
	for _, idxs := range seen {
		if len(idxs) > 1 {
			found = true
			for _, i := range idxs {
				views[i].Error = duplicateSlotCopy
			}
		}
	}
	return found
}

func parseAllDaySlots(dates []string) ([]slotView, []store.NewSlotInput, bool) {
	views := make([]slotView, 0, len(dates))
	slots := make([]store.NewSlotInput, 0, len(dates))
	for _, d := range dates {
		if d == "" {
			continue
		}
		views = append(views, slotView{Date: d})
		slots = append(slots, store.NewSlotInput{StartsAt: d})
	}
	if len(views) == 0 {
		views = []slotView{{}}
	}
	return views, slots, false
}

// parseDateTimeSlots combines a shared per-row date with separate start/end
// time-of-day inputs into full datetime-local values (CONTEXT.md: start and
// end always share the same date for a date_time slot). dates, startTimes,
// and endTimes are parallel arrays from HTML5 <input type="date"> and
// <input type="time"> fields (date: "2006-01-02", time: "15:04").
func parseDateTimeSlots(dates, startTimes, endTimes []string) ([]slotView, []store.NewSlotInput, bool) {
	n := len(dates)
	if len(startTimes) > n {
		n = len(startTimes)
	}
	if len(endTimes) > n {
		n = len(endTimes)
	}

	views := make([]slotView, 0, n)
	slots := make([]store.NewSlotInput, 0, n)
	hasRowError := false

	for i := 0; i < n; i++ {
		var date, startTime, endTime string
		if i < len(dates) {
			date = dates[i]
		}
		if i < len(startTimes) {
			startTime = startTimes[i]
		}
		if i < len(endTimes) {
			endTime = endTimes[i]
		}
		if date == "" && startTime == "" && endTime == "" {
			continue
		}

		row := slotView{Date: date, StartTime: startTime, EndTime: endTime}
		switch {
		case date == "":
			row.Error = dateRequiredCopy
			hasRowError = true
		case startTime == "" || endTime == "":
			row.Error = timesRequiredCopy
			hasRowError = true
		default:
			start := date + "T" + startTime
			end := date + "T" + endTime
			if isValidDateTimeRange(start, end) {
				endCopy := end
				slots = append(slots, store.NewSlotInput{StartsAt: start, EndsAt: &endCopy})
			} else {
				row.Error = rowErrorCopy
				hasRowError = true
			}
		}
		views = append(views, row)
	}

	if len(views) == 0 {
		views = []slotView{{}}
	}
	return views, slots, hasRowError
}

// isValidDateTimeRange reports whether start and end are both present,
// parseable datetime-local values, and end is strictly after start.
func isValidDateTimeRange(start, end string) bool {
	if start == "" || end == "" {
		return false
	}
	startT, errS := time.Parse(dateTimeLayout, start)
	endT, errE := time.Parse(dateTimeLayout, end)
	if errS != nil || errE != nil {
		return false
	}
	return endT.After(startT)
}

func (s *Server) handleLinksPage(w http.ResponseWriter, r *http.Request) {
	ptoken := r.PathValue("ptoken")
	atoken := r.PathValue("atoken")

	poll, slots, err := s.store.PollByTokens(r.Context(), ptoken, atoken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resultParticipants, resultAnswers, err := s.store.ResponsesByPollID(r.Context(), poll.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	participantPath := "/poll/" + poll.ParticipantToken
	adminPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken
	editPath := adminPath + "/edit"

	data := struct {
		Poll            *store.Poll
		ParticipantPath string
		AdminPath       string
		EditPath        string
		ParticipantURL  string
		AdminURL        string
		Results         resultsGridView
	}{
		Poll:            poll,
		ParticipantPath: participantPath,
		AdminPath:       adminPath,
		EditPath:        editPath,
		ParticipantURL:  scheme + "://" + r.Host + participantPath,
		AdminURL:        scheme + "://" + r.Host + adminPath,
		Results:         buildResultsGridView(poll.PollType, slots, resultParticipants, resultAnswers, true, poll.Title, poll.ParticipantToken, poll.AdminToken),
	}

	if err := s.tmpl.links.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// confirmSlotRemovalCopy is the exact UI-SPEC banner shown when a save is
// attempted while a slot with responses is marked for removal but the
// confirmation flag is absent (server-side defense in depth, T-04-04).
const confirmSlotRemovalCopy = "Confirm you understand response data will be deleted before saving."

// editFormView is the data passed to edit.html for both the initial GET
// render (pre-filled from the live poll) and any validation-failure
// re-render (HTTP 200, values preserved) of
// GET/POST /poll/{ptoken}/admin/{atoken}/edit. It mirrors createFormView's
// shape, plus the poll's own tokens (needed for the form action URL and the
// disabled poll-type toggle) — the poll-type value itself is always the
// poll's existing, immutable type (T-04 CONTEXT: locked after creation).
type editFormView struct {
	ParticipantToken string
	AdminToken       string
	Title            string
	Description      string
	OrganizerName    string
	PollType         string // "all_day" or "date_time"; never user-changeable
	Slots            []editSlotView
	BannerError      string
	TitleError       string

	// ResponseCount (Plan 04-03) is the poll's total participant/response
	// count, threaded into the danger-zone's confirm() copy so it always
	// names the real data about to be permanently deleted (04-UI-SPEC.md).
	ResponseCount int
}

// editSlotView is one rendered slot row on the edit form: the create-form
// slot fields (Date/StartTime/EndTime/Error) plus the slot's persisted ID
// (empty/zero for a newly-added, not-yet-saved row) and its ResponseCount,
// which drives both the client-side removal warning (Task 3) and the
// server-side removal-confirmation gate (Task 2) — both read the same count
// so client and server agree on which removals are destructive.
type editSlotView struct {
	ID            int64
	Date          string
	StartTime     string
	EndTime       string
	Error         string
	ResponseCount int
}

// newEditFormView builds the pre-filled edit-form view for a GET render from
// the poll's current row plus its persisted slots and (optionally) their
// response counts. counts may be nil when response counts are not yet known
// (Task 1 wiring, before SlotResponseCounts is called).
func newEditFormView(poll *store.Poll, slots []store.Slot, counts map[int64]int, responseCount int) editFormView {
	return editFormView{
		ParticipantToken: poll.ParticipantToken,
		AdminToken:       poll.AdminToken,
		Title:            poll.Title,
		Description:      poll.Description,
		OrganizerName:    poll.OrganizerName,
		PollType:         poll.PollType,
		Slots:            editSlotViewsFromSlots(poll.PollType, slots, counts),
		ResponseCount:    responseCount,
	}
}

// editSlotViewsFromSlots maps persisted slots into their edit-form row
// views, splitting each slot's StartsAt/EndsAt back into the separate
// Date/StartTime/EndTime fields the edit form (and parseSlots) expect.
func editSlotViewsFromSlots(pollType string, slots []store.Slot, counts map[int64]int) []editSlotView {
	views := make([]editSlotView, len(slots))
	for i, sl := range slots {
		v := editSlotView{ID: sl.ID}
		if counts != nil {
			v.ResponseCount = counts[sl.ID]
		}
		switch pollType {
		case "date_time":
			if len(sl.StartsAt) >= len(dateTimeLayout) {
				v.Date = sl.StartsAt[:len(dateOnlyLayout)]
				v.StartTime = sl.StartsAt[len(dateOnlyLayout)+1:]
			}
			if sl.EndsAt != nil && len(*sl.EndsAt) >= len(dateTimeLayout) {
				v.EndTime = (*sl.EndsAt)[len(dateOnlyLayout)+1:]
			}
		default: // all_day
			v.Date = sl.StartsAt
		}
		views[i] = v
	}
	return views
}

// renderEditForm executes edit.html with the given view. It is used for both
// the fresh GET render and any validation-failure re-render (which must be
// HTTP 200, not a redirect, matching the create-form error contract).
func (s *Server) renderEditForm(w http.ResponseWriter, view editFormView) {
	if err := s.tmpl.edit.ExecuteTemplate(w, "layout", view); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (s *Server) handleEditForm(w http.ResponseWriter, r *http.Request) {
	ptoken := r.PathValue("ptoken")
	atoken := r.PathValue("atoken")

	poll, slots, err := s.store.PollByTokens(r.Context(), ptoken, atoken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.renderNotFound(w)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	counts, err := s.store.SlotResponseCounts(r.Context(), poll.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Danger-zone response count: the same participant count the edit page
	// otherwise has no reason to compute, fetched here solely so the
	// delete-poll confirm() copy names the real data (04-UI-SPEC.md).
	participants, _, err := s.store.ResponsesByPollID(r.Context(), poll.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.renderEditForm(w, newEditFormView(poll, slots, counts, len(participants)))
}

// parseEditSlots classifies the edit form's submitted slot rows into: the
// slotView rows to re-render, the existing slots to update in place (keep),
// the brand-new slots to append (add), and the full set of existing slot
// IDs to remove — either explicitly marked via the slot_removed flag, or
// implicitly removed because their row was dropped from the DOM entirely
// (Task 3's immediate-removal behavior for zero-response rows). Fields are
// parallel arrays aligned by index, mirroring parseSlots' slot_date /
// slot_start_time / slot_end_time contract plus two edit-only fields:
// slot_id (empty for a newly-added row) and slot_removed ("1" when marked).
func parseEditSlots(pollType string, form map[string][]string, existingIDs map[int64]bool) (views []editSlotView, keep []store.SlotEdit, add []store.NewSlotInput, removeIDs []int64, hasRowError bool) {
	slotIDs := form["slot_id"]
	removedFlags := form["slot_removed"]
	dates := form["slot_date"]
	startTimes := form["slot_start_time"]
	endTimes := form["slot_end_time"]

	n := len(dates)
	if len(slotIDs) > n {
		n = len(slotIDs)
	}

	presentIDs := make(map[int64]bool)

	for i := 0; i < n; i++ {
		var idStr, date, startTime, endTime, removedFlag string
		if i < len(slotIDs) {
			idStr = slotIDs[i]
		}
		if i < len(dates) {
			date = dates[i]
		}
		if i < len(startTimes) {
			startTime = startTimes[i]
		}
		if i < len(endTimes) {
			endTime = endTimes[i]
		}
		if i < len(removedFlags) {
			removedFlag = removedFlags[i]
		}
		removed := removedFlag == "1"

		var slotID int64
		hasID := false
		if idStr != "" {
			if parsed, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				slotID = parsed
				hasID = true
				presentIDs[slotID] = true
			}
		}

		if removed && hasID {
			removeIDs = append(removeIDs, slotID)
			views = append(views, editSlotView{ID: slotID, Date: date, StartTime: startTime, EndTime: endTime})
			continue
		}

		if date == "" && startTime == "" && endTime == "" {
			// Entirely blank row — only possible for a new, unfilled "Add
			// another slot" row. Skip it, same as parseSlots does, so a
			// phantom empty slot is never persisted or re-shown.
			continue
		}

		row := editSlotView{ID: slotID, Date: date, StartTime: startTime, EndTime: endTime}
		var start, end string
		valid := true
		if pollType == "date_time" {
			if date != "" && startTime != "" {
				start = date + "T" + startTime
			}
			if date != "" && endTime != "" {
				end = date + "T" + endTime
			}
			valid = isValidDateTimeRange(start, end)
		} else {
			start = date
			valid = date != ""
		}

		if !valid {
			row.Error = rowErrorCopy
			hasRowError = true
			views = append(views, row)
			continue
		}
		views = append(views, row)

		var endCopy *string
		if pollType == "date_time" && end != "" {
			e := end
			endCopy = &e
		}
		if hasID {
			keep = append(keep, store.SlotEdit{ID: slotID, StartsAt: start, EndsAt: endCopy})
		} else {
			add = append(add, store.NewSlotInput{StartsAt: start, EndsAt: endCopy})
		}
	}

	// Any existing slot whose ID never appeared in the submission at all —
	// its row was removed from the DOM client-side (Task 3's zero-response
	// immediate-removal behavior) — is implicitly removed too.
	for id := range existingIDs {
		if !presentIDs[id] {
			removeIDs = append(removeIDs, id)
		}
	}

	if len(views) == 0 {
		views = []editSlotView{{}}
	}

	return views, keep, add, removeIDs, hasRowError
}

func (s *Server) handleUpdatePoll(w http.ResponseWriter, r *http.Request) {
	ptoken := r.PathValue("ptoken")
	atoken := r.PathValue("atoken")

	poll, slots, err := s.store.PollByTokens(r.Context(), ptoken, atoken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.renderNotFound(w)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		if strings.Contains(err.Error(), "too large") {
			http.Error(w, "Request body too large.", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, bannerErrorCopy, http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := r.FormValue("description")
	organizerName := r.FormValue("organizer_name")

	existingIDs := make(map[int64]bool, len(slots))
	for _, sl := range slots {
		existingIDs[sl.ID] = true
	}

	slotViews, keep, add, removeIDs, rowErr := parseEditSlots(poll.PollType, r.Form, existingIDs)

	// Danger-zone response count for the re-rendered view (same as the GET
	// path) — kept accurate even on a validation-failure re-render, since the
	// danger zone reflects live poll data, not the in-progress form edit.
	participants, _, err := s.store.ResponsesByPollID(r.Context(), poll.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	view := editFormView{
		ParticipantToken: poll.ParticipantToken,
		AdminToken:       poll.AdminToken,
		Title:            title,
		Description:      description,
		OrganizerName:    organizerName,
		PollType:         poll.PollType, // immutable — never taken from the form
		Slots:            slotViews,
		ResponseCount:    len(participants),
	}

	hasError := rowErr
	if title == "" {
		view.TitleError = "Poll title is required."
		hasError = true
	}

	counts, err := s.store.SlotResponseCounts(r.Context(), poll.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for i := range view.Slots {
		view.Slots[i].ResponseCount = counts[view.Slots[i].ID]
	}

	if hasError {
		view.BannerError = bannerErrorCopy
		s.renderEditForm(w, view)
		return
	}

	// Server-side defense in depth (T-04-04): a destructive removal (a
	// removeID with at least one response) cannot be saved without the
	// confirmation flag, mirroring the client-side disabled-button gate.
	destructive := false
	for _, id := range removeIDs {
		if counts[id] > 0 {
			destructive = true
			break
		}
	}
	if destructive && r.FormValue("confirm_slot_removal") == "" {
		view.BannerError = confirmSlotRemovalCopy
		s.renderEditForm(w, view)
		return
	}

	if err := s.store.UpdatePollDetails(r.Context(), poll.ID, title, description, organizerName); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.store.UpdatePollSlots(r.Context(), poll.ID, keep, add, removeIDs); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/poll/"+ptoken+"/admin/"+atoken, http.StatusSeeOther)
}

// handleDeleteResponse deletes a single participant's response (Plan
// 04-02, ADM-03). It resolves the poll via PollByTokens using BOTH tokens
// from the path — a mismatched pair 404s before any delete is attempted
// (T-04-02) — then deletes the participant scoped to that poll's ID
// (T-04-03), so a {participantID} belonging to a different poll deletes
// nothing. After a successful (or no-op) delete, it redirects back to the
// admin page so the results grid recomputes tallies/best-fit from current
// DB state, per Phase 3's always-recompute design — no client-side tally
// math is ever needed.
func (s *Server) handleDeleteResponse(w http.ResponseWriter, r *http.Request) {
	ptoken := r.PathValue("ptoken")
	atoken := r.PathValue("atoken")

	poll, _, err := s.store.PollByTokens(r.Context(), ptoken, atoken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.renderNotFound(w)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	participantID, err := strconv.ParseInt(r.PathValue("participantID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid participant id", http.StatusBadRequest)
		return
	}

	if err := s.store.DeleteParticipant(r.Context(), poll.ID, participantID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/poll/"+ptoken+"/admin/"+atoken, http.StatusSeeOther)
}

// handleDeletePoll deletes the entire poll (Plan 04-03, ADM-04) — the last
// and highest-stakes admin action in the app. Exactly like
// handleDeleteResponse, it resolves the poll via PollByTokens using BOTH
// tokens from the path first (a mismatched pair 404s before any delete is
// attempted, T-04-02), then deletes by the resolved poll.ID only — never a
// client-supplied id. After deletion, the redirect target
// (/poll/{ptoken}) itself 404s, because the poll (and thus both tokens) no
// longer resolve to anything — no tombstone state is added, per
// 04-CONTEXT.md's "reuse the generic invalid-link 404" decision.
func (s *Server) handleDeletePoll(w http.ResponseWriter, r *http.Request) {
	ptoken := r.PathValue("ptoken")
	atoken := r.PathValue("atoken")

	poll, _, err := s.store.PollByTokens(r.Context(), ptoken, atoken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.renderNotFound(w)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.store.DeletePoll(r.Context(), poll.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/poll/"+ptoken, http.StatusSeeOther)
}

// voteFormView is the data passed to vote.html for both the initial GET
// render (fresh or pre-filled from a matching cookie) and the post-submit
// redirect-back render (Saved=true).
type voteFormView struct {
	Poll        *store.Poll
	Slots       []voteSlotView
	DisplayName string
	Comment     string
	Saved       bool
	NameError   string
	BannerError string
	Results     resultsGridView
}

// voteSlotView is one rendered slot row: its human-readable label and
// whichever answer ("yes"/"no"/"maybe") is currently selected, if any.
type voteSlotView struct {
	ID       int64
	Label    string
	Selected string
	Error    string
}

// resultsGridView is the data passed to the shared "results" partial: the
// slots x participants grid, its per-slot tallies and best-fit ranking, and
// the comments list. Built by buildResultsGridView from raw store data —
// shared, per T-03-02, between the participant route (this plan) and the
// admin route (Plan 03-02), so both routes render identically off the same
// view-model.
type resultsGridView struct {
	HasResponses bool
	Participants []resultsParticipant
	Rows         []resultsRow
	Comments     []resultsComment

	// Title, IsAdmin, ParticipantToken, and AdminToken (Plan 04-02) drive the
	// admin-only "Remove" response-deletion control: IsAdmin gates whether
	// the control renders at all (the participant route always passes
	// false), and Title/ParticipantToken/AdminToken give the shared partial
	// what it needs to build the delete-route URL and the confirm() copy
	// without needing the caller's whole page-level data struct.
	Title            string
	IsAdmin          bool
	ParticipantToken string
	AdminToken       string
}

// resultsParticipant is one grid column header. Only DisplayName and the
// participant's own store ID are ever exposed here — never CookieToken
// (threat T-03-02). ID is the participant's row id (not the cookie_token),
// safe to render, and needed by the admin route's per-participant "Remove"
// delete-form action URL (T-04-06).
type resultsParticipant struct {
	ID          int64
	DisplayName string
}

// resultsRow is one grid row (one slot): its label, one cell per
// participant in column order, per-answer tallies, and whether this row is
// one of the (possibly tied) top-ranked "best fit" slots.
type resultsRow struct {
	Label      string
	Cells      []resultsCell
	YesCount   int
	NoCount    int
	MaybeCount int
	BestFit    bool
}

// resultsCell is one grid data cell. Answer is "yes", "no", "maybe", or ""
// when the participant has no stored response for this slot.
type resultsCell struct {
	Answer string
}

// resultsComment is one line in the Comments section: a participant's
// display name paired with their non-empty comment.
type resultsComment struct {
	DisplayName string
	Comment     string
}

// buildResultsGridView shapes raw participants/slots/answers (as returned by
// store.ResponsesByPollID) into the results grid's view-model: one column
// per participant (in the given order), one row per slot with per-cell
// answers and Yes/No/Maybe tallies, best-slot ranking via rankBestSlots, and
// the comments list (non-empty comments only, in participant order).
// isAdmin, title, ptoken, and atoken (Plan 04-02) are threaded straight into
// the view so the shared "results" partial can render (or omit, for the
// participant route) the admin-only "Remove" response-deletion control
// without needing any other page-level data.
func buildResultsGridView(pollType string, slots []store.Slot, participants []store.Participant, answers map[int64]map[int64]string, isAdmin bool, title, ptoken, atoken string) resultsGridView {
	view := resultsGridView{
		HasResponses:     len(participants) > 0,
		Participants:     make([]resultsParticipant, len(participants)),
		Rows:             make([]resultsRow, len(slots)),
		Title:            title,
		IsAdmin:          isAdmin,
		ParticipantToken: ptoken,
		AdminToken:       atoken,
	}

	for i, p := range participants {
		view.Participants[i] = resultsParticipant{ID: p.ID, DisplayName: p.DisplayName}
		if comment := strings.TrimSpace(p.Comment); comment != "" {
			view.Comments = append(view.Comments, resultsComment{DisplayName: p.DisplayName, Comment: p.Comment})
		}
	}

	for i, sl := range slots {
		row := resultsRow{
			Label: slotLabel(pollType, sl),
			Cells: make([]resultsCell, len(participants)),
		}
		for j, p := range participants {
			answer := answers[p.ID][sl.ID]
			row.Cells[j] = resultsCell{Answer: answer}
			switch answer {
			case "yes":
				row.YesCount++
			case "no":
				row.NoCount++
			case "maybe":
				row.MaybeCount++
			}
		}
		view.Rows[i] = row
	}

	best := rankBestSlots(view.Rows)
	for i := range view.Rows {
		view.Rows[i].BestFit = best[i]
	}

	return view
}

// rankBestSlots returns the set of row indices (by position in rows) tied
// for the top rank: most Yes (descending), tie-broken by fewest No
// (ascending) — Maybe never factors into the ranking, per 03-CONTEXT.md's
// Best-Slot Highlighting Rule. Returns an empty set when there are zero rows
// or when the total of every tally across every row is zero (i.e. no one
// has responded at all yet) — "best" is meaningless with no data.
func rankBestSlots(rows []resultsRow) map[int]bool {
	best := make(map[int]bool)
	if len(rows) == 0 {
		return best
	}

	total := 0
	for _, r := range rows {
		total += r.YesCount + r.NoCount + r.MaybeCount
	}
	if total == 0 {
		return best
	}

	bestYes, bestNo := -1, 0
	for _, r := range rows {
		if r.YesCount > bestYes || (r.YesCount == bestYes && r.NoCount < bestNo) {
			bestYes, bestNo = r.YesCount, r.NoCount
		}
	}
	for i, r := range rows {
		if r.YesCount == bestYes && r.NoCount == bestNo {
			best[i] = true
		}
	}
	return best
}

// slotLabel formats a persisted slot for display, per poll type. On any
// parse error it falls back to the raw stored value rather than failing the
// whole page render.
func slotLabel(pollType string, sl store.Slot) string {
	switch pollType {
	case "date_time":
		start, err := time.Parse(dateTimeLayout, sl.StartsAt)
		if err != nil {
			return sl.StartsAt
		}
		label := start.Format("Mon, Jan 2, 2006 3:04 PM")
		if sl.EndsAt != nil {
			end, err := time.Parse(dateTimeLayout, *sl.EndsAt)
			if err == nil {
				label += " – " + end.Format("3:04 PM")
			}
		}
		return label
	default: // all_day
		start, err := time.Parse(dateOnlyLayout, sl.StartsAt)
		if err != nil {
			return sl.StartsAt
		}
		return start.Format("Mon, Jan 2, 2006")
	}
}

// requestIsHTTPS reports whether the original client request used HTTPS,
// honoring both a direct TLS connection and the X-Forwarded-Proto header set
// by a reverse proxy (mirrors handleLinksPage's scheme detection).
func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (s *Server) handleVoteForm(w http.ResponseWriter, r *http.Request) {
	ptoken := r.PathValue("ptoken")

	poll, slots, err := s.store.PollByParticipantToken(r.Context(), ptoken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.renderNotFound(w)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	view := voteFormView{
		Poll:  poll,
		Slots: make([]voteSlotView, len(slots)),
		Saved: r.URL.Query().Get("saved") == "1",
	}

	var answers map[int64]string
	if cookie, err := r.Cookie(participantCookieName); err == nil {
		participant, a, err := s.store.ParticipantByCookie(r.Context(), poll.ID, cookie.Value)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if participant != nil {
			view.DisplayName = participant.DisplayName
			view.Comment = participant.Comment
			answers = a
		}
	}

	for i, sl := range slots {
		view.Slots[i] = voteSlotView{
			ID:    sl.ID,
			Label: slotLabel(poll.PollType, sl),
		}
		if answers != nil {
			view.Slots[i].Selected = answers[sl.ID]
		}
	}

	resultParticipants, resultAnswers, err := s.store.ResponsesByPollID(r.Context(), poll.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	view.Results = buildResultsGridView(poll.PollType, slots, resultParticipants, resultAnswers, false, poll.Title, "", "")

	s.renderVoteForm(w, view)
}

// renderVoteForm executes vote.html with the given view.
func (s *Server) renderVoteForm(w http.ResponseWriter, view voteFormView) {
	if err := s.tmpl.vote.ExecuteTemplate(w, "layout", view); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// renderNotFound renders the branded invalid-link 404 page (a plain card
// with no form and no nav, per UI-SPEC) with HTTP 404, replacing the bare
// http.NotFound stub used by the tracer. The status must be written before
// any body bytes, so it is set first regardless of template execution
// outcome.
func (s *Server) renderNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_ = s.tmpl.notfound.ExecuteTemplate(w, "layout", nil)
}

func (s *Server) handleSubmitResponse(w http.ResponseWriter, r *http.Request) {
	ptoken := r.PathValue("ptoken")

	poll, slots, err := s.store.PollByParticipantToken(r.Context(), ptoken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.renderNotFound(w)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		if strings.Contains(err.Error(), "too large") {
			http.Error(w, "Request body too large.", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, bannerErrorCopy, http.StatusBadRequest)
		return
	}

	displayName := strings.TrimSpace(r.FormValue("display_name"))
	comment := r.FormValue("comment")

	// Build the re-render view and validate BEFORE any DB write. Every
	// submitted value is preserved in the view regardless of validity, so a
	// rejected submission re-renders with the participant's own input intact
	// (never silently dropped) — matching the Phase 1 create-poll pattern.
	view := voteFormView{
		Poll:        poll,
		Slots:       make([]voteSlotView, len(slots)),
		DisplayName: displayName,
		Comment:     comment,
	}

	hasError := false

	if displayName == "" {
		view.NameError = nameRequiredCopy
		hasError = true
	}
	if len([]rune(displayName)) > maxDisplayNameRunes || len([]rune(comment)) > maxCommentRunes {
		hasError = true
	}

	answers := make(map[int64]string, len(slots))
	for i, sl := range slots {
		val := r.FormValue(fmt.Sprintf("answer_%d", sl.ID))
		view.Slots[i] = voteSlotView{
			ID:       sl.ID,
			Label:    slotLabel(poll.PollType, sl),
			Selected: val,
		}
		if val == "yes" || val == "no" || val == "maybe" {
			answers[sl.ID] = val
		} else {
			view.Slots[i].Error = slotAnswerRequiredCopy
			hasError = true
		}
	}

	if hasError {
		view.BannerError = bannerErrorCopy
		// Results must still be populated on a validation-error re-render —
		// mirrors handleVoteForm's GET path. Without this the results
		// section rendered blank (empty Participants/Rows) any time a
		// submission was rejected, even though the poll already had votes.
		resultParticipants, resultAnswers, err := s.store.ResponsesByPollID(r.Context(), poll.ID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		view.Results = buildResultsGridView(poll.PollType, slots, resultParticipants, resultAnswers, false, poll.Title, "", "")
		s.renderVoteForm(w, view)
		return
	}

	cookieToken := ""
	if cookie, err := r.Cookie(participantCookieName); err == nil {
		existing, _, err := s.store.ParticipantByCookie(r.Context(), poll.ID, cookie.Value)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if existing != nil {
			cookieToken = existing.CookieToken
		}
	}

	freshCookie := cookieToken == ""
	if freshCookie {
		cookieToken, err = token.New()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if _, err := s.store.SaveResponse(r.Context(), poll.ID, cookieToken, displayName, comment, answers); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if freshCookie {
		http.SetCookie(w, &http.Cookie{
			Name:     participantCookieName,
			Value:    cookieToken,
			Path:     "/poll/" + ptoken,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   participantCookieMaxAge,
			Secure:   requestIsHTTPS(r),
		})
	}

	http.Redirect(w, r, "/poll/"+ptoken+"?saved=1", http.StatusSeeOther)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// isInstanceAdminSession reports whether the request carries a session
// cookie naming a valid (not expired, per instanceAdminSessionMaxAge)
// admin_sessions row. The session token itself is a fresh crypto/rand value
// per login — not the instance-admin secret — so this DB lookup is real
// authentication, not just a cookie-presence check. A store error fails
// closed (treated as unauthenticated).
func (s *Server) isInstanceAdminSession(r *http.Request) bool {
	cookie, err := r.Cookie(instanceAdminCookieName)
	if err != nil {
		return false
	}
	valid, err := s.store.InstanceAdminSessionValid(r.Context(), cookie.Value, instanceAdminSessionMaxAge)
	if err != nil {
		return false
	}
	return valid
}

func (s *Server) renderInstanceAdminLoginForm(w http.ResponseWriter, bannerError string) {
	data := struct {
		BannerError string
		DBPath      string
	}{BannerError: bannerError, DBPath: s.dbPath}
	if err := s.tmpl.adminLogin.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (s *Server) handleInstanceAdminLoginForm(w http.ResponseWriter, r *http.Request) {
	_ = s.store.PruneInstanceAdminSessions(r.Context(), instanceAdminSessionMaxAge)
	if s.isInstanceAdminSession(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	s.renderInstanceAdminLoginForm(w, "")
}

// handleInstanceAdminLogin verifies the submitted secret in constant time
// and, on success, creates a fresh session row (a new crypto/rand token,
// distinct from the instance-admin secret and from any other session) and
// sets a session-only cookie (no MaxAge/Expires) naming it. That cookie is
// only actually honored for instanceAdminSessionMaxAge (24h) from creation
// — checked at auth time in isInstanceAdminSession, regardless of whether
// the browser keeps the cookie around longer (ADMIN-02).
func (s *Server) handleInstanceAdminLogin(w http.ResponseWriter, r *http.Request) {
	_ = s.store.PruneInstanceAdminSessions(r.Context(), instanceAdminSessionMaxAge)

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}

	secret := r.PostFormValue("secret")
	if subtle.ConstantTimeCompare([]byte(secret), []byte(s.instanceAdminSecret)) != 1 {
		s.renderInstanceAdminLoginForm(w, instanceAdminLoginErrorCopy)
		return
	}

	sessionToken, err := token.New()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.store.CreateInstanceAdminSession(r.Context(), sessionToken); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     instanceAdminCookieName,
		Value:    sessionToken,
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   requestIsHTTPS(r),
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// instanceAdminPollRow is one row of the /admin poll list (ADMIN-03).
type instanceAdminPollRow struct {
	Title           string
	CreatedDisplay  string
	ParticipantPath string
	ParticipantURL  string
	AdminPath       string
	AdminURL        string
}

func (s *Server) handleInstanceAdminPage(w http.ResponseWriter, r *http.Request) {
	_ = s.store.PruneInstanceAdminSessions(r.Context(), instanceAdminSessionMaxAge)
	if !s.isInstanceAdminSession(r) {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}

	polls, err := s.store.ListPolls(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if requestIsHTTPS(r) {
		scheme = "https"
	}

	rows := make([]instanceAdminPollRow, len(polls))
	for i, p := range polls {
		participantPath := "/poll/" + p.ParticipantToken
		adminPath := "/poll/" + p.ParticipantToken + "/admin/" + p.AdminToken
		created := p.CreatedAt
		if t, err := time.Parse(time.RFC3339, p.CreatedAt); err == nil {
			created = t.Format("Mon, Jan 2, 2006")
		}
		rows[i] = instanceAdminPollRow{
			Title:           p.Title,
			CreatedDisplay:  created,
			ParticipantPath: participantPath,
			ParticipantURL:  scheme + "://" + r.Host + participantPath,
			AdminPath:       adminPath,
			AdminURL:        scheme + "://" + r.Host + adminPath,
		}
	}

	data := struct {
		Polls []instanceAdminPollRow
	}{Polls: rows}
	if err := s.tmpl.admin.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
