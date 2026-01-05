//go:build integration

package rod_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestFetcher_Fetch_TimeoutTriggersOnSlowPage(t *testing.T) {
	t.Parallel()

	// Server that delays longer than the fetch timeout
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>delayed</body></html>`))
	}))
	defer srv.Close()

	// Use a short timeout for testing (100ms, shorter than server delay)
	fetcher, err := rod.NewFetcher(rod.WithFetchTimeout(100 * time.Millisecond))
	require.NoError(t, err)
	defer fetcher.Close()

	ctx := context.Background()
	_, err = fetcher.Fetch(ctx, srv.URL)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestFetcher_Close_Idempotent(t *testing.T) {
	t.Parallel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)

	// First close should succeed
	err = fetcher.Close()
	require.NoError(t, err)

	// Second close should also succeed (not panic or error)
	err = fetcher.Close()
	require.NoError(t, err)
}

func TestFetcher_Fetch_AfterClose_ReturnsError(t *testing.T) {
	t.Parallel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)

	err = fetcher.Close()
	require.NoError(t, err)

	_, err = fetcher.Fetch(context.Background(), "http://example.com")

	require.Error(t, err)
	assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	assert.Contains(t, locdoc.ErrorMessage(err), "closed")
}

func TestFetcher_Fetch_SerializesShadowDOMContent(t *testing.T) {
	t.Parallel()

	// Serve a page with Web Components that have shadow DOM containing links.
	// The content uses data-shadow-content attribute to mark what we expect
	// to be serialized from the shadow DOM (not from script).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Shadow DOM Test</title></head>
<body>
<nav-menu></nav-menu>
<script>
class NavMenu extends HTMLElement {
  constructor() {
    super();
    const shadow = this.attachShadow({mode: 'open'});
    shadow.innerHTML = '<a href="/shadow-link-1" data-shadow-content="true">Shadow Link 1</a><a href="/shadow-link-2" data-shadow-content="true">Shadow Link 2</a>';
  }
}
customElements.define('nav-menu', NavMenu);
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
	// Count occurrences of the shadow content marker. In the raw script, it appears
	// in string literals (2 times in the innerHTML assignment). If shadow DOM is
	// properly serialized, it should appear additional times as actual DOM elements,
	// giving us 4 total (2 in script + 2 in serialized shadow DOM).
	markerCount := strings.Count(html, `data-shadow-content="true"`)
	assert.Greater(t, markerCount, 2, "shadow DOM content not serialized: marker found %d times (expected >2)", markerCount)
}
