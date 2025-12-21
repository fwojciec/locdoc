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
	activeFetcher := d.probeFetcher(ctx, sourceURL)

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

	err := d.walkFrontier(ctx, sourceURL, urlFilter, activeFetcher, cfg.concurrency, processURL, handleResult)
	if err != nil {
		return nil, err
	}

	return urls, nil
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
func (d *Discoverer) probeFetcher(ctx context.Context, probeURL string) locdoc.Fetcher {
	// Probe with HTTP
	httpHTML, httpErr := d.HTTPFetcher.Fetch(ctx, probeURL)
	if httpErr != nil {
		// HTTP failed, fall back to Rod
		return d.RodFetcher
	}

	// Detect framework
	framework := d.Prober.Detect(httpHTML)
	requiresJS, known := d.Prober.RequiresJS(framework)

	if known {
		if requiresJS {
			return d.RodFetcher
		}
		return d.HTTPFetcher
	}

	// Unknown framework: compare HTTP vs Rod content
	rodHTML, rodErr := d.RodFetcher.Fetch(ctx, probeURL)
	if rodErr != nil {
		// Rod failed, use HTTP
		return d.HTTPFetcher
	}

	if ContentDiffers(httpHTML, rodHTML, d.Extractor) {
		return d.RodFetcher
	}
	return d.HTTPFetcher
}

// walkFrontier manages concurrent URL processing starting from sourceURL.
// It handles frontier management with Bloom filter deduplication and
// a concurrent worker pool.
func (d *Discoverer) walkFrontier(
	ctx context.Context,
	sourceURL string,
	urlFilter *locdoc.URLFilter,
	fetcher locdoc.Fetcher,
	concurrency int,
	processURL walkProcessor,
	handleResult walkResultHandler,
) error {
	// Delegate to Crawler.walkFrontier - see locdoc-e70 for planned refactor
	c := &Crawler{
		Discoverer: &Discoverer{
			Concurrency: concurrency,
		},
	}
	return c.walkFrontier(ctx, sourceURL, urlFilter, fetcher, processURL, handleResult)
}
