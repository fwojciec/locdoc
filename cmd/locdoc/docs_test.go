package main_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("lists documents for project", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "react-docs" {
					return []*locdoc.Project{
						{
							ID:        "proj-123",
							Name:      "react-docs",
							SourceURL: "https://react.dev/docs",
							CreatedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
						},
					}, nil
				}
				return []*locdoc.Project{}, nil
			},
		}

		documents := &mock.DocumentService{
			FindDocumentsFn: func(_ context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				if filter.ProjectID != nil && *filter.ProjectID == "proj-123" {
					return []*locdoc.Document{
						{
							ID:        "doc-1",
							ProjectID: "proj-123",
							Title:     "Getting Started",
							SourceURL: "https://react.dev/docs/getting-started",
						},
						{
							ID:        "doc-2",
							ProjectID: "proj-123",
							Title:     "Components",
							SourceURL: "https://react.dev/docs/components",
						},
					}, nil
				}
				return []*locdoc.Document{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:       context.Background(),
			Stdout:    stdout,
			Stderr:    stderr,
			Projects:  projects,
			Documents: documents,
		}

		cmd := &main.DocsCmd{
			Name: "react-docs",
			Full: false,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "Documents for react-docs")
		assert.Contains(t, stdout.String(), "Getting Started")
		assert.Contains(t, stdout.String(), "https://react.dev/docs/getting-started")
		assert.Contains(t, stdout.String(), "Components")
		assert.Contains(t, stdout.String(), "https://react.dev/docs/components")
		assert.Empty(t, stderr.String())
	})

	t.Run("shows full content with --full flag", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "react-docs" {
					return []*locdoc.Project{
						{
							ID:        "proj-123",
							Name:      "react-docs",
							SourceURL: "https://react.dev/docs",
						},
					}, nil
				}
				return []*locdoc.Project{}, nil
			},
		}

		documents := &mock.DocumentService{
			FindDocumentsFn: func(_ context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				if filter.ProjectID != nil && *filter.ProjectID == "proj-123" {
					return []*locdoc.Document{
						{
							ID:        "doc-1",
							ProjectID: "proj-123",
							Title:     "Getting Started",
							SourceURL: "https://react.dev/docs/getting-started",
							Content:   "# Getting Started\n\nWelcome to the docs.",
						},
						{
							ID:        "doc-2",
							ProjectID: "proj-123",
							Title:     "Functions",
							SourceURL: "https://react.dev/docs/functions",
							Content:   "# Functions\n\nHere are the functions.",
						},
					}, nil
				}
				return []*locdoc.Document{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:       context.Background(),
			Stdout:    stdout,
			Stderr:    stderr,
			Projects:  projects,
			Documents: documents,
		}

		cmd := &main.DocsCmd{
			Name: "react-docs",
			Full: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		// Uses FormatDocuments output
		assert.Contains(t, stdout.String(), "## Document: Getting Started")
		assert.Contains(t, stdout.String(), "# Getting Started")
		assert.Contains(t, stdout.String(), "Welcome to the docs.")
		assert.Contains(t, stdout.String(), "## Document: Functions")
		assert.Contains(t, stdout.String(), "# Functions")
		assert.Contains(t, stdout.String(), "Here are the functions.")
		assert.Empty(t, stderr.String())
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, _ locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
		}

		cmd := &main.DocsCmd{
			Name: "nonexistent",
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
		assert.Contains(t, stderr.String(), "not found")
	})

	t.Run("returns error when project has no documents", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "empty-project" {
					return []*locdoc.Project{
						{
							ID:        "proj-empty",
							Name:      "empty-project",
							SourceURL: "https://example.com/docs",
						},
					}, nil
				}
				return []*locdoc.Project{}, nil
			},
		}

		documents := &mock.DocumentService{
			FindDocumentsFn: func(_ context.Context, _ locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return []*locdoc.Document{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:       context.Background(),
			Stdout:    stdout,
			Stderr:    stderr,
			Projects:  projects,
			Documents: documents,
		}

		cmd := &main.DocsCmd{
			Name: "empty-project",
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
		assert.Contains(t, stderr.String(), "has no documents")
	})

	t.Run("returns error when FindProjects fails", func(t *testing.T) {
		t.Parallel()

		dbErr := errors.New("database connection failed")

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, _ locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return nil, dbErr
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
		}

		cmd := &main.DocsCmd{
			Name: "react-docs",
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, dbErr, err)
		assert.Contains(t, stderr.String(), "error:")
	})

	t.Run("returns error when FindDocuments fails", func(t *testing.T) {
		t.Parallel()

		docErr := errors.New("document query failed")

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "react-docs" {
					return []*locdoc.Project{
						{
							ID:   "proj-123",
							Name: "react-docs",
						},
					}, nil
				}
				return []*locdoc.Project{}, nil
			},
		}

		documents := &mock.DocumentService{
			FindDocumentsFn: func(_ context.Context, _ locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return nil, docErr
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:       context.Background(),
			Stdout:    stdout,
			Stderr:    stderr,
			Projects:  projects,
			Documents: documents,
		}

		cmd := &main.DocsCmd{
			Name: "react-docs",
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, docErr, err)
		assert.Contains(t, stderr.String(), "error:")
	})

	t.Run("uses source URL when title is empty", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "react-docs" {
					return []*locdoc.Project{
						{
							ID:        "proj-123",
							Name:      "react-docs",
							SourceURL: "https://react.dev/docs",
						},
					}, nil
				}
				return []*locdoc.Project{}, nil
			},
		}

		documents := &mock.DocumentService{
			FindDocumentsFn: func(_ context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				if filter.ProjectID != nil && *filter.ProjectID == "proj-123" {
					return []*locdoc.Document{
						{
							ID:        "doc-1",
							ProjectID: "proj-123",
							Title:     "", // Empty title
							SourceURL: "https://react.dev/docs/some-page",
						},
					}, nil
				}
				return []*locdoc.Document{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:       context.Background(),
			Stdout:    stdout,
			Stderr:    stderr,
			Projects:  projects,
			Documents: documents,
		}

		cmd := &main.DocsCmd{
			Name: "react-docs",
			Full: false,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		// The URL should appear as both the title and the link
		output := stdout.String()
		assert.Contains(t, output, "1. https://react.dev/docs/some-page")
		assert.Empty(t, stderr.String())
	})
}
