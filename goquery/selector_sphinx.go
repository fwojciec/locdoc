package goquery

import (
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
	configs := []SelectorConfig{
		// TOC has highest priority (PriorityTOC = 110)
		{Selector: ".toctree-wrapper a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		{Selector: "#localtoc a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		// Navigation sidebars (PriorityNavigation = 100)
		// ReadTheDocs theme
		{Selector: ".wy-nav-side a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		{Selector: ".wy-menu-vertical a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		// Classic Sphinx theme
		{Selector: ".sphinxsidebar a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		// Content links (PriorityContent = 50)
		{Selector: ".document a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: ".body a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: "article a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		// Footer (PriorityFooter = 20)
		{Selector: "footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
	}
	return ExtractLinksWithConfigs(html, baseURL, configs)
}
