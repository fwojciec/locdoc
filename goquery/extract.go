package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

// SelectorConfig defines a CSS selector with its priority and source label.
type SelectorConfig struct {
	Selector string
	Priority locdoc.LinkPriority
	Source   string
}

// ExtractLinksWithConfigs extracts links from HTML using the provided selector configurations.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
// The returned links maintain document order based on first occurrence.
func ExtractLinksWithConfigs(html string, baseURL string, configs []SelectorConfig) ([]locdoc.DiscoveredLink, error) {
	return extractLinksWithConfigs(html, baseURL, configs, false)
}

// ExtractLinksWithConfigsAndFallback is like ExtractLinksWithConfigs but also extracts
// fallback links from any anchor that matches the base URL path prefix.
// Fallback links have PriorityFallback and won't override higher-priority duplicates.
func ExtractLinksWithConfigsAndFallback(html string, baseURL string, configs []SelectorConfig) ([]locdoc.DiscoveredLink, error) {
	return extractLinksWithConfigs(html, baseURL, configs, true)
}

func extractLinksWithConfigs(html string, baseURL string, configs []SelectorConfig, includeFallback bool) ([]locdoc.DiscoveredLink, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, locdoc.Errorf(locdoc.EINVALID, "invalid base URL: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, locdoc.Errorf(locdoc.EINVALID, "failed to parse HTML: %v", err)
	}

	// Track seen URLs with their index in the result slice for O(1) updates
	seen := make(map[string]int)
	var links []locdoc.DiscoveredLink

	for _, config := range configs {
		doc.Find(config.Selector).Each(func(_ int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if !exists || href == "" {
				return
			}

			// Skip non-HTTP links (javascript:, mailto:, etc.)
			if isNonHTTPLink(href) {
				return
			}

			resolved := resolveURL(base, href)
			if resolved == "" {
				return
			}

			// Filter external links (exact host match, subdomains are filtered)
			if !isSameHost(base, resolved) {
				return
			}

			link := locdoc.DiscoveredLink{
				URL:      resolved,
				Priority: config.Priority,
				Text:     strings.TrimSpace(sel.Text()),
				Source:   config.Source,
			}

			if idx, ok := seen[resolved]; ok {
				// Update if this has higher priority
				if config.Priority > links[idx].Priority {
					links[idx] = link
				}
			} else {
				// First occurrence - add to slice and track index
				seen[resolved] = len(links)
				links = append(links, link)
			}
		})
	}

	// Fallback: extract links matching the base URL path prefix with low priority.
	// Links already found via semantic selectors keep their higher priority
	// due to the deduplication logic. This ensures sites with non-semantic
	// HTML (like Tailwind CSS) still get their links discovered.
	if includeFallback {
		basePath := base.Path
		doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if !exists || href == "" {
				return
			}

			if isNonHTTPLink(href) {
				return
			}

			resolved := resolveURL(base, href)
			if resolved == "" {
				return
			}

			if !isSameHost(base, resolved) {
				return
			}

			// For fallback, also filter by base URL path prefix
			resolvedURL, err := url.Parse(resolved)
			if err != nil {
				return
			}
			if basePath != "" && !strings.HasPrefix(resolvedURL.Path, basePath) {
				return
			}

			link := locdoc.DiscoveredLink{
				URL:      resolved,
				Priority: locdoc.PriorityFallback,
				Text:     strings.TrimSpace(sel.Text()),
				Source:   "fallback",
			}

			if idx, ok := seen[resolved]; ok {
				if locdoc.PriorityFallback > links[idx].Priority {
					links[idx] = link
				}
			} else {
				seen[resolved] = len(links)
				links = append(links, link)
			}
		})
	}

	return links, nil
}

// resolveURL resolves a relative URL against a base URL.
// Returns empty string if the href cannot be parsed or if the resolved URL
// is self-referential (same as base URL after stripping fragment).
// Fragments are stripped from the resolved URL for deduplication purposes.
func resolveURL(base *url.URL, href string) string {
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(ref)
	resolved.Fragment = "" // Strip fragment for deduplication

	// Filter self-referential links (e.g., anchor-only links pointing to same page)
	// Compare against base URL with fragment stripped for defensive correctness
	result := resolved.String()
	baseNoFragment := *base
	baseNoFragment.Fragment = ""
	if result == baseNoFragment.String() {
		return ""
	}
	return result
}

// isSameHost checks if the resolved URL has the same host as the base URL.
// This uses exact host matching - subdomains are considered different hosts.
func isSameHost(base *url.URL, resolved string) bool {
	u, err := url.Parse(resolved)
	if err != nil {
		return false
	}
	return u.Host == base.Host
}

// isNonHTTPLink checks if a href is a non-HTTP link that should be skipped.
func isNonHTTPLink(href string) bool {
	href = strings.ToLower(strings.TrimSpace(href))
	return strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "mailto:") ||
		strings.HasPrefix(href, "tel:") ||
		strings.HasPrefix(href, "data:")
}
