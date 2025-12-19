package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*NextraSelector)(nil)

// NextraSelector extracts links from Nextra documentation sites.
// Validated against Nextra v2.x and v3.x.
//
// It targets Nextra-specific navigation elements:
// - .nextra-sidebar for the main navigation
// - .nextra-toc for on-page TOC
// - .nextra-navbar for top navigation
type NextraSelector struct{}

// NewNextraSelector creates a new NextraSelector.
func NewNextraSelector() *NextraSelector {
	return &NextraSelector{}
}

// Name returns the selector's identifier.
func (s *NextraSelector) Name() string {
	return "nextra"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
func (s *NextraSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, locdoc.Errorf(locdoc.EINVALID, "invalid base URL: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, locdoc.Errorf(locdoc.EINVALID, "failed to parse HTML: %v", err)
	}

	seen := make(map[string]int)
	var links []locdoc.DiscoveredLink

	extractLinks := func(selector string, priority locdoc.LinkPriority, source string) {
		doc.Find(selector).Each(func(_ int, sel *goquery.Selection) {
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

			link := locdoc.DiscoveredLink{
				URL:      resolved,
				Priority: priority,
				Text:     strings.TrimSpace(sel.Text()),
				Source:   source,
			}

			if idx, ok := seen[resolved]; ok {
				if priority > links[idx].Priority {
					links[idx] = link
				}
			} else {
				seen[resolved] = len(links)
				links = append(links, link)
			}
		})
	}

	// Extract from Nextra-specific selectors in priority order
	// TOC has highest priority (PriorityTOC = 110)
	extractLinks(".nextra-toc a[href]", locdoc.PriorityTOC, "toc")

	// Navigation (PriorityNavigation = 100)
	extractLinks(".nextra-sidebar a[href]", locdoc.PriorityNavigation, "sidebar")
	extractLinks(".nextra-navbar a[href]", locdoc.PriorityNavigation, "navbar")

	// Content links (PriorityContent = 50)
	extractLinks("main a[href]", locdoc.PriorityContent, "content")
	extractLinks("article a[href]", locdoc.PriorityContent, "content")

	// Footer (PriorityFooter = 20)
	extractLinks("footer a[href]", locdoc.PriorityFooter, "footer")

	return links, nil
}
