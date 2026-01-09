package locdoc

import (
	"context"
	"time"
)

// Document represents a crawled documentation page.
type Document struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	FilePath    string    `json:"filePath"`
	SourceURL   string    `json:"sourceUrl"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ContentHash string    `json:"contentHash"`
	Position    int       `json:"position"`
	FetchedAt   time.Time `json:"fetchedAt"`
}

// Validate returns an error if the document contains invalid fields.
func (d *Document) Validate() error {
	if d.ProjectID == "" {
		return Errorf(EINVALID, "document project ID required")
	}
	if d.SourceURL == "" {
		return Errorf(EINVALID, "document source URL required")
	}
	return nil
}

// DocumentWriter writes documents to storage.
type DocumentWriter interface {
	CreateDocument(ctx context.Context, doc *Document) error
}

// DocumentService represents a service for managing documents.
type DocumentService interface {
	// CreateDocument creates a new document.
	CreateDocument(ctx context.Context, doc *Document) error

	// FindDocumentByID retrieves a document by ID.
	// Returns ENOTFOUND if document does not exist.
	FindDocumentByID(ctx context.Context, id string) (*Document, error)

	// FindDocuments retrieves documents matching the filter.
	FindDocuments(ctx context.Context, filter DocumentFilter) ([]*Document, error)

	// DeleteDocument permanently removes a document and all associated chunks.
	// Returns ENOTFOUND if document does not exist.
	DeleteDocument(ctx context.Context, id string) error

	// DeleteDocumentsByProject removes all documents for a project.
	DeleteDocumentsByProject(ctx context.Context, projectID string) error
}

// SortOrder represents the sort order for document queries.
type SortOrder string

// SortOrder constants for DocumentFilter.
const (
	SortByFetchedAt SortOrder = "fetched_at"
	SortByPosition  SortOrder = "position"
)

// DocumentFilter represents a filter for FindDocuments.
type DocumentFilter struct {
	ID        *string `json:"id"`
	ProjectID *string `json:"projectId"`
	SourceURL *string `json:"sourceUrl"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`

	SortBy SortOrder `json:"sortBy"`
}
