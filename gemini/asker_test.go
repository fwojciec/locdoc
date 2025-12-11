package gemini_test

import (
	"context"
	"strings"
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

func TestBuildUserPrompt_XMLDocumentStructure(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{
		{Title: "Getting Started", SourceURL: "https://htmx.org/docs/", Content: "HTMX is a library."},
	}

	prompt := gemini.BuildUserPrompt(docs, "What is HTMX?")

	assert.Contains(t, prompt, "<documents>")
	assert.Contains(t, prompt, "</documents>")
	assert.Contains(t, prompt, "<document>")
	assert.Contains(t, prompt, "</document>")
	assert.Contains(t, prompt, "<index>1</index>")
	assert.Contains(t, prompt, "<title>Getting Started</title>")
	assert.Contains(t, prompt, "<source>https://htmx.org/docs/</source>")
	assert.Contains(t, prompt, "<content>HTMX is a library.</content>")
}

func TestBuildUserPrompt_TitleFallsBackToSourceURL(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{
		{Title: "", SourceURL: "https://htmx.org/docs/", Content: "Content here."},
	}

	prompt := gemini.BuildUserPrompt(docs, "question")

	assert.Contains(t, prompt, "<title>https://htmx.org/docs/</title>")
}

func TestBuildUserPrompt_MultipleDocuments(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{
		{Title: "Doc One", SourceURL: "https://example.com/1", Content: "First content."},
		{Title: "Doc Two", SourceURL: "https://example.com/2", Content: "Second content."},
	}

	prompt := gemini.BuildUserPrompt(docs, "question")

	assert.Contains(t, prompt, "<index>1</index>")
	assert.Contains(t, prompt, "<index>2</index>")
	assert.Contains(t, prompt, "<title>Doc One</title>")
	assert.Contains(t, prompt, "<title>Doc Two</title>")
}

func TestBuildUserPrompt_QuestionInXMLTags(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{{Title: "Doc", SourceURL: "https://example.com", Content: "Content"}}

	prompt := gemini.BuildUserPrompt(docs, "How do I use this?")

	assert.Contains(t, prompt, "<question>How do I use this?</question>")
}

func TestBuildUserPrompt_TrailingInstructions(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{{Title: "Doc", SourceURL: "https://example.com", Content: "Content"}}

	prompt := gemini.BuildUserPrompt(docs, "question")

	assert.Contains(t, prompt, "<instructions>")
	assert.Contains(t, prompt, "</instructions>")
}

func TestBuildUserPrompt_InstructionsSpecifySourcesFormat(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{{Title: "Doc", SourceURL: "https://example.com", Content: "Content"}}

	prompt := gemini.BuildUserPrompt(docs, "question")

	assert.Contains(t, prompt, "---")
	assert.Contains(t, prompt, "Sources:")
}

func TestBuildUserPrompt_SandwichOrder(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{{Title: "Doc", SourceURL: "https://example.com", Content: "Content"}}

	prompt := gemini.BuildUserPrompt(docs, "question")

	// Verify sandwich pattern: documents -> question -> instructions
	docsEnd := strings.Index(prompt, "</documents>")
	questionStart := strings.Index(prompt, "<question>")
	instructionsStart := strings.Index(prompt, "<instructions>")

	assert.Greater(t, questionStart, docsEnd, "question should come after documents")
	assert.Greater(t, instructionsStart, questionStart, "instructions should come after question")
}

func TestBuildUserPrompt_DoesNotContainSystemInstruction(t *testing.T) {
	t.Parallel()

	docs := []*locdoc.Document{{Title: "Doc", Content: "Content"}}

	prompt := gemini.BuildUserPrompt(docs, "question")

	assert.NotContains(t, prompt, "You are a helpful assistant")
}
