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
// for static sites only.
type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

// Option configures a Fetcher.
type Option func(*Fetcher)

// WithTimeout sets the timeout for HTTP requests.
// Defaults to DefaultFetchTimeout (10s) if not specified.
func WithTimeout(d time.Duration) Option {
	return func(f *Fetcher) {
		f.timeout = d
	}
}

// NewFetcher creates a new HTTP-based Fetcher.
func NewFetcher(opts ...Option) *Fetcher {
	f := &Fetcher{
		timeout: DefaultFetchTimeout,
	}
	for _, opt := range opts {
		opt(f)
	}

	f.client = &http.Client{
		Timeout: f.timeout,
	}

	return f
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
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
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
