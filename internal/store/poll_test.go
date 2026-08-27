package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestListPolls_ReturnsAllPollsNewestFirst(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()

	first, err := st.CreatePoll(ctx, NewPollInput{
		Title:    "First Poll",
		PollType: "all_day",
		Slots:    []NewSlotInput{{StartsAt: "2026-09-01"}},
	}, "ptok-list-1", "atok-list-1")
	if err != nil {
		t.Fatalf("CreatePoll() (first) error: %v", err)
	}
	second, err := st.CreatePoll(ctx, NewPollInput{
		Title:    "Second Poll",
		PollType: "all_day",
		Slots:    []NewSlotInput{{StartsAt: "2026-09-02"}},
	}, "ptok-list-2", "atok-list-2")
	if err != nil {
		t.Fatalf("CreatePoll() (second) error: %v", err)
	}

	polls, err := st.ListPolls(ctx)
	if err != nil {
		t.Fatalf("ListPolls() error: %v", err)
	}
	if len(polls) != 2 {
		t.Fatalf("expected 2 polls, got %d", len(polls))
	}

	byID := make(map[int64]Poll, len(polls))
	for _, p := range polls {
		byID[p.ID] = p
	}
	if _, ok := byID[first.ID]; !ok {
		t.Fatalf("expected first poll (id=%d) in ListPolls() result", first.ID)
	}
	if _, ok := byID[second.ID]; !ok {
		t.Fatalf("expected second poll (id=%d) in ListPolls() result", second.ID)
	}
	if polls[0].ParticipantToken != "ptok-list-2" {
		t.Fatalf("expected the most recently created poll first, got participant token %q", polls[0].ParticipantToken)
	}
}
