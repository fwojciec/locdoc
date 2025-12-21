package goquery

import (
	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*GitBookSelector)(nil)

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
	configs := []SelectorConfig{
		// TOC has highest priority (PriorityTOC = 110)
		{Selector: "[data-testid='page.desktopTableOfContents'] a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		// Navigation sidebars and header (PriorityNavigation = 100)
		{Selector: "[data-testid='space.sidebar'] a[href]", Priority: locdoc.PriorityNavigation, Source: "sidebar"},
		{Selector: "[data-testid='space.header'] a[href]", Priority: locdoc.PriorityNavigation, Source: "header"},
		// Content links (PriorityContent = 50)
		{Selector: "[data-testid='page.contentEditor'] a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: "main a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: "article a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		// Footer (PriorityFooter = 20)
		{Selector: "footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
	}
	return ExtractLinksWithConfigs(html, baseURL, configs)
}
