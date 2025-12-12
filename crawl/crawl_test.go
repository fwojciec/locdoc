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

func TestCrawler_CrawlProject(t *testing.T) {
	t.Parallel()

	t.Run("returns zero result when sitemap returns no URLs", func(t *testing.T) {
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

	t.Run("crawls single URL and saves document", func(t *testing.T) {
		t.Parallel()

		var savedDoc *locdoc.Document
		c := &crawl.Crawler{
			Sitemaps: &mock.SitemapService{
				DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
					return []string{"https://example.com/page1"}, nil
				},
			},
			Fetcher: &mock.Fetcher{
				FetchFn: func(_ context.Context, url string) (string, error) {
					return "<html><body>Test content</body></html>", nil
				},
			},
			Extractor: &mock.Extractor{
				ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
					return &locdoc.ExtractResult{
						Title:       "Test Page",
						ContentHTML: "<p>Test content</p>",
					}, nil
				},
			},
			Converter: &mock.Converter{
				ConvertFn: func(_ string) (string, error) {
					return "Test content", nil
				},
			},
			Documents: &mock.DocumentService{
				CreateDocumentFn: func(_ context.Context, doc *locdoc.Document) error {
					savedDoc = doc
					return nil
				},
			},
			TokenCounter: &mock.TokenCounter{
				CountTokensFn: func(_ context.Context, text string) (int, error) {
					return len(text) / 4, nil // ~4 chars per token
				},
			},
			Concurrency: 1,
			RetryDelays: []time.Duration{0},
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

		c := &crawl.Crawler{
			Sitemaps: &mock.SitemapService{
				DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
					return []string{"https://example.com/page1", "https://example.com/page2"}, nil
				},
			},
			Fetcher: &mock.Fetcher{
				FetchFn: func(_ context.Context, url string) (string, error) {
					if url == "https://example.com/page1" {
						return "", locdoc.Errorf(locdoc.EINTERNAL, "fetch failed")
					}
					return "<html><body>Page 2</body></html>", nil
				},
			},
			Extractor: &mock.Extractor{
				ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
					return &locdoc.ExtractResult{
						Title:       "Page 2",
						ContentHTML: "<p>Page 2 content</p>",
					}, nil
				},
			},
			Converter: &mock.Converter{
				ConvertFn: func(_ string) (string, error) {
					return "Page 2 content", nil
				},
			},
			Documents: &mock.DocumentService{
				CreateDocumentFn: func(_ context.Context, doc *locdoc.Document) error {
					return nil
				},
			},
			TokenCounter: &mock.TokenCounter{
				CountTokensFn: func(_ context.Context, text string) (int, error) {
					return len(text) / 4, nil
				},
			},
			Concurrency: 1,
			RetryDelays: []time.Duration{0}, // no retry delay for tests
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

	t.Run("calls progress callback with events", func(t *testing.T) {
		t.Parallel()

		c := &crawl.Crawler{
			Sitemaps: &mock.SitemapService{
				DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
					return []string{"https://example.com/page1"}, nil
				},
			},
			Fetcher: &mock.Fetcher{
				FetchFn: func(_ context.Context, _ string) (string, error) {
					return "<html><body>Test</body></html>", nil
				},
			},
			Extractor: &mock.Extractor{
				ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
					return &locdoc.ExtractResult{
						Title:       "Test",
						ContentHTML: "<p>Test</p>",
					}, nil
				},
			},
			Converter: &mock.Converter{
				ConvertFn: func(_ string) (string, error) {
					return "Test", nil
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
			Concurrency: 1,
			RetryDelays: []time.Duration{0},
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
}

func TestProgressType_Constants(t *testing.T) {
	t.Parallel()

	// Verify constants are defined and have expected order
	assert.Equal(t, crawl.ProgressStarted, crawl.ProgressType(0))
	assert.Equal(t, crawl.ProgressCompleted, crawl.ProgressType(1))
	assert.Equal(t, crawl.ProgressFailed, crawl.ProgressType(2))
	assert.Equal(t, crawl.ProgressFinished, crawl.ProgressType(3))
}

func TestResult_Fields(t *testing.T) {
	t.Parallel()

	// Verify Result struct has expected fields
	r := crawl.Result{
		Saved:  10,
		Failed: 2,
		Bytes:  1024,
		Tokens: 500,
	}

	assert.Equal(t, 10, r.Saved)
	assert.Equal(t, 2, r.Failed)
	assert.Equal(t, 1024, r.Bytes)
	assert.Equal(t, 500, r.Tokens)
}

func TestProgressEvent_Fields(t *testing.T) {
	t.Parallel()

	// Verify ProgressEvent struct has expected fields
	testErr := locdoc.Errorf(locdoc.EINTERNAL, "test error")
	e := crawl.ProgressEvent{
		Type:      crawl.ProgressFailed,
		Completed: 5,
		Total:     10,
		URL:       "https://example.com/page",
		Error:     testErr,
	}

	assert.Equal(t, crawl.ProgressFailed, e.Type)
	assert.Equal(t, 5, e.Completed)
	assert.Equal(t, 10, e.Total)
	assert.Equal(t, "https://example.com/page", e.URL)
	assert.Equal(t, testErr, e.Error)
}

func TestProgressFunc_Type(t *testing.T) {
	t.Parallel()

	// Verify ProgressFunc is callable
	var called bool
	var fn crawl.ProgressFunc = func(event crawl.ProgressEvent) {
		called = true
	}

	fn(crawl.ProgressEvent{Type: crawl.ProgressStarted})
	assert.True(t, called)
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
