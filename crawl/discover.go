package crawl

import "time"

// DiscoverOption configures DiscoverURLs behavior.
type DiscoverOption func(*discoverConfig)

type discoverConfig struct {
	concurrency int
	retryDelays []time.Duration
	onURL       func(string)
}

// WithConcurrency sets the number of concurrent workers for URL discovery.
// Defaults to 3 if not specified (lower than full crawl to avoid overwhelming browsers).
func WithConcurrency(n int) DiscoverOption {
	return func(c *discoverConfig) {
		c.concurrency = n
	}
}

// WithRetryDelays sets the retry delays for failed fetches.
// Defaults to DefaultRetryDelays() if not specified.
func WithRetryDelays(delays []time.Duration) DiscoverOption {
	return func(c *discoverConfig) {
		c.retryDelays = delays
	}
}

// WithOnURL sets a callback that is invoked for each URL as it is discovered.
// This enables streaming output instead of waiting for all URLs to be collected.
func WithOnURL(fn func(string)) DiscoverOption {
	return func(c *discoverConfig) {
		c.onURL = fn
	}
}
