package goquery

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

// VuePressSelector extracts links from VuePress and VitePress documentation sites.
// It supports both VuePress 1.x/2.x and VitePress:
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

	// Extract from VuePress/VitePress selectors in priority order
	// TOC has highest priority (PriorityTOC = 110)
	// VitePress TOC
	extractLinks(".VPDocAsideOutline a[href]", locdoc.PriorityTOC, "toc")

	// Navigation sidebars (PriorityNavigation = 100)
	// VitePress
	extractLinks(".VPSidebar a[href]", locdoc.PriorityNavigation, "sidebar")
	extractLinks(".VPNav a[href]", locdoc.PriorityNavigation, "nav")
	// VuePress classic
	extractLinks(".sidebar-links a[href]", locdoc.PriorityNavigation, "sidebar")
	extractLinks(".sidebar a[href]", locdoc.PriorityNavigation, "sidebar")

	// Content links (PriorityContent = 50)
	extractLinks(".theme-default-content a[href]", locdoc.PriorityContent, "content")
	extractLinks(".VPDoc a[href]", locdoc.PriorityContent, "content")
	extractLinks("main a[href]", locdoc.PriorityContent, "content")

	// Footer (PriorityFooter = 20)
	extractLinks("footer a[href]", locdoc.PriorityFooter, "footer")

	return links, nil
}
