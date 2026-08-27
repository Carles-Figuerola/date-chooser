package web

import (
	"context"
	"database/sql"
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
