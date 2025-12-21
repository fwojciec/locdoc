package goquery

import (
	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*VuePressSelector)(nil)

// VuePressSelector extracts links from VuePress and VitePress documentation sites.
// Validated against VuePress v1.x/v2.x and VitePress v1.x.
//
// It supports both VuePress and VitePress:
// - .sidebar-links, .sidebar for VuePress classic
// - .VPSidebar, .VPNav for VitePress
// - .VPDocAsideOutline for VitePress TOC
type VuePressSelector struct{}

// NewVuePressSelector creates a new VuePressSelector.
func NewVuePressSelector() *VuePressSelector {
	return &VuePressSelector{}
}

// Name returns the selector's identifier.
func (s *VuePressSelector) Name() string {
	return "vuepress"
}

// ExtractLinks parses HTML and returns discovered links with priority.
// Links are deduplicated by URL, keeping the highest priority version.
// External links (different host than baseURL) are filtered out.
func (s *VuePressSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
	configs := []SelectorConfig{
		// TOC has highest priority (PriorityTOC = 110)
		// VitePress TOC
		{Selector: ".VPDocAsideOutline a[href]", Priority: locdoc.PriorityTOC, Source: "toc"},
		// Navigation sidebars (PriorityNavigation = 100)
		// VitePress
		{Selector: ".VPSidebar a[href]", Priority: locdoc.PriorityNavigation, Source: "sidebar"},
		{Selector: ".VPNav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		// VuePress classic
		{Selector: ".sidebar-links a[href]", Priority: locdoc.PriorityNavigation, Source: "sidebar"},
		{Selector: ".sidebar a[href]", Priority: locdoc.PriorityNavigation, Source: "sidebar"},
		// Content links (PriorityContent = 50)
		{Selector: ".theme-default-content a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: ".VPDoc a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		{Selector: "main a[href]", Priority: locdoc.PriorityContent, Source: "content"},
		// Footer (PriorityFooter = 20)
		{Selector: "footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
	}
	return ExtractLinksWithConfigs(html, baseURL, configs)
}
