//go:build integration

package rod_test

import (
	"context"
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

	// htmx.org is a documentation site that uses JS for some rendering
	html, err := fetcher.Fetch(ctx, "https://htmx.org/docs/")
	require.NoError(t, err)

	// Should contain actual content (not just loading indicators)
	assert.NotEmpty(t, html)
	assert.Contains(t, html, "htmx", "expected htmx content in page")
	t.Logf("Fetched %d bytes from htmx.org/docs/", len(html))
}

func TestFetcher_Integration_ReactDocs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fetcher, err := rod.NewFetcher()
	require.NoError(t, err)
	defer fetcher.Close()

	// React docs is heavily JS-rendered
	html, err := fetcher.Fetch(ctx, "https://react.dev/learn")
	require.NoError(t, err)

	// Should contain actual React content
	assert.NotEmpty(t, html)
	assert.Contains(t, html, "React", "expected React content in page")
	t.Logf("Fetched %d bytes from react.dev/learn", len(html))
}
