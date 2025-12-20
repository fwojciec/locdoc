package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitBookSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewGitBookSelector()
	assert.Equal(t, "gitbook", s.Name())
}

func TestGitBookSelector_ExtractLinks(t *testing.T) {
	t.Parallel()

	t.Run("extracts links from space.sidebar with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en" class="circular-corners theme-clean tint">
<head><title>GitBook</title></head>
<body>
<div data-testid="space.sidebar">
	<nav>
		<ul>
			<li><a href="/docs/intro">Introduction</a></li>
			<li><a href="/docs/guide">User Guide</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewGitBookSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "Introduction", links[0].Text)

		assert.Equal(t, "https://example.com/docs/guide", links[1].URL)
	})

	t.Run("extracts links from page.desktopTableOfContents with TOC priority", func(t *testing.T) {
		t.Parallel()

		// Use actual page paths for TOC links (not anchor-only which are self-referential)
		html := `<!DOCTYPE html>
<html>
<head><title>GitBook</title></head>
<body>
<div data-testid="space.sidebar">
	<nav><ul><li><a href="/docs/other">Other Page</a></li></ul></nav>
</div>
<div data-testid="page.desktopTableOfContents">
	<nav>
		<ul>
			<li><a href="/docs/section-1">Section 1</a></li>
			<li><a href="/docs/section-2">Section 2</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewGitBookSelector()
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

	t.Run("extracts links from header navigation", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>GitBook</title></head>
<body>
<header data-testid="space.header">
	<nav>
		<a href="/docs">Documentation</a>
		<a href="/api">API Reference</a>
	</nav>
</header>
</body>
</html>`

		s := goquery.NewGitBookSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("filters external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>GitBook</title></head>
<body>
<div data-testid="space.sidebar">
	<nav>
		<ul>
			<li><a href="/docs/intro">Internal</a></li>
			<li><a href="https://github.com/project">GitHub</a></li>
		</ul>
	</nav>
</div>
</body>
</html>`

		s := goquery.NewGitBookSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
	})

	t.Run("deduplicates links keeping highest priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>GitBook</title></head>
<body>
<div data-testid="space.sidebar">
	<nav><ul><li><a href="/docs/intro">Intro in Sidebar</a></li></ul></nav>
</div>
<main data-testid="page.contentEditor">
	<p>See <a href="/docs/intro">the intro</a> for more.</p>
</main>
</body>
</html>`

		s := goquery.NewGitBookSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		s := goquery.NewGitBookSelector()
		links, err := s.ExtractLinks("", "https://example.com")

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><div data-testid="space.sidebar"><a href="/docs">Docs</a></div></body></html>`

		s := goquery.NewGitBookSelector()
		_, err := s.ExtractLinks(html, "://invalid")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})
}
