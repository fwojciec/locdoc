package slog_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/fwojciec/locdoc/mock"
	locslog "github.com/fwojciec/locdoc/slog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingFetcher_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("logs fetch with bytes and duration", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		inner := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html>content</html>", nil
			},
		}

		fetcher := locslog.NewLoggingFetcher(inner, logger)
		html, err := fetcher.Fetch(context.Background(), "https://example.com/docs")

		require.NoError(t, err)
		assert.Equal(t, "<html>content</html>", html)
		output := buf.String()
		assert.Contains(t, output, "fetch")
		assert.Contains(t, output, "url=https://example.com/docs")
		assert.Contains(t, output, "bytes=20")
		assert.Contains(t, output, "duration=")
	})

	t.Run("logs error on failure", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		inner := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "", errors.New("network error")
			},
		}

		fetcher := locslog.NewLoggingFetcher(inner, logger)
		_, err := fetcher.Fetch(context.Background(), "https://example.com/docs")

		require.Error(t, err)
		output := buf.String()
		assert.Contains(t, output, "fetch")
		assert.Contains(t, output, "err=\"network error\"")
	})
}

func TestLoggingFetcher_Close(t *testing.T) {
	t.Parallel()

	t.Run("delegates to inner fetcher", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		closeCalled := false
		inner := &mock.Fetcher{
			CloseFn: func() error {
				closeCalled = true
				return nil
			},
		}

		fetcher := locslog.NewLoggingFetcher(inner, logger)
		err := fetcher.Close()

		require.NoError(t, err)
		assert.True(t, closeCalled)
	})
}
