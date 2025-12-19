// Package goquery provides HTML link extraction using CSS selectors.
//
// This package implements the locdoc.LinkSelector interface for extracting
// prioritized links from HTML documents. It uses goquery for CSS selector
// matching and supports different priority levels based on where links appear
// in the document structure (nav, aside, main/article, footer).
package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

// BaseSelector implements the base link extraction logic using CSS selectors.
// It extracts links from common HTML structural elements and assigns priorities
// based on their location in the document.
type BaseSelector struct{}

// NewBaseSelector creates a new BaseSelector.
func NewBaseSelector() *BaseSelector {
	return &BaseSelector{}
}

// Name returns the selector's identifier.
func (s *BaseSelector) Name() string {
	return "base"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
// The returned links maintain document order based on first occurrence.
func (s *BaseSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
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

	extractLinks := func(selector string, priority locdoc.LinkPriority, source string) {
		doc.Find(selector).Each(func(_ int, sel *goquery.Selection) {
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
				Priority: priority,
				Text:     strings.TrimSpace(sel.Text()),
				Source:   source,
			}

			if idx, ok := seen[resolved]; ok {
				// Update if this has higher priority
				if priority > links[idx].Priority {
					links[idx] = link
				}
			} else {
				// First occurrence - add to slice and track index
				seen[resolved] = len(links)
				links = append(links, link)
			}
		})
	}

	// Extract in priority order (highest first for better deduplication)
	extractLinks("nav a[href]", locdoc.PriorityNavigation, "nav")
	extractLinks("aside a[href]", locdoc.PriorityTOC, "sidebar")
	extractLinks("main a[href], article a[href]", locdoc.PriorityContent, "content")
	extractLinks("footer a[href]", locdoc.PriorityFooter, "footer")

	return links, nil
}

// resolveURL resolves a relative URL against a base URL.
// Returns empty string if the href cannot be parsed.
func resolveURL(base *url.URL, href string) string {
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
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
