package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/fwojciec/locdoc"
	locdochttp "github.com/fwojciec/locdoc/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSitemapService_DiscoverURLs_FromRobotsTxt(t *testing.T) {
	t.Parallel()

	robotsTxt := `User-agent: *
Disallow: /private/
Sitemap: {{BASE}}/sitemap.xml
`
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
  <url><loc>{{BASE}}/docs/guide</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/robots.txt":  robotsTxt,
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, nil)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, srv.URL+"/docs/intro")
	assert.Contains(t, urls, srv.URL+"/docs/guide")
}

func TestSitemapService_DiscoverURLs_FallbackToSitemapXML(t *testing.T) {
	t.Parallel()

	// No robots.txt, should fallback to /sitemap.xml
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/page1</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, nil)

	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Contains(t, urls, srv.URL+"/page1")
}

func TestSitemapService_DiscoverURLs_SitemapIndex(t *testing.T) {
	t.Parallel()

	sitemapIndex := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>{{BASE}}/sitemap-docs.xml</loc></sitemap>
  <sitemap><loc>{{BASE}}/sitemap-api.xml</loc></sitemap>
</sitemapindex>`

	sitemapDocs := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
</urlset>`

	sitemapAPI := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/api/reference</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml":      sitemapIndex,
		"/sitemap-docs.xml": sitemapDocs,
		"/sitemap-api.xml":  sitemapAPI,
	})
	defer srv.Close()

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, nil)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, srv.URL+"/docs/intro")
	assert.Contains(t, urls, srv.URL+"/api/reference")
}

func TestSitemapService_DiscoverURLs_WithIncludeFilter(t *testing.T) {
	t.Parallel()

	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
  <url><loc>{{BASE}}/blog/post1</loc></url>
  <url><loc>{{BASE}}/docs/guide</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	filter := &locdoc.URLFilter{
		Include: []*regexp.Regexp{regexp.MustCompile(`/docs/`)},
	}

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, filter)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, srv.URL+"/docs/intro")
	assert.Contains(t, urls, srv.URL+"/docs/guide")
}

func TestSitemapService_DiscoverURLs_WithExcludeFilter(t *testing.T) {
	t.Parallel()

	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
  <url><loc>{{BASE}}/docs/internal/debug</loc></url>
  <url><loc>{{BASE}}/docs/guide</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	filter := &locdoc.URLFilter{
		Exclude: []*regexp.Regexp{regexp.MustCompile(`/internal/`)},
	}

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, filter)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, srv.URL+"/docs/intro")
	assert.Contains(t, urls, srv.URL+"/docs/guide")
}

func TestSitemapService_DiscoverURLs_ContextCancellation(t *testing.T) {
	t.Parallel()

	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/page1</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	svc := locdochttp.NewSitemapService(srv.Client())
	_, err := svc.DiscoverURLs(ctx, srv.URL, nil)

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSitemapService_DiscoverURLs_MultipleSitemapsInRobots(t *testing.T) {
	t.Parallel()

	robotsTxt := `User-agent: *
Sitemap: {{BASE}}/sitemap1.xml
Sitemap: {{BASE}}/sitemap2.xml
`
	sitemap1 := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/page1</loc></url>
</urlset>`

	sitemap2 := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/page2</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/robots.txt":   robotsTxt,
		"/sitemap1.xml": sitemap1,
		"/sitemap2.xml": sitemap2,
	})
	defer srv.Close()

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, nil)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, srv.URL+"/page1")
	assert.Contains(t, urls, srv.URL+"/page2")
}

func TestSitemapService_DiscoverURLs_NoSitemapFound(t *testing.T) {
	t.Parallel()

	// No robots.txt, no sitemap.xml
	srv := newTestServer(t, map[string]string{})
	defer srv.Close()

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, nil)

	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestSitemapService_DiscoverURLs_DeduplicatesURLsAcrossSitemaps(t *testing.T) {
	t.Parallel()

	// Two sitemaps with overlapping URLs
	robotsTxt := `Sitemap: {{BASE}}/sitemap1.xml
Sitemap: {{BASE}}/sitemap2.xml
`
	sitemap1 := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/shared</loc></url>
  <url><loc>{{BASE}}/unique1</loc></url>
</urlset>`

	sitemap2 := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/shared</loc></url>
  <url><loc>{{BASE}}/unique2</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/robots.txt":   robotsTxt,
		"/sitemap1.xml": sitemap1,
		"/sitemap2.xml": sitemap2,
	})
	defer srv.Close()

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, nil)

	require.NoError(t, err)
	// Should have 3 unique URLs, not 4 (shared appears in both sitemaps)
	assert.Len(t, urls, 3)
	assert.Contains(t, urls, srv.URL+"/shared")
	assert.Contains(t, urls, srv.URL+"/unique1")
	assert.Contains(t, urls, srv.URL+"/unique2")
}

// newTestServer creates a test HTTP server with the given path->content mapping.
// Content strings may contain {{BASE}} which is replaced with the server URL.
func newTestServer(t *testing.T, content map[string]string) *httptest.Server {
	t.Helper()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := content[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		// Replace {{BASE}} with actual server URL
		body = replaceBaseURL(body, srv.URL)

		// Set content type based on path
		if r.URL.Path == "/robots.txt" {
			w.Header().Set("Content-Type", "text/plain")
		} else {
			w.Header().Set("Content-Type", "application/xml")
		}
		_, _ = w.Write([]byte(body))
	}))

	return srv
}

func replaceBaseURL(content, baseURL string) string {
	return regexp.MustCompile(`\{\{BASE\}\}`).ReplaceAllString(content, baseURL)
}

func TestSitemapService_DiscoverURLs_FiltersBySourcePathPrefix(t *testing.T) {
	t.Parallel()

	// Sitemap contains URLs from multiple paths
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
  <url><loc>{{BASE}}/docs/guide</loc></url>
  <url><loc>{{BASE}}/examples/basic</loc></url>
  <url><loc>{{BASE}}/essays/htmx</loc></url>
  <url><loc>{{BASE}}/</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	// Request with /docs/ path - should only get /docs/* URLs
	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL+"/docs/", nil)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, srv.URL+"/docs/intro")
	assert.Contains(t, urls, srv.URL+"/docs/guide")
}

func TestSitemapService_DiscoverURLs_NoFilterForRootPath(t *testing.T) {
	t.Parallel()

	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
  <url><loc>{{BASE}}/examples/basic</loc></url>
  <url><loc>{{BASE}}/</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	// Request with root path - should get all URLs
	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL+"/", nil)

	require.NoError(t, err)
	assert.Len(t, urls, 3)
}

func TestSitemapService_DiscoverURLs_PathPrefixCombinesWithExplicitFilter(t *testing.T) {
	t.Parallel()

	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
  <url><loc>{{BASE}}/docs/internal/debug</loc></url>
  <url><loc>{{BASE}}/docs/guide</loc></url>
  <url><loc>{{BASE}}/examples/basic</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	// Request with /docs/ path AND exclude /internal/
	filter := &locdoc.URLFilter{
		Exclude: []*regexp.Regexp{regexp.MustCompile(`/internal/`)},
	}

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL+"/docs/", filter)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, srv.URL+"/docs/intro")
	assert.Contains(t, urls, srv.URL+"/docs/guide")
}

func TestSitemapService_DiscoverURLs_PathPrefixWithoutTrailingSlash(t *testing.T) {
	t.Parallel()

	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
  <url><loc>{{BASE}}/docs/guide</loc></url>
  <url><loc>{{BASE}}/documentation/api</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	// Request with /docs path (no trailing slash) should still work and not match /documentation
	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL+"/docs", nil)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, srv.URL+"/docs/intro")
	assert.Contains(t, urls, srv.URL+"/docs/guide")
	assert.NotContains(t, urls, srv.URL+"/documentation/api")
}

func TestSitemapService_DiscoverURLs_PathPrefixRespectsBoundaries(t *testing.T) {
	t.Parallel()

	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/api/v1/docs</loc></url>
  <url><loc>{{BASE}}/api/v1/reference</loc></url>
  <url><loc>{{BASE}}/api/v2/docs</loc></url>
  <url><loc>{{BASE}}/api/v20/docs</loc></url>
</urlset>`

	srv := newTestServer(t, map[string]string{
		"/sitemap.xml": sitemapXML,
	})
	defer srv.Close()

	// /api/v2/ should match /api/v2/* but not /api/v20/*
	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL+"/api/v2/", nil)

	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Contains(t, urls, srv.URL+"/api/v2/docs")
}

func TestSitemapService_DiscoverURLs_SitemapDeclaredInRobotsBut404(t *testing.T) {
	t.Parallel()

	// robots.txt declares a sitemap, but the sitemap doesn't exist (404)
	// This should return empty URLs (not error) to allow recursive crawling fallback
	robotsTxt := `User-agent: *
Sitemap: {{BASE}}/sitemap.xml
`

	srv := newTestServer(t, map[string]string{
		"/robots.txt": robotsTxt,
		// Note: /sitemap.xml is NOT served (will return 404)
	})
	defer srv.Close()

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL, nil)

	require.NoError(t, err, "404 on declared sitemap should not be an error")
	assert.Empty(t, urls, "should return empty URLs when sitemap doesn't exist")
}

func TestSitemapService_DiscoverURLs_FindsSitemapAtDomainRoot(t *testing.T) {
	t.Parallel()

	// Sitemap exists ONLY at domain root, not under /docs/
	// This test documents that sitemap discovery always happens at the domain root,
	// regardless of the path in the source URL.
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>{{BASE}}/docs/intro</loc></url>
</urlset>`

	var requestedPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)

		if r.URL.Path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			body := replaceBaseURL(sitemapXML, "http://"+r.Host)
			_, _ = w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc := locdochttp.NewSitemapService(srv.Client())
	urls, err := svc.DiscoverURLs(context.Background(), srv.URL+"/docs/", nil)

	require.NoError(t, err)
	assert.Len(t, urls, 1)

	// Verify we looked for sitemap at root, not under /docs/
	assert.Contains(t, requestedPaths, "/robots.txt", "should check robots.txt at root")
	assert.Contains(t, requestedPaths, "/sitemap.xml", "should check sitemap.xml at root")
	assert.NotContains(t, requestedPaths, "/docs/robots.txt", "should NOT check robots.txt under path")
	assert.NotContains(t, requestedPaths, "/docs/sitemap.xml", "should NOT check sitemap.xml under path")
}
