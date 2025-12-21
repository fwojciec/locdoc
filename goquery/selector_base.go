// Package goquery provides HTML link extraction using CSS selectors.
//
// This package implements the locdoc.LinkSelector interface for extracting
// prioritized links from HTML documents. It uses goquery for CSS selector
// matching and supports different priority levels based on where links appear
// in the document structure (nav, aside, main/article, footer).
package goquery

import (
	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*BaseSelector)(nil)

// BaseSelector implements the base link extraction logic using CSS selectors.
// It extracts links from common HTML structural elements and assigns priorities
// based on their location in the document.
type BaseSelector struct{}

// NewBaseSelector creates a new BaseSelector.
func NewBaseSelector() *BaseSelector {
	return &BaseSelector{}
}

// Name returns the selector's identifier.
func (s *BaseSelector) Name() string {
	return "base"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
// The returned links maintain document order based on first occurrence.
func (s *BaseSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
	configs := []SelectorConfig{
		{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		{Selector: "aside a[href]", Priority: locdoc.PriorityTOC, Source: "sidebar"},
		{Selector: "main a[href], article a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: "footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
	}
	return ExtractLinksWithConfigs(html, baseURL, configs)
}
