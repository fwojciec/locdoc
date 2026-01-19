package locdoc

import "time"

// LinkPriority represents crawl priority (higher = more important).
type LinkPriority int

// Link priority levels for crawl ordering.
const (
	PriorityIgnore     LinkPriority = 0
	PriorityFallback   LinkPriority = 10
	PriorityFooter     LinkPriority = 20
	PriorityContent    LinkPriority = 50
	PriorityNavigation LinkPriority = 100
	PriorityTOC        LinkPriority = 110
)

// DiscoveredLink represents a URL with priority metadata.
type DiscoveredLink struct {
	URL      string
	Priority LinkPriority
	Text     string
	Source   string // "nav", "sidebar", "content", "footer"
}

// Framework identifies a documentation framework.
type Framework string

// Supported documentation frameworks.
const (
	FrameworkUnknown    Framework = ""
	FrameworkDocusaurus Framework = "docusaurus"
	FrameworkMkDocs     Framework = "mkdocs"
	FrameworkSphinx     Framework = "sphinx"
	FrameworkVuePress   Framework = "vuepress"
	FrameworkVitePress  Framework = "vitepress"
	FrameworkGitBook    Framework = "gitbook"
	FrameworkNextra     Framework = "nextra"
	FrameworkZeroheight Framework = "zeroheight"
)

// LinkSelector extracts prioritized links from HTML.
type LinkSelector interface {
	// ExtractLinks parses HTML and returns discovered links with priority.
	// The baseURL is used to resolve relative URLs.
	ExtractLinks(html string, baseURL string) ([]DiscoveredLink, error)

	// Name returns the selector's identifier (e.g., "docusaurus", "generic").
	Name() string
}

// FrameworkDetector identifies documentation frameworks from HTML.
type FrameworkDetector interface {
	// Detect analyzes HTML and returns the identified framework.
	// Returns FrameworkUnknown if the framework cannot be determined.
	Detect(html string) Framework
}

// Prober identifies documentation frameworks and determines their rendering requirements.
type Prober interface {
	FrameworkDetector

	// RequiresJS indicates whether a framework requires JavaScript rendering.
	// Returns (requires, known) where:
	//   - requires: true if the framework needs JS to render content
	//   - known: true if the framework is recognized
	// Unknown frameworks return (false, false).
	RequiresJS(framework Framework) (requires bool, known bool)

	// RenderDelay returns the recommended delay after page load for a framework.
	// Some SPA frameworks need additional time for async content to render.
	// Returns 0 for frameworks that don't need extra delay.
	RenderDelay(framework Framework) time.Duration
}

// LinkSelectorRegistry manages framework-specific selectors.
type LinkSelectorRegistry interface {
	// Get returns the selector for a specific framework.
	// Returns nil if no selector is registered for the framework.
	Get(framework Framework) LinkSelector

	// GetForHTML detects the framework from HTML and returns the appropriate selector.
	// Falls back to a generic selector if the framework is unknown.
	GetForHTML(html string) LinkSelector

	// Register adds a selector for a framework.
	Register(framework Framework, selector LinkSelector)

	// List returns all registered frameworks.
	List() []Framework
}
