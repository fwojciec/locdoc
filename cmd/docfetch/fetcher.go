package main

import (
	"context"

	"github.com/fwojciec/locdoc"
)

// ConcurrentFetcher implements locdoc.PageFetcher by orchestrating
// fetching, extraction, and conversion through injected dependencies.
type ConcurrentFetcher struct {
	fetcher   locdoc.Fetcher
	extractor locdoc.Extractor
	converter locdoc.Converter
}

// NewConcurrentFetcher creates a new ConcurrentFetcher with the given dependencies.
func NewConcurrentFetcher(
	fetcher locdoc.Fetcher,
	extractor locdoc.Extractor,
	converter locdoc.Converter,
) *ConcurrentFetcher {
	return &ConcurrentFetcher{
		fetcher:   fetcher,
		extractor: extractor,
		converter: converter,
	}
}

// FetchAll retrieves and converts all pages at the given URLs.
func (cf *ConcurrentFetcher) FetchAll(
	ctx context.Context,
	urls []string,
	progress locdoc.FetchProgressFunc,
) ([]*locdoc.Page, error) {
	var pages []*locdoc.Page
	total := len(urls)

	for i, url := range urls {
		// Check for context cancellation before processing each URL
		if err := ctx.Err(); err != nil {
			return pages, err
		}

		var fetchErr error

		html, err := cf.fetcher.Fetch(ctx, url)
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
