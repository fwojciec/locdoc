package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/beevik/etree"
	"github.com/fwojciec/locdoc"
)

// Ensure SitemapService implements locdoc.SitemapService.
var _ locdoc.SitemapService = (*SitemapService)(nil)

// SitemapService discovers URLs from website sitemaps via HTTP.
type SitemapService struct {
	client *http.Client
}

// NewSitemapService creates a new SitemapService with the given HTTP client.
// If client is nil, http.DefaultClient is used.
func NewSitemapService(client *http.Client) *SitemapService {
	if client == nil {
		client = http.DefaultClient
	}
	return &SitemapService{client: client}
}

// DiscoverURLs finds all URLs from a site's sitemap.
// Returns an empty slice (not nil) if no sitemaps are found.
//
// When baseURL has a non-root path (e.g., https://example.com/docs/),
// only URLs with paths starting with that prefix are returned.
func (s *SitemapService) DiscoverURLs(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
	// Check for context cancellation early
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Extract path prefix for filtering (empty or "/" means no prefix filtering)
	pathPrefix := base.Path
	if pathPrefix == "/" {
		pathPrefix = ""
	}

	// For sitemap discovery, use the root of the domain (strip any path)
	sitemapBase := *base
	sitemapBase.Path = ""

	// Find sitemap URLs from robots.txt or fallback
	sitemapURLs, err := s.findSitemapURLs(ctx, &sitemapBase)
	if err != nil {
		return nil, err
	}

	// If no sitemaps found, return empty list
	if len(sitemapURLs) == 0 {
		return []string{}, nil
	}

	// Process all sitemaps and collect URLs
	var allURLs []string
	seenSitemaps := make(map[string]bool)
	seenURLs := make(map[string]bool)

	for _, sitemapURL := range sitemapURLs {
		urls, err := s.processSitemap(ctx, sitemapURL, seenSitemaps)
		if err != nil {
			return nil, err
		}
		// Deduplicate URLs across sitemaps
		for _, u := range urls {
			if !seenURLs[u] {
				seenURLs[u] = true
				allURLs = append(allURLs, u)
			}
		}
	}

	// Apply path prefix filter if baseURL has a non-root path
	if pathPrefix != "" {
		var filtered []string
		for _, u := range allURLs {
			if matchesPathPrefix(u, pathPrefix) {
				filtered = append(filtered, u)
			}
		}
		allURLs = filtered
	}

	// Apply user-provided filter
	if filter != nil {
		var filtered []string
		for _, u := range allURLs {
			if filter.Match(u) {
				filtered = append(filtered, u)
			}
		}
		return filtered, nil
	}

	return allURLs, nil
}

// matchesPathPrefix checks if a URL's path starts with the given prefix,
// respecting path boundaries. If prefix doesn't end with /, it's normalized
// to do so for matching (e.g., /docs matches /docs/ and /docs/intro but not /documentation).
func matchesPathPrefix(rawURL, prefix string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	path := parsed.Path

	// Normalize prefix to end with / if non-empty and not ending with /
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	// Check if path starts with prefix
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	// If we have a prefix that ends with / (after normalization),
	// it's already at a path boundary, so any match is valid
	if strings.HasSuffix(prefix, "/") {
		return true
	}

	// This shouldn't be reached since we always normalize to end with /
	return false
}

// findSitemapURLs discovers sitemap URLs from robots.txt or falls back to /sitemap.xml.
func (s *SitemapService) findSitemapURLs(ctx context.Context, base *url.URL) ([]string, error) {
	// Try robots.txt first
	robotsURL := base.ResolveReference(&url.URL{Path: "/robots.txt"})
	sitemaps, err := s.parseSitemapsFromRobots(ctx, robotsURL.String())
	if err == nil && len(sitemaps) > 0 {
		return sitemaps, nil
	}

	// Fall back to /sitemap.xml
	sitemapURL := base.ResolveReference(&url.URL{Path: "/sitemap.xml"})
	exists, err := s.urlExists(ctx, sitemapURL.String())
	if err != nil {
		// Propagate context errors, treat other errors as "not found"
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, nil
	}
	if exists {
		return []string{sitemapURL.String()}, nil
	}

	return nil, nil
}

// parseSitemapsFromRobots extracts Sitemap: directives from robots.txt.
func (s *SitemapService) parseSitemapsFromRobots(ctx context.Context, robotsURL string) ([]string, error) {
	body, err := s.fetchURL(ctx, robotsURL)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var sitemaps []string
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Case-insensitive check for Sitemap: directive
		if strings.HasPrefix(strings.ToLower(line), "sitemap:") {
			sitemapURL := strings.TrimSpace(line[8:]) // len("sitemap:") == 8
			if sitemapURL != "" {
				sitemaps = append(sitemaps, sitemapURL)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading robots.txt: %w", err)
	}

	return sitemaps, nil
}

// processSitemap fetches and parses a sitemap, handling both urlset and sitemapindex.
func (s *SitemapService) processSitemap(ctx context.Context, sitemapURL string, seen map[string]bool) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Avoid processing the same sitemap twice
	if seen[sitemapURL] {
		return nil, nil
	}
	seen[sitemapURL] = true

	body, err := s.fetchURL(ctx, sitemapURL)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(body); err != nil {
		return nil, fmt.Errorf("parsing sitemap XML: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("empty sitemap XML")
	}

	// Check if this is a sitemap index
	if root.Tag == "sitemapindex" {
		return s.processSitemapIndex(ctx, root, seen)
	}

	// Otherwise treat as urlset
	return s.parseURLSet(root), nil
}

// processSitemapIndex processes a <sitemapindex> element recursively.
func (s *SitemapService) processSitemapIndex(ctx context.Context, root *etree.Element, seen map[string]bool) ([]string, error) {
	var allURLs []string

	for _, sitemap := range root.SelectElements("sitemap") {
		loc := sitemap.SelectElement("loc")
		if loc == nil {
			continue
		}
		sitemapURL := strings.TrimSpace(loc.Text())
		if sitemapURL == "" {
			continue
		}

		urls, err := s.processSitemap(ctx, sitemapURL, seen)
		if err != nil {
			return nil, err
		}
		allURLs = append(allURLs, urls...)
	}

	return allURLs, nil
}

// parseURLSet extracts URLs from a <urlset> element.
func (s *SitemapService) parseURLSet(root *etree.Element) []string {
	var urls []string
	for _, urlEl := range root.SelectElements("url") {
		loc := urlEl.SelectElement("loc")
		if loc == nil {
			continue
		}
		u := strings.TrimSpace(loc.Text())
		if u != "" {
			urls = append(urls, u)
		}
	}
	return urls
}

// fetchURL fetches a URL and returns the response body.
func (s *SitemapService) fetchURL(ctx context.Context, targetURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, targetURL)
	}

	return resp.Body, nil
}

// urlExists checks if a URL returns 200 OK.
func (s *SitemapService) urlExists(ctx context.Context, targetURL string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL, nil)
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
