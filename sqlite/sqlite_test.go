package sqlite_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc/sqlite"
	"github.com/stretchr/testify/require"
)

func TestDB_Open(t *testing.T) {
	t.Parallel()

	t.Run("creates schema on first open", func(t *testing.T) {
		t.Parallel()

		db := sqlite.NewDB(":memory:")
		err := db.Open()
		require.NoError(t, err)
		defer db.Close()

		// Verify tables exist by querying them
		ctx := context.Background()

		// Check projects table exists
		var projectCount int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&projectCount)
		require.NoError(t, err)

		// Check documents table exists
		var docCount int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents").Scan(&docCount)
		require.NoError(t, err)
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		t.Parallel()

		db := sqlite.NewDB("/nonexistent/path/db.sqlite")
		err := db.Open()
		require.Error(t, err)
	})

	t.Run("enables WAL mode for file-based databases", func(t *testing.T) {
		t.Parallel()

		dbPath := t.TempDir() + "/test.db"
		db := sqlite.NewDB(dbPath)
		err := db.Open()
		require.NoError(t, err)
		defer db.Close()

		ctx := context.Background()
		var journalMode string
		err = db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)
		require.NoError(t, err)
		require.Equal(t, "wal", journalMode)
	})
}
