package crawl

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/fwojciec/locdoc"
)

// DiscoverOption configures DiscoverURLs behavior.
type DiscoverOption func(*discoverConfig)

type discoverConfig struct {
	concurrency int
	retryDelays []time.Duration
	onURL       func(string)
}

// WithConcurrency sets the number of concurrent workers for URL discovery.
// Defaults to 3 if not specified (lower than full crawl to avoid overwhelming browsers).
func WithConcurrency(n int) DiscoverOption {
	return func(c *discoverConfig) {
		c.concurrency = n
	}
}

// WithRetryDelays sets the retry delays for failed fetches.
// Defaults to DefaultRetryDelays() if not specified.
func WithRetryDelays(delays []time.Duration) DiscoverOption {
	return func(c *discoverConfig) {
		c.retryDelays = delays
	}
}

// WithOnURL sets a callback that is invoked for each URL as it is discovered.
// This enables streaming output instead of waiting for all URLs to be collected.
func WithOnURL(fn func(string)) DiscoverOption {
	return func(c *discoverConfig) {
		c.onURL = fn
	}
}

// DiscoverURLs recursively discovers URLs from a documentation site.
// It follows links within the path prefix scope of the source URL.
// This is used for preview mode when sitemap discovery returns no URLs.
//
// Discovery stops after processing maxRecursiveCrawlURLs (1000) URLs
// to prevent runaway crawls on large sites.
//
// URLs are processed concurrently using walkFrontier for improved performance.
// Use WithConcurrency and WithRetryDelays options to configure behavior.
//
// The function probes the source URL to determine whether to use HTTP or
// browser-based fetching based on framework detection.
// The httpFetcher, rodFetcher, prober, and extractor arguments are required and must not be nil.
func DiscoverURLs(
	ctx context.Context,
	sourceURL string,
	urlFilter *locdoc.URLFilter,
	linkSelectors locdoc.LinkSelectorRegistry,
	rateLimiter locdoc.DomainLimiter,
	httpFetcher locdoc.Fetcher,
	rodFetcher locdoc.Fetcher,
	prober locdoc.Prober,
	extractor locdoc.Extractor,
	opts ...DiscoverOption,
) ([]string, error) {
	// Apply options
	cfg := &discoverConfig{
		concurrency: 3, // Lower default for preview mode
		retryDelays: DefaultRetryDelays(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Probe to determine which fetcher to use
	probeCrawler := &Crawler{
		HTTPFetcher: httpFetcher,
		RodFetcher:  rodFetcher,
		Prober:      prober,
		Extractor:   extractor,
	}
	activeFetcher := probeCrawler.probeFetcher(ctx, sourceURL)

	// Create a minimal Crawler with just the dependencies needed for discovery
	c := &Crawler{
		HTTPFetcher:   activeFetcher,
		RodFetcher:    activeFetcher, // Discovery uses the same fetcher for both
		LinkSelectors: linkSelectors,
		RateLimiter:   rateLimiter,
		Concurrency:   cfg.concurrency,
		RetryDelays:   cfg.retryDelays,
	}

	// Collected URLs (handleResult is called sequentially from coordinator)
	var urls []string

	// Discovery processor: fetch page and extract links (no content extraction)
	processURL := func(ctx context.Context, link locdoc.DiscoveredLink, f locdoc.Fetcher) crawlResult {
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

		// Fetch page with retry
		fetchFn := func(ctx context.Context, url string) (string, error) {
			return f.Fetch(ctx, url)
		}
		html, err := FetchWithRetryDelays(ctx, link.URL, fetchFn, nil, cfg.retryDelays)
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
			urls = append(urls, result.url)
			if cfg.onURL != nil {
				cfg.onURL(result.url)
			}
		}
	}

	err := c.walkFrontier(ctx, sourceURL, urlFilter, activeFetcher, processURL, handleResult)
	if err != nil {
		return nil, err
	}

	return urls, nil
}
