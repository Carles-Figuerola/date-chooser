package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const instanceAdminSecretKey = "instance_admin_secret"

// EnsureInstanceAdminSecret returns the persisted instance-admin secret,
// creating it with candidateSecret if none exists yet (ADMIN-01). Follows
// the same caller-generates-the-token convention as CreatePoll's
// participant/admin tokens — this package never calls crypto/rand itself.
// The returned value is the one now persisted: on a fresh database that's
// candidateSecret; on a database that already has one, candidateSecret is
// discarded and the existing value is returned instead.
func (s *Store) EnsureInstanceAdminSecret(ctx context.Context, candidateSecret string) (string, error) {
	var secret string
	err := s.db.QueryRowContext(ctx, `
		SELECT value FROM settings WHERE key = ?
	`, instanceAdminSecretKey).Scan(&secret)
	if err == nil {
		return secret, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("store: reading instance admin secret: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value) VALUES (?, ?)
	`, instanceAdminSecretKey, candidateSecret); err != nil {
		return "", fmt.Errorf("store: persisting instance admin secret: %w", err)
	}
	return candidateSecret, nil
}
