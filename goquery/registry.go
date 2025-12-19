package goquery

import "github.com/fwojciec/locdoc"

var _ locdoc.LinkSelectorRegistry = (*Registry)(nil)

// Registry manages framework-specific link selectors and auto-detects
// frameworks from HTML content. It uses a FrameworkDetector to identify
// the documentation framework and returns the appropriate selector,
// falling back to a generic selector when the framework is unknown
// or no specific selector is registered.
type Registry struct {
	detector  locdoc.FrameworkDetector
	fallback  locdoc.LinkSelector
	selectors map[locdoc.Framework]locdoc.LinkSelector
}

// NewRegistry creates a new Registry with the given detector and fallback selector.
// The fallback selector is used when GetForHTML cannot find a specific selector
// for the detected framework.
func NewRegistry(detector locdoc.FrameworkDetector, fallback locdoc.LinkSelector) *Registry {
	return &Registry{
		detector:  detector,
		fallback:  fallback,
		selectors: make(map[locdoc.Framework]locdoc.LinkSelector),
	}
}

// Get returns the selector for a specific framework.
// Returns nil if no selector is registered for the framework.
func (r *Registry) Get(framework locdoc.Framework) locdoc.LinkSelector {
	return r.selectors[framework]
}

// GetForHTML detects the framework from HTML and returns the appropriate selector.
// Falls back to the fallback selector if the framework is unknown or no selector
// is registered for the detected framework.
func (r *Registry) GetForHTML(html string) locdoc.LinkSelector {
	framework := r.detector.Detect(html)
	if selector, ok := r.selectors[framework]; ok {
		return selector
	}
	return r.fallback
}

// Register adds a selector for a framework.
// If a selector is already registered for the framework, it is replaced.
func (r *Registry) Register(framework locdoc.Framework, selector locdoc.LinkSelector) {
	r.selectors[framework] = selector
}

// List returns all registered frameworks.
func (r *Registry) List() []locdoc.Framework {
	frameworks := make([]locdoc.Framework, 0, len(r.selectors))
	for f := range r.selectors {
		frameworks = append(frameworks, f)
	}
	return frameworks
}
