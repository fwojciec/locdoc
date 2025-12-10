package locdoc_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/stretchr/testify/assert"
)

func TestFormatDocuments(t *testing.T) {
	t.Parallel()

	t.Run("formats single document with title", func(t *testing.T) {
		t.Parallel()

		docs := []*locdoc.Document{
			{Title: "Getting Started", Content: "Welcome to the docs."},
		}

		result := locdoc.FormatDocuments(docs)

		expected := "## Document: Getting Started\nWelcome to the docs."
		assert.Equal(t, expected, result)
	})

	t.Run("uses source URL when title is empty", func(t *testing.T) {
		t.Parallel()

		docs := []*locdoc.Document{
			{SourceURL: "https://example.com/docs", Content: "Some content."},
		}

		result := locdoc.FormatDocuments(docs)

		expected := "## Document: https://example.com/docs\nSome content."
		assert.Equal(t, expected, result)
	})

	t.Run("formats multiple documents with blank line separator", func(t *testing.T) {
		t.Parallel()

		docs := []*locdoc.Document{
			{Title: "Doc One", Content: "First content."},
			{Title: "Doc Two", Content: "Second content."},
		}

		result := locdoc.FormatDocuments(docs)

		expected := "## Document: Doc One\nFirst content.\n\n## Document: Doc Two\nSecond content."
		assert.Equal(t, expected, result)
	})

	t.Run("returns empty string for empty slice", func(t *testing.T) {
		t.Parallel()

		result := locdoc.FormatDocuments([]*locdoc.Document{})

		assert.Empty(t, result)
	})

	t.Run("returns empty string for nil slice", func(t *testing.T) {
		t.Parallel()

		result := locdoc.FormatDocuments(nil)

		assert.Empty(t, result)
	})

	t.Run("preserves markdown content", func(t *testing.T) {
		t.Parallel()

		docs := []*locdoc.Document{
			{Title: "Markdown Doc", Content: "# Heading\n\n- item 1\n- item 2\n\n```go\nfunc main() {}\n```"},
		}

		result := locdoc.FormatDocuments(docs)

		expected := "## Document: Markdown Doc\n# Heading\n\n- item 1\n- item 2\n\n```go\nfunc main() {}\n```"
		assert.Equal(t, expected, result)
	})
}
