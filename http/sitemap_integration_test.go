//go:build integration

package http_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	locdochttp "github.com/fwojciec/locdoc/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSitemapService_Integration_HtmxDocs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	svc := locdochttp.NewSitemapService(nil)

	// htmx.org has a sitemap declared in robots.txt
	urls, err := svc.DiscoverURLs(ctx, "https://htmx.org", nil)
	require.NoError(t, err)

	// Should find at least some URLs
	assert.NotEmpty(t, urls, "expected at least some URLs from htmx.org sitemap")
	t.Logf("Found %d URLs from htmx.org sitemap", len(urls))

	// Verify URLs look reasonable (show first 5)
	for _, u := range urls[:min(5, len(urls))] {
		t.Logf("  - %s", u)
	}
}

func TestSitemapService_Integration_HtmxDocs_WithFilter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	svc := locdochttp.NewSitemapService(nil)

	// Filter to only /docs/ pages
	filter := &locdoc.URLFilter{
		Include: []*regexp.Regexp{regexp.MustCompile(`/docs/`)},
	}

	urls, err := svc.DiscoverURLs(ctx, "https://htmx.org", filter)
	require.NoError(t, err)

	// Should find some docs URLs
	assert.NotEmpty(t, urls, "expected some /docs/ URLs from htmx.org")
	t.Logf("Found %d /docs/ URLs from htmx.org sitemap", len(urls))

	// Verify all URLs match filter
	for _, u := range urls {
		assert.Contains(t, u, "/docs/", "URL should contain /docs/")
	}
}
