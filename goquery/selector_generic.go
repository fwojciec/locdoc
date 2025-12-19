package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

// GenericSelector implements link extraction using universal CSS selectors
// that work across any documentation framework. It uses common HTML patterns
// and class names to identify navigation, TOC, content, and footer areas.
type GenericSelector struct{}

// NewGenericSelector creates a new GenericSelector.
func NewGenericSelector() *GenericSelector {
	return &GenericSelector{}
}

// Name returns the selector's identifier.
func (s *GenericSelector) Name() string {
	return "generic"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
//
// Priority order (highest to lowest):
//   - TOC: .toc, .sidebar, .table-of-contents, aside
//   - Navigation: nav, [role="navigation"], .nav, .menu, .navbar
//   - Content: main, article, .content, .doc-content
//   - Footer: footer, .footer
func (s *GenericSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
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

	// TOC selectors (highest priority after sitemap)
	tocSelectors := ".toc a[href], .table-of-contents a[href], .sidebar a[href], aside a[href]"
	extractLinks(tocSelectors, locdoc.PriorityTOC, "toc")

	// Navigation selectors
	navSelectors := "nav a[href], [role=\"navigation\"] a[href], .nav a[href], .menu a[href], .navbar a[href]"
	extractLinks(navSelectors, locdoc.PriorityNavigation, "nav")

	// Content selectors
	contentSelectors := "main a[href], article a[href], .content a[href], .doc-content a[href]"
	extractLinks(contentSelectors, locdoc.PriorityContent, "content")

	// Footer selectors
	footerSelectors := "footer a[href], .footer a[href]"
	extractLinks(footerSelectors, locdoc.PriorityFooter, "footer")

	return links, nil
}
