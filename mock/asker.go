package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.Asker = (*Asker)(nil)

// Asker is a mock implementation of locdoc.Asker.
type Asker struct {
	AskFn func(ctx context.Context, projectID, question string) (string, error)
}

func (a *Asker) Ask(ctx context.Context, projectID, question string) (string, error) {
	return a.AskFn(ctx, projectID, question)
}
