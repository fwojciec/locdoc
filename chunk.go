package locdoc

import (
	"context"
)

// Chunk represents a section of a document optimized for embedding and retrieval.
type Chunk struct {
	ID         string        `json:"id"`
	DocumentID string        `json:"documentId"`
	ProjectID  string        `json:"projectId"` // Denormalized for efficient filtering
	Content    string        `json:"content"`
	Embedding  []float32     `json:"embedding,omitempty"`
	Metadata   ChunkMetadata `json:"metadata"`
}

// ChunkMetadata contains contextual information about a chunk.
type ChunkMetadata struct {
	// Header hierarchy from the document (e.g., {"h1": "API", "h2": "Auth"})
	Headers map[string]string `json:"headers,omitempty"`

	// Position in the original document
	StartLine int `json:"startLine,omitempty"`
	EndLine   int `json:"endLine,omitempty"`

	// Source URL for citation
	SourceURL string `json:"sourceUrl,omitempty"`
}

// Validate returns an error if the chunk contains invalid fields.
func (c *Chunk) Validate() error {
	if c.DocumentID == "" {
		return Errorf(EINVALID, "chunk document ID required")
	}
	if c.ProjectID == "" {
		return Errorf(EINVALID, "chunk project ID required")
	}
	if c.Content == "" {
		return Errorf(EINVALID, "chunk content required")
	}
	return nil
}

// ChunkService represents a service for managing chunks.
type ChunkService interface {
	// CreateChunk creates a new chunk.
	CreateChunk(ctx context.Context, chunk *Chunk) error

	// CreateChunks creates multiple chunks in a batch.
	CreateChunks(ctx context.Context, chunks []*Chunk) error

	// FindChunkByID retrieves a chunk by ID.
	// Returns ENOTFOUND if chunk does not exist.
	FindChunkByID(ctx context.Context, id string) (*Chunk, error)

	// FindChunks retrieves chunks matching the filter.
	FindChunks(ctx context.Context, filter ChunkFilter) ([]*Chunk, error)

	// DeleteChunk permanently removes a chunk.
	// Returns ENOTFOUND if chunk does not exist.
	DeleteChunk(ctx context.Context, id string) error

	// DeleteChunksByDocument removes all chunks for a document.
	DeleteChunksByDocument(ctx context.Context, documentID string) error

	// DeleteChunksByProject removes all chunks for a project.
	DeleteChunksByProject(ctx context.Context, projectID string) error
}

// ChunkFilter represents a filter for FindChunks.
type ChunkFilter struct {
	ID         *string `json:"id"`
	DocumentID *string `json:"documentId"`
	ProjectID  *string `json:"projectId"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// SearchService provides semantic search over chunks.
type SearchService interface {
	// Search performs semantic search over chunks.
	// Returns chunks ordered by relevance to the query.
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	// Filter results to specific project(s)
	ProjectIDs []string `json:"projectIds,omitempty"`

	// Maximum number of results to return
	Limit int `json:"limit,omitempty"`

	// Minimum similarity score (0-1)
	MinScore float32 `json:"minScore,omitempty"`
}

// SearchResult represents a search match.
type SearchResult struct {
	Chunk *Chunk  `json:"chunk"`
	Score float32 `json:"score"`
}
