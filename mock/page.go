package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

// Compile-time interface verification.
var (
	_ locdoc.URLSource   = (*URLSource)(nil)
	_ locdoc.PageFetcher = (*PageFetcher)(nil)
	_ locdoc.PageStore   = (*PageStore)(nil)
)

// URLSource is a mock implementation of locdoc.URLSource.
type URLSource struct {
	DiscoverFn func(ctx context.Context, sourceURL string) ([]string, error)
}

func (s *URLSource) Discover(ctx context.Context, sourceURL string) ([]string, error) {
	return s.DiscoverFn(ctx, sourceURL)
}

// PageFetcher is a mock implementation of locdoc.PageFetcher.
type PageFetcher struct {
	FetchAllFn func(ctx context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error)
}

func (f *PageFetcher) FetchAll(ctx context.Context, urls []string, progress locdoc.FetchProgressFunc) ([]*locdoc.Page, error) {
	return f.FetchAllFn(ctx, urls, progress)
}

// PageStore is a mock implementation of locdoc.PageStore.
type PageStore struct {
	SaveFn   func(ctx context.Context, page *locdoc.Page) error
	CommitFn func() error
	AbortFn  func() error
}

func (s *PageStore) Save(ctx context.Context, page *locdoc.Page) error {
	return s.SaveFn(ctx, page)
}

func (s *PageStore) Commit() error {
	return s.CommitFn()
}

func (s *PageStore) Abort() error {
	return s.AbortFn()
}
