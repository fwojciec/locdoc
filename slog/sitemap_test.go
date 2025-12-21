package slog_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/mock"
	locslog "github.com/fwojciec/locdoc/slog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingSitemapService_DiscoverURLs(t *testing.T) {
	t.Parallel()

	t.Run("logs discovery with count and duration", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		inner := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				return []string{"https://example.com/a", "https://example.com/b"}, nil
			},
		}

		svc := locslog.NewLoggingSitemapService(inner, logger)
		urls, err := svc.DiscoverURLs(context.Background(), "https://example.com", nil)

		require.NoError(t, err)
		assert.Len(t, urls, 2)
		output := buf.String()
		assert.Contains(t, output, "sitemap discovery")
		assert.Contains(t, output, "url=https://example.com")
		assert.Contains(t, output, "count=2")
		assert.Contains(t, output, "duration=")
	})

	t.Run("logs error on failure", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		inner := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				return nil, errors.New("connection failed")
			},
		}

		svc := locslog.NewLoggingSitemapService(inner, logger)
		_, err := svc.DiscoverURLs(context.Background(), "https://example.com", nil)

		require.Error(t, err)
		output := buf.String()
		assert.Contains(t, output, "sitemap discovery")
		assert.Contains(t, output, "err=\"connection failed\"")
	})
}
