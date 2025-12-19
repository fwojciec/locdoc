package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

// MkDocsSelector extracts links from MkDocs Material documentation sites.
// It targets MkDocs-specific navigation elements:
// - .md-nav--primary for the main navigation
// - .md-sidebar--secondary for the on-page TOC
// - [data-md-component="navigation"] and [data-md-component="toc"]
type MkDocsSelector struct{}

// NewMkDocsSelector creates a new MkDocsSelector.
func NewMkDocsSelector() *MkDocsSelector {
	return &MkDocsSelector{}
}

// Name returns the selector's identifier.
func (s *MkDocsSelector) Name() string {
	return "mkdocs"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
func (s *MkDocsSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
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

	// Extract from MkDocs-specific selectors in priority order
	// TOC has highest priority (PriorityTOC = 110)
	extractLinks(".md-sidebar--secondary a[href]", locdoc.PriorityTOC, "toc")
	extractLinks("[data-md-component='toc'] a[href]", locdoc.PriorityTOC, "toc")

	// Primary navigation (PriorityNavigation = 100)
	extractLinks(".md-nav--primary a[href]", locdoc.PriorityNavigation, "nav")
	extractLinks("[data-md-component='navigation'] a[href]", locdoc.PriorityNavigation, "nav")

	// Content links (PriorityContent = 50)
	extractLinks(".md-content a[href]", locdoc.PriorityContent, "content")
	extractLinks("article a[href]", locdoc.PriorityContent, "content")

	// Footer (PriorityFooter = 20)
	extractLinks("footer a[href]", locdoc.PriorityFooter, "footer")

	return links, nil
}
