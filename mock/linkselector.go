package mock

import (
	"time"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.LinkSelector = (*LinkSelector)(nil)

// LinkSelector is a mock implementation of locdoc.LinkSelector.
type LinkSelector struct {
	ExtractLinksFn func(html string, baseURL string) ([]locdoc.DiscoveredLink, error)
	NameFn         func() string
}

func (s *LinkSelector) ExtractLinks(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
	return s.ExtractLinksFn(html, baseURL)
}

func (s *LinkSelector) Name() string {
	return s.NameFn()
}

var _ locdoc.FrameworkDetector = (*FrameworkDetector)(nil)

// FrameworkDetector is a mock implementation of locdoc.FrameworkDetector.
type FrameworkDetector struct {
	DetectFn func(html string) locdoc.Framework
}

func (d *FrameworkDetector) Detect(html string) locdoc.Framework {
	return d.DetectFn(html)
}

var _ locdoc.Prober = (*Prober)(nil)

// Prober is a mock implementation of locdoc.Prober.
type Prober struct {
	DetectFn      func(html string) locdoc.Framework
	RequiresJSFn  func(framework locdoc.Framework) (requires bool, known bool)
	RenderDelayFn func(framework locdoc.Framework) time.Duration
}

func (p *Prober) Detect(html string) locdoc.Framework {
	return p.DetectFn(html)
}

func (p *Prober) RequiresJS(framework locdoc.Framework) (requires bool, known bool) {
	return p.RequiresJSFn(framework)
}

func (p *Prober) RenderDelay(framework locdoc.Framework) time.Duration {
	if p.RenderDelayFn != nil {
		return p.RenderDelayFn(framework)
	}
	return 0
}

var _ locdoc.LinkSelectorRegistry = (*LinkSelectorRegistry)(nil)

// LinkSelectorRegistry is a mock implementation of locdoc.LinkSelectorRegistry.
type LinkSelectorRegistry struct {
	GetFn        func(framework locdoc.Framework) locdoc.LinkSelector
	GetForHTMLFn func(html string) locdoc.LinkSelector
	RegisterFn   func(framework locdoc.Framework, selector locdoc.LinkSelector)
	ListFn       func() []locdoc.Framework
}

func (r *LinkSelectorRegistry) Get(framework locdoc.Framework) locdoc.LinkSelector {
	return r.GetFn(framework)
}

func (r *LinkSelectorRegistry) GetForHTML(html string) locdoc.LinkSelector {
	return r.GetForHTMLFn(html)
}

func (r *LinkSelectorRegistry) Register(framework locdoc.Framework, selector locdoc.LinkSelector) {
	r.RegisterFn(framework, selector)
}

func (r *LinkSelectorRegistry) List() []locdoc.Framework {
	return r.ListFn()
}
