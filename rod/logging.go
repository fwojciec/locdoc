package rod

import (
	"context"
	"log/slog"
	"time"

	"github.com/fwojciec/locdoc"
)

// Ensure LoggingFetcher implements locdoc.Fetcher.
var _ locdoc.Fetcher = (*LoggingFetcher)(nil)

// LoggingFetcher wraps a Fetcher with debug logging.
type LoggingFetcher struct {
	next   locdoc.Fetcher
	logger *slog.Logger
}

// NewLoggingFetcher creates a new LoggingFetcher.
func NewLoggingFetcher(next locdoc.Fetcher, logger *slog.Logger) *LoggingFetcher {
	return &LoggingFetcher{next: next, logger: logger}
}

// Fetch logs the URL being fetched and delegates to the wrapped fetcher.
func (f *LoggingFetcher) Fetch(ctx context.Context, url string) (html string, err error) {
	defer func(begin time.Time) {
		f.logger.Info("fetch",
			"url", url,
			"bytes", len(html),
			"duration", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return f.next.Fetch(ctx, url)
}

// Close delegates to the wrapped fetcher.
func (f *LoggingFetcher) Close() error {
	return f.next.Close()
}
