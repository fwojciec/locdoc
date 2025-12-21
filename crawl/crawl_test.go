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
		HTTPFetcher: &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return `<html><body><p>Content</p></body></html>`, nil
			},
		},
		RodFetcher: &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return `<html><body><p>Content</p></body></html>`, nil
			},
		},
		Prober: &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, false
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
		HTTPFetcher:   m.HTTPFetcher,
		RodFetcher:    m.RodFetcher,
		Prober:        m.Prober,
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
	HTTPFetcher   *mock.Fetcher
	RodFetcher    *mock.Fetcher
	Prober        *mock.Prober
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
			HTTPFetcher:  &mock.Fetcher{},
			RodFetcher:   &mock.Fetcher{},
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

		fetchFn := func(_ context.Context, url string) (string, error) {
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
		}

		c := &crawl.Crawler{
			Sitemaps: &mock.SitemapService{
				DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
					return []string{}, nil // No sitemap URLs
				},
			},
			HTTPFetcher: &mock.Fetcher{FetchFn: fetchFn},
			RodFetcher:  &mock.Fetcher{FetchFn: fetchFn},
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

		crawlFetchCount := 0
		ctx, cancel := context.WithCancel(context.Background())

		c, m := newTestCrawler()
		// Use known framework to avoid probe comparison fetch
		m.Prober.DetectFn = func(_ string) locdoc.Framework {
			return locdoc.FrameworkSphinx
		}
		m.Prober.RequiresJSFn = func(_ locdoc.Framework) (bool, bool) {
			return false, true
		}
		m.HTTPFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			crawlFetchCount++
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
			// Cancel after first actual crawl URL is processed (probe + 1 crawl = 2)
			if crawlFetchCount >= 2 {
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
		// Probe fetch + 1 actual crawl = 2 fetches
		assert.Equal(t, 2, crawlFetchCount, "should stop after probe + 1 crawl fetch due to cancellation")
	})

	t.Run("crawls single URL and saves document", func(t *testing.T) {
		t.Parallel()

		var savedDoc *locdoc.Document

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1"}, nil
		}
		m.RodFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
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

		fetchFn := func(_ context.Context, url string) (string, error) {
			if url == "https://example.com/page1" {
				return "", locdoc.Errorf(locdoc.EINTERNAL, "fetch failed")
			}
			return "<html><body>Page 2</body></html>", nil
		}

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1", "https://example.com/page2"}, nil
		}
		m.HTTPFetcher.FetchFn = fetchFn
		m.RodFetcher.FetchFn = fetchFn
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
		m.RodFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
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
		m.RodFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
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
		m.RodFetcher.FetchFn = func(_ context.Context, url string) (string, error) {
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

	// Verifies that when CreateDocument fails during recursive crawling,
	// a ProgressFailed event is emitted with the URL and error. This ensures
	// users see feedback about save failures (same as fetch/extract failures).
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

	t.Run("probe uses HTTP fetcher for known HTTP-only framework", func(t *testing.T) {
		t.Parallel()

		var httpFetchCalls, rodFetchCalls int

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1", "https://example.com/page2"}, nil
		}
		m.HTTPFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			httpFetchCalls++
			return `<html><body><p>HTTP Content</p></body></html>`, nil
		}
		m.RodFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			rodFetchCalls++
			return `<html><body><p>Rod Content</p></body></html>`, nil
		}
		m.Prober.DetectFn = func(_ string) locdoc.Framework {
			return locdoc.FrameworkSphinx // Known HTTP-only framework
		}
		m.Prober.RequiresJSFn = func(f locdoc.Framework) (bool, bool) {
			return false, true // Doesn't require JS, is known
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.Saved)
		// Probe uses HTTP once, then HTTP for both pages = 3 total
		assert.Equal(t, 3, httpFetchCalls, "should use HTTP fetcher for probe and all pages")
		assert.Equal(t, 0, rodFetchCalls, "should not use Rod fetcher")
	})

	t.Run("probe uses Rod fetcher for known JS framework", func(t *testing.T) {
		t.Parallel()

		var httpFetchCalls, rodFetchCalls int

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1", "https://example.com/page2"}, nil
		}
		m.HTTPFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			httpFetchCalls++
			return `<html><body><p>HTTP Content</p></body></html>`, nil
		}
		m.RodFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			rodFetchCalls++
			return `<html><body><p>Rod Content</p></body></html>`, nil
		}
		m.Prober.DetectFn = func(_ string) locdoc.Framework {
			return locdoc.FrameworkGitBook // Known JS framework
		}
		m.Prober.RequiresJSFn = func(f locdoc.Framework) (bool, bool) {
			return true, true // Requires JS, is known
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.Saved)
		// Probe uses HTTP once, but then Rod for both pages = 2 Rod fetches
		assert.Equal(t, 1, httpFetchCalls, "should use HTTP fetcher for probe only")
		assert.Equal(t, 2, rodFetchCalls, "should use Rod fetcher for all pages")
	})

	t.Run("probe uses Rod fetcher for unknown framework with different content", func(t *testing.T) {
		t.Parallel()

		var httpFetchCalls, rodFetchCalls int
		httpHTML := `<html><body><p>Short</p></body></html>`
		rodHTML := `<html><body><p>Short plus lots more JavaScript-rendered content that makes this much much longer</p></body></html>`

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1", "https://example.com/page2"}, nil
		}
		m.HTTPFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			httpFetchCalls++
			return httpHTML, nil
		}
		m.RodFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			rodFetchCalls++
			return rodHTML, nil
		}
		// Make extractor return the actual HTML content for comparison
		m.Extractor.ExtractFn = func(html string) (*locdoc.ExtractResult, error) {
			// Return the body content as ContentHTML
			if html == httpHTML {
				return &locdoc.ExtractResult{
					Title:       "Test",
					ContentHTML: "<p>Short</p>",
				}, nil
			}
			return &locdoc.ExtractResult{
				Title:       "Test",
				ContentHTML: "<p>Short plus lots more JavaScript-rendered content that makes this much much longer</p>",
			}, nil
		}
		m.Prober.DetectFn = func(_ string) locdoc.Framework {
			return locdoc.FrameworkUnknown
		}
		m.Prober.RequiresJSFn = func(f locdoc.Framework) (bool, bool) {
			return false, false // Unknown framework
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.Saved)
		// Probe: HTTP once, Rod once (for comparison), then Rod for pages = 1+1+2
		assert.Equal(t, 1, httpFetchCalls, "should use HTTP fetcher for probe only")
		assert.Equal(t, 3, rodFetchCalls, "should use Rod fetcher for comparison probe and all pages")
	})

	t.Run("probe falls back to Rod when HTTP probe fails", func(t *testing.T) {
		t.Parallel()

		var httpFetchCalls, rodFetchCalls int

		c, m := newTestCrawler()
		m.Sitemaps.DiscoverURLsFn = func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
			return []string{"https://example.com/page1", "https://example.com/page2"}, nil
		}
		m.HTTPFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			httpFetchCalls++
			return "", locdoc.Errorf(locdoc.EINTERNAL, "connection refused")
		}
		m.RodFetcher.FetchFn = func(_ context.Context, _ string) (string, error) {
			rodFetchCalls++
			return `<html><body><p>Rod Content</p></body></html>`, nil
		}
		m.Prober.DetectFn = func(_ string) locdoc.Framework {
			return locdoc.FrameworkUnknown
		}
		m.Prober.RequiresJSFn = func(f locdoc.Framework) (bool, bool) {
			return false, false
		}

		project := &locdoc.Project{
			ID:        "proj-123",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.Saved)
		// HTTP fails, fall back to Rod for everything = 2 pages
		assert.Equal(t, 1, httpFetchCalls, "should attempt HTTP probe once")
		assert.Equal(t, 2, rodFetchCalls, "should fall back to Rod for all pages")
	})
}
