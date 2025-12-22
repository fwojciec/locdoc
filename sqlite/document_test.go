package sqlite_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestProject(t *testing.T, db *sqlite.DB) *locdoc.Project {
	t.Helper()
	svc := sqlite.NewProjectService(db)
	project := &locdoc.Project{
		Name:      "test-project",
		SourceURL: "https://example.com/docs",
	}
	require.NoError(t, svc.CreateProject(context.Background(), project))
	return project
}

func TestDocumentService_CreateDocument(t *testing.T) {
	t.Parallel()

	t.Run("creates document with generated ID and timestamp", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		doc := &locdoc.Document{
			ProjectID: project.ID,
			SourceURL: "https://example.com/docs/page1",
			Title:     "Page 1",
			Content:   "# Page 1\n\nThis is the content.",
		}

		err := svc.CreateDocument(ctx, doc)
		require.NoError(t, err)

		assert.NotEmpty(t, doc.ID, "ID should be generated")
		assert.NotEmpty(t, doc.ContentHash, "ContentHash should be generated")
		assert.False(t, doc.FetchedAt.IsZero(), "FetchedAt should be set")
	})

	t.Run("returns error for invalid document", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		doc := &locdoc.Document{} // missing required fields

		err := svc.CreateDocument(ctx, doc)
		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})

	t.Run("stores position field", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		doc := &locdoc.Document{
			ProjectID: project.ID,
			SourceURL: "https://example.com/docs/page1",
			Position:  42,
		}

		err := svc.CreateDocument(ctx, doc)
		require.NoError(t, err)

		found, err := svc.FindDocumentByID(ctx, doc.ID)
		require.NoError(t, err)
		assert.Equal(t, 42, found.Position)
	})
}

func TestDocumentService_FindDocumentByID(t *testing.T) {
	t.Parallel()

	t.Run("returns document when found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		doc := &locdoc.Document{
			ProjectID: project.ID,
			SourceURL: "https://example.com/docs/page1",
			FilePath:  "/docs/page1.md",
			Title:     "Page 1",
			Content:   "# Page 1\n\nContent here.",
		}
		require.NoError(t, svc.CreateDocument(ctx, doc))

		found, err := svc.FindDocumentByID(ctx, doc.ID)
		require.NoError(t, err)
		assert.Equal(t, doc.ID, found.ID)
		assert.Equal(t, doc.ProjectID, found.ProjectID)
		assert.Equal(t, doc.SourceURL, found.SourceURL)
		assert.Equal(t, doc.FilePath, found.FilePath)
		assert.Equal(t, doc.Title, found.Title)
		assert.Equal(t, doc.Content, found.Content)
		assert.Equal(t, doc.ContentHash, found.ContentHash)
	})

	t.Run("returns ENOTFOUND when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		_, err := svc.FindDocumentByID(ctx, "nonexistent-id")
		require.Error(t, err)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	})
}

func TestDocumentService_FindDocuments(t *testing.T) {
	t.Parallel()

	t.Run("returns all documents with empty filter", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		// Create multiple documents
		for i := 0; i < 3; i++ {
			doc := &locdoc.Document{
				ProjectID: project.ID,
				SourceURL: fmt.Sprintf("https://example.com/docs/page%d", i+1),
			}
			require.NoError(t, svc.CreateDocument(ctx, doc))
		}

		docs, err := svc.FindDocuments(ctx, locdoc.DocumentFilter{})
		require.NoError(t, err)
		assert.Len(t, docs, 3)
	})

	t.Run("filters by project ID", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		// Create two projects
		projectSvc := sqlite.NewProjectService(db)
		p1 := &locdoc.Project{Name: "project1", SourceURL: "https://example.com/p1"}
		p2 := &locdoc.Project{Name: "project2", SourceURL: "https://example.com/p2"}
		require.NoError(t, projectSvc.CreateProject(ctx, p1))
		require.NoError(t, projectSvc.CreateProject(ctx, p2))

		// Create documents for each project
		doc1 := &locdoc.Document{ProjectID: p1.ID, SourceURL: "https://example.com/p1/doc1"}
		doc2 := &locdoc.Document{ProjectID: p2.ID, SourceURL: "https://example.com/p2/doc1"}
		require.NoError(t, svc.CreateDocument(ctx, doc1))
		require.NoError(t, svc.CreateDocument(ctx, doc2))

		// Filter by project
		docs, err := svc.FindDocuments(ctx, locdoc.DocumentFilter{ProjectID: &p1.ID})
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, p1.ID, docs[0].ProjectID)
	})

	t.Run("filters by source URL", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		url := "https://example.com/docs/unique-page"
		doc := &locdoc.Document{ProjectID: project.ID, SourceURL: url}
		require.NoError(t, svc.CreateDocument(ctx, doc))
		require.NoError(t, svc.CreateDocument(ctx, &locdoc.Document{
			ProjectID: project.ID,
			SourceURL: "https://example.com/docs/other",
		}))

		docs, err := svc.FindDocuments(ctx, locdoc.DocumentFilter{SourceURL: &url})
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, url, docs[0].SourceURL)
	})

	t.Run("respects limit and offset", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			doc := &locdoc.Document{
				ProjectID: project.ID,
				SourceURL: fmt.Sprintf("https://example.com/docs/page%d", i+1),
			}
			require.NoError(t, svc.CreateDocument(ctx, doc))
		}

		docs, err := svc.FindDocuments(ctx, locdoc.DocumentFilter{Limit: 2, Offset: 1})
		require.NoError(t, err)
		assert.Len(t, docs, 2)
	})

	t.Run("includes position in results", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		doc := &locdoc.Document{
			ProjectID: project.ID,
			SourceURL: "https://example.com/docs/page1",
			Position:  99,
		}
		require.NoError(t, svc.CreateDocument(ctx, doc))

		docs, err := svc.FindDocuments(ctx, locdoc.DocumentFilter{ProjectID: &project.ID})
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, 99, docs[0].Position)
	})

	t.Run("sorts by position when SortBy is position", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		// Create documents with positions out of order
		for i, pos := range []int{3, 1, 2} {
			doc := &locdoc.Document{
				ProjectID: project.ID,
				SourceURL: fmt.Sprintf("https://example.com/docs/page%d", i+1),
				Position:  pos,
			}
			require.NoError(t, svc.CreateDocument(ctx, doc))
		}

		docs, err := svc.FindDocuments(ctx, locdoc.DocumentFilter{
			ProjectID: &project.ID,
			SortBy:    locdoc.SortByPosition,
		})
		require.NoError(t, err)
		require.Len(t, docs, 3)
		assert.Equal(t, 1, docs[0].Position)
		assert.Equal(t, 2, docs[1].Position)
		assert.Equal(t, 3, docs[2].Position)
	})
}

func TestDocumentService_DeleteDocument(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing document", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		project := createTestProject(t, db)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		doc := &locdoc.Document{
			ProjectID: project.ID,
			SourceURL: "https://example.com/docs/page1",
		}
		require.NoError(t, svc.CreateDocument(ctx, doc))

		err := svc.DeleteDocument(ctx, doc.ID)
		require.NoError(t, err)

		_, err = svc.FindDocumentByID(ctx, doc.ID)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	})

	t.Run("returns ENOTFOUND when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		err := svc.DeleteDocument(ctx, "nonexistent-id")
		require.Error(t, err)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	})
}

func TestDocumentService_DeleteDocumentsByProject(t *testing.T) {
	t.Parallel()

	t.Run("deletes all documents for a project", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewDocumentService(db)
		ctx := context.Background()

		// Create two projects
		projectSvc := sqlite.NewProjectService(db)
		p1 := &locdoc.Project{Name: "project1", SourceURL: "https://example.com/p1"}
		p2 := &locdoc.Project{Name: "project2", SourceURL: "https://example.com/p2"}
		require.NoError(t, projectSvc.CreateProject(ctx, p1))
		require.NoError(t, projectSvc.CreateProject(ctx, p2))

		// Create documents for each project
		for i := 0; i < 3; i++ {
			doc := &locdoc.Document{
				ProjectID: p1.ID,
				SourceURL: fmt.Sprintf("https://example.com/p1/doc%d", i+1),
			}
			require.NoError(t, svc.CreateDocument(ctx, doc))
		}
		doc2 := &locdoc.Document{ProjectID: p2.ID, SourceURL: "https://example.com/p2/doc1"}
		require.NoError(t, svc.CreateDocument(ctx, doc2))

		// Delete documents for p1
		err := svc.DeleteDocumentsByProject(ctx, p1.ID)
		require.NoError(t, err)

		// Verify p1 docs are gone
		docs, err := svc.FindDocuments(ctx, locdoc.DocumentFilter{ProjectID: &p1.ID})
		require.NoError(t, err)
		assert.Empty(t, docs)

		// Verify p2 doc still exists
		docs, err = svc.FindDocuments(ctx, locdoc.DocumentFilter{ProjectID: &p2.ID})
		require.NoError(t, err)
		assert.Len(t, docs, 1)
	})
}
