package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*SphinxSelector)(nil)

// SphinxSelector extracts links from Sphinx documentation sites.
// Validated against Sphinx v4.x-v7.x with ReadTheDocs and classic themes.
//
// It supports both the ReadTheDocs theme and the classic Sphinx theme:
// - .wy-nav-side, .wy-menu-vertical for ReadTheDocs theme
// - .sphinxsidebar for classic theme
// - .toctree-wrapper, #localtoc for TOC elements
type SphinxSelector struct{}

// NewSphinxSelector creates a new SphinxSelector.
func NewSphinxSelector() *SphinxSelector {
	return &SphinxSelector{}
}

// Name returns the selector's identifier.
func (s *SphinxSelector) Name() string {
	return "sphinx"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
func (s *SphinxSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
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

	// Extract from Sphinx-specific selectors in priority order
	// TOC has highest priority (PriorityTOC = 110)
	extractLinks(".toctree-wrapper a[href]", locdoc.PriorityTOC, "toc")
	extractLinks("#localtoc a[href]", locdoc.PriorityTOC, "toc")

	// Navigation sidebars (PriorityNavigation = 100)
	// ReadTheDocs theme
	extractLinks(".wy-nav-side a[href]", locdoc.PriorityNavigation, "nav")
	extractLinks(".wy-menu-vertical a[href]", locdoc.PriorityNavigation, "nav")
	// Classic Sphinx theme
	extractLinks(".sphinxsidebar a[href]", locdoc.PriorityNavigation, "nav")

	// Content links (PriorityContent = 50)
	extractLinks(".document a[href]", locdoc.PriorityContent, "content")
	extractLinks(".body a[href]", locdoc.PriorityContent, "content")
	extractLinks("article a[href]", locdoc.PriorityContent, "content")

	// Footer (PriorityFooter = 20)
	extractLinks("footer a[href]", locdoc.PriorityFooter, "footer")

	return links, nil
}
