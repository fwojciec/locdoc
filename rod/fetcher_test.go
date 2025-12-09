//go:build integration

package rod_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/rod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure Fetcher implements locdoc.Fetcher.
var _ locdoc.Fetcher = (*rod.Fetcher)(nil)

func TestFetcher_Fetch_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Server that delays response
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't respond - let context timeout
		select {}
	}))
	defer srv.Close()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)
	defer fetcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = fetcher.Fetch(ctx, srv.URL)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFetcher_Fetch_ReturnsRenderedHTML(t *testing.T) {
	t.Parallel()

	// Serve a page that uses JavaScript to add content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<div id="content">Loading...</div>
<script>
document.getElementById('content').textContent = 'JavaScript Rendered';
</script>
</body>
</html>`))
	}))
	defer srv.Close()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)
	defer fetcher.Close()

	html, err := fetcher.Fetch(context.Background(), srv.URL)

	require.NoError(t, err)
	assert.Contains(t, html, "JavaScript Rendered")
	assert.NotContains(t, html, "Loading...")
}
