package mock_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentWriter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Verify mock can be used where DocumentWriter is expected
	var _ locdoc.DocumentWriter = &mock.DocumentWriter{}
}

func TestDocumentWriter_CreateDocument(t *testing.T) {
	t.Parallel()

	t.Run("delegates to CreateDocumentFn", func(t *testing.T) {
		t.Parallel()

		var calledWith *locdoc.Document
		w := &mock.DocumentWriter{
			CreateDocumentFn: func(_ context.Context, doc *locdoc.Document) error {
				calledWith = doc
				return nil
			},
		}

		doc := &locdoc.Document{
			ProjectID: "test-project",
			SourceURL: "https://example.com/doc",
			Title:     "Test Doc",
			Content:   "Test content",
		}

		err := w.CreateDocument(context.Background(), doc)

		require.NoError(t, err)
		assert.Equal(t, doc, calledWith)
	})
}
