package crawl

import (
	"context"
	"net/url"
	"strings"

	"github.com/fwojciec/locdoc"
)

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

	// Collected URLs (handleResult is called sequentially from coordinator)
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
			urls = append(urls, result.url)
		}
	}

	err := c.walkFrontier(ctx, sourceURL, urlFilter, processURL, handleResult)
	if err != nil {
		return nil, err
	}

	return urls, nil
}
