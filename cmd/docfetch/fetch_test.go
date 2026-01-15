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

// Story: Fetching Documentation
//
// The FetchCmd orchestrates documentation fetching through three interfaces:
// - URLSource: discovers URLs from the documentation site
// - PageFetcher: fetches and converts pages to markdown
// - PageStore: persists pages with atomic semantics

func TestFetch_SavesAllDiscoveredPages(t *testing.T) {
	t.Parallel()

	// Given: source returns multiple URLs
	source := &mock.URLSource{
		DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
			return []string{
				"https://example.com/docs/page1",
				"https://example.com/docs/page2",
				"https://example.com/docs/page3",
			}, nil
		},
	}

	// Given: fetcher returns pages for all URLs
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

	// Given: store tracks saved pages
	var savedPages []*locdoc.Page
	store := &mock.PageStore{
		SaveFn: func(_ context.Context, page *locdoc.Page) error {
			savedPages = append(savedPages, page)
			return nil
		},
		CommitFn: func() error { return nil },
		AbortFn:  func() error { return nil },
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

	// When: running fetch
	err := cmd.Run(deps)

	// Then: all discovered pages are saved
	require.NoError(t, err)
	assert.Len(t, savedPages, 3)
	assert.Equal(t, "https://example.com/docs/page1", savedPages[0].URL)
	assert.Equal(t, "https://example.com/docs/page2", savedPages[1].URL)
	assert.Equal(t, "https://example.com/docs/page3", savedPages[2].URL)
}

func TestFetch_ReportsProgressViaCallback(t *testing.T) {
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

	// Given: fetcher tracks progress calls
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

	// Then: progress was reported for each page
	require.NoError(t, err)
	assert.Len(t, progressCalls, 2)
	assert.Equal(t, 1, progressCalls[0].Completed)
	assert.Equal(t, 2, progressCalls[0].Total)
	assert.Equal(t, 2, progressCalls[1].Completed)
	assert.Equal(t, 2, progressCalls[1].Total)
}

func TestFetch_CommitsStoreOnSuccess(t *testing.T) {
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

	// Given: store tracks commit calls
	var committed bool
	store := &mock.PageStore{
		SaveFn: func(_ context.Context, _ *locdoc.Page) error { return nil },
		CommitFn: func() error {
			committed = true
			return nil
		},
		AbortFn: func() error { return nil },
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

	// When: fetch succeeds
	err := cmd.Run(deps)

	// Then: store is committed
	require.NoError(t, err)
	assert.True(t, committed, "store should be committed on success")
}

func TestFetch_AbortsStoreWhenNoPagesSaved(t *testing.T) {
	t.Parallel()

	// Given: source returns URLs
	source := &mock.URLSource{
		DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
			return []string{"https://example.com/docs/page1"}, nil
		},
	}

	// Given: fetcher returns empty result (all pages failed individually)
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

	// Given: store tracks commit and abort calls
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
}

func TestFetch_ContinuesOnPageFailures(t *testing.T) {
	t.Parallel()

	// Given: source returns 3 URLs
	source := &mock.URLSource{
		DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
			return []string{
				"https://example.com/docs/page1",
				"https://example.com/docs/page2",
				"https://example.com/docs/page3",
			}, nil
		},
	}

	// Given: fetcher reports failure on page2 but returns page1 and page3
	fetcher := &mock.PageFetcher{
		FetchAllFn: func(_ context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error) {
			// Return only successful pages, report failure via progress
			successfulPages := []*locdoc.Page{
				{URL: "https://example.com/docs/page1", Title: "Page 1", Content: "Content 1"},
				{URL: "https://example.com/docs/page3", Title: "Page 3", Content: "Content 3"},
			}

			if progress != nil {
				// page1 success
				progress(locdoc.FetchProgress{
					URL:       "https://example.com/docs/page1",
					Completed: 1,
					Total:     3,
				})
				// page2 failure
				progress(locdoc.FetchProgress{
					URL:       "https://example.com/docs/page2",
					Completed: 2,
					Total:     3,
					Error:     locdoc.Errorf(locdoc.EINTERNAL, "page2 failed"),
				})
				// page3 success
				progress(locdoc.FetchProgress{
					URL:       "https://example.com/docs/page3",
					Completed: 3,
					Total:     3,
				})
			}

			return successfulPages, nil
		},
	}

	// Given: store tracks saved pages
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
		AbortFn: func() error { return nil },
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

	// When: fetch runs with one page failing
	err := cmd.Run(deps)

	// Then: operation succeeds with partial results
	require.NoError(t, err)
	// Then: only successful pages are saved
	assert.Len(t, savedPages, 2)
	assert.Equal(t, "https://example.com/docs/page1", savedPages[0].URL)
	assert.Equal(t, "https://example.com/docs/page3", savedPages[1].URL)
	// Then: store is committed (partial success is still success)
	assert.True(t, committed, "store should be committed when some pages saved")
}

func TestFetch_AbortsStoreOnDiscoveryFailure(t *testing.T) {
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
	assert.Contains(t, err.Error(), "discovery failed")
}

func TestFetch_AbortsStoreOnFetchFailure(t *testing.T) {
	t.Parallel()

	// Given: source succeeds
	source := &mock.URLSource{
		DiscoverFn: func(_ context.Context, sourceURL string) ([]string, error) {
			return []string{"https://example.com/docs/page1"}, nil
		},
	}

	// Given: fetcher fails completely
	fetcher := &mock.PageFetcher{
		FetchAllFn: func(_ context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error) {
			return nil, locdoc.Errorf(locdoc.EINTERNAL, "fetch failed")
		},
	}

	// Given: store tracks abort calls
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

	// Then: error is returned and store is aborted
	require.Error(t, err)
	assert.True(t, aborted, "store should be aborted on fetch failure")
}

func TestFetch_AbortsStoreOnSaveFailure(t *testing.T) {
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
}
