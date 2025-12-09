package locdoc

import "context"

// Fetcher retrieves rendered HTML from URLs.
// Implementations may use browser automation to handle JavaScript-rendered content.
type Fetcher interface {
	// Fetch navigates to the URL, waits for JavaScript to render,
	// and returns the rendered HTML.
	// The context controls timeout and cancellation.
	Fetch(ctx context.Context, url string) (html string, err error)

	// Close releases browser resources.
	// Must be called when the Fetcher is no longer needed.
	Close() error
}
