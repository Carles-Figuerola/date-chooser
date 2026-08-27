package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cfiguerola/date-chooser/internal/token"
)

// newTestStore opens a fresh Store against a t.TempDir() database and seeds
// a poll with two slots, returning the store and the poll's slot IDs.
func newTestStore(t *testing.T) (*Store, *Poll, []Slot) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	endA := "2026-09-01T10:00"
	poll, err := st.CreatePoll(context.Background(), NewPollInput{
		Title:    "Invariant Test Poll",
		PollType: "date_time",
		Slots: []NewSlotInput{
			{StartsAt: "2026-09-01T09:00", EndsAt: &endA},
			{StartsAt: "2026-09-02T14:00"},
		},
	}, "ptok-invariant", "atok-invariant")
	if err != nil {
		t.Fatalf("CreatePoll() error: %v", err)
	}

	_, slots, err := st.PollByParticipantToken(context.Background(), poll.ParticipantToken)
	if err != nil {
		t.Fatalf("PollByParticipantToken() error: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("expected 2 seeded slots, got %d", len(slots))
	}

	return st, poll, slots
}

func TestParticipant_NewCookieCreatesNewParticipant(t *testing.T) {
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

	answers := map[int64]string{slots[0].ID: "yes", slots[1].ID: "no"}

	if _, err := st.SaveResponse(ctx, poll.ID, cookieA, "Same Name", "", answers); err != nil {
		t.Fatalf("SaveResponse() (cookie A) error: %v", err)
	}
	if _, err := st.SaveResponse(ctx, poll.ID, cookieB, "Same Name", "", answers); err != nil {
		t.Fatalf("SaveResponse() (cookie B) error: %v", err)
	}

	var count int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&count); err != nil {
		t.Fatalf("querying participants: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 participants rows for two different cookie tokens with the same display_name, got %d", count)
	}
}

func TestParticipant_SameCookieUpdatesInPlace(t *testing.T) {
	st, poll, slots := newTestStore(t)
	ctx := context.Background()

	cookie, err := token.New()
	if err != nil {
		t.Fatalf("token.New() error: %v", err)
	}

	first := map[int64]string{slots[0].ID: "yes", slots[1].ID: "no"}
	if _, err := st.SaveResponse(ctx, poll.ID, cookie, "Bob", "first comment", first); err != nil {
		t.Fatalf("SaveResponse() (first) error: %v", err)
	}

	second := map[int64]string{slots[0].ID: "maybe", slots[1].ID: "yes"}
	if _, err := st.SaveResponse(ctx, poll.ID, cookie, "Bob", "second comment", second); err != nil {
		t.Fatalf("SaveResponse() (second) error: %v", err)
	}

	var participantCount int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID).Scan(&participantCount); err != nil {
		t.Fatalf("querying participants: %v", err)
	}
	if participantCount != 1 {
		t.Fatalf("expected exactly 1 participants row for repeated same-cookie submissions, got %d", participantCount)
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
		t.Fatalf("expected %d responses rows (no doubling/stale rows), got %d", len(slots), responseCount)
	}

	participant, answers, err := st.ParticipantByCookie(ctx, poll.ID, cookie)
	if err != nil {
		t.Fatalf("ParticipantByCookie() error: %v", err)
	}
	if participant == nil {
		t.Fatal("expected a participant, got nil")
	}
	if participant.Comment != "second comment" {
		t.Fatalf("expected comment to be replaced with 'second comment', got %q", participant.Comment)
	}
	if answers[slots[0].ID] != "maybe" || answers[slots[1].ID] != "yes" {
		t.Fatalf("expected second submission's answers to replace the first, got %v", answers)
	}
}

func TestParticipant_ByCookiePrefillMapping(t *testing.T) {
	st, poll, slots := newTestStore(t)
	ctx := context.Background()

	cookie, err := token.New()
	if err != nil {
		t.Fatalf("token.New() error: %v", err)
	}

	saved := map[int64]string{slots[0].ID: "yes", slots[1].ID: "maybe"}
	if _, err := st.SaveResponse(ctx, poll.ID, cookie, "Carol", "hello", saved); err != nil {
		t.Fatalf("SaveResponse() error: %v", err)
	}

	participant, answers, err := st.ParticipantByCookie(ctx, poll.ID, cookie)
	if err != nil {
		t.Fatalf("ParticipantByCookie() error: %v", err)
	}
	if participant == nil {
		t.Fatal("expected a participant for a known cookie, got nil")
	}
	if participant.DisplayName != "Carol" {
		t.Fatalf("expected display name 'Carol', got %q", participant.DisplayName)
	}
	if len(answers) != len(saved) {
		t.Fatalf("expected %d answers, got %d: %v", len(saved), len(answers), answers)
	}
	for slotID, want := range saved {
		if got := answers[slotID]; got != want {
			t.Fatalf("expected answer %q for slot %d, got %q", want, slotID, got)
		}
	}

	// Unknown cookie token: no match, no error.
	unknownParticipant, unknownAnswers, err := st.ParticipantByCookie(ctx, poll.ID, "does-not-exist")
	if err != nil {
		t.Fatalf("ParticipantByCookie() (unknown) error: %v", err)
	}
	if unknownParticipant != nil || unknownAnswers != nil {
		t.Fatalf("expected (nil, nil) for an unknown cookie, got (%v, %v)", unknownParticipant, unknownAnswers)
	}

	// Empty cookie token: no match, no error, no query surprise.
	emptyParticipant, emptyAnswers, err := st.ParticipantByCookie(ctx, poll.ID, "")
	if err != nil {
		t.Fatalf("ParticipantByCookie() (empty) error: %v", err)
	}
	if emptyParticipant != nil || emptyAnswers != nil {
		t.Fatalf("expected (nil, nil) for an empty cookie token, got (%v, %v)", emptyParticipant, emptyAnswers)
	}
}

func TestResponse_AnswerEnumRejected(t *testing.T) {
	st, poll, slots := newTestStore(t)
	ctx := context.Background()

	cookie, err := token.New()
	if err != nil {
		t.Fatalf("token.New() error: %v", err)
	}

	bad := map[int64]string{slots[0].ID: "definitely"}
	if _, err := st.SaveResponse(ctx, poll.ID, cookie, "Dave", "", bad); err == nil {
		t.Fatal("expected SaveResponse() to fail for an answer outside {yes,no,maybe}, got nil error")
	}
}

func TestParticipant_PollByParticipantTokenNotFound(t *testing.T) {
	st, _, _ := newTestStore(t)
	ctx := context.Background()

	_, _, err := st.PollByParticipantToken(ctx, "no-such-token")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for an unknown participant token, got %v", err)
	}
}
