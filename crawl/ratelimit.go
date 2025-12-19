package crawl

import (
	"context"
	"sync"

	"github.com/fwojciec/locdoc"
	"golang.org/x/time/rate"
)

var _ locdoc.DomainLimiter = (*DomainLimiter)(nil)

// DomainLimiter provides per-domain rate limiting using token buckets.
// It creates a separate rate limiter for each domain, allowing concurrent
// requests to different domains while enforcing rate limits within each domain.
type DomainLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      float64
}

// NewDomainLimiter creates a new DomainLimiter with the specified requests per second limit.
// Each domain gets its own limiter with a burst of 1 (no bursting allowed).
func NewDomainLimiter(rps float64) *DomainLimiter {
	return &DomainLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
	}
}

// Wait blocks until the rate limit allows a request to the domain.
// Returns an error if the context is canceled before the wait completes.
func (d *DomainLimiter) Wait(ctx context.Context, domain string) error {
	d.mu.Lock()
	limiter, ok := d.limiters[domain]
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(d.rps), 1)
		d.limiters[domain] = limiter
	}
	d.mu.Unlock()

	return limiter.Wait(ctx)
}
