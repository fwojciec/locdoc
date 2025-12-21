package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	locdochttp "github.com/fwojciec/locdoc/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("returns HTML body from server", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html><body>Hello World</body></html>"))
		}))
		defer server.Close()

		fetcher := locdochttp.NewFetcher()
		defer fetcher.Close()

		html, err := fetcher.Fetch(context.Background(), server.URL)
		require.NoError(t, err)
		assert.Equal(t, "<html><body>Hello World</body></html>", html)
	})

	t.Run("respects custom timeout option", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			_, _ = w.Write([]byte("response"))
		}))
		defer server.Close()

		// Use a very short timeout that will expire before server responds
		fetcher := locdochttp.NewFetcher(locdochttp.WithTimeout(10 * time.Millisecond))
		defer fetcher.Close()

		_, err := fetcher.Fetch(context.Background(), server.URL)
		require.Error(t, err)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			_, _ = w.Write([]byte("response"))
		}))
		defer server.Close()

		fetcher := locdochttp.NewFetcher()
		defer fetcher.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := fetcher.Fetch(ctx, server.URL)
		require.Error(t, err)
	})

	t.Run("returns error for non-existent host", func(t *testing.T) {
		t.Parallel()

		fetcher := locdochttp.NewFetcher(locdochttp.WithTimeout(100 * time.Millisecond))
		defer fetcher.Close()

		_, err := fetcher.Fetch(context.Background(), "http://non-existent-host.invalid/page")
		require.Error(t, err)
	})

	t.Run("returns error for non-200 status codes", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("404 Not Found"))
		}))
		defer server.Close()

		fetcher := locdochttp.NewFetcher()
		defer fetcher.Close()

		_, err := fetcher.Fetch(context.Background(), server.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

// Compile-time verification that Fetcher implements locdoc.Fetcher
var _ locdoc.Fetcher = (*locdochttp.Fetcher)(nil)
