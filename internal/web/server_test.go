package web

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/cfiguerola/date-chooser/internal/store"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open() error: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	srv, err := NewServer(st)
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts, st
}

var locationRe = regexp.MustCompile(`^/poll/[^/]+/admin/[^/]+$`)

func TestEndToEnd_CreatePollAndFollowLinks(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Team sync")
	form.Set("poll_type", "all_day")
	form.Set("slot_date", "2026-09-01")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}

	loc := resp.Header.Get("Location")
	if !locationRe.MatchString(loc) {
		t.Fatalf("Location header %q did not match %s", loc, locationRe.String())
	}

	// Verify DB state directly: exactly one polls row, one slots row.
	ctx := context.Background()
	var pollCount, slotCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls WHERE title = ? AND poll_type = ?", "Team sync", "all_day").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 1 {
		t.Fatalf("expected exactly 1 polls row for 'Team sync', got %d", pollCount)
	}
	if err := st.DB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM slots s
		JOIN polls p ON p.id = s.poll_id
		WHERE p.title = ? AND s.starts_at = ?
	`, "Team sync", "2026-09-01").Scan(&slotCount); err != nil {
		t.Fatalf("querying slots: %v", err)
	}
	if slotCount != 1 {
		t.Fatalf("expected exactly 1 slots row with starts_at 2026-09-01, got %d", slotCount)
	}

	// Follow the redirect: GET the Location, expect 200 with both URLs present.
	getResp, err := http.Get(ts.URL + loc)
	if err != nil {
		t.Fatalf("GET %s error: %v", loc, err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for %s, got %d", loc, getResp.StatusCode)
	}

	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	// Extract participant path (everything before "/admin/") from the Location.
	adminIdx := strings.Index(loc, "/admin/")
	if adminIdx == -1 {
		t.Fatalf("Location %q unexpectedly missing /admin/ segment", loc)
	}
	participantPath := loc[:adminIdx]

	if !strings.Contains(bodyStr, participantPath) {
		t.Fatalf("expected links page body to contain participant path %q, got: %s", participantPath, bodyStr)
	}
	if !strings.Contains(bodyStr, loc) {
		t.Fatalf("expected links page body to contain admin path %q, got: %s", loc, bodyStr)
	}
}

func TestLinksPage_RendersBothLabelsAndAbsoluteURLs(t *testing.T) {
	ts, _ := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Label Check")
	form.Set("poll_type", "all_day")
	form.Set("slot_date", "2026-09-01")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}
	loc := resp.Header.Get("Location")

	getResp, err := http.Get(ts.URL + loc)
	if err != nil {
		t.Fatalf("GET %s error: %v", loc, err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for %s, got %d", loc, getResp.StatusCode)
	}

	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Participant link (share this with your group)") {
		t.Fatalf("expected exact participant label in body, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Admin link (keep this secret)") {
		t.Fatalf("expected exact admin label in body, got: %s", bodyStr)
	}

	adminIdx := strings.Index(loc, "/admin/")
	if adminIdx == -1 {
		t.Fatalf("Location %q unexpectedly missing /admin/ segment", loc)
	}
	wantParticipantURL := ts.URL + loc[:adminIdx]
	wantAdminURL := ts.URL + loc

	if !strings.Contains(bodyStr, wantParticipantURL) {
		t.Fatalf("expected absolute participant URL %q in body, got: %s", wantParticipantURL, bodyStr)
	}
	if !strings.Contains(bodyStr, wantAdminURL) {
		t.Fatalf("expected absolute admin URL %q in body, got: %s", wantAdminURL, bodyStr)
	}
}

func TestHealthz(t *testing.T) {
	ts, _ := newTestServer(t)

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// noRedirectClient returns an *http.Client that never follows redirects, so
// tests can inspect the raw response (status code, Location header) from a
// single POST /polls call.
func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func TestCreatePoll_ValidAllDay_MultipleSlotsPersistedInOrder(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Retro")
	form.Set("poll_type", "all_day")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_date", "2026-09-02")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}

	ctx := context.Background()
	var pollID int64
	var pollType string
	if err := st.DB().QueryRowContext(ctx, "SELECT id, poll_type FROM polls WHERE title = ?", "Retro").Scan(&pollID, &pollType); err != nil {
		t.Fatalf("querying poll: %v", err)
	}
	if pollType != "all_day" {
		t.Fatalf("expected poll_type all_day, got %q", pollType)
	}

	rows, err := st.DB().QueryContext(ctx, "SELECT starts_at, ends_at FROM slots WHERE poll_id = ? ORDER BY position", pollID)
	if err != nil {
		t.Fatalf("querying slots: %v", err)
	}
	defer rows.Close()

	var starts []string
	for rows.Next() {
		var s string
		var e sql.NullString
		if err := rows.Scan(&s, &e); err != nil {
			t.Fatalf("scanning slot: %v", err)
		}
		if e.Valid {
			t.Fatalf("expected ends_at NULL for all_day slot %q, got %q", s, e.String)
		}
		starts = append(starts, s)
	}
	want := []string{"2026-09-01", "2026-09-02"}
	if !reflect.DeepEqual(starts, want) {
		t.Fatalf("expected slots %v in submitted order, got %v", want, starts)
	}
}

func TestCreatePoll_ValidDateTime_MultipleSlotsPersisted(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Planning")
	form.Set("poll_type", "date_time")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "09:00")
	form.Add("slot_end_time", "10:00")
	form.Add("slot_date", "2026-09-02")
	form.Add("slot_start_time", "14:00")
	form.Add("slot_end_time", "15:00")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}

	ctx := context.Background()
	var pollID int64
	var pollType string
	if err := st.DB().QueryRowContext(ctx, "SELECT id, poll_type FROM polls WHERE title = ?", "Planning").Scan(&pollID, &pollType); err != nil {
		t.Fatalf("querying poll: %v", err)
	}
	if pollType != "date_time" {
		t.Fatalf("expected poll_type date_time, got %q", pollType)
	}

	rows, err := st.DB().QueryContext(ctx, "SELECT starts_at, ends_at FROM slots WHERE poll_id = ? ORDER BY position", pollID)
	if err != nil {
		t.Fatalf("querying slots: %v", err)
	}
	defer rows.Close()

	type pair struct{ start, end string }
	var got []pair
	for rows.Next() {
		var s string
		var e sql.NullString
		if err := rows.Scan(&s, &e); err != nil {
			t.Fatalf("scanning slot: %v", err)
		}
		if !e.Valid {
			t.Fatalf("expected ends_at set for date_time slot %q", s)
		}
		got = append(got, pair{s, e.String})
	}
	want := []pair{
		{"2026-09-01T09:00", "2026-09-01T10:00"},
		{"2026-09-02T14:00", "2026-09-02T15:00"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected slots %v in submitted order, got %v", want, got)
	}
}

func TestCreatePoll_ZeroSlots_RejectedWithoutRedirect(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "No Slots")
	form.Set("poll_type", "all_day")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 re-render for zero-slot submit, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), "Check the highlighted fields and try again.") {
		t.Fatalf("expected validation banner copy in re-rendered body, got: %s", body)
	}

	ctx := context.Background()
	var pollCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls WHERE title = ?", "No Slots").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 0 {
		t.Fatalf("expected no poll row for zero-slot submit, got %d", pollCount)
	}
}

func TestCreatePoll_EmptyTitle_RejectedWithoutRedirect(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "")
	form.Set("poll_type", "all_day")
	form.Add("slot_date", "2026-09-01")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 re-render for empty-title submit, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), "Check the highlighted fields and try again.") {
		t.Fatalf("expected validation banner copy in re-rendered body, got: %s", body)
	}

	ctx := context.Background()
	var pollCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 0 {
		t.Fatalf("expected no poll row for empty-title submit, got %d", pollCount)
	}
}

func TestCreatePoll_DateTimeEndNotAfterStart_RejectedWithoutRedirect(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Bad Range")
	form.Set("poll_type", "date_time")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "10:00")
	form.Add("slot_end_time", "09:00") // end before start

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 re-render for invalid time range submit, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), "End time must be after start time.") {
		t.Fatalf("expected row-level error copy in re-rendered body, got: %s", body)
	}

	ctx := context.Background()
	var pollCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls WHERE title = ?", "Bad Range").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 0 {
		t.Fatalf("expected no poll row for invalid time-range submit, got %d", pollCount)
	}
}

func TestVote_EndToEnd_SubmitAndRevisit(t *testing.T) {
	ts, st := newTestServer(t)

	// Create a poll with two slots directly through the store (reuse
	// CreatePoll rather than the HTTP create flow, since this test is about
	// the voting slice, not poll creation).
	ctx := context.Background()
	endA := "2026-09-01T10:00"
	endB := "2026-09-02T15:00"
	poll, err := st.CreatePoll(ctx, store.NewPollInput{
		Title:    "Team Offsite",
		PollType: "date_time",
		Slots: []store.NewSlotInput{
			{StartsAt: "2026-09-01T09:00", EndsAt: &endA},
			{StartsAt: "2026-09-02T14:00", EndsAt: &endB},
		},
	}, "ptok-vote-e2e", "atok-vote-e2e")
	if err != nil {
		t.Fatalf("CreatePoll() error: %v", err)
	}

	_, slots, err := st.PollByParticipantToken(ctx, poll.ParticipantToken)
	if err != nil {
		t.Fatalf("PollByParticipantToken() error: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}

	votePath := "/poll/" + poll.ParticipantToken

	// GET the voting form: expect the name field and a Yes/No/Maybe control.
	getResp, err := http.Get(ts.URL + votePath)
	if err != nil {
		t.Fatalf("GET %s error: %v", votePath, err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `name="display_name"`) {
		t.Fatalf("expected display_name field in body, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, fmt.Sprintf("answer_%d", slots[0].ID)) {
		t.Fatalf("expected a Yes/No/Maybe control for slot %d in body, got: %s", slots[0].ID, bodyStr)
	}

	// POST a valid response: name + an answer for every slot.
	form := url.Values{}
	form.Set("display_name", "Alice")
	form.Set("comment", "looking forward to it")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	form.Set(fmt.Sprintf("answer_%d", slots[1].ID), "maybe")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	postResp, err := client.PostForm(ts.URL+votePath+"/responses", form)
	if err != nil {
		t.Fatalf("POST %s/responses error: %v", votePath, err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(postResp.Body)
		t.Fatalf("expected 303, got %d; body: %s", postResp.StatusCode, b)
	}
	if got := postResp.Header.Get("Location"); got != votePath+"?saved=1" {
		t.Fatalf("expected redirect to %s?saved=1, got %q", votePath, got)
	}

	var participantCookie *http.Cookie
	for _, c := range postResp.Cookies() {
		if c.Name == participantCookieName {
			participantCookie = c
		}
	}
	if participantCookie == nil {
		t.Fatalf("expected a Set-Cookie for %s, got cookies: %v", participantCookieName, postResp.Cookies())
	}
	if !participantCookie.HttpOnly {
		t.Fatalf("expected %s cookie to be HttpOnly", participantCookieName)
	}

	// DB now holds exactly one participants row and one responses row per slot.
	var participantCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&participantCount); err != nil {
		t.Fatalf("querying participants: %v", err)
	}
	if participantCount != 1 {
		t.Fatalf("expected exactly 1 participants row, got %d", participantCount)
	}
	var responseCount int
	if err := st.DB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM responses r
		JOIN participants p ON p.id = r.participant_id
		WHERE p.poll_id = ?
	`, poll.ID).Scan(&responseCount); err != nil {
		t.Fatalf("querying responses: %v", err)
	}
	if responseCount != len(slots) {
		t.Fatalf("expected %d responses rows, got %d", len(slots), responseCount)
	}

	// Re-GET carrying the cookie: prior answer for slot 0 should be pre-checked.
	req, err := http.NewRequest(http.MethodGet, ts.URL+votePath, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.AddCookie(participantCookie)
	revisitResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s (revisit) error: %v", votePath, err)
	}
	defer revisitResp.Body.Close()
	revisitBody, err := io.ReadAll(revisitResp.Body)
	if err != nil {
		t.Fatalf("reading revisit body: %v", err)
	}
	revisitStr := string(revisitBody)
	if !strings.Contains(revisitStr, `value="Alice"`) {
		t.Fatalf("expected pre-filled display name 'Alice' in revisit body, got: %s", revisitStr)
	}
	wantChecked := fmt.Sprintf(`name="answer_%d" value="yes" checked`, slots[0].ID)
	if !strings.Contains(revisitStr, wantChecked) {
		t.Fatalf("expected %q pre-checked in revisit body, got: %s", wantChecked, revisitStr)
	}

	// Resubmit a changed answer with the same cookie: participants count
	// stays at 1, and the stored answer for slot 0 changes to "no".
	form2 := url.Values{}
	form2.Set("display_name", "Alice")
	form2.Set("comment", "changed my mind")
	form2.Set(fmt.Sprintf("answer_%d", slots[0].ID), "no")
	form2.Set(fmt.Sprintf("answer_%d", slots[1].ID), "maybe")

	req2, err := http.NewRequest(http.MethodPost, ts.URL+votePath+"/responses", strings.NewReader(form2.Encode()))
	if err != nil {
		t.Fatalf("building resubmit request: %v", err)
	}
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.AddCookie(participantCookie)
	resubmitResp, err := client.Do(req2)
	if err != nil {
		t.Fatalf("resubmit POST error: %v", err)
	}
	defer resubmitResp.Body.Close()
	if resubmitResp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resubmitResp.Body)
		t.Fatalf("expected 303 on resubmit, got %d; body: %s", resubmitResp.StatusCode, b)
	}

	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&participantCount); err != nil {
		t.Fatalf("querying participants after resubmit: %v", err)
	}
	if participantCount != 1 {
		t.Fatalf("expected participants count to stay at 1 after resubmit, got %d", participantCount)
	}

	var storedAnswer string
	if err := st.DB().QueryRowContext(ctx, `
		SELECT r.answer FROM responses r
		JOIN participants p ON p.id = r.participant_id
		WHERE p.poll_id = ? AND r.slot_id = ?
	`, poll.ID, slots[0].ID).Scan(&storedAnswer); err != nil {
		t.Fatalf("querying updated response: %v", err)
	}
	if storedAnswer != "no" {
		t.Fatalf("expected stored answer for slot %d to be 'no' after resubmit, got %q", slots[0].ID, storedAnswer)
	}
}

// createVotePollWithSlots creates a poll with n all_day slots via the store
// directly (bypassing the HTTP create flow, since these tests are about vote
// submission validation, not poll creation) and returns the poll and its
// persisted slots (ordered by position).
func createVotePollWithSlots(t *testing.T, st *store.Store, title, ptoken, atoken string, n int) (*store.Poll, []store.Slot) {
	t.Helper()
	ctx := context.Background()
	slots := make([]store.NewSlotInput, n)
	for i := range slots {
		slots[i] = store.NewSlotInput{StartsAt: fmt.Sprintf("2026-09-%02d", i+1)}
	}
	poll, err := st.CreatePoll(ctx, store.NewPollInput{
		Title:    title,
		PollType: "all_day",
		Slots:    slots,
	}, ptoken, atoken)
	if err != nil {
		t.Fatalf("CreatePoll() error: %v", err)
	}
	_, persisted, err := st.PollByParticipantToken(ctx, poll.ParticipantToken)
	if err != nil {
		t.Fatalf("PollByParticipantToken() error: %v", err)
	}
	return poll, persisted
}

func TestVote_BlankName_RejectedNoWrite(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Blank Name Poll", "ptok-blank-name", "atok-blank-name", 1)

	form := url.Values{}
	form.Set("display_name", "   ")
	form.Set("comment", "hello")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")

	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Your name is required.") {
		t.Fatalf("expected name-required copy in body, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Check the highlighted fields and try again.") {
		t.Fatalf("expected banner copy in body, got: %s", bodyStr)
	}
	wantChecked := fmt.Sprintf(`name="answer_%d" value="yes" checked`, slots[0].ID)
	if !strings.Contains(bodyStr, wantChecked) {
		t.Fatalf("expected submitted answer preserved as checked, got: %s", bodyStr)
	}

	ctx := context.Background()
	var participantCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&participantCount); err != nil {
		t.Fatalf("querying participants: %v", err)
	}
	if participantCount != 0 {
		t.Fatalf("expected zero participants rows on blank-name submit, got %d", participantCount)
	}
}

func TestVote_MissingSlotAnswer_RejectedNoWrite(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Missing Answer Poll", "ptok-missing-answer", "atok-missing-answer", 2)

	form := url.Values{}
	form.Set("display_name", "Bob")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	// slots[1] intentionally left unanswered.

	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Choose Yes, No, or Maybe for this slot.") {
		t.Fatalf("expected per-slot error copy in body, got: %s", bodyStr)
	}
	wantChecked := fmt.Sprintf(`name="answer_%d" value="yes" checked`, slots[0].ID)
	if !strings.Contains(bodyStr, wantChecked) {
		t.Fatalf("expected other slot's submitted answer preserved as checked, got: %s", bodyStr)
	}

	ctx := context.Background()
	var participantCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&participantCount); err != nil {
		t.Fatalf("querying participants: %v", err)
	}
	if participantCount != 0 {
		t.Fatalf("expected zero participants rows on missing-slot-answer submit, got %d", participantCount)
	}
}

func TestVote_NameTooLong_RejectedNoWrite(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Name Too Long Poll", "ptok-name-too-long", "atok-name-too-long", 1)

	form := url.Values{}
	form.Set("display_name", strings.Repeat("a", 101))
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")

	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), "Check the highlighted fields and try again.") {
		t.Fatalf("expected banner copy in body, got: %s", body)
	}

	ctx := context.Background()
	var participantCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&participantCount); err != nil {
		t.Fatalf("querying participants: %v", err)
	}
	if participantCount != 0 {
		t.Fatalf("expected zero participants rows on name-too-long submit, got %d", participantCount)
	}
}

func TestVote_CommentTooLong_RejectedNoWrite(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Comment Too Long Poll", "ptok-comment-too-long", "atok-comment-too-long", 1)

	form := url.Values{}
	form.Set("display_name", "Carol")
	form.Set("comment", strings.Repeat("b", 501))
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")

	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), "Check the highlighted fields and try again.") {
		t.Fatalf("expected banner copy in body, got: %s", body)
	}

	ctx := context.Background()
	var participantCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&participantCount); err != nil {
		t.Fatalf("querying participants: %v", err)
	}
	if participantCount != 0 {
		t.Fatalf("expected zero participants rows on comment-too-long submit, got %d", participantCount)
	}
}

func TestVote_AllValid_StillPersists(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "All Valid Poll", "ptok-all-valid", "atok-all-valid", 2)

	form := url.Values{}
	form.Set("display_name", "Dave")
	form.Set("comment", "sounds good")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	form.Set(fmt.Sprintf("answer_%d", slots[1].ID), "no")

	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}

	ctx := context.Background()
	var participantCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&participantCount); err != nil {
		t.Fatalf("querying participants: %v", err)
	}
	if participantCount != 1 {
		t.Fatalf("expected exactly 1 participants row for a fully valid submission, got %d", participantCount)
	}
}

func TestNotFound_InvalidPollLink(t *testing.T) {
	ts, _ := newTestServer(t)

	resp, err := http.Get(ts.URL + "/poll/does-not-exist")
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "This poll link isn't valid. Double-check the link you were given.") {
		t.Fatalf("expected invalid-link copy in body, got: %s", bodyStr)
	}
	if strings.Contains(bodyStr, "Save my response") {
		t.Fatalf("expected no voting form in 404 body, got: %s", bodyStr)
	}
}

func TestResultsView_CommentsPopulated(t *testing.T) {
	participants := []store.Participant{
		{ID: 1, DisplayName: "Alice", Comment: "see you there"},
		{ID: 2, DisplayName: "Bob", Comment: "   "},
	}

	view := buildResultsGridView("all_day", nil, participants, map[int64]map[int64]string{})

	if len(view.Comments) != 1 {
		t.Fatalf("expected exactly 1 comment, got %d: %v", len(view.Comments), view.Comments)
	}
	if view.Comments[0].DisplayName != "Alice" || view.Comments[0].Comment != "see you there" {
		t.Fatalf("expected Alice's comment, got %+v", view.Comments[0])
	}
}

func TestResultsView_CommentsEmpty(t *testing.T) {
	participants := []store.Participant{
		{ID: 1, DisplayName: "Alice", Comment: ""},
		{ID: 2, DisplayName: "Bob", Comment: "   "},
	}

	view := buildResultsGridView("all_day", nil, participants, map[int64]map[int64]string{})

	if len(view.Comments) != 0 {
		t.Fatalf("expected zero comments, got %v", view.Comments)
	}
}

func TestResults_Grid_ZeroResponses_ShowsNote(t *testing.T) {
	ts, st := newTestServer(t)
	poll, _ := createVotePollWithSlots(t, st, "Empty Poll", "ptok-empty-results", "atok-empty-results", 2)

	resp, err := http.Get(ts.URL + "/poll/" + poll.ParticipantToken)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "No responses yet.") {
		t.Fatalf("expected zero-response note, got: %s", bodyStr)
	}
	if strings.Contains(bodyStr, "Best fit") {
		t.Fatalf("expected no Best fit badge at zero responses, got: %s", bodyStr)
	}
}

func TestResults_Grid_MissingCell_ShowsDash(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Missing Cell Poll", "ptok-missing-cell", "atok-missing-cell", 2)

	// Write directly through the store so only slot 0 has an answer,
	// bypassing handleSubmitResponse's "every slot must be answered" gate
	// (that validation is a Phase 2 concern, not this grid's rendering).
	ctx := context.Background()
	if _, err := st.SaveResponse(ctx, poll.ID, "cookie-missing-cell", "Carol", "", map[int64]string{slots[0].ID: "yes"}); err != nil {
		t.Fatalf("SaveResponse() error: %v", err)
	}

	resp, err := http.Get(ts.URL + "/poll/" + poll.ParticipantToken)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "—") { // em dash
		t.Fatalf("expected missing-cell dash fallback, got: %s", bodyStr)
	}
}

func TestResults_Grid_CommentsSection_RendersAndEscapes(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Comment Poll", "ptok-comment-render", "atok-comment-render", 1)

	form := url.Values{}
	form.Set("display_name", "Eve")
	form.Set("comment", "<b>x</b>")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}

	getResp, err := http.Get(ts.URL + "/poll/" + poll.ParticipantToken)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer getResp.Body.Close()
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Comments</h2>") {
		t.Fatalf("expected Comments heading, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "&lt;b&gt;x&lt;/b&gt;") {
		t.Fatalf("expected escaped comment markup, got: %s", bodyStr)
	}
	if strings.Contains(bodyStr, "<b>x</b>") {
		t.Fatalf("expected no live <b> tag from comment, got: %s", bodyStr)
	}
}

func TestResults_Grid_NoComments_SectionOmitted(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "No Comment Poll", "ptok-no-comment", "atok-no-comment", 1)

	form := url.Values{}
	form.Set("display_name", "Frank")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}

	getResp, err := http.Get(ts.URL + "/poll/" + poll.ParticipantToken)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer getResp.Body.Close()
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if strings.Contains(bodyStr, "Comments</h2>") {
		t.Fatalf("expected no Comments heading when no comments exist, got: %s", bodyStr)
	}
}

func TestResultsView_Tally(t *testing.T) {
	slots := []store.Slot{{ID: 1}, {ID: 2}}
	participants := []store.Participant{{ID: 10, DisplayName: "Alice"}, {ID: 20, DisplayName: "Bob"}}
	answers := map[int64]map[int64]string{
		10: {1: "yes", 2: "no"},
		20: {1: "yes", 2: "maybe"},
	}

	view := buildResultsGridView("all_day", slots, participants, answers)

	if len(view.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(view.Rows))
	}
	rowA, rowB := view.Rows[0], view.Rows[1]
	if rowA.YesCount != 2 || rowA.NoCount != 0 || rowA.MaybeCount != 0 {
		t.Fatalf("expected row A tally 2/0/0, got %d/%d/%d", rowA.YesCount, rowA.NoCount, rowA.MaybeCount)
	}
	if rowB.YesCount != 0 || rowB.NoCount != 1 || rowB.MaybeCount != 1 {
		t.Fatalf("expected row B tally 0/1/1, got %d/%d/%d", rowB.YesCount, rowB.NoCount, rowB.MaybeCount)
	}
}

func TestRankBestSlots_ClearWinner(t *testing.T) {
	rows := []resultsRow{{YesCount: 2, NoCount: 0}, {YesCount: 1, NoCount: 1}}
	best := rankBestSlots(rows)
	if !best[0] || best[1] {
		t.Fatalf("expected only row 0 flagged BestFit, got %v", best)
	}
}

func TestRankBestSlots_TieOnYes(t *testing.T) {
	rows := []resultsRow{{YesCount: 2, NoCount: 0}, {YesCount: 2, NoCount: 0}}
	best := rankBestSlots(rows)
	if !best[0] || !best[1] {
		t.Fatalf("expected both rows flagged BestFit on a tie, got %v", best)
	}
}

func TestRankBestSlots_TieBreakFewestNo(t *testing.T) {
	rows := []resultsRow{{YesCount: 2, NoCount: 1}, {YesCount: 2, NoCount: 0}}
	best := rankBestSlots(rows)
	if best[0] || !best[1] {
		t.Fatalf("expected only row 1 (fewest No) flagged BestFit, got %v", best)
	}
}

func TestRankBestSlots_MaybeIrrelevant(t *testing.T) {
	rows := []resultsRow{{YesCount: 1, NoCount: 0, MaybeCount: 3}, {YesCount: 1, NoCount: 0, MaybeCount: 0}}
	best := rankBestSlots(rows)
	if !best[0] || !best[1] {
		t.Fatalf("expected both rows flagged BestFit regardless of Maybe count, got %v", best)
	}
}

func TestRankBestSlots_ZeroResponses(t *testing.T) {
	rows := []resultsRow{{}, {}}
	best := rankBestSlots(rows)
	if len(best) != 0 {
		t.Fatalf("expected no rows flagged BestFit at zero responses, got %v", best)
	}
}

func TestResultsView_MissingCell(t *testing.T) {
	slots := []store.Slot{{ID: 1}, {ID: 2}}
	participants := []store.Participant{{ID: 10, DisplayName: "Alice"}}
	answers := map[int64]map[int64]string{10: {1: "yes"}}

	view := buildResultsGridView("all_day", slots, participants, answers)

	if view.Rows[1].Cells[0].Answer != "" {
		t.Fatalf("expected empty Answer for a missing cell, got %q", view.Rows[1].Cells[0].Answer)
	}
}

func TestResultsView_ZeroParticipants_HasResponsesFalse(t *testing.T) {
	slots := []store.Slot{{ID: 1}}

	view := buildResultsGridView("all_day", slots, nil, nil)

	if view.HasResponses {
		t.Fatal("expected HasResponses false with zero participants")
	}
	if len(view.Rows[0].Cells) != 0 {
		t.Fatalf("expected zero cells per row with zero participants, got %d", len(view.Rows[0].Cells))
	}
}

func TestResults_Grid_EndToEnd(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Grid Poll", "ptok-grid", "atok-grid", 2)

	postAnswer := func(name, slot0Answer, slot1Answer string) {
		form := url.Values{}
		form.Set("display_name", name)
		form.Set(fmt.Sprintf("answer_%d", slots[0].ID), slot0Answer)
		form.Set(fmt.Sprintf("answer_%d", slots[1].ID), slot1Answer)
		resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
		if err != nil {
			t.Fatalf("POST responses (%s) error: %v", name, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 303 for %s, got %d; body: %s", name, resp.StatusCode, body)
		}
	}

	postAnswer("Alice", "yes", "no")
	postAnswer("Bob", "yes", "maybe")

	getResp, err := http.Get(ts.URL + "/poll/" + poll.ParticipantToken)
	if err != nil {
		t.Fatalf("GET vote page error: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Alice") || !strings.Contains(bodyStr, "Bob") {
		t.Fatalf("expected both participant names in the results grid, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "✓") { // ✓
		t.Fatalf("expected the Yes badge glyph in the results grid, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "2 Yes · 0 No · 0 Maybe") { // "2 Yes · 0 No · 0 Maybe"
		t.Fatalf("expected slot-0 tally '2 Yes · 0 No · 0 Maybe' in the results grid, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Best fit") {
		t.Fatalf("expected a Best fit badge in the results grid, got: %s", bodyStr)
	}
}

func TestResults_AdminRouteParity(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Admin Parity Poll", "ptok-admin-parity", "atok-admin-parity", 2)

	postAnswer := func(name, slot0Answer, slot1Answer string) {
		form := url.Values{}
		form.Set("display_name", name)
		form.Set(fmt.Sprintf("answer_%d", slots[0].ID), slot0Answer)
		form.Set(fmt.Sprintf("answer_%d", slots[1].ID), slot1Answer)
		resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
		if err != nil {
			t.Fatalf("POST responses (%s) error: %v", name, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 303 for %s, got %d; body: %s", name, resp.StatusCode, body)
		}
	}

	postAnswer("Alice", "yes", "no")
	postAnswer("Bob", "yes", "maybe")

	adminPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken
	getResp, err := http.Get(ts.URL + adminPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", adminPath, err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Alice") || !strings.Contains(bodyStr, "Bob") {
		t.Fatalf("expected both participant names in the admin route's results grid, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "✓") {
		t.Fatalf("expected the Yes badge glyph in the admin route's results grid, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "2 Yes · 0 No · 0 Maybe") {
		t.Fatalf("expected slot-0 tally '2 Yes · 0 No · 0 Maybe' in the admin route's results grid, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Best fit") {
		t.Fatalf("expected a Best fit badge in the admin route's results grid, got: %s", bodyStr)
	}
	// The admin token legitimately appears earlier in the body (the admin
	// link box itself). Scope the no-leak check to the results grid markup
	// only, per T-03-02: the grid must expose no data beyond what the
	// participant route shows.
	gridIdx := strings.Index(bodyStr, `class="results-section"`)
	if gridIdx == -1 {
		t.Fatalf("expected a results-section in the admin body, got: %s", bodyStr)
	}
	gridStr := bodyStr[gridIdx:]
	if strings.Contains(gridStr, poll.AdminToken) {
		t.Fatalf("admin token leaked into the results grid markup itself, got: %s", gridStr)
	}
}

func TestCreatePoll_OversizedBody_RejectedWithoutPanic(t *testing.T) {
	ts, st := newTestServer(t)

	huge := strings.Repeat("a", 2<<20) // 2 MiB, exceeds the ~1 MiB cap
	form := url.Values{}
	form.Set("title", "Oversized")
	form.Set("poll_type", "all_day")
	form.Add("slot_date", "2026-09-01")
	form.Set("description", huge)

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 413 or a clean 200 re-render for oversized body, got %d", resp.StatusCode)
	}

	ctx := context.Background()
	var pollCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls WHERE title = ?", "Oversized").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 0 {
		t.Fatalf("expected no poll row for oversized-body submit, got %d", pollCount)
	}
}
