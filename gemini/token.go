package gemini

import (
	"context"

	"github.com/fwojciec/locdoc"
	"google.golang.org/genai"
	"google.golang.org/genai/tokenizer"
)

var _ locdoc.TokenCounter = (*TokenCounter)(nil)

// TokenCounter counts tokens using the Gemini tokenizer.
type TokenCounter struct {
	tok *tokenizer.LocalTokenizer
}

// NewTokenCounter creates a new TokenCounter for the given model.
func NewTokenCounter(model string) (*TokenCounter, error) {
	tok, err := tokenizer.NewLocalTokenizer(model)
	if err != nil {
		return nil, err
	}
	return &TokenCounter{tok: tok}, nil
}

// CountTokens counts the number of tokens in the given text.
func (tc *TokenCounter) CountTokens(ctx context.Context, text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	contents := []*genai.Content{
		genai.NewContentFromText(text, "user"),
	}

	result, err := tc.tok.CountTokens(contents, nil)
	if err != nil {
		return 0, err
	}

	return int(result.TotalTokens), nil
}
