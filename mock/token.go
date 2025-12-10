package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.TokenCounter = (*TokenCounter)(nil)

// TokenCounter is a mock implementation of locdoc.TokenCounter.
type TokenCounter struct {
	CountTokensFn func(ctx context.Context, text string) (int, error)
}

func (tc *TokenCounter) CountTokens(ctx context.Context, text string) (int, error) {
	return tc.CountTokensFn(ctx, text)
}
