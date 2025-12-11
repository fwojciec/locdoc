package gemini_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/gemini"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsker_Ask_ReturnsErrorWhenNoDocuments(t *testing.T) {
	t.Parallel()

	docs := &mock.DocumentService{
		FindDocumentsFn: func(context.Context, locdoc.DocumentFilter) ([]*locdoc.Document, error) {
			return []*locdoc.Document{}, nil
		},
	}

	asker := gemini.NewAsker(nil, docs) // nil client ok for this test

	_, err := asker.Ask(context.Background(), "proj-1", "what is this?")

	require.Error(t, err)
	assert.Equal(t, locdoc.ENOTFOUND, locdoc.ErrorCode(err))
	assert.Contains(t, locdoc.ErrorMessage(err), "no documents")
}

func TestAsker_Ask_PropagatesDocumentServiceError(t *testing.T) {
	t.Parallel()

	expectedErr := locdoc.Errorf(locdoc.EINTERNAL, "database error")
	docs := &mock.DocumentService{
		FindDocumentsFn: func(context.Context, locdoc.DocumentFilter) ([]*locdoc.Document, error) {
			return nil, expectedErr
		},
	}

	asker := gemini.NewAsker(nil, docs)

	_, err := asker.Ask(context.Background(), "proj-1", "what is this?")

	require.Error(t, err)
	assert.Equal(t, locdoc.EINTERNAL, locdoc.ErrorCode(err))
	assert.Contains(t, locdoc.ErrorMessage(err), "database error")
}

func TestAsker_Ask_ReturnsErrorWhenProjectIDEmpty(t *testing.T) {
	t.Parallel()

	asker := gemini.NewAsker(nil, nil)

	_, err := asker.Ask(context.Background(), "", "what is this?")

	require.Error(t, err)
	assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	assert.Contains(t, locdoc.ErrorMessage(err), "project ID required")
}

func TestAsker_Ask_ReturnsErrorWhenQuestionEmpty(t *testing.T) {
	t.Parallel()

	asker := gemini.NewAsker(nil, nil)

	_, err := asker.Ask(context.Background(), "proj-1", "")

	require.Error(t, err)
	assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	assert.Contains(t, locdoc.ErrorMessage(err), "question required")
}

func TestBuildConfig_SetsSystemInstruction(t *testing.T) {
	t.Parallel()

	config := gemini.BuildConfig()

	require.NotNil(t, config.SystemInstruction)
	require.Len(t, config.SystemInstruction.Parts, 1)
	assert.Contains(t, config.SystemInstruction.Parts[0].Text, "helpful assistant")
}

func TestBuildConfig_SetsTemperature(t *testing.T) {
	t.Parallel()

	config := gemini.BuildConfig()

	require.NotNil(t, config.Temperature)
	assert.InDelta(t, 0.4, *config.Temperature, 0.001)
}

func TestBuildUserPrompt_ContainsDocumentation(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{
		{Title: "Getting Started", Content: "HTMX is a library."},
	}

	prompt := gemini.BuildUserPrompt(docs, "What is HTMX?")

	assert.Contains(t, prompt, "<documentation>")
	assert.Contains(t, prompt, "Getting Started")
	assert.Contains(t, prompt, "HTMX is a library.")
	assert.Contains(t, prompt, "</documentation>")
}

func TestBuildUserPrompt_ContainsQuestion(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{{Title: "Doc", Content: "Content"}}

	prompt := gemini.BuildUserPrompt(docs, "How do I use this?")

	assert.Contains(t, prompt, "Question: How do I use this?")
}

func TestBuildUserPrompt_DoesNotContainSystemInstruction(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{{Title: "Doc", Content: "Content"}}

	prompt := gemini.BuildUserPrompt(docs, "question")

	assert.NotContains(t, prompt, "You are a helpful assistant")
}
