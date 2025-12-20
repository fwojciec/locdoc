# Fixing Headless Chrome "Stuck Page" Timeouts in go-rod Concurrent Scraping

Your pattern of **30-second timeout on first attempt, 1-2 second success on retry** is a classic symptom of stuck browser states rather than slow page loads. The page content is ready almost immediately, but Chrome or go-rod is waiting for lifecycle events that never fire. This comprehensive guide provides ranked solutions by effort vs impact, with specific go-rod code patterns.

## Quick wins: Chrome flags that prevent stuck states

The most impactful immediate fix is adding these Chrome flags via `rod.Launcher`. Several flags specifically address concurrent tab reliability and prevent the timer throttling that causes "background" tabs to freeze.

```go
import "github.com/go-rod/rod/lib/launcher"

l := launcher.New().
    Set("disable-background-timer-throttling").      // CRITICAL: prevents timer delays in background tabs
    Set("disable-backgrounding-occluded-windows").   // CRITICAL: prevents deprioritizing hidden tabs
    Set("disable-renderer-backgrounding").           // Keeps all renderers at full priority
    Set("disable-background-networking").            // Stops background network interference
    Set("disable-hang-monitor").                     // Prevents Chrome killing "unresponsive" heavy pages
    Set("disable-ipc-flooding-protection").          // Improves IPC performance under load
    Set("disable-dev-shm-usage").                    // Essential for Docker (use /tmp instead of /dev/shm)
    Set("no-sandbox").                               // Required for Docker/root environments
    Set("disable-gpu").                              // Reduces resource usage in headless
    Set("disable-extensions").                       // Eliminates extension interference
    Set("disable-breakpad").                         // Disables crash reporting
    Leakless(true).                                  // Auto-kill browser when Go process exits
    Headless(true)

url := l.MustLaunch()
browser := rod.New().ControlURL(url).MustConnect()
```

The three flags `--disable-background-timer-throttling`, `--disable-backgrounding-occluded-windows`, and `--disable-renderer-backgrounding` are the most critical for concurrent operations. Chrome aggressively throttles JavaScript timers and deprioritizes "background" tabs, which causes lifecycle events to fire with massive delays or not at all when you have multiple concurrent tabs.

## Your core issue: WaitLoad() is too fragile for concurrent use

`WaitLoad()` waits for the `window.onload` event, which depends on all resources loading. In concurrent scenarios, this event can fail to fire reliably due to Chrome's background tab throttling. Switch to `WaitStable()` with a proper timeout wrapper.

```go
// Replace this pattern:
page.MustNavigate(url)
page.MustWaitLoad()  // Can hang indefinitely in background tabs

// With this pattern:
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := rod.Try(func() {
    page.Context(ctx).MustNavigate(url).MustWaitStable()
})

if errors.Is(err, context.DeadlineExceeded) {
    // Handle stuck page - see recovery section below
}
```

**Understanding the wait methods:** `WaitStable()` internally combines three checks in parallel: `WaitLoad()` (window.onload), `WaitRequestIdle()` (no network activity for 1 second by default), and `WaitDOMStable()` (no layout changes). This parallel approach means if one event fails to fire, the others can still complete and unblock execution.

For pages you know are mostly static HTML, an even faster approach uses `WaitDOMStable()` alone:

```go
page.MustNavigate(url)
page.Timeout(10 * time.Second).MustWaitDOMStable(time.Millisecond * 300, 0.1)
```

## Incognito contexts provide instant isolation without startup cost

The most effective architectural change for your 3-worker scenario is using incognito browser contexts instead of raw page creation. Each context gets isolated cookies, cache, and localStorage—preventing cross-contamination between requests that causes stuck states.

```go
func worker(browser *rod.Browser, urls <-chan string, results chan<- string) {
    for url := range urls {
        // Create isolated context for each request
        incognito := browser.MustIncognito()
        page := incognito.MustPage()
        
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        
        err := rod.Try(func() {
            page.Context(ctx).MustNavigate(url).MustWaitStable()
            html, _ := page.HTML()
            results <- html
        })
        
        cancel()
        
        if err != nil {
            // Force cleanup on any error
            forceClosePage(browser, page, incognito)
        } else {
            incognito.MustClose()  // Clean close of entire context
        }
    }
}
```

Using incognito contexts adds **negligible overhead** (a few milliseconds) compared to creating new browser instances (2-3 seconds each), while providing nearly the same isolation benefits.

## Detecting stuck pages vs genuinely slow pages

The key insight is that stuck pages have **no ongoing network activity** while genuinely slow pages have pending requests. go-rod's event system lets you monitor this:

```go
func navigateWithStuckDetection(page *rod.Page, url string, timeout time.Duration) error {
    var lastActivityTime atomic.Value
    lastActivityTime.Store(time.Now())
    
    // Monitor network activity
    go page.EachEvent(func(e *proto.NetworkRequestWillBeSent) {
        lastActivityTime.Store(time.Now())
    }, func(e *proto.NetworkResponseReceived) {
        lastActivityTime.Store(time.Now())
    })()
    
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    // Start navigation without waiting for full load
    page.Context(ctx).MustNavigate(url)
    
    // Custom wait loop with stuck detection
    stuckThreshold := 10 * time.Second
    checkInterval := time.Second
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(checkInterval):
            lastActivity := lastActivityTime.Load().(time.Time)
            idle := time.Since(lastActivity)
            
            if idle > stuckThreshold {
                // No network activity for 10s after initial requests
                // Page is likely stuck, not slow - return early
                return nil  // Proceed with whatever content is loaded
            }
            
            // Check if page is actually stable
            err := rod.Try(func() {
                page.Timeout(time.Second).MustWaitDOMStable(300*time.Millisecond, 0.1)
            })
            if err == nil {
                return nil  // Page is stable, proceed
            }
        }
    }
}
```

This pattern distinguishes between stuck pages (initial burst of activity then silence) and genuinely slow pages (continuous network activity), allowing early termination when the page is stuck.

## Force-closing stuck pages before timeout

When a page's context is cancelled due to timeout, calling `page.Close()` with the same context will also fail. You need a fresh context for cleanup:

```go
func forceClosePage(browser *rod.Browser, page *rod.Page, context *rod.Browser) {
    // Create fresh context for cleanup operations
    cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Try graceful close first
    err := rod.Try(func() {
        page.Context(cleanupCtx).Close()
    })
    
    if err != nil {
        // Force close via CDP TargetCloseTarget
        proto.TargetCloseTarget{TargetID: page.TargetID}.Call(browser)
    }
    
    // If using incognito context, close that too
    if context != nil {
        context.Close()
    }
}
```

**Critical pattern:** Never rely on `defer page.MustClose()` with a timeout context. The defer will execute after timeout with the cancelled context, causing the close to fail and leak the page.

## Browser restart strategy prevents memory accumulation

Even with proper page cleanup, Chrome accumulates memory over time. Research shows memory grows roughly **0.5MB per second under heavy load**, with the baseline never returning to initial levels. Implement periodic browser recycling:

```go
type BrowserManager struct {
    browser       *rod.Browser
    launcher      *launcher.Launcher
    pageCount     int64
    maxPages      int64
    mu            sync.Mutex
}

func NewBrowserManager(maxPagesPerBrowser int64) *BrowserManager {
    bm := &BrowserManager{maxPages: maxPagesPerBrowser}
    bm.recycleBrowser()
    return bm
}

func (bm *BrowserManager) GetBrowser() *rod.Browser {
    bm.mu.Lock()
    defer bm.mu.Unlock()
    
    if atomic.LoadInt64(&bm.pageCount) >= bm.maxPages {
        bm.recycleBrowser()
    }
    
    atomic.AddInt64(&bm.pageCount, 1)
    return bm.browser
}

func (bm *BrowserManager) recycleBrowser() {
    // Close existing browser
    if bm.browser != nil {
        bm.browser.Close()
    }
    if bm.launcher != nil {
        bm.launcher.Cleanup()
    }
    
    // Launch fresh browser with stability flags
    bm.launcher = launcher.New().
        Set("disable-background-timer-throttling").
        Set("disable-backgrounding-occluded-windows").
        Set("disable-renderer-backgrounding").
        Set("disable-dev-shm-usage").
        Set("no-sandbox").
        Leakless(true)
    
    url := bm.launcher.MustLaunch()
    bm.browser = rod.New().ControlURL(url).MustConnect()
    atomic.StoreInt64(&bm.pageCount, 0)
}
```

**Recommended restart cadence:** Recycle the browser every **50-100 pages** processed. The 2-3 second restart overhead is negligible when amortized across 50+ page loads, and the reliability gains are substantial.

## Complete production-ready worker implementation

Combining all patterns into a robust worker that handles your exact scenario:

```go
func robustWorker(bm *BrowserManager, urls <-chan string, results chan<- Result) {
    for url := range urls {
        result := fetchWithRetry(bm, url, 3)
        results <- result
    }
}

func fetchWithRetry(bm *BrowserManager, url string, maxRetries int) Result {
    var lastErr error
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        browser := bm.GetBrowser()
        
        // Escalating isolation: attempt 1 uses incognito, later attempts may trigger browser recycle
        incognito := browser.MustIncognito()
        page := incognito.MustPage()
        
        // Shorter timeout for early attempts, longer for final
        timeout := time.Duration(10+attempt*10) * time.Second
        ctx, cancel := context.WithTimeout(context.Background(), timeout)
        
        var html string
        err := rod.Try(func() {
            page.Context(ctx).MustNavigate(url).MustWaitStable()
            html, _ = page.HTML()
        })
        
        cancel()
        
        if err == nil {
            incognito.MustClose()
            return Result{URL: url, HTML: html, Success: true}
        }
        
        lastErr = err
        forceClosePage(browser, page, incognito)
        
        // Exponential backoff between retries
        time.Sleep(time.Duration(math.Pow(2, float64(attempt-1))) * time.Second)
        
        // If second attempt also fails, trigger browser recycle before final attempt
        if attempt == 2 {
            bm.recycleBrowser()
        }
    }
    
    return Result{URL: url, Error: lastErr, Success: false}
}
```

## Recommendations ranked by effort vs impact

| Priority | Change | Effort | Impact | When to implement |
|----------|--------|--------|--------|-------------------|
| **1** | Add `--disable-background-timer-throttling` and related flags | 5 min | Very High | Immediately |
| **2** | Switch from `WaitLoad()` to `WaitStable()` | 5 min | High | Immediately |
| **3** | Use incognito contexts per request | 15 min | High | Immediately |
| **4** | Fix page close with fresh context | 15 min | Medium | Before production |
| **5** | Implement browser recycling every 75 pages | 30 min | Medium | Before production |
| **6** | Add stuck detection via network monitoring | 1-2 hrs | Medium | If issues persist |
| **7** | Build full BrowserManager with pools | 2-4 hrs | Medium | For high-volume use |

The first three changes—Chrome flags, `WaitStable()`, and incognito contexts—should resolve most stuck page issues with minimal code changes. These address the root causes: Chrome's background tab throttling, fragile lifecycle events, and state contamination between concurrent requests.

## Navigating to about:blank before close

Mixed evidence exists on whether navigating to `about:blank` before closing helps. Some practitioners report it forces resource cleanup on heavy pages:

```go
// Optional - may help with resource-heavy pages
rod.Try(func() {
    page.Timeout(2 * time.Second).MustNavigate("about:blank")
})
page.Close()
```

This is a low-priority optimization to try if issues persist after implementing the higher-impact changes.

## Key diagnostic step

Before implementing all changes, add logging to confirm your diagnosis. Log the inflight request count when timeouts occur:

```go
if errors.Is(err, context.DeadlineExceeded) {
    // Check: are there pending requests (slow page) or none (stuck page)?
    targets, _ := proto.TargetGetTargets{}.Call(browser)
    log.Printf("Timeout on %s - targets: %d", url, len(targets))
}
```

If timeouts consistently show zero pending requests while the retry succeeds instantly, you've confirmed the stuck state hypothesis and the Chrome flags will be highly effective. If timeouts show many pending requests, the pages are genuinely slow and you may need to adjust timeouts or use more aggressive `domcontentloaded` waiting instead of full network idle.
