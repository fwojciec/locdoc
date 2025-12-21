package goquery

import (
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
	configs := []SelectorConfig{
		// TOC has highest priority (PriorityTOC = 110)
		{Selector: ".nextra-toc a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		// Navigation (PriorityNavigation = 100)
		{Selector: ".nextra-sidebar a[href]", Priority: locdoc.PriorityNavigation, Source: "sidebar"},
		{Selector: ".nextra-navbar a[href]", Priority: locdoc.PriorityNavigation, Source: "navbar"},
		// Content links (PriorityContent = 50)
		{Selector: "main a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: "article a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		// Footer (PriorityFooter = 20)
		{Selector: "footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
	}
	return ExtractLinksWithConfigs(html, baseURL, configs)
}
