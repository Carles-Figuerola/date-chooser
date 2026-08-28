package web

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/cfiguerola/date-chooser/internal/store"
)

// testInstanceAdminSecret is the fixed instance-admin secret used across
// every test server — tests that exercise /admin/login submit this exact
// value rather than reading it back out of the database.
const testInstanceAdminSecret = "test-instance-admin-secret"

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open() error: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	if _, err := st.EnsureInstanceAdminSecret(context.Background(), testInstanceAdminSecret); err != nil {
		t.Fatalf("EnsureInstanceAdminSecret() error: %v", err)
	}

	srv, err := NewServer(st, testInstanceAdminSecret, dbPath)
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

	wantEditHref := `href="` + loc + `/edit"`
	if !strings.Contains(bodyStr, wantEditHref) {
		t.Fatalf("expected an %q link to the edit page in body, got: %s", wantEditHref, bodyStr)
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

// TestVote_ValidationError_StillShowsExistingResults proves that rejecting
// a submission (e.g. a missing slot answer) still populates the results
// section from whatever responses already exist for the poll, mirroring
// the GET path's own render. Before this fix, the re-render's Results view
// was left at its zero value, so the results table went blank on any
// rejected submission even when the poll already had other votes.
func TestVote_ValidationError_StillShowsExistingResults(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Existing Results Poll", "ptok-existing-results", "atok-existing-results", 2)

	aliceForm := url.Values{}
	aliceForm.Set("display_name", "Alice")
	aliceForm.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	aliceForm.Set(fmt.Sprintf("answer_%d", slots[1].ID), "no")
	if resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", aliceForm); err != nil {
		t.Fatalf("POST responses (Alice) error: %v", err)
	} else {
		resp.Body.Close()
	}

	bobForm := url.Values{}
	bobForm.Set("display_name", "Bob")
	bobForm.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	// slots[1] intentionally left unanswered, to trigger the rejection.

	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", bobForm)
	if err != nil {
		t.Fatalf("POST responses (Bob) error: %v", err)
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
	if !strings.Contains(bodyStr, "Alice") {
		t.Fatalf("expected Alice's existing response to still appear in the results section, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "results-badge-yes") || !strings.Contains(bodyStr, "results-badge-no") {
		t.Fatalf("expected Alice's yes/no badges to still render in the results table, got: %s", bodyStr)
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

	view := buildResultsGridView("all_day", nil, participants, map[int64]map[int64]string{}, false, "", "", "")

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

	view := buildResultsGridView("all_day", nil, participants, map[int64]map[int64]string{}, false, "", "", "")

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

	view := buildResultsGridView("all_day", slots, participants, answers, false, "", "", "")

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

	view := buildResultsGridView("all_day", slots, participants, answers, false, "", "", "")

	if view.Rows[1].Cells[0].Answer != "" {
		t.Fatalf("expected empty Answer for a missing cell, got %q", view.Rows[1].Cells[0].Answer)
	}
}

func TestResultsView_ZeroParticipants_HasResponsesFalse(t *testing.T) {
	slots := []store.Slot{{ID: 1}}

	view := buildResultsGridView("all_day", slots, nil, nil, false, "", "", "")

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
	// link box itself) AND, as of Plan 04-02, inside the results grid's
	// admin-controls row — each per-participant "Remove" control's delete
	// form action URL necessarily embeds it (T-04-01: authorization is
	// possession of the admin token in the URL). That supersedes Phase 3's
	// blanket "grid never contains the admin token" assumption. What must
	// still hold, per T-03-02/T-04-06, is that no participant's COOKIE token
	// is ever exposed (guaranteed structurally: resultsParticipant only ever
	// carries ID/DisplayName, never CookieToken) and that the admin token
	// appears ONLY inside the admin-controls row, never attached to the
	// participant-data cells above it.
	gridIdx := strings.Index(bodyStr, `class="results-section"`)
	if gridIdx == -1 {
		t.Fatalf("expected a results-section in the admin body, got: %s", bodyStr)
	}
	gridStr := bodyStr[gridIdx:]
	adminRowIdx := strings.Index(gridStr, `class="results-admin-row"`)
	if adminRowIdx == -1 {
		t.Fatalf("expected a results-admin-row in the admin route's grid, got: %s", gridStr)
	}
	beforeAdminRow := gridStr[:adminRowIdx]
	if strings.Contains(beforeAdminRow, poll.AdminToken) {
		t.Fatalf("admin token leaked into the results grid outside the admin-controls row, got: %s", beforeAdminRow)
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

// createEditPollWithSlots creates a poll with n all_day slots via the store
// directly and returns the poll and its persisted slots (ordered by
// position) — the seed helper for edit-route tests (mirrors
// createVotePollWithSlots).
func createEditPollWithSlots(t *testing.T, st *store.Store, title, ptoken, atoken string, n int) (*store.Poll, []store.Slot) {
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

func TestEdit_TitleDescription_EndToEnd(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createEditPollWithSlots(t, st, "Original Title", "ptok-edit-e2e", "atok-edit-e2e", 1)

	editPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken + "/edit"

	getResp, err := http.Get(ts.URL + editPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", editPath, err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for GET %s, got %d", editPath, getResp.StatusCode)
	}
	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading GET body: %v", err)
	}
	getBodyStr := string(getBody)

	if !strings.Contains(getBodyStr, "Edit poll") {
		t.Fatalf("expected 'Edit poll' in body, got: %s", getBodyStr)
	}
	if !strings.Contains(getBodyStr, "Save changes") {
		t.Fatalf("expected 'Save changes' in body, got: %s", getBodyStr)
	}
	if !strings.Contains(getBodyStr, `value="Original Title"`) {
		t.Fatalf("expected the poll's current title pre-filled, got: %s", getBodyStr)
	}
	if !strings.Contains(getBodyStr, `<input type="radio" name="poll_type" value="all_day" checked disabled>`) {
		t.Fatalf("expected the poll-type radio to carry the disabled attribute, got: %s", getBodyStr)
	}

	form := url.Values{}
	form.Set("title", "Updated Title")
	form.Set("description", "Updated description")
	form.Set("organizer_name", "")
	// Resend the existing slot row unchanged, as the real form's hidden
	// slot_id/slot_date inputs always would — a row's ID missing from the
	// submission is treated as removed (Task 2), so this must be present
	// for a title-only edit to leave the slot list untouched.
	form.Add("slot_id", fmt.Sprintf("%d", slots[0].ID))
	form.Add("slot_date", slots[0].StartsAt)

	postResp, err := noRedirectClient().PostForm(ts.URL+editPath, form)
	if err != nil {
		t.Fatalf("POST %s error: %v", editPath, err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(postResp.Body)
		t.Fatalf("expected 303, got %d; body: %s", postResp.StatusCode, b)
	}
	wantLocation := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken
	if loc := postResp.Header.Get("Location"); loc != wantLocation {
		t.Fatalf("expected Location %q, got %q", wantLocation, loc)
	}

	participantResp, err := http.Get(ts.URL + "/poll/" + poll.ParticipantToken)
	if err != nil {
		t.Fatalf("GET participant page error: %v", err)
	}
	defer participantResp.Body.Close()
	participantBody, err := io.ReadAll(participantResp.Body)
	if err != nil {
		t.Fatalf("reading participant body: %v", err)
	}
	if !strings.Contains(string(participantBody), "Updated Title") {
		t.Fatalf("expected updated title on participant page, got: %s", participantBody)
	}
}

func TestEdit_MismatchedTokens_NotFound(t *testing.T) {
	ts, st := newTestServer(t)
	pollA, _ := createEditPollWithSlots(t, st, "Poll A", "ptok-mismatch-a", "atok-mismatch-a", 1)
	pollB, _ := createEditPollWithSlots(t, st, "Poll B", "ptok-mismatch-b", "atok-mismatch-b", 1)

	// Cross the tokens: pollA's participant token with pollB's admin token.
	crossedPath := "/poll/" + pollA.ParticipantToken + "/admin/" + pollB.AdminToken + "/edit"

	getResp, err := http.Get(ts.URL + crossedPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", crossedPath, err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for GET with mismatched tokens, got %d", getResp.StatusCode)
	}

	form := url.Values{}
	form.Set("title", "Hijacked Title")
	postResp, err := noRedirectClient().PostForm(ts.URL+crossedPath, form)
	if err != nil {
		t.Fatalf("POST %s error: %v", crossedPath, err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for POST with mismatched tokens, got %d", postResp.StatusCode)
	}
}

// submitVote POSTs a vote-form response for the given participant name,
// answering every slot "yes", via the real HTTP endpoint (so a real
// participants+responses row pair is created exactly as production would).
func submitVote(t *testing.T, ts *httptest.Server, ptoken, displayName string, slots []store.Slot) {
	t.Helper()
	form := url.Values{}
	form.Set("display_name", displayName)
	for _, sl := range slots {
		form.Set(fmt.Sprintf("answer_%d", sl.ID), "yes")
	}
	resp, err := (&http.Client{}).PostForm(ts.URL+"/poll/"+ptoken+"/responses", form)
	if err != nil {
		t.Fatalf("POST /responses error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 after following the vote redirect, got %d; body: %s", resp.StatusCode, body)
	}
}

func TestEdit_SlotRemovalWithoutConfirm_RejectedNoWrite(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createEditPollWithSlots(t, st, "Removal No Confirm Poll", "ptok-removal-noconfirm", "atok-removal-noconfirm", 2)

	submitVote(t, ts, poll.ParticipantToken, "Alice", slots)
	submitVote(t, ts, poll.ParticipantToken, "Bob", slots)

	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM responses"); got != 4 {
		t.Fatalf("expected 4 responses before the edit attempt, got %d", got)
	}

	editPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken + "/edit"
	form := url.Values{}
	form.Set("title", poll.Title)
	// Only resend slots[1] — slots[0]'s row is dropped from the DOM,
	// implicitly marking it for removal. slots[0] has responses, so this
	// removal is destructive and requires confirm_slot_removal, which is
	// deliberately omitted here.
	form.Add("slot_id", fmt.Sprintf("%d", slots[1].ID))
	form.Add("slot_date", slots[1].StartsAt)

	resp, err := noRedirectClient().PostForm(ts.URL+editPath, form)
	if err != nil {
		t.Fatalf("POST %s error: %v", editPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render without confirm, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), confirmSlotRemovalCopy) {
		t.Fatalf("expected confirm-removal banner copy in body, got: %s", body)
	}

	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", poll.ID); got != 2 {
		t.Fatalf("expected slots unchanged (2) after rejected removal, got %d", got)
	}
	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM responses"); got != 4 {
		t.Fatalf("expected responses unchanged (4) after rejected removal, got %d", got)
	}
}

func TestEdit_SlotRemovalWithConfirm_Cascades(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createEditPollWithSlots(t, st, "Removal With Confirm Poll", "ptok-removal-confirm", "atok-removal-confirm", 2)

	submitVote(t, ts, poll.ParticipantToken, "Alice", slots)
	submitVote(t, ts, poll.ParticipantToken, "Bob", slots)

	editPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken + "/edit"
	form := url.Values{}
	form.Set("title", poll.Title)
	form.Add("slot_id", fmt.Sprintf("%d", slots[1].ID))
	form.Add("slot_date", slots[1].StartsAt)
	form.Set("confirm_slot_removal", "1")

	resp, err := noRedirectClient().PostForm(ts.URL+editPath, form)
	if err != nil {
		t.Fatalf("POST %s error: %v", editPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303 with confirm flag set, got %d; body: %s", resp.StatusCode, body)
	}
	wantLocation := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken
	if loc := resp.Header.Get("Location"); loc != wantLocation {
		t.Fatalf("expected Location %q, got %q", wantLocation, loc)
	}

	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", poll.ID); got != 1 {
		t.Fatalf("expected 1 remaining slot after confirmed removal, got %d", got)
	}
	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM responses"); got != 2 {
		t.Fatalf("expected 2 remaining responses (only the kept slot's) after confirmed removal, got %d", got)
	}
	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM responses WHERE slot_id = ?", slots[0].ID); got != 0 {
		t.Fatalf("expected zero responses left for the removed slot, got %d", got)
	}
}

func TestEdit_AddSlot_ShowsDashForExistingParticipants(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createEditPollWithSlots(t, st, "Add Slot Poll", "ptok-add-slot", "atok-add-slot", 1)

	submitVote(t, ts, poll.ParticipantToken, "Alice", slots)

	editPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken + "/edit"
	form := url.Values{}
	form.Set("title", poll.Title)
	form.Add("slot_id", fmt.Sprintf("%d", slots[0].ID))
	form.Add("slot_date", slots[0].StartsAt)
	// A brand-new row: empty slot_id, a fresh date.
	form.Add("slot_id", "")
	form.Add("slot_date", "2026-09-15")

	resp, err := noRedirectClient().PostForm(ts.URL+editPath, form)
	if err != nil {
		t.Fatalf("POST %s error: %v", editPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}

	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", poll.ID); got != 2 {
		t.Fatalf("expected 2 slots after add, got %d", got)
	}

	adminResp, err := http.Get(ts.URL + "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken)
	if err != nil {
		t.Fatalf("GET admin page error: %v", err)
	}
	defer adminResp.Body.Close()
	adminBody, err := io.ReadAll(adminResp.Body)
	if err != nil {
		t.Fatalf("reading admin body: %v", err)
	}
	adminBodyStr := string(adminBody)

	gridIdx := strings.Index(adminBodyStr, `class="results-section"`)
	if gridIdx == -1 {
		t.Fatalf("expected a results-section in the admin body, got: %s", adminBodyStr)
	}
	gridStr := adminBodyStr[gridIdx:]
	if !strings.Contains(gridStr, `<span class="results-missing">—</span>`) {
		t.Fatalf("expected the missing-cell dash for the existing participant in the new slot's column, got: %s", gridStr)
	}
}

func TestEdit_SlotRemovalWarningMarkupPresent(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createEditPollWithSlots(t, st, "Warning Markup Poll", "ptok-warning-markup", "atok-warning-markup", 1)

	submitVote(t, ts, poll.ParticipantToken, "Alice", slots)

	editPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken + "/edit"
	resp, err := http.Get(ts.URL + editPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", editPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "This slot has") {
		t.Fatalf("expected the per-slot removal-warning scaffold ('This slot has') in body, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `data-response-count="1"`) {
		t.Fatalf("expected data-response-count=\"1\" for the responded slot's row, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `id="confirm-slot-removal"`) {
		t.Fatalf("expected the aggregate confirm checkbox in body, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "I understand this will permanently delete these responses.") {
		t.Fatalf("expected the confirm checkbox label copy in body, got: %s", bodyStr)
	}
}

func TestEdit_ScriptBearingTitle_Escaped(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createEditPollWithSlots(t, st, "Escape Test Poll", "ptok-edit-xss", "atok-edit-xss", 1)

	editPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken + "/edit"
	maliciousTitle := `<script>alert(1)</script>`

	form := url.Values{}
	form.Set("title", maliciousTitle)
	form.Set("description", "")
	form.Set("organizer_name", "")
	form.Add("slot_id", fmt.Sprintf("%d", slots[0].ID))
	form.Add("slot_date", slots[0].StartsAt)

	resp, err := noRedirectClient().PostForm(ts.URL+editPath, form)
	if err != nil {
		t.Fatalf("POST %s error: %v", editPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303, got %d; body: %s", resp.StatusCode, body)
	}

	getResp, err := http.Get(ts.URL + editPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", editPath, err)
	}
	defer getResp.Body.Close()
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if strings.Contains(bodyStr, maliciousTitle) {
		t.Fatalf("expected the script-bearing title to be HTML-escaped, found raw markup in body: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "&lt;script&gt;") {
		t.Fatalf("expected the escaped form '&lt;script&gt;' in body, got: %s", bodyStr)
	}
}

// countRowsWeb is server_test.go's local equivalent of the store package's
// countRows test helper (kept separate — different package, same shape).
func countRowsWeb(t *testing.T, st *store.Store, query string, args ...interface{}) int {
	t.Helper()
	var n int
	if err := st.DB().QueryRowContext(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q error: %v", query, err)
	}
	return n
}

// participantIDByNameWeb returns the id of the participant with the given
// display name in pollID, failing the test if not found. Mirrors the store
// package's participantIDByName helper (different package, same shape).
func participantIDByNameWeb(t *testing.T, st *store.Store, pollID int64, name string) int64 {
	t.Helper()
	var id int64
	if err := st.DB().QueryRowContext(context.Background(), "SELECT id FROM participants WHERE poll_id = ? AND display_name = ?", pollID, name).Scan(&id); err != nil {
		t.Fatalf("querying participant id for %q: %v", name, err)
	}
	return id
}

func TestDeleteResponse_EndToEnd(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Delete Response Poll", "ptok-delresp-e2e", "atok-delresp-e2e", 2)

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

	aliceID := participantIDByNameWeb(t, st, poll.ID, "Alice")

	deletePath := fmt.Sprintf("/poll/%s/admin/%s/responses/%d/delete", poll.ParticipantToken, poll.AdminToken, aliceID)
	delResp, err := noRedirectClient().Post(ts.URL+deletePath, "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST %s error: %v", deletePath, err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(delResp.Body)
		t.Fatalf("expected 303, got %d; body: %s", delResp.StatusCode, body)
	}
	wantLocation := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken
	if got := delResp.Header.Get("Location"); got != wantLocation {
		t.Fatalf("expected Location %q, got %q", wantLocation, got)
	}

	adminPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken
	getResp, err := http.Get(ts.URL + adminPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", adminPath, err)
	}
	defer getResp.Body.Close()
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if strings.Contains(bodyStr, "Alice") {
		t.Fatalf("expected Alice's name gone from the grid after deletion, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Bob") {
		t.Fatalf("expected Bob to remain in the grid, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "1 Yes · 0 No · 0 Maybe") {
		t.Fatalf("expected slot-0 tally recomputed to '1 Yes · 0 No · 0 Maybe' (Bob only), got: %s", bodyStr)
	}

	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID); got != 1 {
		t.Fatalf("expected 1 remaining participant, got %d", got)
	}
}

func TestDeleteResponse_WrongTokenPair_404(t *testing.T) {
	ts, st := newTestServer(t)
	pollA, slotsA := createVotePollWithSlots(t, st, "Wrong Token Poll A", "ptok-delresp-wrong-a", "atok-delresp-wrong-a", 1)
	pollB, _ := createVotePollWithSlots(t, st, "Wrong Token Poll B", "ptok-delresp-wrong-b", "atok-delresp-wrong-b", 1)

	form := url.Values{}
	form.Set("display_name", "Alice")
	form.Set(fmt.Sprintf("answer_%d", slotsA[0].ID), "yes")
	resp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+pollA.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	resp.Body.Close()

	aliceID := participantIDByNameWeb(t, st, pollA.ID, "Alice")

	// Mismatched pair: pollA's participant token with pollB's admin token.
	deletePath := fmt.Sprintf("/poll/%s/admin/%s/responses/%d/delete", pollA.ParticipantToken, pollB.AdminToken, aliceID)
	delResp, err := noRedirectClient().Post(ts.URL+deletePath, "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST %s error: %v", deletePath, err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(delResp.Body)
		t.Fatalf("expected 404 for mismatched token pair, got %d; body: %s", delResp.StatusCode, body)
	}

	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", pollA.ID); got != 1 {
		t.Fatalf("expected pollA's participant untouched by a mismatched-token delete attempt, got %d", got)
	}
}

func TestResults_AdminRow_OnlyOnAdminAndWhenResponses(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Admin Row Poll", "ptok-adminrow", "atok-adminrow", 1)

	adminPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken

	// Zero responses: no admin-controls row rendered even on the admin route.
	zeroRespResp, err := http.Get(ts.URL + adminPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", adminPath, err)
	}
	defer zeroRespResp.Body.Close()
	zeroRespBody, err := io.ReadAll(zeroRespResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if strings.Contains(string(zeroRespBody), "results-admin-row") {
		t.Fatalf("expected no results-admin-row with zero responses, got: %s", zeroRespBody)
	}

	form := url.Values{}
	form.Set("display_name", "Alice")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	voteResp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	voteResp.Body.Close()

	// Admin route with responses: row + Remove button present.
	adminGetResp, err := http.Get(ts.URL + adminPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", adminPath, err)
	}
	defer adminGetResp.Body.Close()
	adminBody, err := io.ReadAll(adminGetResp.Body)
	if err != nil {
		t.Fatalf("reading admin body: %v", err)
	}
	adminBodyStr := string(adminBody)
	if !strings.Contains(adminBodyStr, "results-admin-row") {
		t.Fatalf("expected results-admin-row on admin route with responses, got: %s", adminBodyStr)
	}
	if !strings.Contains(adminBodyStr, ">Remove<") {
		t.Fatalf("expected a Remove button on the admin route, got: %s", adminBodyStr)
	}

	// Participant route with the same data: no admin row at all.
	participantPath := "/poll/" + poll.ParticipantToken
	partResp, err := http.Get(ts.URL + participantPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", participantPath, err)
	}
	defer partResp.Body.Close()
	partBody, err := io.ReadAll(partResp.Body)
	if err != nil {
		t.Fatalf("reading participant body: %v", err)
	}
	if strings.Contains(string(partBody), "results-admin-row") {
		t.Fatalf("expected no results-admin-row on the participant route, got: %s", partBody)
	}
}

// TestAdminJS_ConfirmCopyAndDoubleSubmitGuard proves admin.js exists and
// gates the delete-response form's submission on window.confirm with the
// exact copy shape from 04-UI-SPEC.md, and disables the submit button on
// confirm as the double-submit backstop (T-04-04) — since this behavior
// runs in a browser (not Go), the strongest mechanical check available here
// is asserting the shipped JS source contains it.
func TestAdminJS_ConfirmCopyAndDoubleSubmitGuard(t *testing.T) {
	data, err := os.ReadFile("static/admin.js")
	if err != nil {
		t.Fatalf("reading static/admin.js: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "window.confirm") {
		t.Fatalf("expected admin.js to gate the delete on window.confirm, got: %s", content)
	}
	if !strings.Contains(content, `"Remove " + name + "'s response to \"" + title + "\"? This cannot be undone."`) {
		t.Fatalf("expected admin.js to build the exact confirm() copy shape, got: %s", content)
	}
	if !strings.Contains(content, "button.disabled = true") {
		t.Fatalf("expected admin.js to disable the submit button on confirm (double-submit guard), got: %s", content)
	}
}

// TestVoteJS_SetAllConditionalConfirm proves vote.js's "Set all to X"
// buttons gate window.confirm on whether the slots already share one
// identical answer (skip confirm), are entirely unanswered (skip confirm —
// nothing to overwrite), or have mixed answers (confirm required), and that
// the click handler selects each slot's radio by the button's target value.
func TestVoteJS_SetAllConditionalConfirm(t *testing.T) {
	data, err := os.ReadFile("static/vote.js")
	if err != nil {
		t.Fatalf("reading static/vote.js: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "data-set-all") {
		t.Fatalf("expected vote.js to wire up buttons via data-set-all, got no match")
	}
	if !strings.Contains(content, "window.confirm") {
		t.Fatalf("expected vote.js to gate on window.confirm, got no match")
	}
	if !strings.Contains(content, "allSameAnswer") {
		t.Fatalf("expected vote.js to compute whether all slots share one answer, got no match")
	}
	if !strings.Contains(content, "hasAnyAnswer") {
		t.Fatalf("expected vote.js to compute whether any slot has an answer, so an all-empty poll skips the confirm, got no match")
	}
	if !strings.Contains(content, `input[type="radio"][value="' + target + '"]`) {
		t.Fatalf("expected vote.js to select each slot's radio by the button's target value, got no match")
	}
}

func TestVote_SetAllButtonsRendered(t *testing.T) {
	ts, st := newTestServer(t)
	poll, _ := createVotePollWithSlots(t, st, "Set All Poll", "ptok-set-all", "atok-set-all", 2)

	resp, err := noRedirectClient().Get(ts.URL + "/poll/" + poll.ParticipantToken)
	if err != nil {
		t.Fatalf("GET /poll/{ptoken} error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	for _, id := range []string{"set-all-yes-btn", "set-all-no-btn", "set-all-maybe-btn"} {
		if !strings.Contains(string(body), `id="`+id+`"`) {
			t.Fatalf("expected the rendered vote page to contain the %s button, got: %s", id, body)
		}
	}
}

// TestDeletePoll_EndToEnd proves the whole-poll delete route (Plan 04-03,
// ADM-04) removes the poll and cascades to its responses: a POST to the
// delete route redirects to the (now-invalid) participant link, and both
// that link and the admin link 404 afterward.
func TestDeletePoll_EndToEnd(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createVotePollWithSlots(t, st, "Delete Poll E2E", "ptok-delpoll-e2e", "atok-delpoll-e2e", 1)

	form := url.Values{}
	form.Set("display_name", "Alice")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	voteResp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	voteResp.Body.Close()

	deletePath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken + "/delete"
	delResp, err := noRedirectClient().Post(ts.URL+deletePath, "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST %s error: %v", deletePath, err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(delResp.Body)
		t.Fatalf("expected 303, got %d; body: %s", delResp.StatusCode, body)
	}
	wantLocation := "/poll/" + poll.ParticipantToken
	if got := delResp.Header.Get("Location"); got != wantLocation {
		t.Fatalf("expected Location %q, got %q", wantLocation, got)
	}

	participantPath := "/poll/" + poll.ParticipantToken
	partResp, err := http.Get(ts.URL + participantPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", participantPath, err)
	}
	defer partResp.Body.Close()
	if partResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(partResp.Body)
		t.Fatalf("expected 404 for the participant link after delete, got %d; body: %s", partResp.StatusCode, body)
	}

	adminPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken
	adminResp, err := http.Get(ts.URL + adminPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", adminPath, err)
	}
	defer adminResp.Body.Close()
	if adminResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(adminResp.Body)
		t.Fatalf("expected 404 for the admin link after delete, got %d; body: %s", adminResp.StatusCode, body)
	}

	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM polls WHERE id = ?", poll.ID); got != 0 {
		t.Fatalf("expected the poll row deleted, got %d", got)
	}
}

// TestDeletePoll_WrongTokenPair_404 proves a mismatched ptoken/atoken pair
// on the delete-poll route 404s and leaves the poll intact (T-04-02).
func TestDeletePoll_WrongTokenPair_404(t *testing.T) {
	ts, st := newTestServer(t)
	pollA, _ := createVotePollWithSlots(t, st, "Delete Poll Wrong Token A", "ptok-delpoll-wrong-a", "atok-delpoll-wrong-a", 1)
	pollB, _ := createVotePollWithSlots(t, st, "Delete Poll Wrong Token B", "ptok-delpoll-wrong-b", "atok-delpoll-wrong-b", 1)

	// Mismatched pair: pollA's participant token with pollB's admin token.
	deletePath := "/poll/" + pollA.ParticipantToken + "/admin/" + pollB.AdminToken + "/delete"
	delResp, err := noRedirectClient().Post(ts.URL+deletePath, "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST %s error: %v", deletePath, err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(delResp.Body)
		t.Fatalf("expected 404 for mismatched token pair, got %d; body: %s", delResp.StatusCode, body)
	}

	if got := countRowsWeb(t, st, "SELECT COUNT(*) FROM polls WHERE id = ?", pollA.ID); got != 1 {
		t.Fatalf("expected pollA untouched by a mismatched-token delete attempt, got %d", got)
	}

	// A follow-up GET of the CORRECT admin link still resolves (200).
	correctAdminPath := "/poll/" + pollA.ParticipantToken + "/admin/" + pollA.AdminToken
	correctResp, err := http.Get(ts.URL + correctAdminPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", correctAdminPath, err)
	}
	defer correctResp.Body.Close()
	if correctResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(correctResp.Body)
		t.Fatalf("expected 200 for pollA's correct admin link, got %d; body: %s", correctResp.StatusCode, body)
	}
}

// TestDeletePoll_DangerZoneRendered proves the edit page renders the
// danger-zone section with the exact copy, a btn-danger "Delete poll"
// button whose form posts to the delete route, and the poll title +
// response count carried in data-* attributes for the confirm() copy.
func TestDeletePoll_DangerZoneRendered(t *testing.T) {
	ts, st := newTestServer(t)
	poll, slots := createEditPollWithSlots(t, st, "Danger Zone Poll", "ptok-dangerzone", "atok-dangerzone", 1)

	form := url.Values{}
	form.Set("display_name", "Alice")
	form.Set(fmt.Sprintf("answer_%d", slots[0].ID), "yes")
	voteResp, err := noRedirectClient().PostForm(ts.URL+"/poll/"+poll.ParticipantToken+"/responses", form)
	if err != nil {
		t.Fatalf("POST responses error: %v", err)
	}
	voteResp.Body.Close()

	editPath := "/poll/" + poll.ParticipantToken + "/admin/" + poll.AdminToken + "/edit"
	getResp, err := http.Get(ts.URL + editPath)
	if err != nil {
		t.Fatalf("GET %s error: %v", editPath, err)
	}
	defer getResp.Body.Close()
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Danger zone") {
		t.Fatalf("expected 'Danger zone' heading, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Deleting this poll removes it and all responses permanently. This cannot be undone.") {
		t.Fatalf("expected the danger-zone body copy, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `class="btn-danger"`) || !strings.Contains(bodyStr, ">Delete poll<") {
		t.Fatalf("expected a btn-danger 'Delete poll' button, got: %s", bodyStr)
	}
	wantAction := `action="/poll/` + poll.ParticipantToken + `/admin/` + poll.AdminToken + `/delete"`
	if !strings.Contains(bodyStr, wantAction) {
		t.Fatalf("expected the delete-poll form to post to %q, got: %s", wantAction, bodyStr)
	}
	if !strings.Contains(bodyStr, `data-poll-title="Danger Zone Poll"`) {
		t.Fatalf("expected data-poll-title carrying the poll's title, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `data-response-count="1"`) {
		t.Fatalf("expected data-response-count reflecting the 1 seeded response, got: %s", bodyStr)
	}
}

// TestEditJS_DeletePollConfirmAndDoubleSubmitGuard proves edit.js gates the
// delete-poll form's submission on window.confirm with the exact copy
// shape from 04-UI-SPEC.md, and disables the submit button on confirm as
// the double-submit backstop (T-04-04) — mirrors
// TestAdminJS_ConfirmCopyAndDoubleSubmitGuard's approach of asserting the
// shipped JS source, since this behavior runs in a browser, not Go.
func TestEditJS_DeletePollConfirmAndDoubleSubmitGuard(t *testing.T) {
	data, err := os.ReadFile("static/edit.js")
	if err != nil {
		t.Fatalf("reading static/edit.js: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "delete-poll-form") {
		t.Fatalf("expected edit.js to gate the '.delete-poll-form' submit, got: %s", content)
	}
	if !strings.Contains(content, "window.confirm") {
		t.Fatalf("expected edit.js to gate the delete-poll submit on window.confirm, got: %s", content)
	}
	if !strings.Contains(content, `"Delete \"" + title + "\"? This will permanently delete the poll and its " +`) {
		t.Fatalf("expected edit.js to build the exact delete-poll confirm() copy shape, got: %s", content)
	}
	if !strings.Contains(content, "deleteButton.disabled = true") {
		t.Fatalf("expected edit.js to disable the delete-poll submit button on confirm (double-submit guard), got: %s", content)
	}
}
