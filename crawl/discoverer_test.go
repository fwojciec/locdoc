package crawl_test

import (
	"context"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverer_DiscoverURLs(t *testing.T) {
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

		prober := &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkSphinx
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, true
			},
		}

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Test", ContentHTML: "<p>Test</p>"}, nil
			},
		}

		d := &crawl.Discoverer{
			HTTPFetcher:   fetcher,
			RodFetcher:    fetcher,
			Prober:        prober,
			Extractor:     extractor,
			LinkSelectors: linkSelectors,
			RateLimiter:   rateLimiter,
		}
		urls, err := d.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
		)

		require.NoError(t, err)
		assert.Len(t, urls, 4)
		assert.Contains(t, urls, "https://example.com/docs/")
		assert.Contains(t, urls, "https://example.com/docs/page1")
		assert.Contains(t, urls, "https://example.com/docs/page2")
		assert.Contains(t, urls, "https://example.com/docs/page3")
	})

	t.Run("respects concurrency setting", func(t *testing.T) {
		t.Parallel()

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(_ string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(_ string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						if baseURL == "https://example.com/docs/" {
							return []locdoc.DiscoveredLink{
								{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
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

		prober := &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkSphinx
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, true
			},
		}

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Test", ContentHTML: "<p>Test</p>"}, nil
			},
		}

		d := &crawl.Discoverer{
			HTTPFetcher:   fetcher,
			RodFetcher:    fetcher,
			Prober:        prober,
			Extractor:     extractor,
			LinkSelectors: linkSelectors,
			RateLimiter:   rateLimiter,
			Concurrency:   2,
		}
		urls, err := d.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
		)

		require.NoError(t, err)
		assert.Len(t, urls, 2) // source + page1
	})

	t.Run("uses custom retry delays", func(t *testing.T) {
		t.Parallel()

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(_ string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(_ string, _ string) ([]locdoc.DiscoveredLink, error) {
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

		prober := &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkSphinx
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, true
			},
		}

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Test", ContentHTML: "<p>Test</p>"}, nil
			},
		}

		d := &crawl.Discoverer{
			HTTPFetcher:   fetcher,
			RodFetcher:    fetcher,
			Prober:        prober,
			Extractor:     extractor,
			LinkSelectors: linkSelectors,
			RateLimiter:   rateLimiter,
			RetryDelays:   []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
		}
		urls, err := d.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
		)

		require.NoError(t, err)
		assert.Len(t, urls, 1) // just source
	})
}
