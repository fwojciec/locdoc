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

func TestDeleteCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("deletes project when --force is set", func(t *testing.T) {
		t.Parallel()

		var deletedID string
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
			DeleteProjectFn: func(_ context.Context, id string) error {
				deletedID = id
				return nil
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

		cmd := &main.DeleteCmd{
			Name:  "react-docs",
			Force: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		assert.Equal(t, "proj-123", deletedID)
		assert.Contains(t, stdout.String(), "Deleted")
	})

	t.Run("requires --force flag", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{
						ID:        "proj-123",
						Name:      "react-docs",
						SourceURL: "https://react.dev/docs",
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

		cmd := &main.DeleteCmd{
			Name:  "react-docs",
			Force: false,
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Contains(t, stderr.String(), "--force")
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

		cmd := &main.DeleteCmd{
			Name:  "nonexistent",
			Force: true,
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
		assert.Contains(t, stderr.String(), "not found")
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

		cmd := &main.DeleteCmd{
			Name:  "react-docs",
			Force: true,
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, dbErr, err)
		assert.Contains(t, stderr.String(), "error:")
	})

	t.Run("returns error when DeleteProject fails", func(t *testing.T) {
		t.Parallel()

		deleteErr := errors.New("delete failed")

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, _ locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{
						ID:   "proj-123",
						Name: "react-docs",
					},
				}, nil
			},
			DeleteProjectFn: func(_ context.Context, _ string) error {
				return deleteErr
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

		cmd := &main.DeleteCmd{
			Name:  "react-docs",
			Force: true,
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, deleteErr, err)
		assert.Contains(t, stderr.String(), "error:")
	})
}
