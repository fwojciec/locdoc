package slog

import (
	"log/slog"
	"time"

	"github.com/fwojciec/locdoc"
)

// Ensure LoggingRegistry implements locdoc.LinkSelectorRegistry.
var _ locdoc.LinkSelectorRegistry = (*LoggingRegistry)(nil)

// LoggingRegistry wraps a LinkSelectorRegistry with debug logging for framework detection.
type LoggingRegistry struct {
	next     locdoc.LinkSelectorRegistry
	detector locdoc.FrameworkDetector
	logger   *slog.Logger
}

// NewLoggingRegistry creates a new LoggingRegistry.
func NewLoggingRegistry(next locdoc.LinkSelectorRegistry, detector locdoc.FrameworkDetector, logger *slog.Logger) *LoggingRegistry {
	return &LoggingRegistry{next: next, detector: detector, logger: logger}
}

// Get delegates to the wrapped registry.
func (r *LoggingRegistry) Get(framework locdoc.Framework) locdoc.LinkSelector {
	return r.next.Get(framework)
}

// GetForHTML detects the framework, logs it, and returns the appropriate selector.
func (r *LoggingRegistry) GetForHTML(html string) locdoc.LinkSelector {
	begin := time.Now()
	framework := r.detector.Detect(html)
	frameworkName := string(framework)
	if framework == locdoc.FrameworkUnknown {
		frameworkName = "(unknown)"
	}
	r.logger.Info("framework detection",
		"framework", frameworkName,
		"duration", time.Since(begin),
	)
	return r.next.GetForHTML(html)
}

// Register delegates to the wrapped registry.
func (r *LoggingRegistry) Register(framework locdoc.Framework, selector locdoc.LinkSelector) {
	r.next.Register(framework, selector)
}

// List delegates to the wrapped registry.
func (r *LoggingRegistry) List() []locdoc.Framework {
	return r.next.List()
}
