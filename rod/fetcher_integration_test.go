//go:build integration

package rod_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fwojciec/locdoc/rod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher_Integration_HtmxDocs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)
	defer fetcher.Close()

	html, err := fetcher.Fetch(ctx, "https://htmx.org/docs/")
	require.NoError(t, err)
	assert.NotEmpty(t, html, "expected non-empty HTML response")

	// Verify HTML document structure
	assert.True(t, strings.HasPrefix(strings.TrimSpace(strings.ToLower(html)), "<!doctype html>") ||
		strings.HasPrefix(strings.TrimSpace(strings.ToLower(html)), "<html"),
		"expected valid HTML document start")
	assert.Contains(t, html, "<head>", "expected head tag")
	assert.Contains(t, html, "</head>", "expected closing head tag")
	assert.Contains(t, html, "<body", "expected body tag")
	assert.Contains(t, html, "</body>", "expected closing body tag")
	assert.Contains(t, html, "</html>", "expected closing html tag")

	// Verify JS-rendered navigation content - htmx docs has a sidebar navigation
	// that requires the page to be fully rendered
	assert.Contains(t, html, "htmx in a Nutshell", "expected rendered introduction section")
	assert.Contains(t, html, "Installing", "expected rendered documentation sections")

	// Verify actual documentation content is present (not just placeholders)
	assert.Contains(t, html, "hx-get", "expected htmx attribute documentation")
	assert.Contains(t, html, "hx-post", "expected htmx attribute documentation")

	t.Logf("Fetched %d bytes from htmx.org/docs/", len(html))
}

func TestFetcher_Integration_ReactDocs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)
	defer fetcher.Close()

	// React docs is heavily JS-rendered - requires JavaScript to render content
	html, err := fetcher.Fetch(ctx, "https://react.dev/learn")
	require.NoError(t, err)
	assert.NotEmpty(t, html, "expected non-empty HTML response")

	// Verify HTML document structure
	assert.True(t, strings.HasPrefix(strings.TrimSpace(strings.ToLower(html)), "<!doctype html>") ||
		strings.HasPrefix(strings.TrimSpace(strings.ToLower(html)), "<html"),
		"expected valid HTML document start")
	assert.Contains(t, html, "<head>", "expected head tag")
	assert.Contains(t, html, "</head>", "expected closing head tag")
	assert.Contains(t, html, "<body", "expected body tag")
	assert.Contains(t, html, "</body>", "expected closing body tag")
	assert.Contains(t, html, "</html>", "expected closing html tag")

	// Verify JS-rendered content - React docs is a fully client-rendered React app
	// The page title "Quick Start" only appears after React hydration
	assert.Contains(t, html, "Quick Start", "expected rendered page title")

	// Verify actual tutorial content is present (requires JS execution)
	assert.Contains(t, html, "Creating and nesting components", "expected rendered tutorial content")
	assert.Contains(t, html, "Writing markup with JSX", "expected rendered tutorial content")

	t.Logf("Fetched %d bytes from react.dev/learn", len(html))
}
