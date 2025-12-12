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

	t.Run("returns nil result with stub implementation", func(t *testing.T) {
		t.Parallel()

		c := &crawl.Crawler{
			Sitemaps:     &mock.SitemapService{},
			Fetcher:      &mock.Fetcher{},
			Extractor:    &mock.Extractor{},
			Converter:    &mock.Converter{},
			Documents:    &mock.DocumentService{},
			TokenCounter: &mock.TokenCounter{},
			Concurrency:  10,
			RetryDelays:  []time.Duration{time.Second},
		}

		project := &locdoc.Project{
			ID:        "test-id",
			Name:      "test",
			SourceURL: "https://example.com",
		}

		result, err := c.CrawlProject(context.Background(), project, nil)

		require.NoError(t, err)
		assert.Nil(t, result)
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
