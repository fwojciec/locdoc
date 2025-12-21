package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.DomainLimiter = (*DomainLimiter)(nil)

// DomainLimiter is a mock implementation of locdoc.DomainLimiter.
type DomainLimiter struct {
	WaitFn func(ctx context.Context, domain string) error
}

func (l *DomainLimiter) Wait(ctx context.Context, domain string) error {
	return l.WaitFn(ctx, domain)
}
