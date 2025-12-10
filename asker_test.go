package locdoc_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAsker verifies Asker interface can be implemented.
type mockAsker struct {
	AskFn func(ctx context.Context, projectID, question string) (string, error)
}

func (m *mockAsker) Ask(ctx context.Context, projectID, question string) (string, error) {
	return m.AskFn(ctx, projectID, question)
}

// Compile-time check that mockAsker implements Asker.
var _ locdoc.Asker = (*mockAsker)(nil)

func TestAsker_CanBeImplemented(t *testing.T) {
	t.Parallel()

	asker := &mockAsker{
		AskFn: func(_ context.Context, projectID, question string) (string, error) {
			return "answer to " + question, nil
		},
	}

	answer, err := asker.Ask(context.Background(), "proj-1", "what is this?")

	require.NoError(t, err)
	assert.Equal(t, "answer to what is this?", answer)
}
