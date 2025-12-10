package sqlite

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/fwojciec/locdoc"
	"github.com/google/uuid"
)

// Compile-time interface verification.
var _ locdoc.DocumentService = (*DocumentService)(nil)

// DocumentService implements locdoc.DocumentService using SQLite.
type DocumentService struct {
	db *DB
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(db *DB) *DocumentService {
	return &DocumentService{db: db}
}

// hashContent computes xxHash of content and returns hex string.
func hashContent(content string) string {
	h := xxhash.Sum64String(content)
	b := make([]byte, 8)
	b[0] = byte(h >> 56)
	b[1] = byte(h >> 48)
	b[2] = byte(h >> 40)
	b[3] = byte(h >> 32)
	b[4] = byte(h >> 24)
	b[5] = byte(h >> 16)
	b[6] = byte(h >> 8)
	b[7] = byte(h)
	return hex.EncodeToString(b)
}

// CreateDocument creates a new document.
func (s *DocumentService) CreateDocument(ctx context.Context, doc *locdoc.Document) error {
	if err := doc.Validate(); err != nil {
		return err
	}

	doc.ID = uuid.New().String()
	doc.FetchedAt = time.Now().UTC()
	doc.ContentHash = hashContent(doc.Content)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO documents (id, project_id, file_path, source_url, title, content, content_hash, position, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, doc.ID, doc.ProjectID, doc.FilePath, doc.SourceURL, doc.Title, doc.Content, doc.ContentHash,
		doc.Position, doc.FetchedAt.Format(time.RFC3339))

	return err
}

// FindDocumentByID retrieves a document by ID.
func (s *DocumentService) FindDocumentByID(ctx context.Context, id string) (*locdoc.Document, error) {
	var doc locdoc.Document
	var fetchedAt string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, file_path, source_url, title, content, content_hash, position, fetched_at
		FROM documents
		WHERE id = ?
	`, id).Scan(&doc.ID, &doc.ProjectID, &doc.FilePath, &doc.SourceURL, &doc.Title,
		&doc.Content, &doc.ContentHash, &doc.Position, &fetchedAt)

	if err == sql.ErrNoRows {
		return nil, locdoc.Errorf(locdoc.ENOTFOUND, "document not found")
	}
	if err != nil {
		return nil, err
	}

	var parseErr error
	doc.FetchedAt, parseErr = time.Parse(time.RFC3339, fetchedAt)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse fetched_at: %w", parseErr)
	}

	return &doc, nil
}

// FindDocuments retrieves documents matching the filter.
func (s *DocumentService) FindDocuments(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
	var query strings.Builder
	var args []any

	query.WriteString("SELECT id, project_id, file_path, source_url, title, content, content_hash, position, fetched_at FROM documents WHERE 1=1")

	if filter.ID != nil {
		query.WriteString(" AND id = ?")
		args = append(args, *filter.ID)
	}
	if filter.ProjectID != nil {
		query.WriteString(" AND project_id = ?")
		args = append(args, *filter.ProjectID)
	}
	if filter.SourceURL != nil {
		query.WriteString(" AND source_url = ?")
		args = append(args, *filter.SourceURL)
	}

	switch filter.SortBy {
	case "position":
		query.WriteString(" ORDER BY position ASC")
	default:
		query.WriteString(" ORDER BY fetched_at DESC")
	}

	if filter.Limit > 0 {
		query.WriteString(" LIMIT ?")
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query.WriteString(" OFFSET ?")
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*locdoc.Document
	for rows.Next() {
		var doc locdoc.Document
		var fetchedAt string

		if err := rows.Scan(&doc.ID, &doc.ProjectID, &doc.FilePath, &doc.SourceURL, &doc.Title,
			&doc.Content, &doc.ContentHash, &doc.Position, &fetchedAt); err != nil {
			return nil, err
		}

		var parseErr error
		doc.FetchedAt, parseErr = time.Parse(time.RFC3339, fetchedAt)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse fetched_at: %w", parseErr)
		}

		docs = append(docs, &doc)
	}

	return docs, rows.Err()
}

// UpdateDocument updates an existing document.
func (s *DocumentService) UpdateDocument(ctx context.Context, id string, upd locdoc.DocumentUpdate) (*locdoc.Document, error) {
	// First check if document exists
	doc, err := s.FindDocumentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if upd.Title != nil {
		doc.Title = *upd.Title
	}
	if upd.Content != nil {
		doc.Content = *upd.Content
		doc.ContentHash = hashContent(doc.Content)
	} else if upd.ContentHash != nil {
		// Only allow explicit hash override if content wasn't updated
		doc.ContentHash = *upd.ContentHash
	}
	if upd.Position != nil {
		doc.Position = *upd.Position
	}

	// Validate before persisting (defense-in-depth)
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE documents
		SET title = ?, content = ?, content_hash = ?, position = ?
		WHERE id = ?
	`, doc.Title, doc.Content, doc.ContentHash, doc.Position, id)

	if err != nil {
		return nil, err
	}

	return doc, nil
}

// DeleteDocument permanently removes a document.
func (s *DocumentService) DeleteDocument(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM documents WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return locdoc.Errorf(locdoc.ENOTFOUND, "document not found")
	}

	return nil
}

// DeleteDocumentsByProject removes all documents for a project.
func (s *DocumentService) DeleteDocumentsByProject(ctx context.Context, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM documents WHERE project_id = ?", projectID)
	return err
}
