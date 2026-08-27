package store

import (
	"context"
	"database/sql"
	"fmt"
)

// UpdatePollDetails updates a poll's title, description, and organizer name
// in place. It never touches participant_token, admin_token, or poll_type —
// tokens and poll type are immutable once a poll is created (04-CONTEXT.md).
func (s *Store) UpdatePollDetails(ctx context.Context, pollID int64, title, description, organizerName string) error {
	if _, err := s.db.ExecContext(ctx, `
		UPDATE polls SET title = ?, description = ?, organizer_name = ? WHERE id = ?
	`, title, description, organizerName, pollID); err != nil {
		return fmt.Errorf("store: updating poll details: %w", err)
	}
	return nil
}

// SlotEdit describes an in-place update to an existing slot's value. Its
// position is never touched — only starts_at/ends_at change.
type SlotEdit struct {
	ID       int64
	StartsAt string
	EndsAt   *string
}

// UpdatePollSlots applies a diff to a poll's slot list inside a single
// transaction: keep rows are updated in place (position untouched), removeIDs
// are deleted (cascading, via schema.sql's ON DELETE CASCADE, to exactly
// that slot's response rows), and add rows are appended after the current
// max position. A delete-all-then-reinsert approach is deliberately NOT
// used here, since that would cascade-delete every response on every edit —
// prohibited by 04-CONTEXT.md's cascade-scoping requirement.
//
// Every write is scoped `WHERE ... AND poll_id = ?` using the caller-
// supplied pollID (always the poll resolved via PollByTokens upstream,
// never a client-supplied value), so a slot id belonging to a different
// poll can never be updated or deleted through this method (T-04-03).
func (s *Store) UpdatePollSlots(ctx context.Context, pollID int64, keep []SlotEdit, add []NewSlotInput, removeIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	for _, k := range keep {
		if _, err := tx.ExecContext(ctx, `
			UPDATE slots SET starts_at = ?, ends_at = ? WHERE id = ? AND poll_id = ?
		`, k.StartsAt, k.EndsAt, k.ID, pollID); err != nil {
			return fmt.Errorf("store: updating slot %d: %w", k.ID, err)
		}
	}

	for _, removeID := range removeIDs {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM slots WHERE id = ? AND poll_id = ?
		`, removeID, pollID); err != nil {
			return fmt.Errorf("store: deleting slot %d: %w", removeID, err)
		}
	}

	if len(add) > 0 {
		var maxPosition sql.NullInt64
		if err := tx.QueryRowContext(ctx, `
			SELECT MAX(position) FROM slots WHERE poll_id = ?
		`, pollID).Scan(&maxPosition); err != nil {
			return fmt.Errorf("store: reading max slot position: %w", err)
		}
		nextPosition := 0
		if maxPosition.Valid {
			nextPosition = int(maxPosition.Int64) + 1
		}
		for i, a := range add {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO slots (poll_id, position, starts_at, ends_at)
				VALUES (?, ?, ?, ?)
			`, pollID, nextPosition+i, a.StartsAt, a.EndsAt); err != nil {
				return fmt.Errorf("store: inserting new slot %d: %w", i, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: committing transaction: %w", err)
	}
	return nil
}

// DeleteParticipant removes a single participant, scoped to the owning
// poll, and cascades (via schema.sql's ON DELETE CASCADE on
// responses.participant_id) to remove that participant's response rows —
// response rows are never deleted manually here. The `AND poll_id = ?`
// scope is mandatory (T-04-03): a participantID belonging to a different
// poll deletes nothing. Deleting zero rows (an already-deleted or
// never-existent id) is not an error — a repeated/stale delete request is a
// safe no-op, backing the double-submit guard (T-04-04).
func (s *Store) DeleteParticipant(ctx context.Context, pollID, participantID int64) error {
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM participants WHERE id = ? AND poll_id = ?
	`, participantID, pollID); err != nil {
		return fmt.Errorf("store: deleting participant %d: %w", participantID, err)
	}
	return nil
}

// DeletePoll permanently removes a poll and, via schema.sql's
// ON DELETE CASCADE (slots.poll_id and participants.poll_id, with responses
// cascading transitively through participants and slots), every one of its
// slots, participants, and responses. Child rows are never deleted
// manually here — the schema-level cascade is the single source of truth
// for what "deleting a poll" means. Deleting a non-existent pollID is not
// an error: it is a safe no-op, backing a stale or double delete request
// (the double-submit backstop for this, the most destructive action in the
// app, per T-04-04). Callers must always pass a poll.ID resolved via
// PollByTokens from BOTH the participant and admin tokens — never a
// client-supplied id directly (T-04-02).
func (s *Store) DeletePoll(ctx context.Context, pollID int64) error {
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM polls WHERE id = ?
	`, pollID); err != nil {
		return fmt.Errorf("store: deleting poll %d: %w", pollID, err)
	}
	return nil
}

// SlotResponseCounts returns, for every slot with at least one response, the
// count of response rows tied to it — keyed by slot ID, scoped to pollID.
// This feeds both the edit form's client-side per-slot removal warning and
// the server-side removal-confirmation gate, so both read the same counts.
func (s *Store) SlotResponseCounts(ctx context.Context, pollID int64) (map[int64]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.slot_id, COUNT(*)
		FROM responses r
		JOIN participants p ON p.id = r.participant_id
		WHERE p.poll_id = ?
		GROUP BY r.slot_id
	`, pollID)
	if err != nil {
		return nil, fmt.Errorf("store: querying slot response counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[int64]int)
	for rows.Next() {
		var slotID int64
		var count int
		if err := rows.Scan(&slotID, &count); err != nil {
			return nil, fmt.Errorf("store: scanning slot response count: %w", err)
		}
		counts[slotID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterating slot response counts: %w", err)
	}
	return counts, nil
}
