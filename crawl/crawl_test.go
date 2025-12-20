package crawl_test

import (
	"context"
	"fmt"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCrawler creates a Crawler with sensible test defaults.
// All mocks return minimal successful responses by default.
// Use the returned mocks struct to customize behavior for specific tests.
func newTestCrawler() (*crawl.Crawler, *testMocks) {
	m := &testMocks{
		Sitemaps: &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{}, nil
			},
		},
		Fetcher: &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return `<html><body><p>Content</p></body></html>`, nil
			},
		},
		Extractor: &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{
					Title:       "Test",
					ContentHTML: "<p>Content</p>",
				}, nil
			},
		},
		Converter: &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "Content", nil
			},
		},
		Documents: &mock.DocumentService{
			CreateDocumentFn: func(_ context.Context, _ *locdoc.Document) error {
				return nil
			},
		},
		TokenCounter: &mock.TokenCounter{
			CountTokensFn: func(_ context.Context, _ string) (int, error) {
				return 1, nil
			},
		},
		LinkSelectors: &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(_ string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(_ string, _ string) ([]locdoc.DiscoveredLink, error) {
						return nil, nil
					},
					NameFn: func() string { return "test" },
				}
			},
		},
		RateLimiter: &mock.DomainLimiter{
			WaitFn: func(_ context.Context, _ string) error {
				return nil
			},
		},
	}

	c := &crawl.Crawler{
		Sitemaps:      m.Sitemaps,
		Fetcher:       m.Fetcher,
		Extractor:     m.Extractor,
		Converter:     m.Converter,
		Documents:     m.Documents,
		TokenCounter:  m.TokenCounter,
		LinkSelectors: m.LinkSelectors,
		RateLimiter:   m.RateLimiter,
		Concurrency:   1,
		RetryDelays:   []time.Duration{0},
	}

	return c, m
}

// testMocks holds references to all mocks used by newTestCrawler.
// Tests can modify the function fields to customize behavior.
type testMocks struct {
	Sitemaps      *mock.SitemapService
	Fetcher       *mock.Fetcher
	Extractor     *mock.Extractor
	Converter     *mock.Converter
	Documents     *mock.DocumentService
	TokenCounter  *mock.TokenCounter
	LinkSelectors *mock.LinkSelectorRegistry
	RateLimiter   *mock.DomainLimiter
}

func TestCrawler_CrawlProject(t *testing.T) {
	t.Parallel()

	t.Run("returns zero result when sitemap returns no URLs and no LinkSelectors", func(t *testing.T) {
		t.Parallel()

		c := &crawl.Crawler{
			Sitemaps: &mock.SitemapService{
				DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
					return []string{}, nil
				},
			},
			Fetcher:      &mock.Fetcher{},
			Extractor:    &mock.Extractor{},
			Converter:    &mock.Converter{},
			Documents:    &mock.DocumentService{},
			TokenCounter: &mock.TokenCounter{},
			Concurrency:  10,
			RetryDelays:  []time.Duration{0}, // no delay for tests
			// Note: no LinkSelectors or RateLimiter - no fallback crawling
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.Saved)
		assert.Equal(t, 0, result.Failed)
		assert.Equal(t, 0, result.Bytes)
		assert.Equal(t, 0, result.Tokens)
	})

	t.Run("falls back to recursive crawl when sitemap returns no URLs", func(t *testing.T) {
		t.Parallel()

		var savedDocs []*locdoc.Document
		fetchCalls := 0

		c := &crawl.Crawler{
			Sitemaps: &mock.SitemapService{
				DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
					return []string{}, nil // No sitemap URLs
				},
			},
			Fetcher: &mock.Fetcher{
				FetchFn: func(_ context.Context, url string) (string, error) {
					fetchCalls++
					if url == "https://example.com/docs/" {
						// Return HTML with links to other pages
						return `<html><body>
							<nav><a href="/docs/page1">Page 1</a></nav>
							<p>Content</p>
						</body></html>`, nil
					}
					if url == "https://example.com/docs/page1" {
						return `<html><body><p>Page 1 content</p></body></html>`, nil
					}
					return "", locdoc.Errorf(locdoc.ENOTFOUND, "not found")
				},
			},
			Extractor: &mock.Extractor{
				ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
					return &locdoc.ExtractResult{
						Title:       "Test Page",
						ContentHTML: "<p>Content</p>",
					}, nil
				},
			},
			Converter: &mock.Converter{
				ConvertFn: func(_ string) (string, error) {
					return "Content", nil
				},
			},
			Documents: &mock.DocumentService{
				CreateDocumentFn: func(_ context.Context, doc *locdoc.Document) error {
					savedDocs = append(savedDocs, doc)
					return nil
				},
			},
			TokenCounter: &mock.TokenCounter{
				CountTokensFn: func(_ context.Context, text string) (int, error) {
					return len(text) / 4, nil
				},
			},
			LinkSelectors: &mock.LinkSelectorRegistry{
				GetForHTMLFn: func(html string) locdoc.LinkSelector {
					return &mock.LinkSelector{
						ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
							// Return a link to page1 from the main page
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
			},
			RateLimiter: &mock.DomainLimiter{
				WaitFn: func(_ context.Context, _ string) error {
					return nil
				},
			},
			Concurrency: 1,
			RetryDelays: []time.Duration{0},
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.Saved, "should save seed URL and discovered page")
		assert.Equal(t, 2, fetchCalls, "should fetch seed URL and discovered page")
	})

	t.Run("recursive crawl respects path prefix scope", func(t *testing.T) {
		t.Parallel()

		var savedURLs []string

		c, m := newTestCrawler()
		m.Documents.CreateDocumentFn = func(_ context.Context, doc *locdoc.Document) error {
			savedURLs = append(savedURLs, doc.SourceURL)
			return nil
		}
		m.LinkSelectors.GetForHTMLFn = func(_ string) locdoc.LinkSelector {
			return &mock.LinkSelector{
				ExtractLinksFn: func(_ string, _ string) ([]locdoc.DiscoveredLink, error) {
					// Return links - one in scope, one out of scope
					return []locdoc.DiscoveredLink{
						{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
						{URL: "https://example.com/other/page", Priority: locdoc.PriorityNavigation}, // out of scope
						{URL: "https://other.com/docs/page", Priority: locdoc.PriorityNavigation},    // different host
					}, nil
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
		// Should only save the seed URL and the in-scope page
		assert.Equal(t, 2, result.Saved)
		assert.Contains(t, savedURLs, "https://example.com/docs/")
		assert.Contains(t, savedURLs, "https://example.com/docs/page1")
		// Should NOT contain out-of-scope URLs
		for _, u := range savedURLs {
			assert.NotContains(t, u, "other.com")
			assert.NotContains(t, u, "/other/")
		}
	})

	t.Run("recursive crawl uses rate limiter", func(t *testing.T) {
		t.Parallel()

		var waitCalls []string

		c, m := newTestCrawler()
		m.RateLimiter.WaitFn = func(_ context.Context, domain string) error {
			waitCalls = append(waitCalls, domain)
			return nil
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
		}

		_, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		assert.Len(t, waitCalls, 1)
		assert.Equal(t, "example.com", waitCalls[0])
	})

	t.Run("recursive crawl applies URL filter", func(t *testing.T) {
		t.Parallel()

		var savedURLs []string

		c, m := newTestCrawler()
		m.Documents.CreateDocumentFn = func(_ context.Context, doc *locdoc.Document) error {
			savedURLs = append(savedURLs, doc.SourceURL)
			return nil
		}
		m.LinkSelectors.GetForHTMLFn = func(_ string) locdoc.LinkSelector {
			return &mock.LinkSelector{
				ExtractLinksFn: func(_ string, _ string) ([]locdoc.DiscoveredLink, error) {
					// Return links - one matches filter, one doesn't
					return []locdoc.DiscoveredLink{
						{URL: "https://example.com/docs/guide/intro", Priority: locdoc.PriorityNavigation},
						{URL: "https://example.com/docs/api/ref", Priority: locdoc.PriorityNavigation},
					}, nil
				},
				NameFn: func() string { return "test" },
			}
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
			Filter:    ".*/guide/.*", // Only allow URLs containing /guide/
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should save seed URL and only the /guide/ URL
		assert.Equal(t, 2, result.Saved)
		assert.Contains(t, savedURLs, "https://example.com/docs/")
		assert.Contains(t, savedURLs, "https://example.com/docs/guide/intro")
		// Should NOT contain /api/ URL
		for _, u := range savedURLs {
			assert.NotContains(t, u, "/api/")
		}
	})

	t.Run("recursive crawl stops on context cancellation", func(t *testing.T) {
		t.Parallel()

		fetchCount := 0
		ctx, cancel := context.WithCancel(context.Background())

		c, m := newTestCrawler()
		m.Fetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			fetchCount++
			return `<html><body><p>Content</p></body></html>`, nil
		}
		m.LinkSelectors.GetForHTMLFn = func(_ string) locdoc.LinkSelector {
			return &mock.LinkSelector{
				ExtractLinksFn: func(_ string, _ string) ([]locdoc.DiscoveredLink, error) {
					// Return many links to ensure there's work queued
					return []locdoc.DiscoveredLink{
						{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
						{URL: "https://example.com/docs/page2", Priority: locdoc.PriorityNavigation},
						{URL: "https://example.com/docs/page3", Priority: locdoc.PriorityNavigation},
					}, nil
				},
				NameFn: func() string { return "test" },
			}
		}
		m.RateLimiter.WaitFn = func(ctx context.Context, _ string) error {
			// Cancel after first URL is processed
			if fetchCount >= 1 {
				cancel()
			}
			return ctx.Err()
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
		}

		result, err := c.CrawlProject(ctx, project, nil)

		// Should return without error (partial results)
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have processed exactly 1 URL (seed) before cancellation stopped further processing
		assert.Equal(t, 1, result.Saved)
		assert.Equal(t, 1, fetchCount, "should stop after first fetch due to cancellation")
	})

	t.Run("crawls single URL and saves document", func(t *testing.T) {
		t.Parallel()

		var savedDoc *locdoc.Document

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1"}, nil
		}
		m.Fetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			return "<html><body>Test content</body></html>", nil
		}
		m.Extractor.ExtractFn = func(_ string) (*locdoc.ExtractResult, error) {
			return &locdoc.ExtractResult{
				Title:       "Test Page",
				ContentHTML: "<p>Test content</p>",
			}, nil
		}
		m.Converter.ConvertFn = func(_ string) (string, error) {
			return "Test content", nil
		}
		m.Documents.CreateDocumentFn = func(_ context.Context, doc *locdoc.Document) error {
			savedDoc = doc
			return nil
		}
		m.TokenCounter.CountTokensFn = func(_ context.Context, text string) (int, error) {
			return len(text) / 4, nil // ~4 chars per token
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.Saved)
		assert.Equal(t, 0, result.Failed)
		assert.Equal(t, len("Test content"), result.Bytes)
		assert.Equal(t, 3, result.Tokens) // 12 chars / 4 = 3

		// Verify saved document
		require.NotNil(t, savedDoc)
		assert.Equal(t, "proj-123", savedDoc.ProjectID)
		assert.Equal(t, "https://example.com/page1", savedDoc.SourceURL)
		assert.Equal(t, "Test Page", savedDoc.Title)
		assert.Equal(t, "Test content", savedDoc.Content)
		assert.Equal(t, 0, savedDoc.Position)
		assert.NotEmpty(t, savedDoc.ContentHash)
	})

	t.Run("counts failed URLs when fetch fails", func(t *testing.T) {
		t.Parallel()

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1", "https://example.com/page2"}, nil
		}
		m.Fetcher.FetchFn = func(_ context.Context, url string) (string, error) {
			if url == "https://example.com/page1" {
				return "", locdoc.Errorf(locdoc.EINTERNAL, "fetch failed")
			}
			return "<html><body>Page 2</body></html>", nil
		}
		m.Extractor.ExtractFn = func(_ string) (*locdoc.ExtractResult, error) {
			return &locdoc.ExtractResult{
				Title:       "Page 2",
				ContentHTML: "<p>Page 2 content</p>",
			}, nil
		}
		m.Converter.ConvertFn = func(_ string) (string, error) {
			return "Page 2 content", nil
		}
		m.TokenCounter.CountTokensFn = func(_ context.Context, text string) (int, error) {
			return len(text) / 4, nil
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.Saved)
		assert.Equal(t, 1, result.Failed)
	})

	t.Run("counts failed URLs when CreateDocument fails", func(t *testing.T) {
		t.Parallel()

		createCallCount := 0

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1", "https://example.com/page2"}, nil
		}
		m.Fetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			return "<html><body>Content</body></html>", nil
		}
		m.Documents.CreateDocumentFn = func(_ context.Context, doc *locdoc.Document) error {
			createCallCount++
			// Fail on first document, succeed on second
			if doc.SourceURL == "https://example.com/page1" {
				return locdoc.Errorf(locdoc.EINTERNAL, "database error")
			}
			return nil
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.Saved)  // Only page2 saved
		assert.Equal(t, 1, result.Failed) // page1 failed during save
		assert.Equal(t, 2, createCallCount)
	})

	t.Run("calls progress callback with events", func(t *testing.T) {
		t.Parallel()

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1"}, nil
		}
		m.Fetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			return "<html><body>Test</body></html>", nil
		}
		m.Extractor.ExtractFn = func(_ string) (*locdoc.ExtractResult, error) {
			return &locdoc.ExtractResult{
				Title:       "Test",
				ContentHTML: "<p>Test</p>",
			}, nil
		}
		m.Converter.ConvertFn = func(_ string) (string, error) {
			return "Test", nil
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		var events []crawl.ProgressEvent
		progress := func(e crawl.ProgressEvent) {
			events = append(events, e)
		}

		_, err := c.CrawlProject(context.Background(), project, progress)

		require.NoError(t, err)
		require.Len(t, events, 3) // Started, Completed, Finished

		// First event: Started
		assert.Equal(t, crawl.ProgressStarted, events[0].Type)
		assert.Equal(t, 1, events[0].Total)

		// Second event: Completed for the URL
		assert.Equal(t, crawl.ProgressCompleted, events[1].Type)
		assert.Equal(t, 1, events[1].Completed)
		assert.Equal(t, 1, events[1].Total)
		assert.Equal(t, "https://example.com/page1", events[1].URL)

		// Third event: Finished
		assert.Equal(t, crawl.ProgressFinished, events[2].Type)
		assert.Equal(t, 1, events[2].Total)
	})

	t.Run("recursive crawl reports completed count in progress events", func(t *testing.T) {
		t.Parallel()

		c, m := newTestCrawler()
		m.Fetcher.FetchFn = func(_ context.Context, url string) (string, error) {
			if url == "https://example.com/docs/" {
				return `<html><body><nav><a href="/docs/page1">Page 1</a></nav><p>Content</p></body></html>`, nil
			}
			return `<html><body><p>Page content</p></body></html>`, nil
		}
		m.LinkSelectors.GetForHTMLFn = func(_ string) locdoc.LinkSelector {
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
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
		}

		var events []crawl.ProgressEvent
		progress := func(e crawl.ProgressEvent) {
			events = append(events, e)
		}

		result, err := c.CrawlProject(context.Background(), project, progress)

		require.NoError(t, err)
		require.Equal(t, 2, result.Saved, "should save 2 pages")

		// Recursive crawling doesn't emit ProgressStarted (no known total)
		// It should emit ProgressCompleted events with incrementing Completed count
		var completedEvents []crawl.ProgressEvent
		for _, e := range events {
			if e.Type == crawl.ProgressCompleted {
				completedEvents = append(completedEvents, e)
			}
		}

		require.Len(t, completedEvents, 2, "should have 2 completed events")

		// Completed count should increment: 1, 2
		assert.Equal(t, 1, completedEvents[0].Completed, "first completed should be 1")
		assert.Equal(t, 2, completedEvents[1].Completed, "second completed should be 2")

		// Total should be 0 (unknown) for recursive crawling
		assert.Equal(t, 0, completedEvents[0].Total, "total should be 0 (unknown)")
		assert.Equal(t, 0, completedEvents[1].Total, "total should be 0 (unknown)")

		// Should have a Finished event at the end
		lastEvent := events[len(events)-1]
		assert.Equal(t, crawl.ProgressFinished, lastEvent.Type, "last event should be Finished")
	})

	t.Run("recursive crawl emits ProgressFailed when CreateDocument fails", func(t *testing.T) {
		t.Parallel()

		c, m := newTestCrawler()
		m.Documents.CreateDocumentFn = func(_ context.Context, doc *locdoc.Document) error {
			// Fail on one specific URL
			if doc.SourceURL == "https://example.com/docs/page1" {
				return locdoc.Errorf(locdoc.EINTERNAL, "database error")
			}
			return nil
		}
		m.LinkSelectors.GetForHTMLFn = func(_ string) locdoc.LinkSelector {
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
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com/docs/",
		}

		var events []crawl.ProgressEvent
		progress := func(e crawl.ProgressEvent) {
			events = append(events, e)
		}

		result, err := c.CrawlProject(context.Background(), project, progress)

		require.NoError(t, err)
		assert.Equal(t, 1, result.Saved, "should save seed URL only")
		assert.Equal(t, 1, result.Failed, "should count page1 as failed")

		// Find the ProgressFailed event
		var failedEvents []crawl.ProgressEvent
		for _, e := range events {
			if e.Type == crawl.ProgressFailed {
				failedEvents = append(failedEvents, e)
			}
		}

		require.Len(t, failedEvents, 1, "should emit exactly one ProgressFailed event")
		assert.Equal(t, "https://example.com/docs/page1", failedEvents[0].URL, "failed event should have correct URL")
		require.Error(t, failedEvents[0].Error, "failed event should have error")
		assert.Contains(t, failedEvents[0].Error.Error(), "database error", "error should contain original message")
	})
}

func TestTruncateURL(t *testing.T) {
	t.Parallel()

	t.Run("returns URL unchanged when shorter than max", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "https://x.com", crawl.TruncateURL("https://x.com", 50))
	})

	t.Run("truncates with ellipsis when longer than max", func(t *testing.T) {
		t.Parallel()
		url := "https://example.com/very/long/path/to/documentation"
		result := crawl.TruncateURL(url, 20)
		assert.Equal(t, ".../to/documentation", result)
		assert.Len(t, result, 20)
	})

	t.Run("returns URL unchanged when exactly max length", func(t *testing.T) {
		t.Parallel()
		url := "https://example.com"
		assert.Equal(t, url, crawl.TruncateURL(url, len(url)))
	})

	t.Run("returns empty string when maxLen is zero", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, crawl.TruncateURL("https://example.com", 0))
	})

	t.Run("returns empty string when maxLen is negative", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, crawl.TruncateURL("https://example.com", -1))
	})

	t.Run("returns prefix of URL when maxLen is very small", func(t *testing.T) {
		t.Parallel()
		// When maxLen < 4, we can't fit "..." prefix, so return URL prefix
		assert.Equal(t, "htt", crawl.TruncateURL("https://example.com", 3))
		assert.Equal(t, "ht", crawl.TruncateURL("https://example.com", 2))
		assert.Equal(t, "h", crawl.TruncateURL("https://example.com", 1))
	})

	t.Run("handles short URL with small maxLen", func(t *testing.T) {
		t.Parallel()
		// URL shorter than maxLen should return unchanged
		assert.Equal(t, "ab", crawl.TruncateURL("ab", 3))
		assert.Equal(t, "a", crawl.TruncateURL("a", 2))
	})
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	t.Run("formats bytes as B", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "512 B", crawl.FormatBytes(512))
	})

	t.Run("formats kilobytes as KB", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "1.5 KB", crawl.FormatBytes(1536))
	})

	t.Run("formats megabytes as MB", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "2.0 MB", crawl.FormatBytes(2*1024*1024))
	})
}

func TestFormatTokens(t *testing.T) {
	t.Parallel()

	t.Run("formats small token counts", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "~500 tokens", crawl.FormatTokens(500))
	})

	t.Run("formats large token counts as k", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "~10k tokens", crawl.FormatTokens(10000))
	})

	t.Run("rounds token counts", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "~2k tokens", crawl.FormatTokens(1500))
	})
}

func TestComputeHash(t *testing.T) {
	t.Parallel()

	t.Run("returns consistent hash for same content", func(t *testing.T) {
		t.Parallel()
		content := "test content"
		hash1 := crawl.ComputeHash(content)
		hash2 := crawl.ComputeHash(content)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("returns different hashes for different content", func(t *testing.T) {
		t.Parallel()
		hash1 := crawl.ComputeHash("content a")
		hash2 := crawl.ComputeHash("content b")
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("returns hex string", func(t *testing.T) {
		t.Parallel()
		hash := crawl.ComputeHash("test")
		assert.Regexp(t, `^[0-9a-f]+$`, hash)
	})
}

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
