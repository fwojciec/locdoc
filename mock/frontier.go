package mock

import (
	"context"

	"github.com/fwojciec/locdoc"
)

var _ locdoc.URLFrontier = (*URLFrontier)(nil)

// URLFrontier is a mock implementation of locdoc.URLFrontier.
type URLFrontier struct {
	PushFn func(link locdoc.DiscoveredLink) bool
	PopFn  func() (locdoc.DiscoveredLink, bool)
	LenFn  func() int
	SeenFn func(url string) bool
}

func (f *URLFrontier) Push(link locdoc.DiscoveredLink) bool {
	return f.PushFn(link)
}

func (f *URLFrontier) Pop() (locdoc.DiscoveredLink, bool) {
	return f.PopFn()
}

func (f *URLFrontier) Len() int {
	return f.LenFn()
}

func (f *URLFrontier) Seen(url string) bool {
	return f.SeenFn(url)
}

var _ locdoc.DomainLimiter = (*DomainLimiter)(nil)

// DomainLimiter is a mock implementation of locdoc.DomainLimiter.
type DomainLimiter struct {
	WaitFn func(ctx context.Context, domain string) error
}

func (l *DomainLimiter) Wait(ctx context.Context, domain string) error {
	return l.WaitFn(ctx, domain)
}
