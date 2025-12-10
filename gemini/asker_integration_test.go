//go:build integration

package gemini_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/gemini"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestAsker_Integration_ReturnsAnswer(t *testing.T) {
	t.Parallel()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	require.NoError(t, err)

	docs := &mock.DocumentService{
		FindDocumentsFn: func(context.Context, locdoc.DocumentFilter) ([]*locdoc.Document, error) {
			return []*locdoc.Document{
				{
					Title:   "Getting Started",
					Content: "HTMX is a library that allows you to access modern browser features directly from HTML.",
				},
			}, nil
		},
	}

	asker := gemini.NewAsker(client, docs)

	answer, err := asker.Ask(ctx, "proj-1", "What is HTMX?")

	require.NoError(t, err)
	assert.NotEmpty(t, answer)
	assert.Contains(t, answer, "HTMX")
}
