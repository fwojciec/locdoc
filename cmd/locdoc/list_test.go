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

func TestListCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("lists projects with ID, name, and URL", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, _ locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{
						ID:        "proj-123",
						Name:      "react-docs",
						SourceURL: "https://react.dev/docs",
						CreatedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
					},
					{
						ID:        "proj-456",
						Name:      "go-docs",
						SourceURL: "https://go.dev/doc",
						CreatedAt: time.Date(2025, 1, 16, 11, 0, 0, 0, time.UTC),
					},
				}, nil
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

		cmd := &main.ListCmd{}

		err := cmd.Run(deps)

		require.NoError(t, err)

		output := stdout.String()
		// Should contain project IDs
		assert.Contains(t, output, "proj-123")
		assert.Contains(t, output, "proj-456")
		// Should contain project names
		assert.Contains(t, output, "react-docs")
		assert.Contains(t, output, "go-docs")
		// Should contain source URLs
		assert.Contains(t, output, "https://react.dev/docs")
		assert.Contains(t, output, "https://go.dev/doc")
	})

	t.Run("shows helpful message when no projects exist", func(t *testing.T) {
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

		cmd := &main.ListCmd{}

		err := cmd.Run(deps)

		require.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "No projects")
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

		cmd := &main.ListCmd{}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, dbErr, err)
		assert.Contains(t, stderr.String(), "error:")
	})
}
