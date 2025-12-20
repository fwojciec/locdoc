package crawl_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverURLs(t *testing.T) {
	t.Parallel()

	t.Run("uses default concurrency of 3 when not specified", func(t *testing.T) {
		t.Parallel()

		// Track concurrent fetch count using atomics
		var maxConcurrent atomic.Int32
		var currentConcurrent atomic.Int32

		const numPages = 10

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				current := currentConcurrent.Add(1)
				for {
					max := maxConcurrent.Load()
					if current <= max || maxConcurrent.CompareAndSwap(max, current) {
						break
					}
				}

				// Simulate work to allow concurrency to build up
				time.Sleep(20 * time.Millisecond)
				currentConcurrent.Add(-1)
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(_ string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(_ string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						if baseURL == "https://example.com/docs/" {
							var links []locdoc.DiscoveredLink
							for i := 1; i <= numPages; i++ {
								links = append(links, locdoc.DiscoveredLink{
									URL:      fmt.Sprintf("https://example.com/docs/page%d", i),
									Priority: locdoc.PriorityNavigation,
								})
							}
							return links, nil
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

		// Call without WithConcurrency option - should use default of 3
		urls, err := crawl.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
			fetcher,
			linkSelectors,
			rateLimiter,
		)

		require.NoError(t, err)
		assert.Len(t, urls, numPages+1)

		// Default concurrency is 3, so we should never see more than 3 concurrent fetches
		assert.LessOrEqual(t, maxConcurrent.Load(), int32(3),
			"default concurrency should be 3, got max concurrent of %d", maxConcurrent.Load())
	})

	t.Run("respects concurrency setting", func(t *testing.T) {
		t.Parallel()

		// Track concurrent fetch count using atomics
		var maxConcurrent atomic.Int32
		var currentConcurrent atomic.Int32

		const numPages = 10
		const concurrency = 2

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				current := currentConcurrent.Add(1)
				for {
					max := maxConcurrent.Load()
					if current <= max || maxConcurrent.CompareAndSwap(max, current) {
						break
					}
				}

				// Simulate work to allow concurrency to build up
				time.Sleep(20 * time.Millisecond)
				currentConcurrent.Add(-1)
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(_ string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(_ string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						if baseURL == "https://example.com/docs/" {
							var links []locdoc.DiscoveredLink
							for i := 1; i <= numPages; i++ {
								links = append(links, locdoc.DiscoveredLink{
									URL:      fmt.Sprintf("https://example.com/docs/page%d", i),
									Priority: locdoc.PriorityNavigation,
								})
							}
							return links, nil
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
			crawl.WithConcurrency(concurrency),
		)

		require.NoError(t, err)
		assert.Len(t, urls, numPages+1)

		// With concurrency=2, we should never see more than 2 concurrent fetches
		assert.LessOrEqual(t, maxConcurrent.Load(), int32(concurrency),
			"should not exceed concurrency limit of %d, got %d", concurrency, maxConcurrent.Load())
	})

	t.Run("retries failed fetches", func(t *testing.T) {
		t.Parallel()

		attempts := make(map[string]int)
		var mu sync.Mutex

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				mu.Lock()
				attempts[url]++
				count := attempts[url]
				mu.Unlock()

				// Fail first 2 attempts for page1
				if url == "https://example.com/docs/page1" && count < 3 {
					return "", errors.New("timeout")
				}
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

		// Use zero delays for fast tests
		noDelays := []time.Duration{0, 0, 0}

		urls, err := crawl.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
			fetcher,
			linkSelectors,
			rateLimiter,
			crawl.WithRetryDelays(noDelays),
		)

		require.NoError(t, err)
		// Should include both pages - page1 succeeds on 3rd attempt
		assert.Len(t, urls, 2)
		assert.Contains(t, urls, "https://example.com/docs/")
		assert.Contains(t, urls, "https://example.com/docs/page1")

		// Verify page1 was attempted 3 times
		mu.Lock()
		page1Attempts := attempts["https://example.com/docs/page1"]
		mu.Unlock()
		assert.Equal(t, 3, page1Attempts, "page1 should be retried")
	})

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

		// Use zero delays for fast tests
		noDelays := []time.Duration{0, 0, 0}

		urls, err := crawl.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
			fetcher,
			linkSelectors,
			rateLimiter,
			crawl.WithRetryDelays(noDelays),
		)

		require.NoError(t, err)
		// Only source is included, failed fetch is skipped (after all retries)
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

	t.Run("calls OnURL callback for each discovered URL", func(t *testing.T) {
		t.Parallel()

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
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

		// Track URLs as they are streamed
		var streamedURLs []string
		var mu sync.Mutex

		urls, err := crawl.DiscoverURLs(
			context.Background(),
			"https://example.com/docs/",
			nil,
			fetcher,
			linkSelectors,
			rateLimiter,
			crawl.WithOnURL(func(url string) {
				mu.Lock()
				streamedURLs = append(streamedURLs, url)
				mu.Unlock()
			}),
		)

		require.NoError(t, err)
		assert.Len(t, urls, 3) // source + 2 pages
		assert.Len(t, streamedURLs, 3)
		assert.Contains(t, streamedURLs, "https://example.com/docs/")
		assert.Contains(t, streamedURLs, "https://example.com/docs/page1")
		assert.Contains(t, streamedURLs, "https://example.com/docs/page2")
	})
}
