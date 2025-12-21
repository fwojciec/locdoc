package slog

import (
	"context"
	"log/slog"
	"time"

	"github.com/fwojciec/locdoc"
)

// Ensure LoggingSitemapService implements locdoc.SitemapService.
var _ locdoc.SitemapService = (*LoggingSitemapService)(nil)

// LoggingSitemapService wraps a SitemapService with debug logging.
type LoggingSitemapService struct {
	next   locdoc.SitemapService
	logger *slog.Logger
}

// NewLoggingSitemapService creates a new LoggingSitemapService.
func NewLoggingSitemapService(next locdoc.SitemapService, logger *slog.Logger) *LoggingSitemapService {
	return &LoggingSitemapService{next: next, logger: logger}
}

// DiscoverURLs delegates to the wrapped service and logs the operation.
func (s *LoggingSitemapService) DiscoverURLs(ctx context.Context, baseURL string, filter *locdoc.URLFilter) (urls []string, err error) {
	defer func(begin time.Time) {
		s.logger.Info("sitemap discovery",
			"url", baseURL,
			"count", len(urls),
			"duration", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return s.next.DiscoverURLs(ctx, baseURL, filter)
}
