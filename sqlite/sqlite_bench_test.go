package sqlite_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/sqlite"
	"github.com/stretchr/testify/require"
)

// BenchmarkWALMode compares write performance between WAL and rollback journal modes.
// This simulates a crawl workload: creating a project and inserting many documents.
func BenchmarkWALMode(b *testing.B) {
	b.Run("rollback_journal", func(b *testing.B) {
		benchmarkDocumentInserts(b, false)
	})

	b.Run("wal_mode", func(b *testing.B) {
		benchmarkDocumentInserts(b, true)
	})
}

func benchmarkDocumentInserts(b *testing.B, useWAL bool) {
	b.Helper()

	// Create a temporary file for the database
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db := sqlite.NewDB(dbPath)
	require.NoError(b, db.Open())

	// Enable WAL mode if requested
	if useWAL {
		ctx := context.Background()
		_, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL")
		require.NoError(b, err)
	}

	defer func() {
		db.Close()
		// Clean up WAL files if they exist
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}()

	// Create a project for the documents
	ctx := context.Background()
	projectSvc := sqlite.NewProjectService(db)
	project := &locdoc.Project{
		Name:      "benchmark-project",
		SourceURL: "https://example.com/docs",
	}
	require.NoError(b, projectSvc.CreateProject(ctx, project))

	docSvc := sqlite.NewDocumentService(db)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		doc := &locdoc.Document{
			ProjectID: project.ID,
			SourceURL: fmt.Sprintf("https://example.com/docs/page%d", i),
			Title:     fmt.Sprintf("Page %d", i),
			Content:   fmt.Sprintf("# Page %d\n\nThis is the content of page %d with some additional text to make it more realistic. Lorem ipsum dolor sit amet, consectetur adipiscing elit.", i, i),
			Position:  i,
		}
		if err := docSvc.CreateDocument(ctx, doc); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBulkInserts tests inserting a batch of documents (simulating a full crawl).
func BenchmarkBulkInserts(b *testing.B) {
	const docsPerCrawl = 100

	b.Run("rollback_journal", func(b *testing.B) {
		benchmarkBulkInserts(b, false, docsPerCrawl)
	})

	b.Run("wal_mode", func(b *testing.B) {
		benchmarkBulkInserts(b, true, docsPerCrawl)
	})
}

func benchmarkBulkInserts(b *testing.B, useWAL bool, docsPerCrawl int) {
	b.Helper()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		tmpDir := b.TempDir()
		dbPath := filepath.Join(tmpDir, fmt.Sprintf("bench%d.db", i))

		db := sqlite.NewDB(dbPath)
		require.NoError(b, db.Open())

		if useWAL {
			ctx := context.Background()
			_, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL")
			require.NoError(b, err)
		}

		ctx := context.Background()
		projectSvc := sqlite.NewProjectService(db)
		project := &locdoc.Project{
			Name:      "benchmark-project",
			SourceURL: "https://example.com/docs",
		}
		require.NoError(b, projectSvc.CreateProject(ctx, project))

		docSvc := sqlite.NewDocumentService(db)

		b.StartTimer()

		// Insert batch of documents
		for j := 0; j < docsPerCrawl; j++ {
			doc := &locdoc.Document{
				ProjectID: project.ID,
				SourceURL: fmt.Sprintf("https://example.com/docs/page%d", j),
				Title:     fmt.Sprintf("Page %d", j),
				Content:   fmt.Sprintf("# Page %d\n\nContent for page %d. Lorem ipsum dolor sit amet.", j, j),
				Position:  j,
			}
			if err := docSvc.CreateDocument(ctx, doc); err != nil {
				b.Fatal(err)
			}
		}

		b.StopTimer()
		db.Close()
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}
}
