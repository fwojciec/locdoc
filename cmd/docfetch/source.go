package main

import (
	"context"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
)

// Compile-time interface verification.
var (
	_ locdoc.URLSource    = (*CompositeSource)(nil)
	_ RecursiveDiscoverer = (*DiscovererAdapter)(nil)
)

// RecursiveDiscoverer discovers URLs by recursively crawling a site.
type RecursiveDiscoverer interface {
	DiscoverURLs(ctx context.Context, sourceURL string, filter *locdoc.URLFilter) ([]string, error)
}

// DiscovererAdapter adapts crawl.Discoverer to the RecursiveDiscoverer interface.
type DiscovererAdapter struct {
	Discoverer *crawl.Discoverer
}

// DiscoverURLs calls the underlying Discoverer without options.
func (a *DiscovererAdapter) DiscoverURLs(ctx context.Context, sourceURL string, filter *locdoc.URLFilter) ([]string, error) {
	return a.Discoverer.DiscoverURLs(ctx, sourceURL, filter)
}

// CompositeSource implements locdoc.URLSource by trying sitemap discovery
// first and falling back to recursive crawling if the sitemap is empty.
type CompositeSource struct {
	sitemap   locdoc.SitemapService
	recursive RecursiveDiscoverer
}

// NewCompositeSource creates a new CompositeSource.
// The sitemap parameter is used for sitemap-based discovery.
// The recursive parameter is used when sitemap returns no URLs.
func NewCompositeSource(sitemap locdoc.SitemapService, recursive RecursiveDiscoverer) *CompositeSource {
	return &CompositeSource{
		sitemap:   sitemap,
		recursive: recursive,
	}
}

// Discover implements locdoc.URLSource.
func (s *CompositeSource) Discover(ctx context.Context, sourceURL string) ([]string, error) {
	urls, err := s.sitemap.DiscoverURLs(ctx, sourceURL, nil)
	if err != nil {
		return nil, err
	}

	if len(urls) > 0 {
		return urls, nil
	}

	// Fallback to recursive discovery
	if s.recursive != nil {
		return s.recursive.DiscoverURLs(ctx, sourceURL, nil)
	}

	return urls, nil
}
