package crawl

import (
	"context"
	"time"
)

// FetchFunc is the signature for a fetch function.
type FetchFunc func(ctx context.Context, url string) (string, error)

// LogFunc is the signature for a logging function.
type LogFunc func(format string, args ...any)

// DefaultRetryDelays returns the backoff delays for fetch retries: 1s, 2s, 4s.
func DefaultRetryDelays() []time.Duration {
	return []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
}

// FetchWithRetry attempts to fetch a URL with exponential backoff retry logic.
// It retries up to 3 times (4 total attempts) with delays of 1s, 2s, 4s.
// The logger function, if provided, is called for each retry attempt.
func FetchWithRetry(ctx context.Context, url string, fetch FetchFunc, logger LogFunc) (string, error) {
	return FetchWithRetryDelays(ctx, url, fetch, logger, DefaultRetryDelays())
}

// FetchWithRetryDelays is like FetchWithRetry but allows configurable delays.
// This is useful for testing without waiting for real delays.
func FetchWithRetryDelays(ctx context.Context, url string, fetch FetchFunc, logger LogFunc, delays []time.Duration) (string, error) {
	maxAttempts := len(delays) + 1 // 1 initial + N retries

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		html, err := fetch(ctx, url)
		if err == nil {
			return html, nil
		}
		lastErr = err

		// Don't retry after the last attempt
		if attempt >= maxAttempts-1 {
			break
		}

		// Check context before sleeping
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Log retry
		if logger != nil {
			logger("  retry %s (attempt %d): %v", url, attempt+2, err)
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delays[attempt]):
		}
	}

	return "", lastErr
}
