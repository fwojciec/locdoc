package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.SitemapService = (*SitemapService)(nil)

// SitemapService is a mock implementation of locdoc.SitemapService.
type SitemapService struct {
	DiscoverURLsFn func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error)
}

func (s *SitemapService) DiscoverURLs(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
	return s.DiscoverURLsFn(ctx, baseURL, filter)
}
