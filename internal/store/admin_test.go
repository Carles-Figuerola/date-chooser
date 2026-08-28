package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cfiguerola/date-chooser/internal/token"
)

// newAdminTestStore opens a fresh Store against a t.TempDir() database and
// seeds a poll with n all_day slots, returning the store, poll, and its
// persisted slots (ordered by position).
func newAdminTestStore(t *testing.T, title, ptoken, atoken string, n int) (*Store, *Poll, []Slot) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	slots := make([]NewSlotInput, n)
	for i := range slots {
		slots[i] = NewSlotInput{StartsAt: "2026-09-0" + string(rune('1'+i))}
	}
	poll, err := st.CreatePoll(context.Background(), NewPollInput{
		Title:    title,
		PollType: "all_day",
		Slots:    slots,
	}, ptoken, atoken)
	if err != nil {
		t.Fatalf("CreatePoll() error: %v", err)
	}

	_, persisted, err := st.PollByTokens(context.Background(), poll.ParticipantToken, poll.AdminToken)
	if err != nil {
		t.Fatalf("PollByTokens() error: %v", err)
	}
	return st, poll, persisted
}

// addParticipantAnsweringAll seeds a participant who answers every given
// slot "yes", returning the display name for later comment-integrity
// assertions.
func addParticipantAnsweringAll(t *testing.T, st *Store, pollID int64, name string, slots []Slot, comment string) {
	t.Helper()
	cookie, err := token.New()
	if err != nil {
		t.Fatalf("token.New() error: %v", err)
	}
	answers := make(map[int64]string, len(slots))
	for _, sl := range slots {
		answers[sl.ID] = "yes"
	}
	if _, err := st.SaveResponse(context.Background(), pollID, cookie, name, comment, answers); err != nil {
		t.Fatalf("SaveResponse() error: %v", err)
	}
}

func countRows(t *testing.T, st *Store, query string, args ...interface{}) int {
	t.Helper()
	var n int
	if err := st.DB().QueryRowContext(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q error: %v", query, err)
	}
	return n
}

func TestAdmin_UpdatePollSlots_RemoveCascadesOnlyThatSlotsResponses(t *testing.T) {
	st, poll, slots := newAdminTestStore(t, "Remove Cascade Poll", "ptok-admin-remove", "atok-admin-remove", 2)
	ctx := context.Background()

	addParticipantAnsweringAll(t, st, poll.ID, "Alice", slots, "alice's comment")
	addParticipantAnsweringAll(t, st, poll.ID, "Bob", slots, "bob's comment")

	if got := countRows(t, st, "SELECT COUNT(*) FROM responses"); got != 4 {
		t.Fatalf("expected 4 responses before removal, got %d", got)
	}

	removedSlotID := slots[0].ID
	keptSlotID := slots[1].ID

	if err := st.UpdatePollSlots(ctx, poll.ID, nil, nil, []int64{removedSlotID}); err != nil {
		t.Fatalf("UpdatePollSlots() error: %v", err)
	}

	if got := countRows(t, st, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", poll.ID); got != 1 {
		t.Fatalf("expected 1 remaining slot, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses"); got != 2 {
		t.Fatalf("expected 2 remaining responses, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses WHERE slot_id != ?", keptSlotID); got != 0 {
		t.Fatalf("expected zero responses for any slot other than the kept slot, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID); got != 2 {
		t.Fatalf("expected both participants to remain, got %d", got)
	}

	var aliceComment, bobComment string
	if err := st.DB().QueryRowContext(ctx, "SELECT comment FROM participants WHERE poll_id = ? AND display_name = ?", poll.ID, "Alice").Scan(&aliceComment); err != nil {
		t.Fatalf("querying Alice's comment: %v", err)
	}
	if err := st.DB().QueryRowContext(ctx, "SELECT comment FROM participants WHERE poll_id = ? AND display_name = ?", poll.ID, "Bob").Scan(&bobComment); err != nil {
		t.Fatalf("querying Bob's comment: %v", err)
	}
	if aliceComment != "alice's comment" || bobComment != "bob's comment" {
		t.Fatalf("expected comments unchanged, got alice=%q bob=%q", aliceComment, bobComment)
	}
}

func TestAdmin_UpdatePollSlots_AddAppendsAtEndKeepsResponses(t *testing.T) {
	st, poll, slots := newAdminTestStore(t, "Add Slot Poll", "ptok-admin-add", "atok-admin-add", 2)
	ctx := context.Background()

	addParticipantAnsweringAll(t, st, poll.ID, "Alice", slots, "")

	beforeResponses := countRows(t, st, "SELECT COUNT(*) FROM responses")

	if err := st.UpdatePollSlots(ctx, poll.ID, nil, []NewSlotInput{{StartsAt: "2026-09-10"}}, nil); err != nil {
		t.Fatalf("UpdatePollSlots() error: %v", err)
	}

	afterResponses := countRows(t, st, "SELECT COUNT(*) FROM responses")
	if afterResponses != beforeResponses {
		t.Fatalf("expected response count unchanged after adding a slot, before=%d after=%d", beforeResponses, afterResponses)
	}

	var newSlotID int64
	var newPosition int
	if err := st.DB().QueryRowContext(ctx, "SELECT id, position FROM slots WHERE poll_id = ? AND starts_at = ?", poll.ID, "2026-09-10").Scan(&newSlotID, &newPosition); err != nil {
		t.Fatalf("querying new slot: %v", err)
	}
	if newPosition != 2 {
		t.Fatalf("expected new slot position 2 (prior max %d + 1), got %d", 1, newPosition)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses WHERE slot_id = ?", newSlotID); got != 0 {
		t.Fatalf("expected the new slot to have zero responses, got %d", got)
	}
}

func TestAdmin_UpdatePollSlots_EditInPlaceReordersAndPreservesResponses(t *testing.T) {
	st, poll, slots := newAdminTestStore(t, "Edit In Place Poll", "ptok-admin-edit", "atok-admin-edit", 2)
	ctx := context.Background()

	addParticipantAnsweringAll(t, st, poll.ID, "Alice", slots, "")

	beforeResponses := countRows(t, st, "SELECT COUNT(*) FROM responses WHERE slot_id = ?", slots[0].ID)

	// slots[0] starts at "2026-09-01", slots[1] at "2026-09-02". Editing
	// slots[0] to a later date than slots[1] should push it to position 1
	// — slots are re-sequenced by date on every save, not left in
	// creation order.
	if err := st.UpdatePollSlots(ctx, poll.ID, []SlotEdit{{ID: slots[0].ID, StartsAt: "2026-10-05"}}, nil, nil); err != nil {
		t.Fatalf("UpdatePollSlots() error: %v", err)
	}

	var starts string
	var position int
	if err := st.DB().QueryRowContext(ctx, "SELECT starts_at, position FROM slots WHERE id = ?", slots[0].ID).Scan(&starts, &position); err != nil {
		t.Fatalf("querying edited slot: %v", err)
	}
	if starts != "2026-10-05" {
		t.Fatalf("expected starts_at updated to '2026-10-05', got %q", starts)
	}
	if position != 1 {
		t.Fatalf("expected the edited (now-latest-dated) slot to move to position 1, got %d", position)
	}

	var otherPosition int
	if err := st.DB().QueryRowContext(ctx, "SELECT position FROM slots WHERE id = ?", slots[1].ID).Scan(&otherPosition); err != nil {
		t.Fatalf("querying other slot: %v", err)
	}
	if otherPosition != 0 {
		t.Fatalf("expected the untouched, earlier-dated slot to move to position 0, got %d", otherPosition)
	}

	afterResponses := countRows(t, st, "SELECT COUNT(*) FROM responses WHERE slot_id = ?", slots[0].ID)
	if afterResponses != beforeResponses {
		t.Fatalf("expected response count for edited slot unchanged, before=%d after=%d", beforeResponses, afterResponses)
	}
}

func TestAdmin_UpdatePollSlots_ScopedByPoll(t *testing.T) {
	stA, pollA, slotsA := newAdminTestStore(t, "Poll A", "ptok-admin-scope-a", "atok-admin-scope-a", 1)
	ctx := context.Background()

	// Seed a second poll in the SAME database file so a slot ID from poll B
	// can be attempted against poll A's UpdatePollSlots call.
	slotsB := make([]NewSlotInput, 1)
	slotsB[0] = NewSlotInput{StartsAt: "2026-09-20"}
	pollB, err := stA.CreatePoll(ctx, NewPollInput{Title: "Poll B", PollType: "all_day", Slots: slotsB}, "ptok-admin-scope-b", "atok-admin-scope-b")
	if err != nil {
		t.Fatalf("CreatePoll() (poll B) error: %v", err)
	}
	_, persistedB, err := stA.PollByTokens(ctx, pollB.ParticipantToken, pollB.AdminToken)
	if err != nil {
		t.Fatalf("PollByTokens() (poll B) error: %v", err)
	}

	// Attempt to remove poll B's slot through poll A's UpdatePollSlots call.
	if err := stA.UpdatePollSlots(ctx, pollA.ID, nil, nil, []int64{persistedB[0].ID}); err != nil {
		t.Fatalf("UpdatePollSlots() error: %v", err)
	}

	if got := countRows(t, stA, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", pollA.ID); got != 1 {
		t.Fatalf("expected poll A's own slot untouched, got %d slots", got)
	}
	if got := countRows(t, stA, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", pollB.ID); got != 1 {
		t.Fatalf("expected poll B's slot NOT deleted via poll A's edit, got %d slots", got)
	}

	// Attempt to edit poll B's slot through poll A's UpdatePollSlots call.
	if err := stA.UpdatePollSlots(ctx, pollA.ID, []SlotEdit{{ID: persistedB[0].ID, StartsAt: "1999-01-01"}}, nil, nil); err != nil {
		t.Fatalf("UpdatePollSlots() (edit attempt) error: %v", err)
	}
	var stillOriginal string
	if err := stA.DB().QueryRowContext(ctx, "SELECT starts_at FROM slots WHERE id = ?", persistedB[0].ID).Scan(&stillOriginal); err != nil {
		t.Fatalf("querying poll B's slot: %v", err)
	}
	if stillOriginal != "2026-09-20" {
		t.Fatalf("expected poll B's slot NOT edited via poll A's edit, got starts_at=%q", stillOriginal)
	}

	_ = slotsA
}

// participantIDByName returns the id of the participant with the given
// display name in pollID, failing the test if not found.
func participantIDByName(t *testing.T, st *Store, pollID int64, name string) int64 {
	t.Helper()
	var id int64
	if err := st.DB().QueryRowContext(context.Background(), "SELECT id FROM participants WHERE poll_id = ? AND display_name = ?", pollID, name).Scan(&id); err != nil {
		t.Fatalf("querying participant id for %q: %v", name, err)
	}
	return id
}

func TestAdmin_DeleteParticipant_CascadesOnlyThatParticipant(t *testing.T) {
	st, poll, slots := newAdminTestStore(t, "Delete Participant Poll", "ptok-admin-delpart", "atok-admin-delpart", 2)
	ctx := context.Background()

	addParticipantAnsweringAll(t, st, poll.ID, "Alice", slots, "")
	addParticipantAnsweringAll(t, st, poll.ID, "Bob", slots, "")

	if got := countRows(t, st, "SELECT COUNT(*) FROM responses"); got != 4 {
		t.Fatalf("expected 4 responses before deletion, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID); got != 2 {
		t.Fatalf("expected 2 participants before deletion, got %d", got)
	}

	aliceID := participantIDByName(t, st, poll.ID, "Alice")
	bobID := participantIDByName(t, st, poll.ID, "Bob")

	if err := st.DeleteParticipant(ctx, poll.ID, aliceID); err != nil {
		t.Fatalf("DeleteParticipant() error: %v", err)
	}

	if got := countRows(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID); got != 1 {
		t.Fatalf("expected 1 remaining participant, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses"); got != 2 {
		t.Fatalf("expected 2 remaining responses, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses WHERE participant_id != ?", bobID); got != 0 {
		t.Fatalf("expected zero responses for any participant other than Bob, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", poll.ID); got != 2 {
		t.Fatalf("expected slots count unchanged, got %d", got)
	}
}

func TestAdmin_DeleteParticipant_ScopedByPoll(t *testing.T) {
	stA, pollA, slotsA := newAdminTestStore(t, "Poll A", "ptok-admin-delpart-scope-a", "atok-admin-delpart-scope-a", 1)
	ctx := context.Background()

	addParticipantAnsweringAll(t, stA, pollA.ID, "Alice", slotsA, "")

	slotsB := []NewSlotInput{{StartsAt: "2026-09-20"}}
	pollB, err := stA.CreatePoll(ctx, NewPollInput{Title: "Poll B", PollType: "all_day", Slots: slotsB}, "ptok-admin-delpart-scope-b", "atok-admin-delpart-scope-b")
	if err != nil {
		t.Fatalf("CreatePoll() (poll B) error: %v", err)
	}
	_, persistedB, err := stA.PollByTokens(ctx, pollB.ParticipantToken, pollB.AdminToken)
	if err != nil {
		t.Fatalf("PollByTokens() (poll B) error: %v", err)
	}
	addParticipantAnsweringAll(t, stA, pollB.ID, "Carol", persistedB, "")

	carolID := participantIDByName(t, stA, pollB.ID, "Carol")

	// Attempt to delete poll B's participant through poll A's DeleteParticipant call.
	if err := stA.DeleteParticipant(ctx, pollA.ID, carolID); err != nil {
		t.Fatalf("DeleteParticipant() error: %v", err)
	}

	if got := countRows(t, stA, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", pollA.ID); got != 1 {
		t.Fatalf("expected poll A's own participant untouched, got %d", got)
	}
	if got := countRows(t, stA, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", pollB.ID); got != 1 {
		t.Fatalf("expected poll B's participant NOT deleted via poll A's DeleteParticipant, got %d", got)
	}
	if got := countRows(t, stA, "SELECT COUNT(*) FROM responses"); got != 2 {
		t.Fatalf("expected both polls' responses untouched, got %d", got)
	}
}

func TestAdmin_DeleteParticipant_NonExistent_NoError(t *testing.T) {
	st, poll, slots := newAdminTestStore(t, "Nonexistent Participant Poll", "ptok-admin-delpart-none", "atok-admin-delpart-none", 1)
	ctx := context.Background()

	addParticipantAnsweringAll(t, st, poll.ID, "Alice", slots, "")

	beforeParticipants := countRows(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID)
	beforeResponses := countRows(t, st, "SELECT COUNT(*) FROM responses")

	if err := st.DeleteParticipant(ctx, poll.ID, 999999); err != nil {
		t.Fatalf("DeleteParticipant() error for non-existent id: %v", err)
	}

	if got := countRows(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID); got != beforeParticipants {
		t.Fatalf("expected participant count unchanged, before=%d after=%d", beforeParticipants, got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses"); got != beforeResponses {
		t.Fatalf("expected response count unchanged, before=%d after=%d", beforeResponses, got)
	}
}

func TestAdmin_DeletePoll_CascadesEverything(t *testing.T) {
	st, poll, slots := newAdminTestStore(t, "Delete Poll Cascade", "ptok-admin-delpoll-cascade", "atok-admin-delpoll-cascade", 2)
	ctx := context.Background()

	addParticipantAnsweringAll(t, st, poll.ID, "Alice", slots, "alice's comment")
	addParticipantAnsweringAll(t, st, poll.ID, "Bob", slots, "bob's comment")

	if got := countRows(t, st, "SELECT COUNT(*) FROM polls WHERE id = ?", poll.ID); got != 1 {
		t.Fatalf("expected 1 poll row before deletion, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", poll.ID); got != 2 {
		t.Fatalf("expected 2 slot rows before deletion, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID); got != 2 {
		t.Fatalf("expected 2 participant rows before deletion, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses r JOIN participants p ON p.id = r.participant_id WHERE p.poll_id = ?", poll.ID); got != 4 {
		t.Fatalf("expected 4 response rows before deletion, got %d", got)
	}

	if err := st.DeletePoll(ctx, poll.ID); err != nil {
		t.Fatalf("DeletePoll() error: %v", err)
	}

	if got := countRows(t, st, "SELECT COUNT(*) FROM polls WHERE id = ?", poll.ID); got != 0 {
		t.Fatalf("expected 0 poll rows after deletion, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", poll.ID); got != 0 {
		t.Fatalf("expected 0 slot rows after deletion, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", poll.ID); got != 0 {
		t.Fatalf("expected 0 participant rows after deletion, got %d", got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses r JOIN participants p ON p.id = r.participant_id WHERE p.poll_id = ?", poll.ID); got != 0 {
		t.Fatalf("expected 0 response rows after deletion, got %d", got)
	}
}

func TestAdmin_DeletePoll_OtherPollUntouched(t *testing.T) {
	stA, pollA, slotsA := newAdminTestStore(t, "Delete Poll A", "ptok-admin-delpoll-a", "atok-admin-delpoll-a", 2)
	ctx := context.Background()

	addParticipantAnsweringAll(t, stA, pollA.ID, "Alice", slotsA, "")

	slotsBInput := []NewSlotInput{{StartsAt: "2026-09-20"}, {StartsAt: "2026-09-21"}}
	pollB, err := stA.CreatePoll(ctx, NewPollInput{Title: "Delete Poll B", PollType: "all_day", Slots: slotsBInput}, "ptok-admin-delpoll-b", "atok-admin-delpoll-b")
	if err != nil {
		t.Fatalf("CreatePoll() (poll B) error: %v", err)
	}
	_, persistedB, err := stA.PollByTokens(ctx, pollB.ParticipantToken, pollB.AdminToken)
	if err != nil {
		t.Fatalf("PollByTokens() (poll B) error: %v", err)
	}
	addParticipantAnsweringAll(t, stA, pollB.ID, "Carol", persistedB, "carol's comment")

	beforeSlotsB := countRows(t, stA, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", pollB.ID)
	beforeParticipantsB := countRows(t, stA, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", pollB.ID)
	beforeResponsesB := countRows(t, stA, "SELECT COUNT(*) FROM responses r JOIN participants p ON p.id = r.participant_id WHERE p.poll_id = ?", pollB.ID)

	if err := stA.DeletePoll(ctx, pollA.ID); err != nil {
		t.Fatalf("DeletePoll() error: %v", err)
	}

	if got := countRows(t, stA, "SELECT COUNT(*) FROM polls WHERE id = ?", pollB.ID); got != 1 {
		t.Fatalf("expected poll B's poll row untouched, got %d", got)
	}
	if got := countRows(t, stA, "SELECT COUNT(*) FROM slots WHERE poll_id = ?", pollB.ID); got != beforeSlotsB {
		t.Fatalf("expected poll B's slot count unchanged, before=%d after=%d", beforeSlotsB, got)
	}
	if got := countRows(t, stA, "SELECT COUNT(*) FROM participants WHERE poll_id = ?", pollB.ID); got != beforeParticipantsB {
		t.Fatalf("expected poll B's participant count unchanged, before=%d after=%d", beforeParticipantsB, got)
	}
	if got := countRows(t, stA, "SELECT COUNT(*) FROM responses r JOIN participants p ON p.id = r.participant_id WHERE p.poll_id = ?", pollB.ID); got != beforeResponsesB {
		t.Fatalf("expected poll B's response count unchanged, before=%d after=%d", beforeResponsesB, got)
	}
	if got := countRows(t, stA, "SELECT COUNT(*) FROM polls WHERE id = ?", pollA.ID); got != 0 {
		t.Fatalf("expected poll A deleted, got %d rows", got)
	}
}

func TestAdmin_DeletePoll_NonExistent_NoError(t *testing.T) {
	st, poll, slots := newAdminTestStore(t, "Delete Poll Nonexistent", "ptok-admin-delpoll-none", "atok-admin-delpoll-none", 1)
	ctx := context.Background()

	addParticipantAnsweringAll(t, st, poll.ID, "Alice", slots, "")

	beforePolls := countRows(t, st, "SELECT COUNT(*) FROM polls")
	beforeParticipants := countRows(t, st, "SELECT COUNT(*) FROM participants")
	beforeResponses := countRows(t, st, "SELECT COUNT(*) FROM responses")

	if err := st.DeletePoll(ctx, 999999); err != nil {
		t.Fatalf("DeletePoll() error for non-existent id: %v", err)
	}

	if got := countRows(t, st, "SELECT COUNT(*) FROM polls"); got != beforePolls {
		t.Fatalf("expected poll count unchanged, before=%d after=%d", beforePolls, got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM participants"); got != beforeParticipants {
		t.Fatalf("expected participant count unchanged, before=%d after=%d", beforeParticipants, got)
	}
	if got := countRows(t, st, "SELECT COUNT(*) FROM responses"); got != beforeResponses {
		t.Fatalf("expected response count unchanged, before=%d after=%d", beforeResponses, got)
	}
}

func TestAdmin_SlotResponseCounts(t *testing.T) {
	st, poll, slots := newAdminTestStore(t, "Response Counts Poll", "ptok-admin-counts", "atok-admin-counts", 2)
	ctx := context.Background()

	addParticipantAnsweringAll(t, st, poll.ID, "Alice", slots[:1], "")
	addParticipantAnsweringAll(t, st, poll.ID, "Bob", slots[:1], "")

	counts, err := st.SlotResponseCounts(ctx, poll.ID)
	if err != nil {
		t.Fatalf("SlotResponseCounts() error: %v", err)
	}
	if counts[slots[0].ID] != 2 {
		t.Fatalf("expected 2 responses for slot 0, got %d", counts[slots[0].ID])
	}
	if counts[slots[1].ID] != 0 {
		t.Fatalf("expected 0 responses for slot 1, got %d", counts[slots[1].ID])
	}
}
