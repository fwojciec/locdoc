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

func TestAskCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("asks question and prints answer", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "react-docs" {
					return []*locdoc.Project{{ID: "proj-123", Name: "react-docs"}}, nil
				}
				return []*locdoc.Project{}, nil
			},
		}

		asker := &mock.Asker{
			AskFn: func(_ context.Context, projectID, question string) (string, error) {
				if projectID == "proj-123" && question == "What is useState?" {
					return "useState is a React Hook.", nil
				}
				return "", nil
			},
		}

		stdout := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   &bytes.Buffer{},
			Projects: projects,
			Asker:    asker,
		}

		cmd := &main.AskCmd{Name: "react-docs", Question: "What is useState?"}
		err := cmd.Run(deps)

		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "useState is a React Hook.")
	})
}
