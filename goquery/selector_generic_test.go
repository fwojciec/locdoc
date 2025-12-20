package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewGenericSelector()
	assert.Equal(t, "generic", s.Name())
}

func TestGenericSelector_ExtractLinks(t *testing.T) {
	t.Parallel()

	t.Run("extracts links from TOC elements with TOC priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="toc">
	<a href="/docs/section1">Section 1</a>
	<a href="/docs/section2">Section 2</a>
</div>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/section1", links[0].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
		assert.Equal(t, "toc", links[0].Source)
	})

	t.Run("extracts links from sidebar elements with TOC priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="sidebar">
	<a href="/docs/intro">Introduction</a>
</div>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
		assert.Equal(t, "toc", links[0].Source)
	})

	t.Run("extracts links from nav elements with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/guide">Guide</a>
</nav>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "nav", links[0].Source)
	})

	t.Run("extracts links from role=navigation with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div role="navigation">
	<a href="/docs/api">API Reference</a>
</div>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/docs/api", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "nav", links[0].Source)
	})

	t.Run("extracts links from content areas with content priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<main>
	<a href="/docs/related">Related docs</a>
</main>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/docs/related", links[0].URL)
		assert.Equal(t, locdoc.PriorityContent, links[0].Priority)
		assert.Equal(t, "content", links[0].Source)
	})

	t.Run("extracts links from footer with footer priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<footer>
	<a href="/privacy">Privacy</a>
</footer>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/privacy", links[0].URL)
		assert.Equal(t, locdoc.PriorityFooter, links[0].Priority)
		assert.Equal(t, "footer", links[0].Source)
	})

	t.Run("prioritizes TOC over navigation for same link", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/guide">Guide in Nav</a>
</nav>
<div class="toc">
	<a href="/docs/guide">Guide in TOC</a>
</div>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		// TOC has higher priority than nav
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
		assert.Equal(t, "toc", links[0].Source)
	})

	t.Run("does not downgrade TOC to navigation priority", func(t *testing.T) {
		t.Parallel()

		// Link appears in both TOC and nav; TOC is processed first
		// Navigation should not downgrade the priority
		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="toc">
	<a href="/docs/guide">Guide in TOC</a>
</div>
<nav>
	<a href="/docs/guide">Guide in Nav</a>
</nav>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		// Should keep TOC priority (not downgraded to nav)
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
		assert.Equal(t, "toc", links[0].Source)
	})

	t.Run("filters external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/internal">Internal</a>
	<a href="https://external.com/page">External</a>
</nav>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/docs/internal", links[0].URL)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><nav><a href="/docs">Docs</a></nav></body></html>`

		s := goquery.NewGenericSelector()
		_, err := s.ExtractLinks(html, "://invalid-url")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks("", "https://example.com")

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("extracts from menu class elements", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<ul class="menu">
	<a href="/docs/item1">Item 1</a>
</ul>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/docs/item1", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("skips non-HTTP links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/valid">Valid</a>
	<a href="javascript:void(0)">JS Link</a>
	<a href="mailto:test@example.com">Email</a>
</nav>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/docs/valid", links[0].URL)
	})

	t.Run("falls back to path-filtered links when semantic selectors find nothing", func(t *testing.T) {
		t.Parallel()

		// Simulates a Tailwind CSS site with no semantic HTML elements
		// All navigation is in plain divs with utility classes
		// Page has docs links AND other site links - only docs links should be extracted
		html := `<!DOCTYPE html>
<html>
<head><title>TanStack-like Docs</title></head>
<body>
<div class="flex flex-col gap-4">
	<a href="/query/v5/docs/overview">Overview</a>
	<a href="/query/v5/docs/installation">Installation</a>
	<a href="/query/v5/docs/quick-start">Quick Start</a>
	<a href="/query/v4/docs/old-version">Old Version</a>
	<a href="/router/docs/intro">Router Docs</a>
	<a href="https://github.com/tanstack/query">GitHub</a>
</div>
</body>
</html>`

		s := goquery.NewGenericSelector()
		// Base URL includes path - fallback should only include links under this path
		links, err := s.ExtractLinks(html, "https://tanstack.com/query/v5/docs")

		require.NoError(t, err)
		require.Len(t, links, 3) // Only /query/v5/docs/* links

		// All should have fallback priority
		for _, link := range links {
			assert.Equal(t, locdoc.PriorityFallback, link.Priority)
			assert.Equal(t, "fallback", link.Source)
		}

		// Verify the URLs are correct - only v5 docs, not v4 or router
		urls := make([]string, len(links))
		for i, l := range links {
			urls[i] = l.URL
		}
		assert.Contains(t, urls, "https://tanstack.com/query/v5/docs/overview")
		assert.Contains(t, urls, "https://tanstack.com/query/v5/docs/installation")
		assert.Contains(t, urls, "https://tanstack.com/query/v5/docs/quick-start")
		assert.NotContains(t, urls, "https://tanstack.com/query/v4/docs/old-version")
		assert.NotContains(t, urls, "https://tanstack.com/router/docs/intro")
	})

	t.Run("semantic links keep priority while fallback adds additional links", func(t *testing.T) {
		t.Parallel()

		// Page has both nav links AND links in plain divs
		// Nav link keeps higher priority, div link is added with fallback priority
		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/guide">Guide</a>
</nav>
<div class="random-div">
	<a href="/docs/extra">Extra Link in Div</a>
</div>
</body>
</html>`

		s := goquery.NewGenericSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2) // Both links found

		// Find the links by URL
		var navLink, divLink locdoc.DiscoveredLink
		for _, l := range links {
			switch l.URL {
			case "https://example.com/docs/guide":
				navLink = l
			case "https://example.com/docs/extra":
				divLink = l
			}
		}

		// Nav link keeps its navigation priority
		assert.Equal(t, locdoc.PriorityNavigation, navLink.Priority)
		assert.Equal(t, "nav", navLink.Source)

		// Div link gets fallback priority
		assert.Equal(t, locdoc.PriorityFallback, divLink.Priority)
		assert.Equal(t, "fallback", divLink.Source)
	})
}
