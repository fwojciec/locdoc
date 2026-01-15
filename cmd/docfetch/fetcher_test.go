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

// Story: Page Processing Pipeline
// The fetcher processes URLs through fetch → extract → convert stages

func TestConcurrentFetcher_FetchAll(t *testing.T) {
	t.Parallel()

	t.Run("fetches single page through pipeline", func(t *testing.T) {
		t.Parallel()

		// Given a pipeline with fetch, extract, and convert stages
		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return "<html><body>Test</body></html>", nil
			},
		}
		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{
					Title:       "Test Page",
					ContentHTML: "<p>Extracted content</p>",
				}, nil
			},
		}
		converter := &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "# Converted markdown", nil
			},
		}

		cf := main.NewConcurrentFetcher(fetcher, extractor, converter)

		// When I fetch a single URL
		pages, err := cf.FetchAll(context.Background(), []string{"https://example.com/page"}, nil)

		// Then the page is processed through all stages
		require.NoError(t, err)
		require.Len(t, pages, 1)
		assert.Equal(t, "https://example.com/page", pages[0].URL)
		assert.Equal(t, "Test Page", pages[0].Title)
		assert.Equal(t, "# Converted markdown", pages[0].Content)
	})

	t.Run("fetches multiple pages", func(t *testing.T) {
		t.Parallel()

		// Given a pipeline that tracks fetched URLs
		fetchedURLs := make(chan string, 3)

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				fetchedURLs <- url
				return "<html><body>" + url + "</body></html>", nil
			},
		}
		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{
					Title:       "Page",
					ContentHTML: html,
				}, nil
			},
		}
		converter := &mock.Converter{
			ConvertFn: func(html string) (string, error) {
				return "md:" + html, nil
			},
		}

		cf := main.NewConcurrentFetcher(fetcher, extractor, converter)

		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2",
			"https://example.com/page3",
		}

		// When I fetch multiple URLs
		pages, err := cf.FetchAll(context.Background(), urls, nil)

		// Then all pages are processed
		require.NoError(t, err)
		require.Len(t, pages, 3)

		// And all URLs were fetched
		close(fetchedURLs)
		fetched := make(map[string]bool)
		for url := range fetchedURLs {
			fetched[url] = true
		}
		for _, url := range urls {
			assert.True(t, fetched[url], "URL should have been fetched: %s", url)
		}
	})

	t.Run("reports progress for each page", func(t *testing.T) {
		t.Parallel()

		// Given a working pipeline
		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return "<html></html>", nil
			},
		}
		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Page", ContentHTML: "<p></p>"}, nil
			},
		}
		converter := &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "content", nil
			},
		}

		cf := main.NewConcurrentFetcher(fetcher, extractor, converter)

		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2",
		}

		var progressReports []locdoc.FetchProgress
		progress := func(p locdoc.FetchProgress) {
			progressReports = append(progressReports, p)
		}

		// When I fetch with a progress callback
		_, err := cf.FetchAll(context.Background(), urls, progress)

		// Then progress is reported for each page
		require.NoError(t, err)
		require.Len(t, progressReports, 2, "should report progress for each page")

		// And progress reports have correct values
		for i, p := range progressReports {
			assert.Equal(t, urls[i], p.URL, "URL should match")
			assert.Equal(t, 2, p.Total, "total should be 2")
			assert.Equal(t, i+1, p.Completed, "completed should increment")
			assert.NoError(t, p.Error, "no error expected")
		}
	})

	t.Run("continues on individual page failures", func(t *testing.T) {
		t.Parallel()

		// Given a fetcher that fails for one URL
		fetchErr := locdoc.Errorf(locdoc.EINTERNAL, "fetch failed")

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				if url == "https://example.com/fail" {
					return "", fetchErr
				}
				return "<html></html>", nil
			},
		}
		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Page", ContentHTML: "<p></p>"}, nil
			},
		}
		converter := &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "content", nil
			},
		}

		cf := main.NewConcurrentFetcher(fetcher, extractor, converter)

		urls := []string{
			"https://example.com/ok1",
			"https://example.com/fail",
			"https://example.com/ok2",
		}

		var progressReports []locdoc.FetchProgress
		progress := func(p locdoc.FetchProgress) {
			progressReports = append(progressReports, p)
		}

		// When I fetch URLs including the failing one
		pages, err := cf.FetchAll(context.Background(), urls, progress)

		// Then no overall error occurs
		require.NoError(t, err)

		// And only successful pages are returned
		require.Len(t, pages, 2)
		assert.Equal(t, "https://example.com/ok1", pages[0].URL)
		assert.Equal(t, "https://example.com/ok2", pages[1].URL)

		// And progress reports the failure for that page
		require.Len(t, progressReports, 3)
		require.NoError(t, progressReports[0].Error)
		assert.Equal(t, fetchErr, progressReports[1].Error)
		require.NoError(t, progressReports[2].Error)
	})

	t.Run("returns error on context cancellation", func(t *testing.T) {
		t.Parallel()

		// Given a pipeline with a cancelled context
		fetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, _ string) (string, error) {
				return "", ctx.Err()
			},
		}
		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Page", ContentHTML: "<p></p>"}, nil
			},
		}
		converter := &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "content", nil
			},
		}

		cf := main.NewConcurrentFetcher(fetcher, extractor, converter)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// When I fetch with a cancelled context
		_, err := cf.FetchAll(ctx, []string{"https://example.com/page"}, nil)

		// Then a cancellation error is returned
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}
