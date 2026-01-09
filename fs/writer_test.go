package fs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLToPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "simple path",
			url:  "https://example.com/docs/api/users",
			want: "docs/api/users.md",
		},
		{
			name: "trailing slash becomes index",
			url:  "https://example.com/docs/",
			want: "docs/index.md",
		},
		{
			name: "root path becomes index",
			url:  "https://example.com/",
			want: "index.md",
		},
		{
			name: "no trailing slash",
			url:  "https://example.com/docs",
			want: "docs.md",
		},
		{
			name: "ignores query string",
			url:  "https://example.com/docs/api?version=2",
			want: "docs/api.md",
		},
		{
			name: "ignores fragment",
			url:  "https://example.com/docs/api#section",
			want: "docs/api.md",
		},
		{
			name: "root without trailing slash",
			url:  "https://example.com",
			want: "index.md",
		},
		{
			name: "deep nesting",
			url:  "https://example.com/a/b/c/d/e/f",
			want: "a/b/c/d/e/f.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := fs.URLToPath(tt.url)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatDocument(t *testing.T) {
	t.Parallel()

	t.Run("formats document with frontmatter", func(t *testing.T) {
		t.Parallel()

		doc := &locdoc.Document{
			SourceURL: "https://example.com/docs/api",
			Title:     "API Reference",
			Content:   "# API Reference\n\nThis is the API documentation.",
			FetchedAt: time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
		}

		got := fs.FormatDocument(doc)

		want := `---
source: https://example.com/docs/api
title: API Reference
crawled: 2025-01-08
---

# API Reference

This is the API documentation.`

		assert.Equal(t, want, got)
	})
}

func TestWriter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ locdoc.DocumentWriter = &fs.Writer{}
}

func TestWriter_CreateDocument(t *testing.T) {
	t.Parallel()

	t.Run("writes document to correct path with frontmatter", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		w := fs.NewWriter(baseDir)

		doc := &locdoc.Document{
			ProjectID: "test-project",
			SourceURL: "https://example.com/docs/api/users",
			Title:     "Users API",
			Content:   "# Users API\n\nManage users.",
			FetchedAt: time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
		}

		err := w.CreateDocument(context.Background(), doc)

		require.NoError(t, err)

		// Verify file was created at correct path
		filePath := filepath.Join(baseDir, "docs/api/users.md")
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		want := `---
source: https://example.com/docs/api/users
title: Users API
crawled: 2025-01-08
---

# Users API

Manage users.`

		assert.Equal(t, want, string(content))
	})

	t.Run("creates parent directories", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		w := fs.NewWriter(baseDir)

		doc := &locdoc.Document{
			ProjectID: "test-project",
			SourceURL: "https://example.com/deeply/nested/path/doc",
			Title:     "Nested Doc",
			Content:   "Content",
			FetchedAt: time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
		}

		err := w.CreateDocument(context.Background(), doc)

		require.NoError(t, err)

		filePath := filepath.Join(baseDir, "deeply/nested/path/doc.md")
		_, err = os.Stat(filePath)
		require.NoError(t, err)
	})

	t.Run("trailing slash creates index.md", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		w := fs.NewWriter(baseDir)

		doc := &locdoc.Document{
			ProjectID: "test-project",
			SourceURL: "https://example.com/docs/",
			Title:     "Docs Index",
			Content:   "Index content",
			FetchedAt: time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
		}

		err := w.CreateDocument(context.Background(), doc)

		require.NoError(t, err)

		filePath := filepath.Join(baseDir, "docs/index.md")
		_, err = os.Stat(filePath)
		require.NoError(t, err)
	})

	t.Run("validates document", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		w := fs.NewWriter(baseDir)

		doc := &locdoc.Document{
			// Missing ProjectID and SourceURL
			Title:   "Invalid Doc",
			Content: "Content",
		}

		err := w.CreateDocument(context.Background(), doc)

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})
}
