package goquery

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fwojciec/locdoc"
)

// Detector identifies documentation frameworks from HTML content.
// It checks for framework-specific CSS classes, data attributes, meta tags,
// and structural markers that are unique to each documentation generator.
type Detector struct{}

// NewDetector creates a new Detector.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes HTML and returns the identified framework.
// Returns FrameworkUnknown if the framework cannot be determined.
func (d *Detector) Detect(html string) locdoc.Framework {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return locdoc.FrameworkUnknown
	}

	// Check meta generator tags first - most reliable when present
	if framework := d.detectFromMetaGenerator(doc); framework != locdoc.FrameworkUnknown {
		return framework
	}

	// Check for Docusaurus markers
	// __docusaurus_skipToContent_fallback is highly specific
	if d.hasSelector(doc, "#__docusaurus_skipToContent_fallback") ||
		d.hasSelector(doc, ".theme-doc-sidebar-container") ||
		d.hasSelector(doc, "[data-rh]") && d.hasSelector(doc, "[data-theme]") {
		return locdoc.FrameworkDocusaurus
	}

	// Check for MkDocs Material markers
	// data-md-color-* attributes are unique to MkDocs Material
	if d.hasSelector(doc, "[data-md-color-scheme]") ||
		d.hasSelector(doc, "[data-md-component]") ||
		d.hasSelector(doc, ".md-nav--primary") {
		return locdoc.FrameworkMkDocs
	}

	// Check for Sphinx markers (including ReadTheDocs theme)
	if d.hasSelector(doc, ".toctree-wrapper") ||
		d.hasSelector(doc, ".wy-nav-side") ||
		d.hasSelector(doc, ".wy-menu-vertical") ||
		d.hasSelector(doc, ".sphinxsidebar") {
		return locdoc.FrameworkSphinx
	}

	// Check for VitePress markers (before VuePress since VitePress is a VuePress successor)
	// #VPContent and .VPDoc are unique to VitePress
	if d.hasSelector(doc, "#VPContent") ||
		d.hasSelector(doc, ".VPDoc") ||
		d.hasSelector(doc, ".VPDocAsideOutline") {
		return locdoc.FrameworkVitePress
	}

	// Check for VuePress markers
	if d.hasSelector(doc, ".theme-default-content") ||
		d.hasSelector(doc, ".sidebar-links") ||
		d.hasSelector(doc, ".vuepress-navbar") {
		return locdoc.FrameworkVuePress
	}

	// Check for GitBook markers
	// GitBook uses specific classes on html element: circular-corners, theme-clean, tint
	if d.hasSelector(doc, "[data-testid='space.sidebar']") ||
		d.hasSelector(doc, "[data-testid='page.desktopTableOfContents']") ||
		d.hasGitBookClasses(doc) {
		return locdoc.FrameworkGitBook
	}

	// Check for Nextra markers
	if d.hasSelector(doc, ".nextra-navbar") ||
		d.hasSelector(doc, ".nextra-sidebar") ||
		d.hasSelector(doc, ".nextra-toc") {
		return locdoc.FrameworkNextra
	}

	return locdoc.FrameworkUnknown
}

// detectFromMetaGenerator checks the meta generator tag for framework identification.
func (d *Detector) detectFromMetaGenerator(doc *goquery.Document) locdoc.Framework {
	generator := ""
	doc.Find("meta[name='generator']").Each(func(_ int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			generator = strings.ToLower(content)
		}
	})

	if generator == "" {
		return locdoc.FrameworkUnknown
	}

	switch {
	case strings.Contains(generator, "sphinx"):
		return locdoc.FrameworkSphinx
	case strings.Contains(generator, "gitbook"):
		return locdoc.FrameworkGitBook
	case strings.Contains(generator, "docusaurus"):
		return locdoc.FrameworkDocusaurus
	case strings.Contains(generator, "mkdocs"):
		return locdoc.FrameworkMkDocs
	case strings.Contains(generator, "vitepress"):
		return locdoc.FrameworkVitePress
	case strings.Contains(generator, "vuepress"):
		return locdoc.FrameworkVuePress
	case strings.Contains(generator, "nextra"):
		return locdoc.FrameworkNextra
	}

	return locdoc.FrameworkUnknown
}

// hasSelector checks if the document contains at least one element matching the selector.
func (d *Detector) hasSelector(doc *goquery.Document, selector string) bool {
	return doc.Find(selector).Length() > 0
}

// hasGitBookClasses checks for GitBook-specific classes on the html element.
// GitBook uses a combination of: circular-corners, theme-clean, tint
func (d *Detector) hasGitBookClasses(doc *goquery.Document) bool {
	htmlClass := ""
	doc.Find("html").Each(func(_ int, s *goquery.Selection) {
		if class, exists := s.Attr("class"); exists {
			htmlClass = class
		}
	})

	if htmlClass == "" {
		return false
	}

	// GitBook has a distinctive combination of classes
	hasCircularCorners := strings.Contains(htmlClass, "circular-corners")
	hasThemeClean := strings.Contains(htmlClass, "theme-clean")
	hasTint := strings.Contains(htmlClass, "tint")

	// Require at least two of these GitBook-specific classes
	count := 0
	if hasCircularCorners {
		count++
	}
	if hasThemeClean {
		count++
	}
	if hasTint {
		count++
	}

	return count >= 2
}
