package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CreateInstanceAdminSession persists a new session row for token, created
// now. Follows the same caller-generates-the-token convention as every
// other token in this package.
func (s *Store) CreateInstanceAdminSession(ctx context.Context, token string) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_sessions (token, created_at) VALUES (?, ?)
	`, token, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("store: creating instance admin session: %w", err)
	}
	return nil
}

// InstanceAdminSessionValid reports whether token names an admin_sessions
// row created within maxAge of now. A missing row (never existed, or
// already pruned) is simply invalid — not an error.
func (s *Store) InstanceAdminSessionValid(ctx context.Context, token string, maxAge time.Duration) (bool, error) {
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT created_at FROM admin_sessions WHERE token = ?
	`, token).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: reading instance admin session: %w", err)
	}

	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return false, fmt.Errorf("store: parsing instance admin session timestamp: %w", err)
	}
	return time.Since(created) <= maxAge, nil
}

// PruneInstanceAdminSessions deletes every admin_sessions row older than
// maxAge. Best-effort garbage collection, called on every /admin and
// /admin/login request — NOT the security boundary itself, which is
// InstanceAdminSessionValid's own age check at auth time.
func (s *Store) PruneInstanceAdminSessions(ctx context.Context, maxAge time.Duration) error {
	cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM admin_sessions WHERE created_at < ?
	`, cutoff); err != nil {
		return fmt.Errorf("store: pruning instance admin sessions: %w", err)
	}
	return nil
}
