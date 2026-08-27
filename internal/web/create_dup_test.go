package web

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
)

// duplicateSlotText is the exact SLOT-05 row-level error copy asserted
// against the rendered form body. Kept as a literal here (rather than
// referencing the server package's duplicateSlotCopy constant) so these
// tests fail loudly if the shipped copy ever drifts from what this file
// documents.
const duplicateSlotText = "This slot is a duplicate of another slot."

func TestCreatePoll_AllDayDuplicate_Rejected(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Dup All Day")
	form.Set("poll_type", "all_day")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_date", "2026-09-01")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render for duplicate all_day submit, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), duplicateSlotText) {
		t.Fatalf("expected duplicate-slot copy in re-rendered body, got: %s", body)
	}

	ctx := context.Background()
	var pollCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls WHERE title = ?", "Dup All Day").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 0 {
		t.Fatalf("expected no poll row for duplicate all_day submit, got %d", pollCount)
	}
}

func TestCreatePoll_DateTimeDuplicate_Rejected(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Dup Date Time")
	form.Set("poll_type", "date_time")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "09:00")
	form.Add("slot_end_time", "10:00")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "09:00")
	form.Add("slot_end_time", "10:00")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render for duplicate date_time submit, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), duplicateSlotText) {
		t.Fatalf("expected duplicate-slot copy in re-rendered body, got: %s", body)
	}

	ctx := context.Background()
	var pollCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls WHERE title = ?", "Dup Date Time").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 0 {
		t.Fatalf("expected no poll row for duplicate date_time submit, got %d", pollCount)
	}
}

func TestCreatePoll_ThreeWayDuplicate_AllFlagged(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Triple Dup")
	form.Set("poll_type", "all_day")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_date", "2026-09-01")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render for three-way duplicate submit, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if got := strings.Count(string(body), duplicateSlotText); got != 3 {
		t.Fatalf("expected duplicate-slot copy to appear exactly 3 times (all rows flagged), got %d; body: %s", got, body)
	}

	ctx := context.Background()
	var pollCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls WHERE title = ?", "Triple Dup").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 0 {
		t.Fatalf("expected no poll row for three-way duplicate submit, got %d", pollCount)
	}
}

func TestCreatePoll_TwoIndependentDuplicatePairs_AllFlagged(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Two Pairs")
	form.Set("poll_type", "date_time")
	// Pair A: 09:00-10:00 x2
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "09:00")
	form.Add("slot_end_time", "10:00")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "09:00")
	form.Add("slot_end_time", "10:00")
	// Pair B: 14:00-15:00 x2 (different start/end from pair A)
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "14:00")
	form.Add("slot_end_time", "15:00")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "14:00")
	form.Add("slot_end_time", "15:00")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 re-render for two independent duplicate pairs submit, got %d; body: %s", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if got := strings.Count(string(body), duplicateSlotText); got != 4 {
		t.Fatalf("expected duplicate-slot copy to appear exactly 4 times (both pairs flagged), got %d; body: %s", got, body)
	}

	ctx := context.Background()
	var pollCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM polls WHERE title = ?", "Two Pairs").Scan(&pollCount); err != nil {
		t.Fatalf("querying polls: %v", err)
	}
	if pollCount != 0 {
		t.Fatalf("expected no poll row for two independent duplicate pairs submit, got %d", pollCount)
	}
}

func TestCreatePoll_DateTimeSameDateDifferentStart_NotDuplicate(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Same Date Diff Start")
	form.Set("poll_type", "date_time")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "09:00")
	form.Add("slot_end_time", "11:00")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_start_time", "10:00")
	form.Add("slot_end_time", "11:00")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 303 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303 redirect for same-date-different-start submit, got %d; body: %s", resp.StatusCode, body)
	}

	ctx := context.Background()
	var pollID int64
	if err := st.DB().QueryRowContext(ctx, "SELECT id FROM polls WHERE title = ?", "Same Date Diff Start").Scan(&pollID); err != nil {
		t.Fatalf("querying poll: %v", err)
	}
	var slotCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", pollID).Scan(&slotCount); err != nil {
		t.Fatalf("querying slots: %v", err)
	}
	if slotCount != 2 {
		t.Fatalf("expected 2 persisted slots for same-date-different-start submit, got %d", slotCount)
	}
}

func TestCreatePoll_AllDayDistinctDates_NotDuplicate(t *testing.T) {
	ts, st := newTestServer(t)

	form := url.Values{}
	form.Set("title", "Distinct Dates")
	form.Set("poll_type", "all_day")
	form.Add("slot_date", "2026-09-01")
	form.Add("slot_date", "2026-09-02")

	resp, err := noRedirectClient().PostForm(ts.URL+"/polls", form)
	if err != nil {
		t.Fatalf("POST /polls error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 303 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 303 redirect for distinct-dates submit, got %d; body: %s", resp.StatusCode, body)
	}

	ctx := context.Background()
	var pollID int64
	if err := st.DB().QueryRowContext(ctx, "SELECT id FROM polls WHERE title = ?", "Distinct Dates").Scan(&pollID); err != nil {
		t.Fatalf("querying poll: %v", err)
	}
	var slotCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", pollID).Scan(&slotCount); err != nil {
		t.Fatalf("querying slots: %v", err)
	}
	if slotCount != 2 {
		t.Fatalf("expected 2 persisted slots for distinct-dates submit, got %d", slotCount)
	}
}
