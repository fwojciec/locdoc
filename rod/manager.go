package rod

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// DefaultMaxPages is the default number of pages before browser recycling.
const DefaultMaxPages = 75

// BrowserManager manages browser lifecycle with automatic recycling to prevent
// memory accumulation. Chrome accumulates memory over time (~0.5MB/s under load),
// and the baseline never returns to initial levels even with proper page cleanup.
// Recycling the browser periodically addresses this issue.
//
// BrowserManager is safe for concurrent use.
type BrowserManager struct {
	browser   *rod.Browser
	launcher  *launcher.Launcher
	pageCount int64
	maxPages  int64
	mu        sync.Mutex
	closed    atomic.Bool
}

// ManagerOption configures a BrowserManager.
type ManagerOption func(*BrowserManager)

// WithMaxPages sets the maximum number of pages before the browser is recycled.
// Defaults to 75 if not specified.
func WithMaxPages(n int64) ManagerOption {
	return func(bm *BrowserManager) {
		bm.maxPages = n
	}
}

// NewBrowserManager creates a new BrowserManager that launches a headless Chrome browser.
// The browser will be recycled after maxPages (default 75) pages have been processed.
// Close must be called when the BrowserManager is no longer needed.
func NewBrowserManager(opts ...ManagerOption) (*BrowserManager, error) {
	bm := &BrowserManager{
		maxPages: DefaultMaxPages,
	}
	for _, opt := range opts {
		opt(bm)
	}

	if err := bm.launchBrowser(); err != nil {
		return nil, err
	}

	return bm, nil
}

// Browser returns the current browser instance, recycling if the page count
// has reached maxPages. Callers should call IncrementPageCount after using
// the browser to process a page.
func (bm *BrowserManager) Browser() *rod.Browser {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if atomic.LoadInt64(&bm.pageCount) >= bm.maxPages {
		bm.recycleBrowser()
	}

	return bm.browser
}

// IncrementPageCount increments the page counter. Call this after successfully
// processing a page to track progress toward the recycling threshold.
func (bm *BrowserManager) IncrementPageCount() {
	atomic.AddInt64(&bm.pageCount, 1)
}

// Close releases browser resources. Close is safe to call multiple times.
func (bm *BrowserManager) Close() error {
	if !bm.closed.CompareAndSwap(false, true) {
		return nil
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	return bm.closeBrowser()
}

// launchBrowser starts a new browser instance with stability flags.
func (bm *BrowserManager) launchBrowser() error {
	lnchr := launcher.New().
		Set("disable-background-timer-throttling").
		Set("disable-backgrounding-occluded-windows").
		Set("disable-renderer-backgrounding").
		Set("disable-dev-shm-usage").
		Set("disable-hang-monitor").
		Leakless(true).
		Headless(true)

	u, err := lnchr.Launch()
	if err != nil {
		return fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		lnchr.Kill()
		return fmt.Errorf("connecting to browser: %w", err)
	}

	bm.browser = browser
	bm.launcher = lnchr
	return nil
}

// closeBrowser shuts down the current browser and launcher.
// Must be called with mu held.
func (bm *BrowserManager) closeBrowser() error {
	var err error
	if bm.browser != nil {
		err = bm.browser.Close()
		bm.browser = nil
	}
	if bm.launcher != nil {
		bm.launcher.Kill()
		bm.launcher = nil
	}
	return err
}

// recycleBrowser starts a fresh browser and closes the old one.
// If launching the new browser fails, the old browser is kept.
// Must be called with mu held.
func (bm *BrowserManager) recycleBrowser() {
	// Save old instances in case new launch fails
	oldBrowser := bm.browser
	oldLauncher := bm.launcher
	bm.browser = nil
	bm.launcher = nil

	// Try to launch new browser
	if err := bm.launchBrowser(); err != nil {
		// Restore old instances if new launch fails
		bm.browser = oldBrowser
		bm.launcher = oldLauncher
		return
	}

	// Successfully launched new browser, clean up old one
	if oldBrowser != nil {
		_ = oldBrowser.Close()
	}
	if oldLauncher != nil {
		oldLauncher.Kill()
	}
	atomic.StoreInt64(&bm.pageCount, 0)
}

// LauncherPID returns the process ID of the browser launcher.
// This method exists for testing purposes to verify proper cleanup.
func (bm *BrowserManager) LauncherPID() int {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	if bm.launcher == nil {
		return 0
	}
	return bm.launcher.PID()
}
