package main_test

import (
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/docfetch"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Story: Composite URL Discovery
// The source tries multiple strategies to find documentation URLs

func TestCompositeSource_UsesSitemapWhenAvailable(t *testing.T) {
	t.Parallel()

	// Given a sitemap service returns URLs
	sitemap := &mock.SitemapService{
		DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}, nil
		},
	}
	source := main.NewCompositeSource(sitemap, nil)

	// When I discover URLs
	urls, err := source.Discover(context.Background(), "https://example.com")

	// Then sitemap URLs are returned
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}, urls)
}

func TestCompositeSource_FallsBackToRecursiveWhenSitemapEmpty(t *testing.T) {
	t.Parallel()

	// Given sitemap returns no URLs
	sitemap := &mock.SitemapService{
		DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{}, nil
		},
	}
	// And recursive discoverer finds some
	recursive := &mockRecursiveDiscoverer{
		urls: []string{"https://example.com/x", "https://example.com/y"},
	}
	source := main.NewCompositeSource(sitemap, recursive)

	// When I discover URLs
	urls, err := source.Discover(context.Background(), "https://example.com")

	// Then recursive URLs are returned
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/x", "https://example.com/y"}, urls)
}

// mockRecursiveDiscoverer is a test helper that implements main.RecursiveDiscoverer.
type mockRecursiveDiscoverer struct {
	urls []string
	err  error
}

func (m *mockRecursiveDiscoverer) DiscoverURLs(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
	return m.urls, m.err
}

func TestCompositeSource_ReturnsEmptyWhenBothFail(t *testing.T) {
	t.Parallel()

	// Given both discovery methods find nothing
	sitemap := &mock.SitemapService{
		DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{}, nil
		},
	}
	recursive := &mockRecursiveDiscoverer{
		urls: []string{},
	}
	source := main.NewCompositeSource(sitemap, recursive)

	// When I discover URLs
	urls, err := source.Discover(context.Background(), "https://example.com")

	// Then empty list is returned (not an error)
	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestCompositeSource_PropagatesSitemapError(t *testing.T) {
	t.Parallel()

	// Given sitemap service returns an error
	sitemap := &mock.SitemapService{
		DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return nil, assert.AnError
		},
	}
	source := main.NewCompositeSource(sitemap, nil)

	// When I discover URLs
	_, err := source.Discover(context.Background(), "https://example.com")

	// Then the error is returned
	assert.Error(t, err)
}

func TestCompositeSource_PropagatesRecursiveError(t *testing.T) {
	t.Parallel()

	// Given sitemap returns empty
	sitemap := &mock.SitemapService{
		DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{}, nil
		},
	}
	// And recursive discoverer returns an error
	recursive := &mockRecursiveDiscoverer{
		err: assert.AnError,
	}
	source := main.NewCompositeSource(sitemap, recursive)

	// When I discover URLs
	_, err := source.Discover(context.Background(), "https://example.com")

	// Then the error is returned
	assert.Error(t, err)
}
