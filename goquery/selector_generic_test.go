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
}
