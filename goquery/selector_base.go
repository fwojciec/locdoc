package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

// BaseSelector implements the base link extraction logic using CSS selectors.
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
func (s *BaseSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, locdoc.Errorf(locdoc.EINVALID, "invalid base URL: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, locdoc.Errorf(locdoc.EINVALID, "failed to parse HTML: %v", err)
	}

	// Use a map to deduplicate and keep highest priority
	seen := make(map[string]locdoc.DiscoveredLink)

	extractLinks := func(selector string, priority locdoc.LinkPriority, source string) {
		doc.Find(selector).Each(func(_ int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if !exists || href == "" {
				return
			}

			resolved := resolveURL(base, href)
			if resolved == "" {
				return
			}

			// Filter external links
			if !isSameHost(base, resolved) {
				return
			}

			link := locdoc.DiscoveredLink{
				URL:      resolved,
				Priority: priority,
				Text:     strings.TrimSpace(sel.Text()),
				Source:   source,
			}

			// Keep only if not seen or this has higher priority
			if existing, ok := seen[resolved]; !ok || priority > existing.Priority {
				seen[resolved] = link
			}
		})
	}

	// Extract in priority order (highest first)
	extractLinks("nav a[href]", locdoc.PriorityNavigation, "nav")
	extractLinks("aside a[href]", locdoc.PriorityTOC, "sidebar")
	extractLinks("main a[href], article a[href]", locdoc.PriorityContent, "content")
	extractLinks("footer a[href]", locdoc.PriorityFooter, "footer")

	// Convert map to slice maintaining insertion order by re-scanning
	var links []locdoc.DiscoveredLink
	addedURLs := make(map[string]bool)

	addFromSelector := func(selector string) {
		doc.Find(selector).Each(func(_ int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if !exists || href == "" {
				return
			}

			resolved := resolveURL(base, href)
			if resolved == "" {
				return
			}

			if addedURLs[resolved] {
				return
			}

			if link, ok := seen[resolved]; ok {
				links = append(links, link)
				addedURLs[resolved] = true
			}
		})
	}

	addFromSelector("nav a[href]")
	addFromSelector("aside a[href]")
	addFromSelector("main a[href], article a[href]")
	addFromSelector("footer a[href]")

	return links, nil
}

// resolveURL resolves a relative URL against a base URL.
func resolveURL(base *url.URL, href string) string {
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}

// isSameHost checks if the resolved URL has the same host as the base URL.
func isSameHost(base *url.URL, resolved string) bool {
	u, err := url.Parse(resolved)
	if err != nil {
		return false
	}
	return u.Host == base.Host
}
