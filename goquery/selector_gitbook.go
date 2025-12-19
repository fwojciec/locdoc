package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

// GitBookSelector extracts links from GitBook documentation sites.
// Validated against GitBook's current (2024) hosted platform.
//
// It targets GitBook-specific navigation elements:
// - [data-testid="space.sidebar"] for the main navigation
// - [data-testid="page.desktopTableOfContents"] for on-page TOC
// - [data-testid="space.header"] for header navigation
type GitBookSelector struct{}

// NewGitBookSelector creates a new GitBookSelector.
func NewGitBookSelector() *GitBookSelector {
	return &GitBookSelector{}
}

// Name returns the selector's identifier.
func (s *GitBookSelector) Name() string {
	return "gitbook"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
func (s *GitBookSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
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

	// Extract from GitBook-specific selectors in priority order
	// TOC has highest priority (PriorityTOC = 110)
	extractLinks("[data-testid='page.desktopTableOfContents'] a[href]", locdoc.PriorityTOC, "toc")

	// Navigation sidebars and header (PriorityNavigation = 100)
	extractLinks("[data-testid='space.sidebar'] a[href]", locdoc.PriorityNavigation, "sidebar")
	extractLinks("[data-testid='space.header'] a[href]", locdoc.PriorityNavigation, "header")

	// Content links (PriorityContent = 50)
	extractLinks("[data-testid='page.contentEditor'] a[href]", locdoc.PriorityContent, "content")
	extractLinks("main a[href]", locdoc.PriorityContent, "content")
	extractLinks("article a[href]", locdoc.PriorityContent, "content")

	// Footer (PriorityFooter = 20)
	extractLinks("footer a[href]", locdoc.PriorityFooter, "footer")

	return links, nil
}
