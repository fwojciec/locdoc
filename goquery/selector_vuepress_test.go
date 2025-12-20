package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVuePressSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewVuePressSelector()
	assert.Equal(t, "vuepress", s.Name())
}

func TestVuePressSelector_ExtractLinks(t *testing.T) {
	t.Parallel()

	// VuePress 1.x / Classic theme tests
	t.Run("extracts links from sidebar-links with navigation priority (VuePress 1.x)", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en-US">
<head><title>VuePress Docs</title></head>
<body>
<aside class="sidebar">
	<ul class="sidebar-links">
		<li class="sidebar-link"><a href="/guide/">Guide</a></li>
		<li class="sidebar-link"><a href="/api/">API</a></li>
	</ul>
</aside>
</body>
</html>`

		s := goquery.NewVuePressSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/guide/", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "Guide", links[0].Text)

		assert.Equal(t, "https://example.com/api/", links[1].URL)
	})

	t.Run("extracts links from theme-default-content with content priority (VuePress 1.x)", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>VuePress</title></head>
<body>
<div class="theme-default-content">
	<h1>Getting Started</h1>
	<p>See <a href="/guide/intro">the introduction</a> for details.</p>
</div>
</body>
</html>`

		s := goquery.NewVuePressSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, locdoc.PriorityContent, links[0].Priority)
	})

	// VitePress tests
	t.Run("extracts links from VPSidebar with navigation priority (VitePress)", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en-US" class="dark">
<head><title>VitePress</title></head>
<body>
<aside class="VPSidebar">
	<nav class="VPSidebarNav">
		<ul>
			<li><a class="link" href="/guide/getting-started">Getting Started</a></li>
			<li><a class="link" href="/guide/configuration">Configuration</a></li>
		</ul>
	</nav>
</aside>
</body>
</html>`

		s := goquery.NewVuePressSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/guide/getting-started", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("extracts links from VPDocAsideOutline with TOC priority (VitePress)", func(t *testing.T) {
		t.Parallel()

		// Use actual page paths for TOC links (not anchor-only which are self-referential)
		html := `<!DOCTYPE html>
<html>
<head><title>VitePress</title></head>
<body>
<aside class="VPSidebar">
	<nav><ul><li><a href="/guide">Guide</a></li></ul></nav>
</aside>
<div class="VPDocAsideOutline">
	<nav class="VPDocAsideOutlineItem">
		<ul>
			<li><a href="/guide/overview">Overview</a></li>
			<li><a href="/guide/installation">Installation</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewVuePressSelector()
		links, err := s.ExtractLinks(html, "https://example.com/guide/intro")

		require.NoError(t, err)
		require.Len(t, links, 3)

		// Check TOC links have correct priority
		var tocLinks []locdoc.DiscoveredLink
		for _, l := range links {
			if l.Priority == locdoc.PriorityTOC {
				tocLinks = append(tocLinks, l)
			}
		}
		assert.Len(t, tocLinks, 2)
	})

	t.Run("extracts links from VPNav with navigation priority (VitePress)", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>VitePress</title></head>
<body>
<nav class="VPNav">
	<div class="VPNavBar">
		<a class="VPNavBarTitle" href="/">Home</a>
		<a class="VPNavBarMenuLink" href="/guide/">Guide</a>
	</div>
</nav>
</body>
</html>`

		s := goquery.NewVuePressSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("filters external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>VuePress</title></head>
<body>
<aside class="sidebar">
	<ul class="sidebar-links">
		<li><a href="/guide/">Internal</a></li>
		<li><a href="https://github.com/project">GitHub</a></li>
	</ul>
</aside>
</body>
</html>`

		s := goquery.NewVuePressSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/guide/", links[0].URL)
	})

	t.Run("deduplicates links keeping highest priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>VuePress</title></head>
<body>
<aside class="sidebar">
	<ul class="sidebar-links">
		<li><a href="/guide/intro">Intro in Sidebar</a></li>
	</ul>
</aside>
<div class="theme-default-content">
	<p>See <a href="/guide/intro">the intro</a> for more.</p>
</div>
</body>
</html>`

		s := goquery.NewVuePressSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		s := goquery.NewVuePressSelector()
		links, err := s.ExtractLinks("", "https://example.com")

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><aside class="sidebar"><a href="/docs">Docs</a></aside></body></html>`

		s := goquery.NewVuePressSelector()
		_, err := s.ExtractLinks(html, "://invalid")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})
}
