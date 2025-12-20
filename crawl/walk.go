package crawl

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fwojciec/locdoc"
)

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
