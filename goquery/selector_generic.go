package goquery

import (
	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*GenericSelector)(nil)

// GenericSelector implements link extraction using universal CSS selectors
// that work across any documentation framework. It uses common HTML patterns
// and class names to identify navigation, TOC, content, and footer areas.
type GenericSelector struct{}

// NewGenericSelector creates a new GenericSelector.
func NewGenericSelector() *GenericSelector {
	return &GenericSelector{}
}

// Name returns the selector's identifier.
func (s *GenericSelector) Name() string {
	return "generic"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
//
// Priority order (highest to lowest):
//   - TOC: .toc, .sidebar, .table-of-contents, aside
//   - Navigation: nav, [role="navigation"], .nav, .menu, .navbar
//   - Content: main, article, .content, .doc-content
//   - Footer: footer, .footer
//   - Fallback: a[href] matching base URL path (catches links in non-semantic HTML)
func (s *GenericSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
	configs := []SelectorConfig{
		// TOC selectors (highest priority after sitemap)
		{Selector: ".toc a[href], .table-of-contents a[href], .sidebar a[href], aside a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		// Navigation selectors
		{Selector: "nav a[href], [role=\"navigation\"] a[href], .nav a[href], .menu a[href], .navbar a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		// Content selectors
		{Selector: "main a[href], article a[href], .content a[href], .doc-content a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		// Footer selectors
		{Selector: "footer a[href], .footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
	}
	// Use the fallback variant to also extract links matching the base URL path prefix
	return ExtractLinksWithConfigsAndFallback(html, baseURL, configs)
}
