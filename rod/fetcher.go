package rod

import (
	"context"
	"fmt"

	"github.com/fwojciec/locdoc"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Ensure Fetcher implements locdoc.Fetcher at compile time.
var _ locdoc.Fetcher = (*Fetcher)(nil)

// Fetcher retrieves rendered HTML from URLs using Chrome browser automation.
// Fetcher is safe for concurrent use by multiple goroutines.
type Fetcher struct {
	browser *rod.Browser
}

// NewFetcher creates a new Fetcher that launches a headless Chrome browser.
// Close must be called when the Fetcher is no longer needed.
//
// Returns an error if Chrome/Chromium cannot be found or launched.
func NewFetcher() (*Fetcher, error) {
	// Launch browser using rod's launcher (finds or downloads Chrome)
	l := launcher.New().Headless(true)
	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		l.Kill() // Clean up launched process on connection failure
		return nil, fmt.Errorf("connecting to browser: %w", err)
	}

	return &Fetcher{browser: browser}, nil
}

// Fetch navigates to the URL and returns the rendered HTML.
func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error) {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// Create a new page
	page, err := f.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return "", err
	}
	defer page.Close()

	// Set context for all subsequent operations
	page = page.Context(ctx)

	// Navigate to URL
	if err := page.Navigate(url); err != nil {
		return "", err
	}

	// Wait for page to load
	if err := page.WaitLoad(); err != nil {
		return "", err
	}

	// Get rendered HTML
	html, err := page.HTML()
	if err != nil {
		return "", err
	}

	return html, nil
}

// Close releases browser resources.
func (f *Fetcher) Close() error {
	return f.browser.Close()
}
