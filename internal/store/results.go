package store

import (
	"context"
	"fmt"
)

// ResponsesByPollID returns every participant for a poll, ordered by
// (created_at, id) for a stable, deterministic column order shared by the
// results grid's headers and its comments list, plus a nested map of
// participant ID -> (slot ID -> answer) built from every stored response for
// that poll. Tallies and best-slot ranking are computed in Go, by
// buildResultsGridView/rankBestSlots in internal/web — per 03-CONTEXT.md's
// "Claude's Discretion", this method returns raw rows only.
func (s *Store) ResponsesByPollID(ctx context.Context, pollID int64) ([]Participant, map[int64]map[int64]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, poll_id, display_name, comment, cookie_token, created_at, updated_at
		FROM participants
		WHERE poll_id = ?
		ORDER BY created_at, id
	`, pollID)
	if err != nil {
		return nil, nil, fmt.Errorf("store: querying participants: %w", err)
	}
	defer rows.Close()

	var participants []Participant
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.PollID, &p.DisplayName, &p.Comment, &p.CookieToken, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("store: scanning participant: %w", err)
		}
		participants = append(participants, p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("store: iterating participants: %w", err)
	}

	respRows, err := s.db.QueryContext(ctx, `
		SELECT r.participant_id, r.slot_id, r.answer
		FROM responses r
		JOIN participants p ON p.id = r.participant_id
		WHERE p.poll_id = ?
	`, pollID)
	if err != nil {
		return nil, nil, fmt.Errorf("store: querying responses: %w", err)
	}
	defer respRows.Close()

	answers := make(map[int64]map[int64]string)
	for respRows.Next() {
		var participantID, slotID int64
		var answer string
		if err := respRows.Scan(&participantID, &slotID, &answer); err != nil {
			return nil, nil, fmt.Errorf("store: scanning response: %w", err)
		}
		if answers[participantID] == nil {
			answers[participantID] = make(map[int64]string)
		}
		answers[participantID][slotID] = answer
	}
	if err := respRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("store: iterating responses: %w", err)
	}

	return participants, answers, nil
}
