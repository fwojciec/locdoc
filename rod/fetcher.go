package rod

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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

// NewFetcher creates a new Fetcher that launches a headless Chrome browser.
// Close must be called when the Fetcher is no longer needed.
//
// Returns an error if Chrome/Chromium cannot be found or launched.
func NewFetcher(opts ...Option) (*Fetcher, error) {
	// Launch browser using rod's launcher (finds or downloads Chrome).
	// Chrome aggressively throttles "background" tabs during concurrent operations,
	// causing lifecycle events to fire with massive delays or not at all.
	// See docs/go-rod-reliability.md for full context.
	lnchr := launcher.New().
		Set("disable-background-timer-throttling").    // Prevents timer delays in background tabs
		Set("disable-backgrounding-occluded-windows"). // Prevents deprioritizing hidden tabs
		Set("disable-renderer-backgrounding").         // Keeps all renderers at full priority
		Set("disable-dev-shm-usage").                  // Uses /tmp instead of /dev/shm (essential for Docker)
		Set("disable-hang-monitor").                   // Prevents killing "unresponsive" heavy pages
		Leakless(true).                                // Auto-kill browser when Go process exits
		Headless(true)
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
	// Check if fetcher is closed
	if f.closed.Load() {
		return "", locdoc.Errorf(locdoc.EINVALID, "fetcher is closed")
	}

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

// Close releases browser resources. Close is safe to call multiple times.
func (f *Fetcher) Close() error {
	f.closeOnce.Do(func() {
		f.closed.Store(true)
		f.closeErr = f.browser.Close()
		f.launcher.Kill()
	})
	return f.closeErr
}

// LauncherPID returns the process ID of the browser launcher.
// This method exists for testing purposes to verify proper cleanup.
func (f *Fetcher) LauncherPID() int {
	return f.launcher.PID()
}
