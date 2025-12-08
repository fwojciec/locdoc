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
