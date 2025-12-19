package crawl_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainLimiter(t *testing.T) {
	t.Parallel()

	t.Run("implements locdoc.DomainLimiter interface", func(t *testing.T) {
		t.Parallel()
		var _ locdoc.DomainLimiter = crawl.NewDomainLimiter(1)
	})

	t.Run("allows immediate request when under limit", func(t *testing.T) {
		t.Parallel()

		limiter := crawl.NewDomainLimiter(10) // 10 req/sec

		start := time.Now()
		err := limiter.Wait(context.Background(), "example.com")
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, elapsed, 50*time.Millisecond, "first request should be immediate")
	})

	t.Run("rate limits requests to same domain", func(t *testing.T) {
		t.Parallel()

		limiter := crawl.NewDomainLimiter(10) // 10 req/sec = 100ms between requests

		// First request is immediate
		err := limiter.Wait(context.Background(), "example.com")
		require.NoError(t, err)

		// Second request should wait
		start := time.Now()
		err = limiter.Wait(context.Background(), "example.com")
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, elapsed, 80*time.Millisecond, "should wait for rate limit")
	})

	t.Run("different domains have independent limits", func(t *testing.T) {
		t.Parallel()

		limiter := crawl.NewDomainLimiter(10) // 10 req/sec

		// First request to domain A
		err := limiter.Wait(context.Background(), "example.com")
		require.NoError(t, err)

		// First request to domain B should be immediate
		start := time.Now()
		err = limiter.Wait(context.Background(), "other.com")
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, elapsed, 50*time.Millisecond, "different domain should not wait")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()

		limiter := crawl.NewDomainLimiter(1) // 1 req/sec = 1000ms between requests

		// First request exhausts the token
		err := limiter.Wait(context.Background(), "example.com")
		require.NoError(t, err)

		// Second request with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err = limiter.Wait(ctx, "example.com")
		assert.Error(t, err, "should fail when context times out")
	})

	t.Run("concurrent requests are serialized per domain", func(t *testing.T) {
		t.Parallel()

		limiter := crawl.NewDomainLimiter(100) // 100 req/sec = 10ms between requests

		var wg sync.WaitGroup
		var completed atomic.Int32

		// Launch 5 concurrent requests to same domain
		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := limiter.Wait(context.Background(), "example.com")
				if err == nil {
					completed.Add(1)
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, int32(5), completed.Load(), "all requests should complete")
	})
}
