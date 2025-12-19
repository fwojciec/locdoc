package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure NextraSelector implements locdoc.LinkSelector at compile time.
var _ locdoc.LinkSelector = (*goquery.NextraSelector)(nil)

func TestNextraSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewNextraSelector()
	assert.Equal(t, "nextra", s.Name())
}

func TestNextraSelector_ExtractLinks(t *testing.T) {
	t.Parallel()

	t.Run("extracts links from nextra-sidebar with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en" dir="ltr">
<head><title>Nextra Docs</title></head>
<body>
<nav class="nextra-sidebar">
	<ul>
		<li><a href="/docs/getting-started">Getting Started</a></li>
		<li><a href="/docs/configuration">Configuration</a></li>
	</ul>
</nav>
</body>
</html>`

		s := goquery.NewNextraSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/getting-started", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "Getting Started", links[0].Text)

		assert.Equal(t, "https://example.com/docs/configuration", links[1].URL)
	})

	t.Run("extracts links from nextra-toc with TOC priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Nextra</title></head>
<body>
<nav class="nextra-sidebar">
	<ul><li><a href="/docs/page">Page</a></li></ul>
</nav>
<div class="nextra-toc">
	<nav>
		<ul>
			<li><a href="#overview">Overview</a></li>
			<li><a href="#installation">Installation</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewNextraSelector()
		links, err := s.ExtractLinks(html, "https://example.com/docs/page")

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

	t.Run("extracts links from nextra-navbar with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Nextra</title></head>
<body>
<nav class="nextra-navbar">
	<ul>
		<li><a href="/docs">Docs</a></li>
		<li><a href="/blog">Blog</a></li>
	</ul>
</nav>
</body>
</html>`

		s := goquery.NewNextraSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("filters external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Nextra</title></head>
<body>
<nav class="nextra-sidebar">
	<ul>
		<li><a href="/docs/intro">Internal</a></li>
		<li><a href="https://github.com/project">GitHub</a></li>
	</ul>
</nav>
</body>
</html>`

		s := goquery.NewNextraSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
	})

	t.Run("deduplicates links keeping highest priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Nextra</title></head>
<body>
<nav class="nextra-sidebar">
	<ul><li><a href="/docs/intro">Intro in Sidebar</a></li></ul>
</nav>
<main class="nx-w-full">
	<p>See <a href="/docs/intro">the intro</a> for more.</p>
</main>
</body>
</html>`

		s := goquery.NewNextraSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		s := goquery.NewNextraSelector()
		links, err := s.ExtractLinks("", "https://example.com")

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><nav class="nextra-sidebar"><a href="/docs">Docs</a></nav></body></html>`

		s := goquery.NewNextraSelector()
		_, err := s.ExtractLinks(html, "://invalid")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})
}
