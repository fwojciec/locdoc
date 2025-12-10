package locdoc

import "strings"

// FormatDocuments formats documents for display or LLM context.
// Uses title if available, falls back to source URL.
// Documents are separated by blank lines.
func FormatDocuments(docs []*Document) string {
	if len(docs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(docs))
	for _, doc := range docs {
		header := doc.Title
		if header == "" {
			header = doc.SourceURL
		}
		parts = append(parts, "## Document: "+header+"\n"+doc.Content)
	}

	return strings.Join(parts, "\n\n")
}
