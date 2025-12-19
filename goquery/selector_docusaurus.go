package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*DocusaurusSelector)(nil)

// DocusaurusSelector extracts links from Docusaurus documentation sites.
// Validated against Docusaurus v2.x and v3.x.
//
// It targets Docusaurus-specific navigation elements:
// - .theme-doc-sidebar-container for the docs sidebar
// - .table-of-contents for on-page TOC
// - .navbar for the top navigation bar
type DocusaurusSelector struct{}

// NewDocusaurusSelector creates a new DocusaurusSelector.
func NewDocusaurusSelector() *DocusaurusSelector {
	return &DocusaurusSelector{}
}

// Name returns the selector's identifier.
func (s *DocusaurusSelector) Name() string {
	return "docusaurus"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
func (s *DocusaurusSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
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

			// Filter external links (exact host match)
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

	// Extract from Docusaurus-specific selectors in priority order
	// TOC has highest priority (PriorityTOC = 110)
	extractLinks(".table-of-contents a[href]", locdoc.PriorityTOC, "toc")

	// Sidebar navigation (PriorityNavigation = 100)
	extractLinks(".theme-doc-sidebar-container a[href]", locdoc.PriorityNavigation, "sidebar")
	extractLinks("nav.navbar a[href]", locdoc.PriorityNavigation, "navbar")

	// Content links (PriorityContent = 50)
	extractLinks("article a[href]", locdoc.PriorityContent, "content")
	extractLinks("main a[href]", locdoc.PriorityContent, "content")

	// Footer (PriorityFooter = 20)
	extractLinks("footer a[href]", locdoc.PriorityFooter, "footer")

	return links, nil
}
