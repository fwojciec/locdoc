package crawl_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverURLs(t *testing.T) {
	t.Parallel()

	t.Run("discovers URLs recursively from source", func(t *testing.T) {
		t.Parallel()

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				if url == "https://example.com/docs/" {
					return `<html><body><nav><a href="/docs/page1">Page 1</a><a href="/docs/page2">Page 2</a></nav></body></html>`, nil
				}
				if url == "https://example.com/docs/page1" {
					return `<html><body><nav><a href="/docs/page3">Page 3</a></nav></body></html>`, nil
				}
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						if baseURL == "https://example.com/docs/" {
							return []locdoc.DiscoveredLink{
								{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
								{URL: "https://example.com/docs/page2", Priority: locdoc.PriorityNavigation},
							}, nil
						}
						if baseURL == "https://example.com/docs/page1" {
							return []locdoc.DiscoveredLink{
								{URL: "https://example.com/docs/page3", Priority: locdoc.PriorityNavigation},
							}, nil
						}
						return nil, nil
					},
					NameFn: func() string { return "test" },
				}
			},
		}

		rateLimiter := &mock.DomainLimiter{
			WaitFn: func(_ context.Context, _ string) error {
				return nil
			},
		}

		urls, err := crawl.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
			fetcher,
			linkSelectors,
			rateLimiter,
		)

		require.NoError(t, err)
		assert.Len(t, urls, 4)
		assert.Contains(t, urls, "https://example.com/docs/")
		assert.Contains(t, urls, "https://example.com/docs/page1")
		assert.Contains(t, urls, "https://example.com/docs/page2")
		assert.Contains(t, urls, "https://example.com/docs/page3")
	})

	t.Run("respects path prefix scope", func(t *testing.T) {
		t.Parallel()

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						// Return links both inside and outside scope
						return []locdoc.DiscoveredLink{
							{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
							{URL: "https://example.com/other/page", Priority: locdoc.PriorityNavigation},  // Out of scope
							{URL: "https://different.com/docs/page", Priority: locdoc.PriorityNavigation}, // Different host
						}, nil
					},
					NameFn: func() string { return "test" },
				}
			},
		}

		rateLimiter := &mock.DomainLimiter{
			WaitFn: func(_ context.Context, _ string) error {
				return nil
			},
		}

		urls, err := crawl.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
			fetcher,
			linkSelectors,
			rateLimiter,
		)

		require.NoError(t, err)
		// Should only include source and in-scope page
		assert.Len(t, urls, 2)
		assert.Contains(t, urls, "https://example.com/docs/")
		assert.Contains(t, urls, "https://example.com/docs/page1")
		assert.NotContains(t, urls, "https://example.com/other/page")
		assert.NotContains(t, urls, "https://different.com/docs/page")
	})

	t.Run("applies URL filter", func(t *testing.T) {
		t.Parallel()

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						return []locdoc.DiscoveredLink{
							{URL: "https://example.com/docs/api/v1", Priority: locdoc.PriorityNavigation},
							{URL: "https://example.com/docs/guide/intro", Priority: locdoc.PriorityNavigation},
						}, nil
					},
					NameFn: func() string { return "test" },
				}
			},
		}

		rateLimiter := &mock.DomainLimiter{
			WaitFn: func(_ context.Context, _ string) error {
				return nil
			},
		}

		filter := &locdoc.URLFilter{
			Include: []*regexp.Regexp{regexp.MustCompile(`/api/`)},
		}

		urls, err := crawl.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			filter,
			fetcher,
			linkSelectors,
			rateLimiter,
		)

		require.NoError(t, err)
		// Source is always included, plus filtered matches
		assert.Len(t, urls, 2)
		assert.Contains(t, urls, "https://example.com/docs/")
		assert.Contains(t, urls, "https://example.com/docs/api/v1")
		assert.NotContains(t, urls, "https://example.com/docs/guide/intro")
	})

	t.Run("skips failed fetches without error", func(t *testing.T) {
		t.Parallel()

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				if url == "https://example.com/docs/" {
					return `<html><body></body></html>`, nil
				}
				return "", locdoc.Errorf(locdoc.ENOTFOUND, "not found")
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						return []locdoc.DiscoveredLink{
							{URL: "https://example.com/docs/missing", Priority: locdoc.PriorityNavigation},
						}, nil
					},
					NameFn: func() string { return "test" },
				}
			},
		}

		rateLimiter := &mock.DomainLimiter{
			WaitFn: func(_ context.Context, _ string) error {
				return nil
			},
		}

		urls, err := crawl.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
			fetcher,
			linkSelectors,
			rateLimiter,
		)

		require.NoError(t, err)
		// Only source is included, failed fetch is skipped
		assert.Len(t, urls, 1)
		assert.Contains(t, urls, "https://example.com/docs/")
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						// Generate many links to ensure we'd normally continue
						return []locdoc.DiscoveredLink{
							{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
							{URL: "https://example.com/docs/page2", Priority: locdoc.PriorityNavigation},
						}, nil
					},
					NameFn: func() string { return "test" },
				}
			},
		}

		rateLimiter := &mock.DomainLimiter{
			WaitFn: func(ctx context.Context, _ string) error {
				// Cancel after first rate limit wait
				cancel()
				return ctx.Err()
			},
		}

		urls, err := crawl.DiscoverURLs(
			ctx,
			"https://example.com/docs/",
			nil,
			fetcher,
			linkSelectors,
			rateLimiter,
		)

		require.NoError(t, err)
		// Should stop early due to cancellation
		assert.Empty(t, urls)
	})
}
