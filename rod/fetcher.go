package rod

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// DefaultFetchTimeout is the default timeout for page navigation and loading.
const DefaultFetchTimeout = 30 * time.Second

// Ensure Fetcher implements locdoc.Fetcher at compile time.
var _ locdoc.Fetcher = (*Fetcher)(nil)

// Fetcher retrieves rendered HTML from URLs using Chrome browser automation.
// The browser is automatically recycled after processing a configurable number
// of pages (default 75) to prevent memory accumulation.
// Fetcher is safe for concurrent use by multiple goroutines.
type Fetcher struct {
	manager      *BrowserManager
	fetchTimeout time.Duration
	maxPages     int64
	closed       atomic.Bool
	closeOnce    sync.Once
	closeErr     error
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

// WithRecycleAfter sets the number of pages after which the browser is recycled.
// Defaults to 75 if not specified. Chrome accumulates memory over time, and
// recycling the browser periodically prevents unbounded memory growth.
func WithRecycleAfter(n int64) Option {
	return func(f *Fetcher) {
		f.maxPages = n
	}
}

// NewFetcher creates a new Fetcher that launches a headless Chrome browser.
// The browser is automatically recycled after processing maxPages (default 75)
// to prevent memory accumulation.
// Close must be called when the Fetcher is no longer needed.
//
// Returns an error if Chrome/Chromium cannot be found or launched.
func NewFetcher(opts ...Option) (*Fetcher, error) {
	f := &Fetcher{
		fetchTimeout: DefaultFetchTimeout,
		maxPages:     DefaultMaxPages,
	}
	for _, opt := range opts {
		opt(f)
	}

	manager, err := NewBrowserManager(WithMaxPages(f.maxPages))
	if err != nil {
		return nil, err
	}
	f.manager = manager

	return f, nil
}

// Fetch navigates to the URL and returns the rendered HTML.
func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error) {
	// Check if fetcher is closed
	if f.closed.Load() {
		return "", locdoc.Errorf(locdoc.EINVALID, "fetcher is closed")
	}

	// Check context before starting
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// Get browser from manager (may trigger recycling if page limit reached)
	browser := f.manager.Browser()

	// Use incognito context for isolation. Each fetch gets isolated cookies, cache,
	// and localStorage, preventing cross-contamination between concurrent requests.
	incognito, err := browser.Incognito()
	if err != nil {
		return "", err
	}

	page, err := incognito.Page(proto.TargetCreateTarget{})
	if err != nil {
		_ = incognito.Close()
		return "", err
	}

	// Create timeout context for entire fetch operation (navigate + wait + HTML)
	fetchCtx, cancel := context.WithTimeout(ctx, f.fetchTimeout)
	defer cancel()

	// Set context for all subsequent operations
	page = page.Context(fetchCtx)

	// Navigate to URL
	if err := page.Navigate(url); err != nil {
		f.closePageAndContext(page, incognito)
		return "", err
	}

	// Wait for page to load. We use WaitLoad instead of WaitStable because WaitStable
	// requires the DOM to be unchanged for the specified duration, which never happens
	// on React/JS-heavy sites with continuous animations or state updates.
	if err := page.WaitLoad(); err != nil {
		f.closePageAndContext(page, incognito)
		return "", err
	}

	// Get rendered HTML
	html, err := page.HTML()
	if err != nil {
		f.closePageAndContext(page, incognito)
		return "", err
	}

	// Clean close of entire incognito context (error intentionally ignored)
	_ = incognito.Close()

	// Track page count for browser recycling
	f.manager.IncrementPageCount()

	return html, nil
}

// closePageAndContext closes a page and its incognito context using a fresh context.
// When a page's context is cancelled due to timeout, page.Close() with that context
// will also fail. This method uses a fresh context for cleanup operations.
// Errors are intentionally ignored as this is best-effort cleanup during error recovery.
func (f *Fetcher) closePageAndContext(page *rod.Page, incognito *rod.Browser) {
	// Create fresh context for cleanup operations since the page's context may be cancelled
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = page.Context(cleanupCtx).Close()
	_ = incognito.Close()
}

// Close releases browser resources. Close is safe to call multiple times.
func (f *Fetcher) Close() error {
	f.closeOnce.Do(func() {
		f.closed.Store(true)
		f.closeErr = f.manager.Close()
	})
	return f.closeErr
}

// LauncherPID returns the process ID of the browser launcher.
// This method exists for testing purposes to verify proper cleanup.
func (f *Fetcher) LauncherPID() int {
	return f.manager.LauncherPID()
}
