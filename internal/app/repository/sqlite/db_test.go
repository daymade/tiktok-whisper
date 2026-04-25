package sqlite

import (
	"testing"
)

// TestGetConnection only verifies that GetConnection returns a usable
// connection — it does NOT assert anything about schema. A previous version
// of this test queried sqlite_master for the `transcriptions` table, which
// silently passed on a developer machine that had a populated DB from prior
// runs but failed in any fresh clone or worktree. Schema-shape verification
// belongs in a migration / integration test that explicitly bootstraps the
// table first.
func TestGetConnection(t *testing.T) {
	db, err := GetConnection()
	if err != nil {
		t.Fatalf("GetConnection() error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Errorf("DB ping failed after GetConnection: %v", err)
	}
}
