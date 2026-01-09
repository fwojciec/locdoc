package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.DocumentWriter = (*DocumentWriter)(nil)

// DocumentWriter is a mock implementation of locdoc.DocumentWriter.
type DocumentWriter struct {
	CreateDocumentFn func(ctx context.Context, doc *locdoc.Document) error
}

func (w *DocumentWriter) CreateDocument(ctx context.Context, doc *locdoc.Document) error {
	return w.CreateDocumentFn(ctx, doc)
}
