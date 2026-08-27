package web

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
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
// slot row whose end is not strictly after its start (or is incomplete).
const rowErrorCopy = "End time must be after start time."

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
	store *store.Store
	tmpl  *pageTemplates
}

// NewServer constructs a Server backed by the given store, parsing the
// embedded templates once.
func NewServer(st *store.Store) (*Server, error) {
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, fmt.Errorf("web: parsing templates: %w", err)
	}
	return &Server{store: st, tmpl: tmpl}, nil
}

// Routes builds the HTTP handler for all Date Chooser routes.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", s.handleCreateForm)
	mux.HandleFunc("POST /polls", s.handleCreatePoll)
	mux.HandleFunc("GET /poll/{ptoken}/admin/{atoken}", s.handleLinksPage)
	mux.HandleFunc("GET /healthz", s.handleHealthz)

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
		start, end := "", ""
		if date != "" && startTime != "" {
			start = date + "T" + startTime
		}
		if date != "" && endTime != "" {
			end = date + "T" + endTime
		}
		if valid := isValidDateTimeRange(start, end); valid {
			endCopy := end
			slots = append(slots, store.NewSlotInput{StartsAt: start, EndsAt: &endCopy})
		} else {
			row.Error = rowErrorCopy
			hasRowError = true
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

	poll, _, err := s.store.PollByTokens(r.Context(), ptoken, atoken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	participantPath := "/poll/" + poll.ParticipantToken
	adminPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken

	data := struct {
		Poll            *store.Poll
		ParticipantPath string
		AdminPath       string
		ParticipantURL  string
		AdminURL        string
	}{
		Poll:            poll,
		ParticipantPath: participantPath,
		AdminPath:       adminPath,
		ParticipantURL:  scheme + "://" + r.Host + participantPath,
		AdminURL:        scheme + "://" + r.Host + adminPath,
	}

	if err := s.tmpl.links.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
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
