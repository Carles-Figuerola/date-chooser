package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when a poll cannot be located by its tokens.
var ErrNotFound = errors.New("store: poll not found")

// Poll is a persisted poll record.
type Poll struct {
	ID               int64
	ParticipantToken string
	AdminToken       string
	Title            string
	Description      string
	OrganizerName    string
	PollType         string
	CreatedAt        string
}

// Slot is a persisted candidate date/time option belonging to a poll.
type Slot struct {
	ID       int64
	PollID   int64
	Position int
	StartsAt string
	EndsAt   *string
}

// NewSlotInput describes one candidate slot supplied at poll-creation time.
type NewSlotInput struct {
	StartsAt string
	EndsAt   *string
}

// NewPollInput describes a poll to be created, along with its candidate
// slots.
type NewPollInput struct {
	Title         string
	Description   string
	OrganizerName string
	PollType      string
	Slots         []NewSlotInput
}

// CreatePoll inserts a new poll and its slots inside a single transaction,
// using the given, independently-generated participant and admin tokens.
func (s *Store) CreatePoll(ctx context.Context, in NewPollInput, participantToken, adminToken string) (*Poll, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	createdAt := time.Now().UTC().Format(time.RFC3339)

	res, err := tx.ExecContext(ctx, `
		INSERT INTO polls (participant_token, admin_token, title, description, organizer_name, poll_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, participantToken, adminToken, in.Title, in.Description, in.OrganizerName, in.PollType, createdAt)
	if err != nil {
		return nil, fmt.Errorf("store: inserting poll: %w", err)
	}

	pollID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("store: reading poll id: %w", err)
	}

	for i, slot := range in.Slots {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO slots (poll_id, position, starts_at, ends_at)
			VALUES (?, ?, ?, ?)
		`, pollID, i, slot.StartsAt, slot.EndsAt); err != nil {
			return nil, fmt.Errorf("store: inserting slot %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: committing transaction: %w", err)
	}

	return &Poll{
		ID:               pollID,
		ParticipantToken: participantToken,
		AdminToken:       adminToken,
		Title:            in.Title,
		Description:      in.Description,
		OrganizerName:    in.OrganizerName,
		PollType:         in.PollType,
		CreatedAt:        createdAt,
	}, nil
}

// PollByTokens looks up a poll (and its slots, ordered by position) by its
// participant and admin tokens. Both tokens must match the same poll row;
// otherwise ErrNotFound is returned.
func (s *Store) PollByTokens(ctx context.Context, participantToken, adminToken string) (*Poll, []Slot, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, participant_token, admin_token, title, description, organizer_name, poll_type, created_at
		FROM polls
		WHERE participant_token = ? AND admin_token = ?
	`, participantToken, adminToken)

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

// ListPolls returns every poll, newest first, for the instance-admin page
// (ADMIN-03). It never scopes by token — callers must gate access to this
// data themselves (the instance-admin secret/session, not a poll token).
func (s *Store) ListPolls(ctx context.Context) ([]Poll, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, participant_token, admin_token, title, description, organizer_name, poll_type, created_at
		FROM polls
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: querying polls: %w", err)
	}
	defer rows.Close()

	var polls []Poll
	for rows.Next() {
		var p Poll
		if err := rows.Scan(&p.ID, &p.ParticipantToken, &p.AdminToken, &p.Title, &p.Description, &p.OrganizerName, &p.PollType, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scanning poll: %w", err)
		}
		polls = append(polls, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterating polls: %w", err)
	}
	return polls, nil
}
