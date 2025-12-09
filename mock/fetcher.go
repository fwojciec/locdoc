package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.Fetcher = (*Fetcher)(nil)

// Fetcher is a mock implementation of locdoc.Fetcher.
type Fetcher struct {
	FetchFn func(ctx context.Context, url string) (string, error)
	CloseFn func() error
}

func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error) {
	return f.FetchFn(ctx, url)
}

func (f *Fetcher) Close() error {
	return f.CloseFn()
}
