package rod

import (
	"context"
	"fmt"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// DefaultFetchTimeout is the default timeout for page navigation and loading.
const DefaultFetchTimeout = 30 * time.Second

// Ensure Fetcher implements locdoc.Fetcher at compile time.
var _ locdoc.Fetcher = (*Fetcher)(nil)

// Fetcher retrieves rendered HTML from URLs using Chrome browser automation.
// Fetcher is safe for concurrent use by multiple goroutines.
type Fetcher struct {
	browser      *rod.Browser
	launcher     *launcher.Launcher
	fetchTimeout time.Duration
}

// Option configures a Fetcher.
type Option func(*Fetcher)

// WithFetchTimeout sets the timeout for page navigation and loading.
// Defaults to 30 seconds if not specified.
func WithFetchTimeout(d time.Duration) Option {
	return func(f *Fetcher) {
		f.fetchTimeout = d
	}
}

// NewFetcher creates a new Fetcher that launches a headless Chrome browser.
// Close must be called when the Fetcher is no longer needed.
//
// Returns an error if Chrome/Chromium cannot be found or launched.
func NewFetcher(opts ...Option) (*Fetcher, error) {
	// Launch browser using rod's launcher (finds or downloads Chrome)
	lnchr := launcher.New().Headless(true)
	u, err := lnchr.Launch()
	if err != nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		lnchr.Kill() // Clean up launched process on connection failure
		return nil, fmt.Errorf("connecting to browser: %w", err)
	}

	f := &Fetcher{
		browser:      browser,
		launcher:     lnchr,
		fetchTimeout: DefaultFetchTimeout,
	}
	for _, opt := range opts {
		opt(f)
	}

	return f, nil
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

	// Create timeout context for entire fetch operation (navigate + wait + HTML)
	fetchCtx, cancel := context.WithTimeout(ctx, f.fetchTimeout)
	defer cancel()

	// Set context for all subsequent operations
	page = page.Context(fetchCtx)

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
	err := f.browser.Close()
	f.launcher.Kill()
	return err
}

// LauncherPID returns the process ID of the browser launcher.
// This method exists for testing purposes to verify proper cleanup.
func (f *Fetcher) LauncherPID() int {
	return f.launcher.PID()
}
