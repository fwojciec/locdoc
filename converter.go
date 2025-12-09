package locdoc

// Converter converts HTML to Markdown.
type Converter interface {
	// Convert transforms HTML content into Markdown.
	// The input should be clean HTML (e.g., from an Extractor).
	// Returns the Markdown representation of the content.
	Convert(html string) (string, error)
}
