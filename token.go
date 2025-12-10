package locdoc

import "context"

// TokenCounter counts tokens in text for a specific model.
type TokenCounter interface {
	CountTokens(ctx context.Context, text string) (int, error)
}
