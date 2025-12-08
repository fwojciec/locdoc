package sqlite_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db := sqlite.NewDB(":memory:")
	require.NoError(t, db.Open())
	t.Cleanup(func() { db.Close() })
	return db
}

func TestProjectService_CreateProject(t *testing.T) {
	t.Parallel()

	t.Run("creates project with generated ID and timestamps", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		project := &locdoc.Project{
			Name:      "test-project",
			SourceURL: "https://example.com/docs",
		}

		err := svc.CreateProject(ctx, project)
		require.NoError(t, err)

		assert.NotEmpty(t, project.ID, "ID should be generated")
		assert.False(t, project.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.False(t, project.UpdatedAt.IsZero(), "UpdatedAt should be set")
	})

	t.Run("returns error for invalid project", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		project := &locdoc.Project{} // missing required fields

		err := svc.CreateProject(ctx, project)
		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})
}

func TestProjectService_FindProjectByID(t *testing.T) {
	t.Parallel()

	t.Run("returns project when found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		// Create a project first
		project := &locdoc.Project{
			Name:      "test-project",
			SourceURL: "https://example.com/docs",
			LocalPath: "/path/to/docs",
		}
		require.NoError(t, svc.CreateProject(ctx, project))

		// Find by ID
		found, err := svc.FindProjectByID(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, project.ID, found.ID)
		assert.Equal(t, project.Name, found.Name)
		assert.Equal(t, project.SourceURL, found.SourceURL)
		assert.Equal(t, project.LocalPath, found.LocalPath)
	})

	t.Run("returns ENOTFOUND when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		_, err := svc.FindProjectByID(ctx, "nonexistent-id")
		require.Error(t, err)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	})
}

func TestProjectService_FindProjects(t *testing.T) {
	t.Parallel()

	t.Run("returns all projects with empty filter", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		// Create multiple projects
		for i := 0; i < 3; i++ {
			project := &locdoc.Project{
				Name:      "project-" + string(rune('a'+i)),
				SourceURL: "https://example.com/docs",
			}
			require.NoError(t, svc.CreateProject(ctx, project))
		}

		projects, err := svc.FindProjects(ctx, locdoc.ProjectFilter{})
		require.NoError(t, err)
		assert.Len(t, projects, 3)
	})

	t.Run("filters by name", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		// Create projects
		p1 := &locdoc.Project{Name: "alpha", SourceURL: "https://example.com/alpha"}
		p2 := &locdoc.Project{Name: "beta", SourceURL: "https://example.com/beta"}
		require.NoError(t, svc.CreateProject(ctx, p1))
		require.NoError(t, svc.CreateProject(ctx, p2))

		name := "alpha"
		projects, err := svc.FindProjects(ctx, locdoc.ProjectFilter{Name: &name})
		require.NoError(t, err)
		require.Len(t, projects, 1)
		assert.Equal(t, "alpha", projects[0].Name)
	})

	t.Run("respects limit and offset", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		// Create 5 projects
		for i := 0; i < 5; i++ {
			project := &locdoc.Project{
				Name:      "project-" + string(rune('a'+i)),
				SourceURL: "https://example.com/docs",
			}
			require.NoError(t, svc.CreateProject(ctx, project))
		}

		projects, err := svc.FindProjects(ctx, locdoc.ProjectFilter{Limit: 2, Offset: 1})
		require.NoError(t, err)
		assert.Len(t, projects, 2)
	})
}

func TestProjectService_UpdateProject(t *testing.T) {
	t.Parallel()

	t.Run("updates project fields", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		// Create a project first
		project := &locdoc.Project{
			Name:      "original-name",
			SourceURL: "https://example.com/docs",
		}
		require.NoError(t, svc.CreateProject(ctx, project))
		originalUpdatedAt := project.UpdatedAt

		// Update it
		newName := "updated-name"
		newURL := "https://example.com/new-docs"
		updated, err := svc.UpdateProject(ctx, project.ID, locdoc.ProjectUpdate{
			Name:      &newName,
			SourceURL: &newURL,
		})
		require.NoError(t, err)

		assert.Equal(t, "updated-name", updated.Name)
		assert.Equal(t, "https://example.com/new-docs", updated.SourceURL)
		assert.True(t, updated.UpdatedAt.After(originalUpdatedAt) || updated.UpdatedAt.Equal(originalUpdatedAt))
	})

	t.Run("returns ENOTFOUND when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		name := "test"
		_, err := svc.UpdateProject(ctx, "nonexistent-id", locdoc.ProjectUpdate{Name: &name})
		require.Error(t, err)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	})
}

func TestProjectService_DeleteProject(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing project", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		// Create a project first
		project := &locdoc.Project{
			Name:      "test-project",
			SourceURL: "https://example.com/docs",
		}
		require.NoError(t, svc.CreateProject(ctx, project))

		// Delete it
		err := svc.DeleteProject(ctx, project.ID)
		require.NoError(t, err)

		// Verify it's gone
		_, err = svc.FindProjectByID(ctx, project.ID)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	})

	t.Run("returns ENOTFOUND when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := sqlite.NewProjectService(db)
		ctx := context.Background()

		err := svc.DeleteProject(ctx, "nonexistent-id")
		require.Error(t, err)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	})
}
