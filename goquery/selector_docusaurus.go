package goquery

import (
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
	configs := []SelectorConfig{
		// TOC has highest priority (PriorityTOC = 110)
		{Selector: ".table-of-contents a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		// Sidebar navigation (PriorityNavigation = 100)
		{Selector: ".theme-doc-sidebar-container a[href]", Priority: locdoc.PriorityNavigation, Source: "sidebar"},
		{Selector: "nav.navbar a[href]", Priority: locdoc.PriorityNavigation, Source: "navbar"},
		// Content links (PriorityContent = 50)
		{Selector: "article a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: "main a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		// Footer (PriorityFooter = 20)
		{Selector: "footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
	}
	return ExtractLinksWithConfigs(html, baseURL, configs)
}
