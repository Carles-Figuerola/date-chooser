package store

import (
	"context"
	"testing"

	"github.com/cfiguerola/date-chooser/internal/token"
)

func TestResults_ResponsesByPollID_OrderedParticipantsAndAnswers(t *testing.T) {
	st, poll, slots := newTestStore(t)
	ctx := context.Background()

	cookieA, err := token.New()
	if err != nil {
		t.Fatalf("token.New() error: %v", err)
	}
	cookieB, err := token.New()
	if err != nil {
		t.Fatalf("token.New() error: %v", err)
	}

	if _, err := st.SaveResponse(ctx, poll.ID, cookieA, "Alice", "", map[int64]string{slots[0].ID: "yes"}); err != nil {
		t.Fatalf("SaveResponse() (Alice) error: %v", err)
	}
	if _, err := st.SaveResponse(ctx, poll.ID, cookieB, "Bob", "", map[int64]string{slots[0].ID: "yes", slots[1].ID: "maybe"}); err != nil {
		t.Fatalf("SaveResponse() (Bob) error: %v", err)
	}

	participants, answers, err := st.ResponsesByPollID(ctx, poll.ID)
	if err != nil {
		t.Fatalf("ResponsesByPollID() error: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}
	if participants[0].DisplayName != "Alice" || participants[1].DisplayName != "Bob" {
		t.Fatalf("expected participants ordered [Alice, Bob] by (created_at, id), got [%s, %s]", participants[0].DisplayName, participants[1].DisplayName)
	}

	aliceAnswers := answers[participants[0].ID]
	if aliceAnswers[slots[0].ID] != "yes" {
		t.Fatalf("expected Alice's slot 0 answer to be yes, got %q", aliceAnswers[slots[0].ID])
	}
	if _, ok := aliceAnswers[slots[1].ID]; ok {
		t.Fatalf("expected Alice to have no stored answer for slot 1, got %q", aliceAnswers[slots[1].ID])
	}

	bobAnswers := answers[participants[1].ID]
	if bobAnswers[slots[0].ID] != "yes" || bobAnswers[slots[1].ID] != "maybe" {
		t.Fatalf("expected Bob's answers {slot0:yes, slot1:maybe}, got %v", bobAnswers)
	}
}

func TestResults_ResponsesByPollID_NoParticipants(t *testing.T) {
	st, poll, _ := newTestStore(t)
	ctx := context.Background()

	participants, answers, err := st.ResponsesByPollID(ctx, poll.ID)
	if err != nil {
		t.Fatalf("ResponsesByPollID() error: %v", err)
	}
	if len(participants) != 0 {
		t.Fatalf("expected zero participants, got %d", len(participants))
	}
	if len(answers) != 0 {
		t.Fatalf("expected zero answers, got %v", answers)
	}
}
