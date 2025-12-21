// Package http provides an HTTP-based implementation of locdoc.Fetcher
// for fetching content from static sites that don't require JavaScript rendering.
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fwojciec/locdoc"
)

// DefaultFetchTimeout is the default timeout for HTTP requests.
// Kept consistent with rod.DefaultFetchTimeout (10s).
const DefaultFetchTimeout = 10 * time.Second

// Ensure Fetcher implements locdoc.Fetcher at compile time.
var _ locdoc.Fetcher = (*Fetcher)(nil)

// Fetcher retrieves HTML content from URLs using HTTP requests.
// Unlike rod.Fetcher, this does not execute JavaScript and is suitable
// for static sites only. Fetcher is safe for concurrent use by multiple
// goroutines.
type Fetcher struct {
	client *http.Client
}

// config holds the configuration options for a Fetcher.
type config struct {
	timeout time.Duration
}

// Option configures a Fetcher.
type Option func(*config)

// WithTimeout sets the timeout for HTTP requests.
// Defaults to DefaultFetchTimeout (10s) if not specified.
func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}

// NewFetcher creates a new HTTP-based Fetcher.
func NewFetcher(opts ...Option) *Fetcher {
	cfg := &config{
		timeout: DefaultFetchTimeout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Fetcher{
		client: &http.Client{
			Timeout: cfg.timeout,
		},
	}
}

// Fetch retrieves the HTML content from the given URL.
func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Drain body to enable connection reuse
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("HTTP %d %s for %s", resp.StatusCode, http.StatusText(resp.StatusCode), url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Close releases resources. For HTTP fetcher this is a no-op since
// http.Client doesn't require explicit cleanup.
func (f *Fetcher) Close() error {
	return nil
}
