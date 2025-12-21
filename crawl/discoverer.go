package crawl

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/fwojciec/locdoc"
)

// Discoverer handles URL discovery for documentation sites.
// It probes sites to determine the best fetching strategy and
// recursively crawls to discover all documentation URLs.
type Discoverer struct {
	HTTPFetcher   locdoc.Fetcher
	RodFetcher    locdoc.Fetcher
	Prober        locdoc.Prober
	Extractor     locdoc.Extractor
	LinkSelectors locdoc.LinkSelectorRegistry
	RateLimiter   locdoc.DomainLimiter
	Concurrency   int
	RetryDelays   []time.Duration
}

// DiscoverURLs recursively discovers URLs from a documentation site.
// It follows links within the path prefix scope of the source URL.
//
// Discovery stops after processing maxRecursiveCrawlURLs (1000) URLs
// to prevent runaway crawls on large sites.
//
// URLs are processed concurrently using walkFrontier for improved performance.
func (d *Discoverer) DiscoverURLs(
	ctx context.Context,
	sourceURL string,
	urlFilter *locdoc.URLFilter,
	opts ...DiscoverOption,
) ([]string, error) {
	// Apply options
	cfg := &discoverConfig{
		concurrency: d.Concurrency,
		retryDelays: d.RetryDelays,
	}
	if cfg.concurrency <= 0 {
		cfg.concurrency = 3 // Lower default for preview mode
	}
	if cfg.retryDelays == nil {
		cfg.retryDelays = DefaultRetryDelays()
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Probe to determine which fetcher to use
	probeCfg := probeConfig{
		HTTPFetcher: d.HTTPFetcher,
		RodFetcher:  d.RodFetcher,
		Prober:      d.Prober,
		Extractor:   d.Extractor,
	}
	activeFetcher := probeFetcher(ctx, sourceURL, probeCfg)

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
		if err := d.RateLimiter.Wait(ctx, linkURL.Host); err != nil {
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
		selector := d.LinkSelectors.GetForHTML(html)
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

	err := walkFrontier(ctx, sourceURL, urlFilter, activeFetcher, cfg.concurrency, processURL, handleResult)
	if err != nil {
		return nil, err
	}

	return urls, nil
}
