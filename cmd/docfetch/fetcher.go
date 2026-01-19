package main

import (
	"context"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
)

// Ensure ConcurrentFetcher implements locdoc.PageFetcher at compile time.
var _ locdoc.PageFetcher = (*ConcurrentFetcher)(nil)

// ConcurrentFetcher implements locdoc.PageFetcher by orchestrating
// fetching, extraction, and conversion through injected dependencies.
type ConcurrentFetcher struct {
	fetcher     locdoc.Fetcher
	extractor   locdoc.Extractor
	converter   locdoc.Converter
	retryDelays []time.Duration
}

// FetcherOption configures a ConcurrentFetcher.
type FetcherOption func(*ConcurrentFetcher)

// WithRetryDelays sets the retry delays for failed fetches.
// Defaults to crawl.DefaultRetryDelays() if not specified.
func WithRetryDelays(delays []time.Duration) FetcherOption {
	return func(cf *ConcurrentFetcher) {
		cf.retryDelays = delays
	}
}

// NewConcurrentFetcher creates a new ConcurrentFetcher with the given dependencies.
func NewConcurrentFetcher(
	fetcher locdoc.Fetcher,
	extractor locdoc.Extractor,
	converter locdoc.Converter,
	opts ...FetcherOption,
) *ConcurrentFetcher {
	cf := &ConcurrentFetcher{
		fetcher:   fetcher,
		extractor: extractor,
		converter: converter,
	}
	for _, opt := range opts {
		opt(cf)
	}
	return cf
}

// FetchAll retrieves and converts all pages at the given URLs.
func (cf *ConcurrentFetcher) FetchAll(
	ctx context.Context,
	urls []string,
	progress locdoc.FetchProgressFunc,
) ([]*locdoc.Page, error) {
	var pages []*locdoc.Page
	total := len(urls)

	delays := cf.retryDelays
	if delays == nil {
		delays = crawl.DefaultRetryDelays()
	}

	for i, url := range urls {
		// Check for context cancellation before processing each URL
		if err := ctx.Err(); err != nil {
			return pages, err
		}

		var fetchErr error

		fetchFn := func(ctx context.Context, url string) (string, error) {
			return cf.fetcher.Fetch(ctx, url)
		}
		html, err := crawl.FetchWithRetryDelays(ctx, url, fetchFn, nil, delays)
		if err != nil {
			fetchErr = err
		}

		var result *locdoc.ExtractResult
		if fetchErr == nil {
			result, err = cf.extractor.Extract(html)
			if err != nil {
				fetchErr = err
			}
		}

		var content string
		if fetchErr == nil {
			content, err = cf.converter.Convert(result.ContentHTML)
			if err != nil {
				fetchErr = err
			}
		}

		if fetchErr == nil {
			pages = append(pages, &locdoc.Page{
				URL:     url,
				Title:   result.Title,
				Content: content,
			})
		}

		if progress != nil {
			progress(locdoc.FetchProgress{
				URL:       url,
				Completed: i + 1,
				Total:     total,
				Error:     fetchErr,
			})
		}
	}

	return pages, nil
}
