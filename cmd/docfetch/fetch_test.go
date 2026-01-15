package main_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/docfetch"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Story: FetchCmd uses 3 interfaces for clean separation
//
// The FetchCmd orchestrates documentation fetching through three interfaces:
// - URLSource: discovers URLs from the documentation site
// - PageFetcher: fetches and converts pages to markdown
// - PageStore: persists pages with atomic semantics

func TestFetchCmd_ThreeInterfaces(t *testing.T) {
	t.Parallel()

	t.Run("preview mode only needs URLSource", func(t *testing.T) {
		t.Parallel()

		// Given: a URL source that returns discovered URLs
		source := &mock.URLSource{
			DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:    context.Background(),
			Stdout: stdout,
			Stderr: &bytes.Buffer{},
			Source: source,
			// Fetcher and Store not needed for preview
		}

		cmd := &main.FetchCmd{
			URL:     "https://example.com/docs",
			Preview: true,
		}

		// When: running in preview mode
		err := cmd.Run(deps)

		// Then: URLs are printed without fetching or storing
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "https://example.com/docs/page1")
		assert.Contains(t, output, "https://example.com/docs/page2")
	})

	t.Run("fetch mode uses all three interfaces", func(t *testing.T) {
		t.Parallel()

		// Given: source returns URLs
		source := &mock.URLSource{
			DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
				}, nil
			},
		}

		// Given: fetcher returns pages
		fetcher := &mock.PageFetcher{
			FetchAllFn: func(_ context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error) {
				pages := make([]*locdoc.Page, len(urls))
				for i, url := range urls {
					pages[i] = &locdoc.Page{
						URL:     url,
						Title:   "Test Page",
						Content: "Test content",
					}
					if progress != nil {
						progress(locdoc.FetchProgress{
							URL:       url,
							Completed: i + 1,
							Total:     len(urls),
						})
					}
				}
				return pages, nil
			},
		}

		// Given: store saves pages
		var savedPages []*locdoc.Page
		var committed bool
		store := &mock.PageStore{
			SaveFn: func(_ context.Context, page *locdoc.Page) error {
				savedPages = append(savedPages, page)
				return nil
			},
			CommitFn: func() error {
				committed = true
				return nil
			},
			AbortFn: func() error {
				return nil
			},
		}

		stdout := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:     context.Background(),
			Stdout:  stdout,
			Stderr:  &bytes.Buffer{},
			Source:  source,
			Fetcher: fetcher,
			Store:   store,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
		}

		// When: running fetch mode
		err := cmd.Run(deps)

		// Then: pages are fetched, saved, and committed
		require.NoError(t, err)
		assert.Len(t, savedPages, 2)
		assert.True(t, committed, "store should be committed on success")
	})

	t.Run("reports progress during fetch", func(t *testing.T) {
		t.Parallel()

		source := &mock.URLSource{
			DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
				}, nil
			},
		}

		var progressCalls []locdoc.FetchProgress
		fetcher := &mock.PageFetcher{
			FetchAllFn: func(_ context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error) {
				pages := make([]*locdoc.Page, len(urls))
				for i, url := range urls {
					pages[i] = &locdoc.Page{URL: url, Title: "Test", Content: "Content"}
					if progress != nil {
						p := locdoc.FetchProgress{
							URL:       url,
							Completed: i + 1,
							Total:     len(urls),
						}
						progressCalls = append(progressCalls, p)
						progress(p)
					}
				}
				return pages, nil
			},
		}

		store := &mock.PageStore{
			SaveFn:   func(_ context.Context, _ *locdoc.Page) error { return nil },
			CommitFn: func() error { return nil },
			AbortFn:  func() error { return nil },
		}

		stdout := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:     context.Background(),
			Stdout:  stdout,
			Stderr:  &bytes.Buffer{},
			Source:  source,
			Fetcher: fetcher,
			Store:   store,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
		}

		// When: running fetch
		err := cmd.Run(deps)

		// Then: progress was reported
		require.NoError(t, err)
		assert.Len(t, progressCalls, 2)
		assert.Equal(t, 1, progressCalls[0].Completed)
		assert.Equal(t, 2, progressCalls[1].Completed)
	})

	t.Run("aborts store on fetch failure", func(t *testing.T) {
		t.Parallel()

		source := &mock.URLSource{
			DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
				return []string{"https://example.com/docs/page1"}, nil
			},
		}

		fetcher := &mock.PageFetcher{
			FetchAllFn: func(_ context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error) {
				return nil, locdoc.Errorf(locdoc.EINTERNAL, "fetch failed")
			},
		}

		var aborted bool
		store := &mock.PageStore{
			SaveFn:   func(_ context.Context, _ *locdoc.Page) error { return nil },
			CommitFn: func() error { return nil },
			AbortFn: func() error {
				aborted = true
				return nil
			},
		}

		deps := &main.Dependencies{
			Ctx:     context.Background(),
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			Source:  source,
			Fetcher: fetcher,
			Store:   store,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
		}

		// When: fetch fails
		err := cmd.Run(deps)

		// Then: store is aborted
		require.Error(t, err)
		assert.True(t, aborted, "store should be aborted on failure")
	})

	t.Run("returns error on discovery failure", func(t *testing.T) {
		t.Parallel()

		// Given: source fails to discover URLs
		source := &mock.URLSource{
			DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
				return nil, locdoc.Errorf(locdoc.EINTERNAL, "discovery failed")
			},
		}

		deps := &main.Dependencies{
			Ctx:    context.Background(),
			Stdout: &bytes.Buffer{},
			Stderr: &bytes.Buffer{},
			Source: source,
			// Fetcher and Store not called when discovery fails
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
		}

		// When: discovery fails
		err := cmd.Run(deps)

		// Then: error is returned
		require.Error(t, err)
	})

	t.Run("aborts store on save failure", func(t *testing.T) {
		t.Parallel()

		// Given: source and fetcher succeed
		source := &mock.URLSource{
			DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
				return []string{"https://example.com/docs/page1"}, nil
			},
		}

		fetcher := &mock.PageFetcher{
			FetchAllFn: func(_ context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error) {
				return []*locdoc.Page{{URL: urls[0], Title: "Test", Content: "Content"}}, nil
			},
		}

		// Given: store fails to save
		var aborted bool
		store := &mock.PageStore{
			SaveFn: func(_ context.Context, _ *locdoc.Page) error {
				return locdoc.Errorf(locdoc.EINTERNAL, "save failed")
			},
			CommitFn: func() error { return nil },
			AbortFn: func() error {
				aborted = true
				return nil
			},
		}

		deps := &main.Dependencies{
			Ctx:     context.Background(),
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			Source:  source,
			Fetcher: fetcher,
			Store:   store,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
		}

		// When: save fails
		err := cmd.Run(deps)

		// Then: error is returned and store is aborted
		require.Error(t, err)
		assert.True(t, aborted, "store should be aborted on save failure")
	})

	t.Run("aborts store when no pages saved", func(t *testing.T) {
		t.Parallel()

		source := &mock.URLSource{
			DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
				return []string{"https://example.com/docs/page1"}, nil
			},
		}

		// All pages fail individually but FetchAll succeeds with empty result
		fetcher := &mock.PageFetcher{
			FetchAllFn: func(_ context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error) {
				// Report failures via progress but return empty pages
				for i, url := range urls {
					if progress != nil {
						progress(locdoc.FetchProgress{
							URL:       url,
							Completed: i + 1,
							Total:     len(urls),
							Error:     locdoc.Errorf(locdoc.EINTERNAL, "page failed"),
						})
					}
				}
				return []*locdoc.Page{}, nil
			},
		}

		var committed, aborted bool
		store := &mock.PageStore{
			SaveFn: func(_ context.Context, _ *locdoc.Page) error { return nil },
			CommitFn: func() error {
				committed = true
				return nil
			},
			AbortFn: func() error {
				aborted = true
				return nil
			},
		}

		deps := &main.Dependencies{
			Ctx:     context.Background(),
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			Source:  source,
			Fetcher: fetcher,
			Store:   store,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
		}

		// When: no pages are successfully fetched
		err := cmd.Run(deps)

		// Then: store is aborted, not committed
		require.NoError(t, err) // Command succeeds even if no pages saved
		assert.False(t, committed, "store should not be committed when no pages saved")
		assert.True(t, aborted, "store should be aborted when no pages saved")
	})
}
