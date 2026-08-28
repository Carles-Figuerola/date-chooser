package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestEnsureInstanceAdminSecret_CreatesOnFirstCall(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	got, err := st.EnsureInstanceAdminSecret(context.Background(), "candidate-secret")
	if err != nil {
		t.Fatalf("EnsureInstanceAdminSecret() error: %v", err)
	}
	if got != "candidate-secret" {
		t.Fatalf("expected the candidate to be persisted on first call, got %q", got)
	}

	var count int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM settings WHERE key = 'instance_admin_secret'").Scan(&count); err != nil {
		t.Fatalf("querying settings: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 settings row, got %d", count)
	}
}

func TestEnsureInstanceAdminSecret_ReturnsExistingOnSubsequentCalls(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()

	first, err := st.EnsureInstanceAdminSecret(ctx, "first-candidate")
	if err != nil {
		t.Fatalf("EnsureInstanceAdminSecret() first call error: %v", err)
	}

	second, err := st.EnsureInstanceAdminSecret(ctx, "second-candidate")
	if err != nil {
		t.Fatalf("EnsureInstanceAdminSecret() second call error: %v", err)
	}
	if second != first {
		t.Fatalf("expected the second call to return the existing secret %q, got %q", first, second)
	}
	if second == "second-candidate" {
		t.Fatalf("expected the second candidate to be discarded, not persisted")
	}
}
