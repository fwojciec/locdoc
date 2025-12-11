package main_test

import (
	"context"
	"errors"
	"testing"
	"time"

	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noDelays is used for fast unit tests.
var noDelays = []time.Duration{0, 0, 0}

func TestFetchWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("succeeds on first attempt", func(t *testing.T) {
		t.Parallel()

		var attempts int
		fetcher := func(ctx context.Context, url string) (string, error) {
			attempts++
			return "<html>content</html>", nil
		}

		html, err := main.FetchWithRetryDelays(testContext(), "https://example.com", fetcher, nil, noDelays)

		require.NoError(t, err)
		assert.Equal(t, "<html>content</html>", html)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries on failure and succeeds", func(t *testing.T) {
		t.Parallel()

		var attempts int
		fetcher := func(ctx context.Context, url string) (string, error) {
			attempts++
			if attempts < 4 {
				return "", errors.New("transient error")
			}
			return "<html>success</html>", nil
		}

		html, err := main.FetchWithRetryDelays(testContext(), "https://example.com", fetcher, nil, noDelays)

		require.NoError(t, err)
		assert.Equal(t, "<html>success</html>", html)
		assert.Equal(t, 4, attempts)
	})

	t.Run("returns error after max retries", func(t *testing.T) {
		t.Parallel()

		var attempts int
		fetcher := func(ctx context.Context, url string) (string, error) {
			attempts++
			return "", errors.New("persistent error")
		}

		_, err := main.FetchWithRetryDelays(testContext(), "https://example.com", fetcher, nil, noDelays)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "persistent error")
		assert.Equal(t, 4, attempts) // 1 initial + 3 retries = 4 total attempts
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		var attempts int
		fetcher := func(ctx context.Context, url string) (string, error) {
			attempts++
			if attempts == 1 {
				cancel() // Cancel after first attempt
			}
			return "", errors.New("transient error")
		}

		_, err := main.FetchWithRetryDelays(ctx, "https://example.com", fetcher, nil, noDelays)

		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled) || attempts <= 2, "should stop on context cancellation")
	})

	t.Run("logs retry attempts", func(t *testing.T) {
		t.Parallel()

		var attempts int
		fetcher := func(ctx context.Context, url string) (string, error) {
			attempts++
			if attempts < 2 {
				return "", errors.New("transient error")
			}
			return "<html>success</html>", nil
		}

		var logs []string
		logger := func(format string, args ...any) {
			logs = append(logs, format)
		}

		html, err := main.FetchWithRetryDelays(testContext(), "https://example.com/page", fetcher, logger, noDelays)

		require.NoError(t, err)
		assert.Equal(t, "<html>success</html>", html)
		assert.Len(t, logs, 1, "should log one retry")
	})

	t.Run("logs multiple retry attempts", func(t *testing.T) {
		t.Parallel()

		var attempts int
		fetcher := func(ctx context.Context, url string) (string, error) {
			attempts++
			if attempts < 4 {
				return "", errors.New("transient error")
			}
			return "<html>success</html>", nil
		}

		var logs []string
		logger := func(format string, args ...any) {
			logs = append(logs, format)
		}

		html, err := main.FetchWithRetryDelays(testContext(), "https://example.com/page", fetcher, logger, noDelays)

		require.NoError(t, err)
		assert.Equal(t, "<html>success</html>", html)
		assert.Len(t, logs, 3, "should log 3 retries")
	})

	t.Run("number of retries matches delay count", func(t *testing.T) {
		t.Parallel()

		var attempts int
		fetcher := func(ctx context.Context, url string) (string, error) {
			attempts++
			return "", errors.New("always fail")
		}

		// With 2 delays, we should have 3 total attempts (1 + 2 retries)
		twoDelays := []time.Duration{0, 0}
		_, err := main.FetchWithRetryDelays(testContext(), "https://example.com", fetcher, nil, twoDelays)

		require.Error(t, err)
		assert.Equal(t, 3, attempts)
	})
}
