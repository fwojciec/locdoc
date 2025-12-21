package goquery

import (
	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*MkDocsSelector)(nil)

// MkDocsSelector extracts links from MkDocs Material documentation sites.
// Validated against MkDocs Material v8.x and v9.x.
//
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
	configs := []SelectorConfig{
		// TOC has highest priority (PriorityTOC = 110)
		{Selector: ".md-sidebar--secondary a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		{Selector: "[data-md-component='toc'] a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		// Primary navigation (PriorityNavigation = 100)
		{Selector: ".md-nav--primary a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		{Selector: "[data-md-component='navigation'] a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		// Content links (PriorityContent = 50)
		{Selector: ".md-content a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: "article a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		// Footer (PriorityFooter = 20)
		{Selector: "footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
	}
	return ExtractLinksWithConfigs(html, baseURL, configs)
}
