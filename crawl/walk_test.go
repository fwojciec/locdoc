package crawl_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecursiveCrawl_Concurrency(t *testing.T) {
	t.Parallel()

	t.Run("processes URLs in parallel with multiple workers", func(t *testing.T) {
		t.Parallel()

		// Track concurrent fetch count using atomics to avoid data races
		var maxConcurrent atomic.Int32
		var currentConcurrent atomic.Int32

		// Create enough URLs to see parallelism
		const numPages = 10
		const concurrency = 3

		c, m := newTestCrawler()
		c.Concurrency = concurrency
		m.Fetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			// Track concurrent fetches using atomic compare-and-swap for max
			current := currentConcurrent.Add(1)
			for {
				max := maxConcurrent.Load()
				if current <= max || maxConcurrent.CompareAndSwap(max, current) {
					break
				}
			}

			// Simulate work to allow concurrency to build up
			time.Sleep(50 * time.Millisecond)

			currentConcurrent.Add(-1)
			return `<html><body><p>Content</p></body></html>`, nil
		}
		m.LinkSelectors.GetForHTMLFn = func(_ string) locdoc.LinkSelector {
			return &mock.LinkSelector{
				ExtractLinksFn: func(_ string, baseURL string) ([]locdoc.DiscoveredLink, error) {
					// Only the seed page discovers links
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
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, numPages+1, result.Saved, "should save seed URL and all discovered pages")

		// The key assertion: we should see concurrent processing
		// With concurrency=3, we should see at least 2 concurrent fetches at some point
		assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(2),
			"expected at least 2 concurrent fetches, got %d (should see parallelism with concurrency=%d)",
			maxConcurrent.Load(), concurrency)
	})

	t.Run("respects max URL limit with concurrent workers", func(t *testing.T) {
		t.Parallel()

		var fetchCount atomic.Int32

		c, m := newTestCrawler()
		c.Concurrency = 5
		m.Fetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			fetchCount.Add(1)
			return `<html><body><p>Content</p></body></html>`, nil
		}
		m.LinkSelectors.GetForHTMLFn = func(_ string) locdoc.LinkSelector {
			return &mock.LinkSelector{
				ExtractLinksFn: func(_ string, _ string) ([]locdoc.DiscoveredLink, error) {
					// Always return more links than the max URL limit
					// This would cause infinite crawling without the limit
					var links []locdoc.DiscoveredLink
					for i := 0; i < 100; i++ {
						links = append(links, locdoc.DiscoveredLink{
							URL:      fmt.Sprintf("https://example.com/docs/page%d_%d", fetchCount.Load(), i),
							Priority: locdoc.PriorityNavigation,
						})
					}
					return links, nil
				},
				NameFn: func() string { return "test" },
			}
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)

		// The max URL limit is 1000 (maxRecursiveCrawlURLs constant)
		// We should not process more than that
		assert.LessOrEqual(t, int(fetchCount.Load()), 1000,
			"should not process more than maxRecursiveCrawlURLs (1000)")
		assert.LessOrEqual(t, result.Saved, 1000,
			"should not save more than maxRecursiveCrawlURLs (1000)")
	})

	t.Run("rate limiter enforced per worker", func(t *testing.T) {
		t.Parallel()

		var waitCalls atomic.Int32

		c, m := newTestCrawler()
		c.Concurrency = 3
		m.LinkSelectors.GetForHTMLFn = func(_ string) locdoc.LinkSelector {
			return &mock.LinkSelector{
				ExtractLinksFn: func(_ string, baseURL string) ([]locdoc.DiscoveredLink, error) {
					if baseURL == "https://example.com/docs/" {
						return []locdoc.DiscoveredLink{
							{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
							{URL: "https://example.com/docs/page2", Priority: locdoc.PriorityNavigation},
							{URL: "https://example.com/docs/page3", Priority: locdoc.PriorityNavigation},
						}, nil
					}
					return nil, nil
				},
				NameFn: func() string { return "test" },
			}
		}
		m.RateLimiter.WaitFn = func(_ context.Context, _ string) error {
			waitCalls.Add(1)
			return nil
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 4, result.Saved, "should save seed + 3 pages")

		// Rate limiter should be called for each URL
		assert.Equal(t, int32(4), waitCalls.Load(),
			"rate limiter should be called once per URL")
	})
}
