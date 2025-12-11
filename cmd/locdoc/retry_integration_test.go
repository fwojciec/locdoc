//go:build integration

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

func TestFetchWithRetry_ExponentialBackoff(t *testing.T) {
	t.Parallel()

	t.Run("uses exponential backoff delays", func(t *testing.T) {
		t.Parallel()

		var timestamps []time.Time
		fetcher := func(ctx context.Context, url string) (string, error) {
			timestamps = append(timestamps, time.Now())
			if len(timestamps) < 4 {
				return "", errors.New("transient error")
			}
			return "<html>success</html>", nil
		}

		html, err := main.FetchWithRetry(context.Background(), "https://example.com", fetcher, nil)

		require.NoError(t, err)
		assert.Equal(t, "<html>success</html>", html)
		require.Len(t, timestamps, 4)

		// Check delays are approximately 1s, 2s, and 4s (with some tolerance)
		delay1 := timestamps[1].Sub(timestamps[0])
		delay2 := timestamps[2].Sub(timestamps[1])
		delay3 := timestamps[3].Sub(timestamps[2])

		assert.InDelta(t, 1.0, delay1.Seconds(), 0.2, "first retry should be ~1s")
		assert.InDelta(t, 2.0, delay2.Seconds(), 0.4, "second retry should be ~2s")
		assert.InDelta(t, 4.0, delay3.Seconds(), 0.8, "third retry should be ~4s")
	})
}
