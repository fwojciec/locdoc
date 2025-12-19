package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocusaurusSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewDocusaurusSelector()
	assert.Equal(t, "docusaurus", s.Name())
}

func TestDocusaurusSelector_ExtractLinks(t *testing.T) {
	t.Parallel()

	t.Run("extracts links from theme-doc-sidebar-container with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en" data-theme="light">
<head><title>Docusaurus Docs</title></head>
<body>
<div class="theme-doc-sidebar-container">
	<nav class="menu">
		<ul class="menu__list">
			<li class="menu__list-item"><a class="menu__link" href="/docs/intro">Introduction</a></li>
			<li class="menu__list-item"><a class="menu__link" href="/docs/getting-started">Getting Started</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewDocusaurusSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "Introduction", links[0].Text)

		assert.Equal(t, "https://example.com/docs/getting-started", links[1].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[1].Priority)
	})

	t.Run("extracts links from table-of-contents with TOC priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Docusaurus</title></head>
<body>
<div class="theme-doc-sidebar-container">
	<nav class="menu">
		<ul><li><a href="/docs/intro">Intro</a></li></ul>
	</nav>
</div>
<aside class="col col--3">
	<div class="table-of-contents">
		<ul>
			<li><a href="#section-1">Section 1</a></li>
			<li><a href="#section-2">Section 2</a></li>
		</ul>
	</div>
</aside>
</body>
</html>`

		s := goquery.NewDocusaurusSelector()
		links, err := s.ExtractLinks(html, "https://example.com/docs/page")

		require.NoError(t, err)
		// Should have sidebar link + TOC links
		require.Len(t, links, 3)

		// TOC links should have higher priority
		var tocLinks []locdoc.DiscoveredLink
		for _, l := range links {
			if l.Priority == locdoc.PriorityTOC {
				tocLinks = append(tocLinks, l)
			}
		}
		assert.Len(t, tocLinks, 2)
	})

	t.Run("extracts links from navbar with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Docusaurus</title></head>
<body>
<nav class="navbar navbar--fixed-top">
	<div class="navbar__items">
		<a class="navbar__item navbar__link" href="/docs">Docs</a>
		<a class="navbar__item navbar__link" href="/blog">Blog</a>
	</div>
</nav>
</body>
</html>`

		s := goquery.NewDocusaurusSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("filters external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Docusaurus</title></head>
<body>
<div class="theme-doc-sidebar-container">
	<nav class="menu">
		<ul>
			<li><a href="/docs/intro">Internal</a></li>
			<li><a href="https://github.com/project">GitHub</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewDocusaurusSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
	})

	t.Run("deduplicates links keeping highest priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Docusaurus</title></head>
<body>
<div class="theme-doc-sidebar-container">
	<nav class="menu">
		<ul><li><a href="/docs/intro">Intro in Sidebar</a></li></ul>
	</nav>
</div>
<article>
	<p>See <a href="/docs/intro">the introduction</a> for more.</p>
</article>
</body>
</html>`

		s := goquery.NewDocusaurusSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		// Should keep navigation priority (higher than content)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("deduplicates across three priority levels keeping highest", func(t *testing.T) {
		t.Parallel()

		// Same link appears in TOC (110), sidebar (100), and content (50)
		// Should keep TOC priority as it's highest
		html := `<!DOCTYPE html>
<html>
<head><title>Docusaurus</title></head>
<body>
<div class="theme-doc-sidebar-container">
	<nav class="menu">
		<ul><li><a href="/docs/intro">Intro in Sidebar</a></li></ul>
	</nav>
</div>
<aside class="col col--3">
	<div class="table-of-contents">
		<ul><li><a href="/docs/intro">Intro in TOC</a></li></ul>
	</div>
</aside>
<article>
	<p>See <a href="/docs/intro">the introduction</a> in content.</p>
</article>
</body>
</html>`

		s := goquery.NewDocusaurusSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		// Should keep TOC priority (highest: 110 > 100 > 50)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
		assert.Equal(t, "toc", links[0].Source)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		s := goquery.NewDocusaurusSelector()
		links, err := s.ExtractLinks("", "https://example.com")

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><nav class="navbar"><a href="/docs">Docs</a></nav></body></html>`

		s := goquery.NewDocusaurusSelector()
		_, err := s.ExtractLinks(html, "://invalid")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})
}
