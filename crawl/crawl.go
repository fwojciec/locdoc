// Package crawl provides documentation crawling orchestration.
// It coordinates sitemap discovery, fetching, extraction, and storage
// of documentation pages.
package crawl

import (
	"context"
	"time"

	"github.com/fwojciec/locdoc"
)

// Crawler orchestrates the crawling of documentation sites.
type Crawler struct {
	Sitemaps     locdoc.SitemapService
	Fetcher      locdoc.Fetcher
	Extractor    locdoc.Extractor
	Converter    locdoc.Converter
	Documents    locdoc.DocumentService
	TokenCounter locdoc.TokenCounter
	Concurrency  int
	RetryDelays  []time.Duration
}

// Result holds the outcome of a crawl operation.
type Result struct {
	Saved  int
	Failed int
	Bytes  int
	Tokens int
}

// ProgressEvent reports progress during a crawl operation.
type ProgressEvent struct {
	Type      ProgressType
	Completed int
	Total     int
	URL       string
	Error     error
}

// ProgressType indicates the type of progress event.
type ProgressType int

const (
	ProgressStarted ProgressType = iota
	ProgressCompleted
	ProgressFailed
	ProgressFinished
)

// ProgressFunc is a callback for reporting crawl progress.
type ProgressFunc func(event ProgressEvent)

// CrawlProject crawls all pages for a project and saves them as documents.
// The progress callback, if provided, receives events as crawling proceeds.
func (c *Crawler) CrawlProject(_ context.Context, _ *locdoc.Project, _ ProgressFunc) (*Result, error) {
	return nil, nil
}
