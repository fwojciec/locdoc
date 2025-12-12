package main_test

import (
	"bytes"
	"context"
	"testing"

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
					return []*locdoc.Project{{ID: "proj-123", Name: "react-docs"}}, nil
				}
				return []*locdoc.Project{}, nil
			},
		}

		documents := &mock.DocumentService{
			FindDocumentsFn: func(_ context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				if filter.ProjectID != nil && *filter.ProjectID == "proj-123" {
					return []*locdoc.Document{
						{ID: "doc-1", Title: "Getting Started", SourceURL: "https://react.dev/docs/getting-started"},
						{ID: "doc-2", Title: "Components", SourceURL: "https://react.dev/docs/components"},
					}, nil
				}
				return []*locdoc.Document{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:       context.Background(),
			Stdout:    stdout,
			Stderr:    &bytes.Buffer{},
			Projects:  projects,
			Documents: documents,
		}

		cmd := &main.DocsCmd{Name: "react-docs"}
		err := cmd.Run(deps)

		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "Getting Started")
		assert.Contains(t, stdout.String(), "Components")
	})

	t.Run("shows full content with --full flag", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "react-docs" {
					return []*locdoc.Project{{ID: "proj-123", Name: "react-docs"}}, nil
				}
				return []*locdoc.Project{}, nil
			},
		}

		documents := &mock.DocumentService{
			FindDocumentsFn: func(_ context.Context, _ locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return []*locdoc.Document{
					{ID: "doc-1", Title: "Getting Started", Content: "# Getting Started\n\nWelcome."},
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:       context.Background(),
			Stdout:    stdout,
			Stderr:    &bytes.Buffer{},
			Projects:  projects,
			Documents: documents,
		}

		cmd := &main.DocsCmd{Name: "react-docs", Full: true}
		err := cmd.Run(deps)

		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "# Getting Started")
		assert.Contains(t, stdout.String(), "Welcome.")
	})
}
