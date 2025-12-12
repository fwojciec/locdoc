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

func TestAskCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("asks question and prints answer", func(t *testing.T) {
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

		asker := &mock.Asker{
			AskFn: func(_ context.Context, projectID, question string) (string, error) {
				if projectID == "proj-123" && question == "What is useState?" {
					return "useState is a React Hook that lets you add state to function components.", nil
				}
				return "", errors.New("unexpected call")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
			Asker:    asker,
		}

		cmd := &main.AskCmd{
			Name:     "react-docs",
			Question: "What is useState?",
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "useState is a React Hook that lets you add state to function components.")
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

		cmd := &main.AskCmd{
			Name:     "nonexistent",
			Question: "What is useState?",
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

		cmd := &main.AskCmd{
			Name:     "react-docs",
			Question: "What is useState?",
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, dbErr, err)
		assert.Contains(t, stderr.String(), "error:")
	})

	t.Run("returns error when Asker.Ask fails", func(t *testing.T) {
		t.Parallel()

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

		askErr := errors.New("LLM service unavailable")

		asker := &mock.Asker{
			AskFn: func(_ context.Context, _, _ string) (string, error) {
				return "", askErr
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
			Asker:    asker,
		}

		cmd := &main.AskCmd{
			Name:     "react-docs",
			Question: "What is useState?",
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Equal(t, askErr, err)
		assert.Contains(t, stderr.String(), "error:")
	})
}
