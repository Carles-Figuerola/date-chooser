package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newAdminSessionTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestInstanceAdminSessionValid_FreshSessionIsValid(t *testing.T) {
	st := newAdminSessionTestStore(t)
	ctx := context.Background()

	if err := st.CreateInstanceAdminSession(ctx, "fresh-token"); err != nil {
		t.Fatalf("CreateInstanceAdminSession() error: %v", err)
	}

	valid, err := st.InstanceAdminSessionValid(ctx, "fresh-token", 24*time.Hour)
	if err != nil {
		t.Fatalf("InstanceAdminSessionValid() error: %v", err)
	}
	if !valid {
		t.Fatalf("expected a freshly created session to be valid")
	}
}

func TestInstanceAdminSessionValid_UnknownTokenIsInvalid(t *testing.T) {
	st := newAdminSessionTestStore(t)

	valid, err := st.InstanceAdminSessionValid(context.Background(), "never-created", 24*time.Hour)
	if err != nil {
		t.Fatalf("InstanceAdminSessionValid() error: %v", err)
	}
	if valid {
		t.Fatalf("expected an unknown token to be invalid")
	}
}

func TestInstanceAdminSessionValid_ExpiredSessionIsInvalid(t *testing.T) {
	st := newAdminSessionTestStore(t)
	ctx := context.Background()

	old := time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339)
	if _, err := st.DB().ExecContext(ctx, "INSERT INTO admin_sessions (token, created_at) VALUES (?, ?)", "old-token", old); err != nil {
		t.Fatalf("seeding old session: %v", err)
	}

	valid, err := st.InstanceAdminSessionValid(ctx, "old-token", 24*time.Hour)
	if err != nil {
		t.Fatalf("InstanceAdminSessionValid() error: %v", err)
	}
	if valid {
		t.Fatalf("expected a 25h-old session to be invalid against a 24h max age")
	}
}

func TestPruneInstanceAdminSessions_DeletesOnlyExpiredRows(t *testing.T) {
	st := newAdminSessionTestStore(t)
	ctx := context.Background()

	old := time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339)
	fresh := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB().ExecContext(ctx, "INSERT INTO admin_sessions (token, created_at) VALUES (?, ?)", "old-token", old); err != nil {
		t.Fatalf("seeding old session: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx, "INSERT INTO admin_sessions (token, created_at) VALUES (?, ?)", "fresh-token", fresh); err != nil {
		t.Fatalf("seeding fresh session: %v", err)
	}

	if err := st.PruneInstanceAdminSessions(ctx, 24*time.Hour); err != nil {
		t.Fatalf("PruneInstanceAdminSessions() error: %v", err)
	}

	var count int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_sessions WHERE token = 'old-token'").Scan(&count); err != nil {
		t.Fatalf("querying old token: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the old session to be pruned, still found %d row(s)", count)
	}

	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_sessions WHERE token = 'fresh-token'").Scan(&count); err != nil {
		t.Fatalf("querying fresh token: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the fresh session to survive pruning, got %d row(s)", count)
	}
}
