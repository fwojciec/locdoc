// Package crawl provides documentation crawling orchestration.
// It coordinates sitemap discovery, fetching, extraction, and storage
// of documentation pages.
package crawl

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/fwojciec/locdoc"
	"golang.org/x/sync/errgroup"
)

// Crawler orchestrates the crawling of documentation sites.
type Crawler struct {
	*Discoverer
	Sitemaps     locdoc.SitemapService
	Converter    locdoc.Converter
	Documents    locdoc.DocumentService
	TokenCounter locdoc.TokenCounter
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

// probeConfig holds dependencies for probeFetcher.
type probeConfig struct {
	HTTPFetcher locdoc.Fetcher
	RodFetcher  locdoc.Fetcher
	Prober      locdoc.Prober
	Extractor   locdoc.Extractor
}

// probeFetcher determines which fetcher to use for crawling by probing the first URL.
// Returns the fetcher to use for subsequent requests.
//
// Logic:
// 1. HTTP fetch first URL
// 2. Detect framework
// 3. If known framework → use HTTP or Rod based on RequiresJS
// 4. If unknown → Rod fetch, compare content, choose based on differences
// 5. If HTTP fails → fall back to Rod
func probeFetcher(ctx context.Context, probeURL string, cfg probeConfig) locdoc.Fetcher {
	// Probe with HTTP
	httpHTML, httpErr := cfg.HTTPFetcher.Fetch(ctx, probeURL)
	if httpErr != nil {
		// HTTP failed, fall back to Rod
		return cfg.RodFetcher
	}

	// Detect framework
	framework := cfg.Prober.Detect(httpHTML)
	requiresJS, known := cfg.Prober.RequiresJS(framework)

	if known {
		if requiresJS {
			return cfg.RodFetcher
		}
		return cfg.HTTPFetcher
	}

	// Unknown framework: compare HTTP vs Rod content
	rodHTML, rodErr := cfg.RodFetcher.Fetch(ctx, probeURL)
	if rodErr != nil {
		// Rod failed, use HTTP
		return cfg.HTTPFetcher
	}

	if ContentDiffers(httpHTML, rodHTML, cfg.Extractor) {
		return cfg.RodFetcher
	}
	return cfg.HTTPFetcher
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
			cfg := probeConfig{
				HTTPFetcher: c.HTTPFetcher,
				RodFetcher:  c.RodFetcher,
				Prober:      c.Prober,
				Extractor:   c.Extractor,
			}
			fetcher := probeFetcher(ctx, project.SourceURL, cfg)
			return c.recursiveCrawl(ctx, project, urlFilter, fetcher, progress)
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

	// Probe first URL to determine which fetcher to use
	cfg := probeConfig{
		HTTPFetcher: c.HTTPFetcher,
		RodFetcher:  c.RodFetcher,
		Prober:      c.Prober,
		Extractor:   c.Extractor,
	}
	fetcher := probeFetcher(ctx, urls[0], cfg)

	// Start workers
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	go func() {
		for i, url := range urls {
			i, url := i, url
			g.Go(func() error {
				result := c.processURL(gctx, i, url, fetcher)
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
func (c *Crawler) processURL(ctx context.Context, position int, url string, fetcher locdoc.Fetcher) crawlResult {
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
		return fetcher.Fetch(ctx, url)
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
