package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.DocumentService = (*DocumentService)(nil)

// DocumentService is a mock implementation of locdoc.DocumentService.
type DocumentService struct {
	CreateDocumentFn           func(ctx context.Context, doc *locdoc.Document) error
	FindDocumentByIDFn         func(ctx context.Context, id string) (*locdoc.Document, error)
	FindDocumentsFn            func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error)
	UpdateDocumentFn           func(ctx context.Context, id string, upd locdoc.DocumentUpdate) (*locdoc.Document, error)
	DeleteDocumentFn           func(ctx context.Context, id string) error
	DeleteDocumentsByProjectFn func(ctx context.Context, projectID string) error
}

func (s *DocumentService) CreateDocument(ctx context.Context, doc *locdoc.Document) error {
	return s.CreateDocumentFn(ctx, doc)
}

func (s *DocumentService) FindDocumentByID(ctx context.Context, id string) (*locdoc.Document, error) {
	return s.FindDocumentByIDFn(ctx, id)
}

func (s *DocumentService) FindDocuments(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
	return s.FindDocumentsFn(ctx, filter)
}

func (s *DocumentService) UpdateDocument(ctx context.Context, id string, upd locdoc.DocumentUpdate) (*locdoc.Document, error) {
	return s.UpdateDocumentFn(ctx, id, upd)
}

func (s *DocumentService) DeleteDocument(ctx context.Context, id string) error {
	return s.DeleteDocumentFn(ctx, id)
}

func (s *DocumentService) DeleteDocumentsByProject(ctx context.Context, projectID string) error {
	return s.DeleteDocumentsByProjectFn(ctx, projectID)
}
