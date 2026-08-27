package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Participant is a persisted respondent record for one poll, identified by
// an HttpOnly cookie token (not a name or account) so a participant can
// revisit the same link later and have their prior response pre-filled.
type Participant struct {
	ID          int64
	PollID      int64
	DisplayName string
	Comment     string
	CookieToken string
	CreatedAt   string
	UpdatedAt   string
}

// PollByParticipantToken looks up a poll (and its slots, ordered by
// position) by its participant token alone — unlike PollByTokens, no admin
// token is required, since this is the public voting-page lookup. Returns
// ErrNotFound when no poll matches.
func (s *Store) PollByParticipantToken(ctx context.Context, participantToken string) (*Poll, []Slot, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, participant_token, admin_token, title, description, organizer_name, poll_type, created_at
		FROM polls
		WHERE participant_token = ?
	`, participantToken)

	var p Poll
	if err := row.Scan(&p.ID, &p.ParticipantToken, &p.AdminToken, &p.Title, &p.Description, &p.OrganizerName, &p.PollType, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("store: querying poll: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, poll_id, position, starts_at, ends_at
		FROM slots
		WHERE poll_id = ?
		ORDER BY position
	`, p.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("store: querying slots: %w", err)
	}
	defer rows.Close()

	var slots []Slot
	for rows.Next() {
		var sl Slot
		if err := rows.Scan(&sl.ID, &sl.PollID, &sl.Position, &sl.StartsAt, &sl.EndsAt); err != nil {
			return nil, nil, fmt.Errorf("store: scanning slot: %w", err)
		}
		slots = append(slots, sl)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("store: iterating slots: %w", err)
	}

	return &p, slots, nil
}

// ParticipantByCookie returns the participant row for (pollID, cookieToken)
// plus a map of slot_id -> answer for that participant's stored responses.
// When cookieToken is empty, or no participant matches, it returns
// (nil, nil, nil) — an empty cookie token must never match a stored row
// (participants.cookie_token is always non-empty by construction), so this
// is checked explicitly rather than relying on the query to find no rows.
func (s *Store) ParticipantByCookie(ctx context.Context, pollID int64, cookieToken string) (*Participant, map[int64]string, error) {
	if cookieToken == "" {
		return nil, nil, nil
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, poll_id, display_name, comment, cookie_token, created_at, updated_at
		FROM participants
		WHERE poll_id = ? AND cookie_token = ?
	`, pollID, cookieToken)

	var p Participant
	if err := row.Scan(&p.ID, &p.PollID, &p.DisplayName, &p.Comment, &p.CookieToken, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("store: querying participant: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT slot_id, answer
		FROM responses
		WHERE participant_id = ?
	`, p.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("store: querying responses: %w", err)
	}
	defer rows.Close()

	answers := make(map[int64]string)
	for rows.Next() {
		var slotID int64
		var answer string
		if err := rows.Scan(&slotID, &answer); err != nil {
			return nil, nil, fmt.Errorf("store: scanning response: %w", err)
		}
		answers[slotID] = answer
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("store: iterating responses: %w", err)
	}

	return &p, answers, nil
}

// SaveResponse upserts a participant's response for a poll, keyed strictly
// on (pollID, cookieToken) — never on displayName. A request with no
// matching cookie token ALWAYS inserts a new participant row, even if the
// display name collides with an existing participant (the anti-overwrite
// rule; see threat T-02-03). After this call, the participant's stored
// answers exactly equal the submitted map (any previously stored answer for
// a slot no longer present in answers is removed).
func (s *Store) SaveResponse(ctx context.Context, pollID int64, cookieToken, displayName, comment string, answers map[int64]string) (*Participant, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	now := time.Now().UTC().Format(time.RFC3339)

	var participantID int64
	var createdAt string

	row := tx.QueryRowContext(ctx, `
		SELECT id, created_at FROM participants WHERE poll_id = ? AND cookie_token = ?
	`, pollID, cookieToken)
	err = row.Scan(&participantID, &createdAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		res, err := tx.ExecContext(ctx, `
			INSERT INTO participants (poll_id, display_name, comment, cookie_token, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, pollID, displayName, comment, cookieToken, now, now)
		if err != nil {
			return nil, fmt.Errorf("store: inserting participant: %w", err)
		}
		participantID, err = res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("store: reading participant id: %w", err)
		}
		createdAt = now
	case err != nil:
		return nil, fmt.Errorf("store: querying participant: %w", err)
	default:
		if _, err := tx.ExecContext(ctx, `
			UPDATE participants SET display_name = ?, comment = ?, updated_at = ?
			WHERE id = ?
		`, displayName, comment, now, participantID); err != nil {
			return nil, fmt.Errorf("store: updating participant: %w", err)
		}
	}

	// Make the stored answers exactly equal the submitted map: delete
	// anything no longer present, then upsert every submitted answer.
	if _, err := tx.ExecContext(ctx, `DELETE FROM responses WHERE participant_id = ?`, participantID); err != nil {
		return nil, fmt.Errorf("store: clearing prior responses: %w", err)
	}
	for slotID, answer := range answers {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO responses (participant_id, slot_id, answer) VALUES (?, ?, ?)
		`, participantID, slotID, answer); err != nil {
			return nil, fmt.Errorf("store: inserting response for slot %d: %w", slotID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: committing transaction: %w", err)
	}

	return &Participant{
		ID:          participantID,
		PollID:      pollID,
		DisplayName: displayName,
		Comment:     comment,
		CookieToken: cookieToken,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}, nil
}
