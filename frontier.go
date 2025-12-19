package locdoc

import "context"

// URLFrontier manages a crawl queue with deduplication.
type URLFrontier interface {
	// Push adds a link to the frontier.
	// Returns false if the URL has already been seen.
	Push(link DiscoveredLink) bool

	// Pop returns the next URL by priority.
	// Returns false if the frontier is empty.
	Pop() (DiscoveredLink, bool)

	// Len returns the number of URLs in the queue.
	Len() int

	// Seen returns true if the URL has been processed or queued.
	Seen(url string) bool
}

// DomainLimiter provides per-domain rate limiting.
type DomainLimiter interface {
	// Wait blocks until the rate limit allows a request to the domain.
	// Returns an error if the context is canceled.
	Wait(ctx context.Context, domain string) error
}
