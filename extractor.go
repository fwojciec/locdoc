package locdoc

// ExtractResult holds the extracted content from an HTML page.
type ExtractResult struct {
	// Title is the page title extracted from metadata.
	Title string

	// ContentHTML is the main content as clean HTML.
	// Boilerplate (nav, footer, sidebar, ads) has been removed.
	ContentHTML string
}

// Extractor extracts main content from HTML pages, removing boilerplate.
type Extractor interface {
	// Extract processes raw HTML and returns the main content.
	// The title comes from page metadata (meta tags, JSON+LD, etc.).
	// The content HTML has boilerplate removed but preserves structure.
	Extract(html string) (*ExtractResult, error)
}
