package locdoc

import "context"

// Asker provides natural language question answering over documentation.
type Asker interface {
	// Ask answers a natural language question about a project's documentation.
	// Returns ENOTFOUND if the project does not exist.
	Ask(ctx context.Context, projectID string, question string) (string, error)
}
