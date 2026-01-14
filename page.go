package locdoc

import "context"

// Page represents a fetched documentation page.
type Page struct {
	URL     string
	Title   string
	Content string // Markdown
}

// FetchProgress reports progress during page fetching.
type FetchProgress struct {
	URL       string
	Completed int
	Total     int
	Error     error
}

// FetchProgressFunc is called as pages are processed.
type FetchProgressFunc func(FetchProgress)

// URLSource discovers documentation URLs from a site.
// Implementations hide the complexity of sitemap vs recursive discovery.
type URLSource interface {
	Discover(ctx context.Context, sourceURL string) ([]string, error)
}

// PageFetcher retrieves and converts documentation pages.
// Implementations hide HTTP vs browser selection, retry logic,
// content extraction, and markdown conversion.
type PageFetcher interface {
	FetchAll(ctx context.Context, urls []string, progress FetchProgressFunc) ([]*Page, error)
}

// PageStore persists pages to storage with atomic semantics.
// Save writes to a temporary location; Commit makes changes permanent;
// Abort discards pending changes.
type PageStore interface {
	Save(ctx context.Context, page *Page) error
	Commit() error
	Abort() error
}
