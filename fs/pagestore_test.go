package fs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Story: Atomic File Storage
// The store uses temp directory for atomic updates

func TestFileStore_SaveWritesToTempDirectory(t *testing.T) {
	t.Parallel()

	// Given a store targeting a directory
	base := t.TempDir()
	store := fs.NewFileStore(base, "output")

	// When I save a page
	err := store.Save(context.Background(), &locdoc.Page{
		URL:     "https://example.com/docs/api",
		Title:   "API Reference",
		Content: "# API\n\nWelcome to the API.",
	})

	// Then no error occurs
	require.NoError(t, err)

	// And the file exists in the temp directory (not final)
	tempPath := filepath.Join(base, "output.tmp", "docs", "api.md")
	_, err = os.Stat(tempPath)
	require.NoError(t, err, "file should exist in temp directory")

	// And final directory does not exist yet
	finalPath := filepath.Join(base, "output", "docs", "api.md")
	_, err = os.Stat(finalPath)
	assert.True(t, os.IsNotExist(err), "final directory should not exist until commit")
}

func TestFileStore_CommitMovesFromTempToFinal(t *testing.T) {
	t.Parallel()

	// Given a store with saved pages
	base := t.TempDir()
	store := fs.NewFileStore(base, "output")
	err := store.Save(context.Background(), &locdoc.Page{
		URL:     "https://example.com/a",
		Title:   "A",
		Content: "# A",
	})
	require.NoError(t, err)

	// When I commit
	err = store.Commit()

	// Then no error occurs
	require.NoError(t, err)

	// And final directory exists with content
	finalPath := filepath.Join(base, "output", "a.md")
	_, err = os.Stat(finalPath)
	require.NoError(t, err, "file should exist in final directory after commit")

	// And temp directory is gone
	tempDir := filepath.Join(base, "output.tmp")
	_, err = os.Stat(tempDir)
	assert.True(t, os.IsNotExist(err), "temp directory should be removed after commit")
}

func TestFileStore_AbortCleansUpTempDirectory(t *testing.T) {
	t.Parallel()

	// Given a store with saved pages
	base := t.TempDir()
	store := fs.NewFileStore(base, "output")
	err := store.Save(context.Background(), &locdoc.Page{
		URL:     "https://example.com/a",
		Title:   "A",
		Content: "# A",
	})
	require.NoError(t, err)

	// When I abort
	err = store.Abort()

	// Then no error occurs
	require.NoError(t, err)

	// And temp directory is cleaned up
	tempDir := filepath.Join(base, "output.tmp")
	_, err = os.Stat(tempDir)
	assert.True(t, os.IsNotExist(err), "temp directory should be removed after abort")

	// And final directory doesn't exist
	finalDir := filepath.Join(base, "output")
	_, err = os.Stat(finalDir)
	assert.True(t, os.IsNotExist(err), "final directory should not exist after abort")
}

func TestFileStore_IncludesFrontmatter(t *testing.T) {
	t.Parallel()

	// Given a page with metadata
	base := t.TempDir()
	store := fs.NewFileStore(base, "output")
	err := store.Save(context.Background(), &locdoc.Page{
		URL:     "https://example.com/intro",
		Title:   "Introduction",
		Content: "# Welcome",
	})
	require.NoError(t, err)
	err = store.Commit()
	require.NoError(t, err)

	// When I read the file
	content, err := os.ReadFile(filepath.Join(base, "output", "intro.md"))
	require.NoError(t, err)

	// Then it has YAML frontmatter
	assert.Contains(t, string(content), "---")
	assert.Contains(t, string(content), "source: https://example.com/intro")
	assert.Contains(t, string(content), "title: Introduction")
	// And content follows the frontmatter
	assert.Contains(t, string(content), "# Welcome")
}

func TestFileStore_PreservesURLPathStructure(t *testing.T) {
	t.Parallel()

	// Given pages with nested paths
	base := t.TempDir()
	store := fs.NewFileStore(base, "output")
	err := store.Save(context.Background(), &locdoc.Page{
		URL:     "https://example.com/docs/api/users",
		Title:   "Users API",
		Content: "# Users",
	})
	require.NoError(t, err)
	err = store.Commit()
	require.NoError(t, err)

	// Then nested directories are created
	expectedPath := filepath.Join(base, "output", "docs", "api", "users.md")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "nested path structure should be preserved")
}

func TestFileStore_RejectsPathTraversal(t *testing.T) {
	t.Parallel()

	// Given a store
	base := t.TempDir()
	store := fs.NewFileStore(base, "output")

	// When I try to save a page with path traversal
	err := store.Save(context.Background(), &locdoc.Page{
		URL:     "https://example.com/../../../etc/passwd",
		Title:   "Malicious",
		Content: "bad content",
	})

	// Then an error is returned
	require.Error(t, err, "path traversal should be rejected")
	assert.Contains(t, err.Error(), "path traversal")
}
