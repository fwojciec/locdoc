package gemini_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenCounter_CountTokens(t *testing.T) {
	t.Parallel()

	// Use a real model name that the tokenizer supports
	tc, err := gemini.NewTokenCounter("gemini-2.0-flash")
	require.NoError(t, err)

	// Verify it implements the interface
	var _ locdoc.TokenCounter = tc

	t.Run("counts tokens in text", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		count, err := tc.CountTokens(ctx, "Hello, world!")

		require.NoError(t, err)
		assert.Positive(t, count)
	})

	t.Run("empty string returns zero", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		count, err := tc.CountTokens(ctx, "")

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("longer text returns more tokens", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		shortCount, err := tc.CountTokens(ctx, "Hello")
		require.NoError(t, err)

		longCount, err := tc.CountTokens(ctx, "Hello, this is a much longer piece of text that should have more tokens than just a single word.")
		require.NoError(t, err)

		assert.Greater(t, longCount, shortCount)
	})
}
