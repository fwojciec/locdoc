package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure SphinxSelector implements locdoc.LinkSelector at compile time.
var _ locdoc.LinkSelector = (*goquery.SphinxSelector)(nil)

func TestSphinxSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewSphinxSelector()
	assert.Equal(t, "sphinx", s.Name())
}

func TestSphinxSelector_ExtractLinks(t *testing.T) {
	t.Parallel()

	t.Run("extracts links from wy-nav-side with navigation priority (ReadTheDocs theme)", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Sphinx Docs</title></head>
<body>
<nav class="wy-nav-side">
	<div class="wy-side-scroll">
		<div class="wy-menu wy-menu-vertical">
			<ul class="current">
				<li><a class="reference internal" href="intro.html">Introduction</a></li>
				<li><a class="reference internal" href="install.html">Installation</a></li>
			</ul>
		</div>
	</div>
</nav>
</body>
</html>`

		s := goquery.NewSphinxSelector()
		links, err := s.ExtractLinks(html, "https://example.com/docs/")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/intro.html", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "Introduction", links[0].Text)

		assert.Equal(t, "https://example.com/docs/install.html", links[1].URL)
	})

	t.Run("extracts links from sphinxsidebar with navigation priority (classic theme)", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Sphinx Classic</title></head>
<body>
<div class="sphinxsidebar">
	<div class="sphinxsidebarwrapper">
		<h3><a href="index.html">Table of Contents</a></h3>
		<ul>
			<li><a class="reference internal" href="intro.html">Intro</a></li>
			<li><a class="reference internal" href="api.html">API</a></li>
		</ul>
	</div>
</div>
</body>
</html>`

		s := goquery.NewSphinxSelector()
		links, err := s.ExtractLinks(html, "https://example.com/")

		require.NoError(t, err)
		require.Len(t, links, 3) // Including the TOC header link

		// All navigation links should have navigation priority
		for _, l := range links {
			assert.Equal(t, locdoc.PriorityNavigation, l.Priority)
		}
	})

	t.Run("extracts links from toctree-wrapper with TOC priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Sphinx Docs</title></head>
<body>
<nav class="wy-nav-side">
	<div class="wy-menu-vertical">
		<ul><li><a href="page.html">Page</a></li></ul>
	</div>
</nav>
<div class="toctree-wrapper compound">
	<ul>
		<li class="toctree-l1"><a class="reference internal" href="#section-1">Section 1</a></li>
		<li class="toctree-l1"><a class="reference internal" href="#section-2">Section 2</a></li>
	</ul>
</div>
</body>
</html>`

		s := goquery.NewSphinxSelector()
		links, err := s.ExtractLinks(html, "https://example.com/docs/page.html")

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

	t.Run("extracts links from localtoc with TOC priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Sphinx</title></head>
<body>
<div id="localtoc">
	<ul>
		<li><a href="#overview">Overview</a></li>
		<li><a href="#details">Details</a></li>
	</ul>
</div>
</body>
</html>`

		s := goquery.NewSphinxSelector()
		links, err := s.ExtractLinks(html, "https://example.com/page.html")

		require.NoError(t, err)
		require.Len(t, links, 2)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
	})

	t.Run("filters external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Sphinx</title></head>
<body>
<nav class="wy-nav-side">
	<div class="wy-menu-vertical">
		<ul>
			<li><a href="intro.html">Internal</a></li>
			<li><a href="https://github.com/project">GitHub</a></li>
		</ul>
	</div>
</nav>
</body>
</html>`

		s := goquery.NewSphinxSelector()
		links, err := s.ExtractLinks(html, "https://example.com/")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/intro.html", links[0].URL)
	})

	t.Run("deduplicates links keeping highest priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Sphinx</title></head>
<body>
<nav class="wy-nav-side">
	<div class="wy-menu-vertical">
		<ul><li><a href="intro.html">Intro in Nav</a></li></ul>
	</div>
</nav>
<div class="document">
	<div class="body">
		<p>See <a href="intro.html">the intro</a> for more.</p>
	</div>
</div>
</body>
</html>`

		s := goquery.NewSphinxSelector()
		links, err := s.ExtractLinks(html, "https://example.com/")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		s := goquery.NewSphinxSelector()
		links, err := s.ExtractLinks("", "https://example.com")

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><nav class="wy-nav-side"><a href="docs.html">Docs</a></nav></body></html>`

		s := goquery.NewSphinxSelector()
		_, err := s.ExtractLinks(html, "://invalid")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})
}
