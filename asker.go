package locdoc

import "context"

// Asker provides natural language question answering over documentation.
type Asker interface {
	Ask(ctx context.Context, projectID string, question string) (string, error)
}
