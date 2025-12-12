package main_test

import (
	"bytes"
	"context"
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
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   &bytes.Buffer{},
			Projects: projects,
		}

		err := (&main.ListCmd{}).Run(deps)

		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "react-docs")
		assert.Contains(t, stdout.String(), "https://react.dev/docs")
	})

	t.Run("shows helpful message when no projects exist", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, _ locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   &bytes.Buffer{},
			Projects: projects,
		}

		err := (&main.ListCmd{}).Run(deps)

		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "No projects")
	})
}
