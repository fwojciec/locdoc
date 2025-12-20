// Package crawl provides documentation crawling orchestration.
// It coordinates sitemap discovery, fetching, extraction, and storage
// of documentation pages.
package crawl

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/fwojciec/locdoc"
	"golang.org/x/sync/errgroup"
)

// Crawler orchestrates the crawling of documentation sites.
type Crawler struct {
	Sitemaps      locdoc.SitemapService
	Fetcher       locdoc.Fetcher
	Extractor     locdoc.Extractor
	Converter     locdoc.Converter
	Documents     locdoc.DocumentService
	TokenCounter  locdoc.TokenCounter
	LinkSelectors locdoc.LinkSelectorRegistry
	RateLimiter   locdoc.DomainLimiter
	Concurrency   int
	RetryDelays   []time.Duration
}

// Result holds the outcome of a crawl operation.
type Result struct {
	Saved  int
	Failed int
	Bytes  int
	Tokens int
}

// ProgressEvent reports progress during a crawl operation.
type ProgressEvent struct {
	Type      ProgressType
	Completed int
	Total     int
	URL       string
	Error     error
}

// ProgressType indicates the type of progress event.
type ProgressType int

const (
	ProgressStarted ProgressType = iota
	ProgressCompleted
	ProgressFailed
	ProgressFinished
)

// ProgressFunc is a callback for reporting crawl progress.
type ProgressFunc func(event ProgressEvent)

// crawlResult holds the outcome of processing a single URL.
type crawlResult struct {
	position   int
	url        string
	title      string
	markdown   string
	hash       string
	err        error
	discovered []locdoc.DiscoveredLink // Links discovered on this page (for recursive crawling)
}

// CrawlProject crawls all pages for a project and saves them as documents.
// The progress callback, if provided, receives events as crawling proceeds.
func (c *Crawler) CrawlProject(ctx context.Context, project *locdoc.Project, progress ProgressFunc) (*Result, error) {
	// Reconstruct URLFilter from project's stored filter patterns
	var urlFilter *locdoc.URLFilter
	if project.Filter != "" {
		urlFilter = &locdoc.URLFilter{}
		for _, pattern := range strings.Split(project.Filter, "\n") {
			if pattern == "" {
				continue
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid filter pattern %q: %w", pattern, err)
			}
			urlFilter.Include = append(urlFilter.Include, re)
		}
	}

	// Discover URLs from sitemap
	urls, err := c.Sitemaps.DiscoverURLs(ctx, project.SourceURL, urlFilter)
	if err != nil {
		return nil, fmt.Errorf("sitemap discovery: %w", err)
	}

	if len(urls) == 0 {
		// Fall back to recursive crawling if LinkSelectors is configured
		if c.LinkSelectors != nil && c.RateLimiter != nil {
			return c.recursiveCrawl(ctx, project, urlFilter, progress)
		}
		return &Result{}, nil
	}

	// Set up concurrency
	concurrency := c.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	// Channel for collecting results
	resultCh := make(chan crawlResult, len(urls))

	// Progress tracking
	var completed atomic.Int64
	total := len(urls)

	// Notify start
	if progress != nil {
		progress(ProgressEvent{
			Type:  ProgressStarted,
			Total: total,
		})
	}

	// Start workers
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	go func() {
		for i, url := range urls {
			i, url := i, url
			g.Go(func() error {
				result := c.processURL(gctx, i, url)
				resultCh <- result
				return nil
			})
		}
		_ = g.Wait()
		close(resultCh)
	}()

	// Collect results in order
	results := make([]crawlResult, len(urls))
	var failedCount int
	for result := range resultCh {
		completed.Add(1)
		results[result.position] = result

		if result.err != nil {
			failedCount++
			if progress != nil {
				progress(ProgressEvent{
					Type:      ProgressFailed,
					Completed: int(completed.Load()),
					Total:     total,
					URL:       result.url,
					Error:     result.err,
				})
			}
		} else {
			if progress != nil {
				progress(ProgressEvent{
					Type:      ProgressCompleted,
					Completed: int(completed.Load()),
					Total:     total,
					URL:       result.url,
				})
			}
		}
	}

	// Save documents and accumulate stats
	var savedCount int
	var totalBytes int
	var totalTokens int

	for _, result := range results {
		if result.err != nil {
			continue
		}

		doc := &locdoc.Document{
			ProjectID:   project.ID,
			SourceURL:   result.url,
			Title:       result.title,
			Content:     result.markdown,
			ContentHash: result.hash,
			Position:    result.position,
		}

		if err := c.Documents.CreateDocument(ctx, doc); err != nil {
			failedCount++
			continue
		}

		savedCount++
		totalBytes += len(result.markdown)
		if c.TokenCounter != nil {
			if tokens, err := c.TokenCounter.CountTokens(ctx, result.markdown); err == nil {
				totalTokens += tokens
			}
		}
	}

	// Notify finished
	if progress != nil {
		progress(ProgressEvent{
			Type:      ProgressFinished,
			Completed: total,
			Total:     total,
		})
	}

	return &Result{
		Saved:  savedCount,
		Failed: failedCount,
		Bytes:  totalBytes,
		Tokens: totalTokens,
	}, nil
}

// processURL fetches and processes a single URL.
func (c *Crawler) processURL(ctx context.Context, position int, url string) crawlResult {
	result := crawlResult{
		position: position,
		url:      url,
	}

	// Fetch with retry
	delays := c.RetryDelays
	if delays == nil {
		delays = DefaultRetryDelays()
	}
	fetchFn := func(ctx context.Context, url string) (string, error) {
		return c.Fetcher.Fetch(ctx, url)
	}
	html, err := FetchWithRetryDelays(ctx, url, fetchFn, nil, delays)
	if err != nil {
		result.err = err
		return result
	}

	// Extract content
	extracted, err := c.Extractor.Extract(html)
	if err != nil {
		result.err = err
		return result
	}

	// Convert to markdown
	markdown, err := c.Converter.Convert(extracted.ContentHTML)
	if err != nil {
		result.err = err
		return result
	}

	result.title = extracted.Title
	result.markdown = markdown
	result.hash = computeHash(markdown)

	return result
}

// Frontier configuration for recursive crawling.
const (
	// frontierExpectedURLs is the expected number of URLs for Bloom filter sizing.
	frontierExpectedURLs = 10000
	// frontierFalsePositiveRate is the acceptable false positive rate for deduplication.
	frontierFalsePositiveRate = 0.01
	// maxRecursiveCrawlURLs limits the number of URLs processed to prevent runaway crawls.
	maxRecursiveCrawlURLs = 1000
)

// walkProcessor processes a URL and returns a crawlResult.
type walkProcessor func(ctx context.Context, link locdoc.DiscoveredLink) crawlResult

// walkResultHandler handles a completed crawlResult.
// It should add discovered links to the frontier (after filtering) and handle the result.
type walkResultHandler func(result *crawlResult, frontier *Frontier, parsedSourceURL *url.URL, pathPrefix string, urlFilter *locdoc.URLFilter)

// walkFrontier manages concurrent URL processing starting from sourceURL.
// It handles the shared logic between DiscoverURLs and recursiveCrawl:
// - Frontier management with Bloom filter deduplication
// - Concurrent worker pool
// - Work dispatch and result collection
//
// The processURL function is called for each URL to fetch and process it.
// The handleResult function is called for each result to filter links and handle the outcome.
func (c *Crawler) walkFrontier(
	ctx context.Context,
	sourceURL string,
	urlFilter *locdoc.URLFilter,
	processURL walkProcessor,
	handleResult walkResultHandler,
) error {
	// Parse source URL to get base path for scope limiting
	parsedSourceURL, err := url.Parse(sourceURL)
	if err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	}
	pathPrefix := parsedSourceURL.Path

	// Create frontier and seed with source URL
	frontier := NewFrontier(frontierExpectedURLs, frontierFalsePositiveRate)
	frontier.Push(locdoc.DiscoveredLink{
		URL:      sourceURL,
		Priority: locdoc.PriorityNavigation,
	})

	// Set up concurrency
	concurrency := c.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	// Channels for worker coordination
	workCh := make(chan locdoc.DiscoveredLink, concurrency)
	resultCh := make(chan crawlResult)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for link := range workCh {
				result := processURL(ctx, link)
				select {
				case resultCh <- result:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Close result channel when all workers are done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Coordinator loop
	processedCount := 0 // URLs dispatched to workers
	pending := 0        // URLs currently being processed
	var nextLink *locdoc.DiscoveredLink

	// Get first link
	if link, ok := frontier.Pop(); ok {
		nextLink = &link
	}

coordinatorLoop:
	for {
		// Check termination conditions
		if nextLink == nil && pending == 0 {
			break coordinatorLoop
		}

		// Check context cancellation
		if ctx.Err() != nil {
			break coordinatorLoop
		}

		// Try to dispatch work or receive results
		if nextLink != nil && processedCount < maxRecursiveCrawlURLs {
			select {
			case <-ctx.Done():
				break coordinatorLoop
			case workCh <- *nextLink:
				processedCount++
				pending++
				nextLink = nil
			case crawlRes := <-resultCh:
				pending--
				handleResult(&crawlRes, frontier, parsedSourceURL, pathPrefix, urlFilter)
			}
		} else {
			// No more work to dispatch, just receive results
			select {
			case <-ctx.Done():
				break coordinatorLoop
			case crawlRes, ok := <-resultCh:
				if !ok {
					break coordinatorLoop
				}
				pending--
				handleResult(&crawlRes, frontier, parsedSourceURL, pathPrefix, urlFilter)
			}
		}

		// Try to get next link if we don't have one
		if nextLink == nil && processedCount < maxRecursiveCrawlURLs {
			if link, ok := frontier.Pop(); ok {
				nextLink = &link
			}
		}
	}

	// Signal workers to stop and drain remaining results
	close(workCh)

	// Drain any remaining results with timeout
	drainTimeout := time.After(5 * time.Second)
drainLoop:
	for {
		select {
		case crawlRes, ok := <-resultCh:
			if !ok {
				break drainLoop
			}
			handleResult(&crawlRes, frontier, parsedSourceURL, pathPrefix, urlFilter)
		case <-drainTimeout:
			break drainLoop
		}
	}

	return nil
}

// recursiveCrawl performs recursive link-following when sitemap discovery fails.
// It starts from the project's source URL and follows links within the path prefix scope.
// URLs are processed concurrently using walkFrontier.
func (c *Crawler) recursiveCrawl(ctx context.Context, project *locdoc.Project, urlFilter *locdoc.URLFilter, progress ProgressFunc) (*Result, error) {
	var result Result
	var position int
	completedCount := 0

	// Result handler that saves documents and reports progress
	handleResult := func(crawlRes *crawlResult, frontier *Frontier, sourceURL *url.URL, pathPrefix string, filter *locdoc.URLFilter) {
		c.processRecursiveResult(ctx, crawlRes, &result, &position, &completedCount, project, progress, frontier, sourceURL, pathPrefix, filter)
	}

	err := c.walkFrontier(ctx, project.SourceURL, urlFilter, c.processRecursiveURL, handleResult)
	if err != nil {
		return nil, err
	}

	if progress != nil {
		progress(ProgressEvent{
			Type: ProgressFinished,
		})
	}

	return &result, nil
}

// processRecursiveURL fetches and processes a single URL for recursive crawling.
func (c *Crawler) processRecursiveURL(ctx context.Context, link locdoc.DiscoveredLink) crawlResult {
	result := crawlResult{
		url: link.URL,
	}

	// Parse URL for rate limiting
	linkURL, err := url.Parse(link.URL)
	if err != nil {
		result.err = err
		return result
	}

	// Rate limit
	if err := c.RateLimiter.Wait(ctx, linkURL.Host); err != nil {
		result.err = err
		return result
	}

	// Fetch with retry
	delays := c.RetryDelays
	if delays == nil {
		delays = DefaultRetryDelays()
	}
	fetchFn := func(ctx context.Context, url string) (string, error) {
		return c.Fetcher.Fetch(ctx, url)
	}
	html, err := FetchWithRetryDelays(ctx, link.URL, fetchFn, nil, delays)
	if err != nil {
		result.err = err
		return result
	}

	// Extract links (coordinator will filter for scope)
	selector := c.LinkSelectors.GetForHTML(html)
	links, err := selector.ExtractLinks(html, link.URL)
	if err == nil {
		result.discovered = links
	}

	// Extract content
	extracted, err := c.Extractor.Extract(html)
	if err != nil {
		result.err = err
		return result
	}

	// Convert to markdown
	markdown, err := c.Converter.Convert(extracted.ContentHTML)
	if err != nil {
		result.err = err
		return result
	}

	result.title = extracted.Title
	result.markdown = markdown
	result.hash = computeHash(markdown)

	return result
}

// processRecursiveResult handles a completed crawl result from a worker.
func (c *Crawler) processRecursiveResult(
	ctx context.Context,
	crawlRes *crawlResult,
	result *Result,
	position *int,
	completedCount *int,
	project *locdoc.Project,
	progress ProgressFunc,
	frontier *Frontier,
	sourceURL *url.URL,
	pathPrefix string,
	urlFilter *locdoc.URLFilter,
) {
	// Add discovered links to frontier (after scope filtering)
	for _, discovered := range crawlRes.discovered {
		discoveredURL, err := url.Parse(discovered.URL)
		if err != nil {
			continue
		}
		if discoveredURL.Host != sourceURL.Host {
			continue
		}
		if !strings.HasPrefix(discoveredURL.Path, pathPrefix) {
			continue
		}
		if urlFilter != nil && !matchesFilter(discovered.URL, urlFilter) {
			continue
		}
		frontier.Push(discovered)
	}

	if crawlRes.err != nil {
		result.Failed++
		*completedCount++
		if progress != nil {
			progress(ProgressEvent{
				Type:      ProgressFailed,
				Completed: *completedCount,
				URL:       crawlRes.url,
				Error:     crawlRes.err,
			})
		}
		return
	}

	// Save document
	doc := &locdoc.Document{
		ProjectID:   project.ID,
		SourceURL:   crawlRes.url,
		Title:       crawlRes.title,
		Content:     crawlRes.markdown,
		ContentHash: crawlRes.hash,
		Position:    *position,
	}
	*position++

	if err := c.Documents.CreateDocument(ctx, doc); err != nil {
		result.Failed++
		*completedCount++
		if progress != nil {
			progress(ProgressEvent{
				Type:      ProgressFailed,
				Completed: *completedCount,
				URL:       crawlRes.url,
				Error:     err,
			})
		}
		return
	}

	result.Saved++
	result.Bytes += len(crawlRes.markdown)
	if c.TokenCounter != nil {
		if tokens, err := c.TokenCounter.CountTokens(ctx, crawlRes.markdown); err == nil {
			result.Tokens += tokens
		}
	}

	*completedCount++
	if progress != nil {
		progress(ProgressEvent{
			Type:      ProgressCompleted,
			Completed: *completedCount,
			URL:       crawlRes.url,
		})
	}
}

// matchesFilter checks if a URL matches the include patterns.
func matchesFilter(rawURL string, filter *locdoc.URLFilter) bool {
	if filter == nil || len(filter.Include) == 0 {
		return true
	}
	for _, re := range filter.Include {
		if re.MatchString(rawURL) {
			return true
		}
	}
	return false
}

// computeHash computes a hash of the content using xxhash.
func computeHash(content string) string {
	h := xxhash.Sum64String(content)
	return fmt.Sprintf("%x", h)
}

// ComputeHash computes a hash of the content using xxhash.
// This is the exported version for use in CLI commands.
func ComputeHash(content string) string {
	return computeHash(content)
}

// TruncateURL shortens a URL for display, keeping the end which is more informative.
func TruncateURL(url string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if maxLen < 4 {
		// Too short for "..." prefix, just return dots
		return url[:min(len(url), maxLen)]
	}
	if len(url) <= maxLen {
		return url
	}
	return "..." + url[len(url)-maxLen+3:]
}

// FormatBytes formats bytes in human-readable form.
func FormatBytes(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatTokens formats token count in human-readable form.
func FormatTokens(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("~%d tokens", tokens)
	}
	return fmt.Sprintf("~%dk tokens", (tokens+500)/1000)
}

// DiscoverURLs recursively discovers URLs from a documentation site.
// It follows links within the path prefix scope of the source URL.
// This is used for preview mode when sitemap discovery returns no URLs.
//
// Discovery stops after processing maxRecursiveCrawlURLs (1000) URLs
// to prevent runaway crawls on large sites.
//
// URLs are processed concurrently using walkFrontier for improved performance.
func DiscoverURLs(
	ctx context.Context,
	sourceURL string,
	urlFilter *locdoc.URLFilter,
	fetcher locdoc.Fetcher,
	linkSelectors locdoc.LinkSelectorRegistry,
	rateLimiter locdoc.DomainLimiter,
) ([]string, error) {
	// Create a minimal Crawler with just the dependencies needed for discovery
	c := &Crawler{
		Fetcher:       fetcher,
		LinkSelectors: linkSelectors,
		RateLimiter:   rateLimiter,
	}

	// Thread-safe collection of discovered URLs
	var mu sync.Mutex
	var urls []string

	// Discovery processor: fetch page and extract links (no content extraction)
	processURL := func(ctx context.Context, link locdoc.DiscoveredLink) crawlResult {
		result := crawlResult{
			url: link.URL,
		}

		// Parse URL for rate limiting
		linkURL, err := url.Parse(link.URL)
		if err != nil {
			result.err = err
			return result
		}

		// Rate limit
		if err := rateLimiter.Wait(ctx, linkURL.Host); err != nil {
			result.err = err
			return result
		}

		// Fetch page (no retry in discovery mode - keep it fast)
		html, err := fetcher.Fetch(ctx, link.URL)
		if err != nil {
			result.err = err
			return result
		}

		// Extract links for frontier
		selector := linkSelectors.GetForHTML(html)
		links, err := selector.ExtractLinks(html, link.URL)
		if err == nil {
			result.discovered = links
		}

		return result
	}

	// Discovery handler: collect URLs and add links to frontier
	handleResult := func(result *crawlResult, frontier *Frontier, parsedSourceURL *url.URL, pathPrefix string, filter *locdoc.URLFilter) {
		// Add discovered links to frontier (after scope filtering)
		for _, discovered := range result.discovered {
			discoveredURL, err := url.Parse(discovered.URL)
			if err != nil {
				continue
			}
			if discoveredURL.Host != parsedSourceURL.Host {
				continue
			}
			if !strings.HasPrefix(discoveredURL.Path, pathPrefix) {
				continue
			}
			if filter != nil && !matchesFilter(discovered.URL, filter) {
				continue
			}
			frontier.Push(discovered)
		}

		// Collect successfully fetched URLs
		if result.err == nil {
			mu.Lock()
			urls = append(urls, result.url)
			mu.Unlock()
		}
	}

	err := c.walkFrontier(ctx, sourceURL, urlFilter, processURL, handleResult)
	if err != nil {
		return nil, err
	}

	return urls, nil
}
