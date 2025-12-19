package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure MkDocsSelector implements locdoc.LinkSelector at compile time.
var _ locdoc.LinkSelector = (*goquery.MkDocsSelector)(nil)

func TestMkDocsSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewMkDocsSelector()
	assert.Equal(t, "mkdocs", s.Name())
}

func TestMkDocsSelector_ExtractLinks(t *testing.T) {
	t.Parallel()

	t.Run("extracts links from md-nav--primary with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en">
<head><title>MkDocs Material</title></head>
<body data-md-color-scheme="default">
<nav class="md-nav md-nav--primary">
	<ul class="md-nav__list">
		<li class="md-nav__item"><a class="md-nav__link" href="/getting-started/">Getting Started</a></li>
		<li class="md-nav__item"><a class="md-nav__link" href="/configuration/">Configuration</a></li>
	</ul>
</nav>
</body>
</html>`

		s := goquery.NewMkDocsSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/getting-started/", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "Getting Started", links[0].Text)

		assert.Equal(t, "https://example.com/configuration/", links[1].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[1].Priority)
	})

	t.Run("extracts links from data-md-component navigation", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>MkDocs</title></head>
<body>
<div data-md-component="navigation">
	<nav class="md-nav">
		<ul class="md-nav__list">
			<li><a href="/docs/intro">Intro</a></li>
			<li><a href="/docs/guide">Guide</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewMkDocsSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("extracts links from md-sidebar--secondary with TOC priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>MkDocs Material</title></head>
<body>
<nav class="md-nav md-nav--primary">
	<ul><li><a href="/docs/page">Page</a></li></ul>
</nav>
<aside class="md-sidebar md-sidebar--secondary">
	<nav class="md-nav md-nav--secondary">
		<ul class="md-nav__list">
			<li><a href="#overview">Overview</a></li>
			<li><a href="#installation">Installation</a></li>
		</ul>
	</nav>
</aside>
</body>
</html>`

		s := goquery.NewMkDocsSelector()
		links, err := s.ExtractLinks(html, "https://example.com/docs/page")

		require.NoError(t, err)
		// Should have primary nav link + TOC links
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

	t.Run("extracts links from data-md-component toc", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>MkDocs</title></head>
<body>
<div data-md-component="toc">
	<nav class="md-nav">
		<ul>
			<li><a href="#section-1">Section 1</a></li>
			<li><a href="#section-2">Section 2</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewMkDocsSelector()
		links, err := s.ExtractLinks(html, "https://example.com/page")

		require.NoError(t, err)
		require.Len(t, links, 2)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
	})

	t.Run("filters external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>MkDocs</title></head>
<body>
<nav class="md-nav md-nav--primary">
	<ul>
		<li><a href="/docs/intro">Internal</a></li>
		<li><a href="https://github.com/project">GitHub</a></li>
	</ul>
</nav>
</body>
</html>`

		s := goquery.NewMkDocsSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
	})

	t.Run("deduplicates links keeping highest priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>MkDocs</title></head>
<body>
<nav class="md-nav md-nav--primary">
	<ul><li><a href="/docs/intro">Intro in Nav</a></li></ul>
</nav>
<article class="md-content__inner">
	<p>See <a href="/docs/intro">the intro</a> for more.</p>
</article>
</body>
</html>`

		s := goquery.NewMkDocsSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		s := goquery.NewMkDocsSelector()
		links, err := s.ExtractLinks("", "https://example.com")

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><nav class="md-nav"><a href="/docs">Docs</a></nav></body></html>`

		s := goquery.NewMkDocsSelector()
		_, err := s.ExtractLinks(html, "://invalid")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})
}
