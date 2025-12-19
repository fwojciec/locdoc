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
	position int
	url      string
	title    string
	markdown string
	hash     string
	err      error
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

// recursiveCrawl performs recursive link-following when sitemap discovery fails.
// It starts from the project's source URL and follows links within the path prefix scope.
//
// Note: URLs are processed sequentially (not concurrently) to simplify rate limiting
// and frontier management. For sites requiring high throughput, use sitemap-based crawling.
func (c *Crawler) recursiveCrawl(ctx context.Context, project *locdoc.Project, urlFilter *locdoc.URLFilter, progress ProgressFunc) (*Result, error) {
	// Parse source URL to get base path for scope limiting
	sourceURL, err := url.Parse(project.SourceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid source URL: %w", err)
	}
	pathPrefix := sourceURL.Path

	// Create frontier and seed with source URL
	frontier := NewFrontier(frontierExpectedURLs, frontierFalsePositiveRate)
	frontier.Push(locdoc.DiscoveredLink{
		URL:      project.SourceURL,
		Priority: locdoc.PriorityNavigation,
	})

	var result Result
	position := 0
	processedCount := 0

	// Process URLs from frontier
	for {
		link, ok := frontier.Pop()
		if !ok {
			break // Frontier empty
		}

		// Safety limit to prevent runaway crawls
		if processedCount >= maxRecursiveCrawlURLs {
			break
		}
		processedCount++

		// Check context cancellation
		if ctx.Err() != nil {
			break
		}

		// Rate limit
		linkURL, err := url.Parse(link.URL)
		if err != nil {
			result.Failed++
			continue
		}
		if err := c.RateLimiter.Wait(ctx, linkURL.Host); err != nil {
			break // Context canceled
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
			result.Failed++
			if progress != nil {
				progress(ProgressEvent{
					Type:  ProgressFailed,
					URL:   link.URL,
					Error: err,
				})
			}
			continue
		}

		// Extract links and add to frontier
		selector := c.LinkSelectors.GetForHTML(html)
		links, err := selector.ExtractLinks(html, link.URL)
		if err == nil {
			for _, discovered := range links {
				// Check scope: must be same host and within path prefix
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
				// Apply URL filter if configured
				if urlFilter != nil && !matchesFilter(discovered.URL, urlFilter) {
					continue
				}
				frontier.Push(discovered)
			}
		}

		// Extract content
		extracted, err := c.Extractor.Extract(html)
		if err != nil {
			result.Failed++
			if progress != nil {
				progress(ProgressEvent{
					Type:  ProgressFailed,
					URL:   link.URL,
					Error: err,
				})
			}
			continue
		}

		// Convert to markdown
		markdown, err := c.Converter.Convert(extracted.ContentHTML)
		if err != nil {
			result.Failed++
			if progress != nil {
				progress(ProgressEvent{
					Type:  ProgressFailed,
					URL:   link.URL,
					Error: err,
				})
			}
			continue
		}

		// Save document
		doc := &locdoc.Document{
			ProjectID:   project.ID,
			SourceURL:   link.URL,
			Title:       extracted.Title,
			Content:     markdown,
			ContentHash: computeHash(markdown),
			Position:    position,
		}
		position++

		if err := c.Documents.CreateDocument(ctx, doc); err != nil {
			result.Failed++
			continue
		}

		result.Saved++
		result.Bytes += len(markdown)
		if c.TokenCounter != nil {
			if tokens, err := c.TokenCounter.CountTokens(ctx, markdown); err == nil {
				result.Tokens += tokens
			}
		}

		if progress != nil {
			progress(ProgressEvent{
				Type: ProgressCompleted,
				URL:  link.URL,
			})
		}
	}

	if progress != nil {
		progress(ProgressEvent{
			Type: ProgressFinished,
		})
	}

	return &result, nil
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
