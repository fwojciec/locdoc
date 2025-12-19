package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure BaseSelector implements locdoc.LinkSelector at compile time.
var _ locdoc.LinkSelector = (*goquery.BaseSelector)(nil)

func TestBaseSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewBaseSelector()
	if got := s.Name(); got != "base" {
		t.Errorf("Name() = %q, want %q", got, "base")
	}
}

func TestBaseSelector_ExtractLinks(t *testing.T) {
	t.Parallel()

	t.Run("extracts links from nav with navigation priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/intro">Introduction</a>
	<a href="/docs/guide">Guide</a>
</nav>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "Introduction", links[0].Text)

		assert.Equal(t, "https://example.com/docs/guide", links[1].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[1].Priority)
	})

	t.Run("extracts links from aside with TOC priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<aside>
	<a href="/docs/section1">Section 1</a>
	<a href="/docs/section2">Section 2</a>
</aside>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/section1", links[0].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
		assert.Equal(t, "sidebar", links[0].Source)

		assert.Equal(t, "https://example.com/docs/section2", links[1].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[1].Priority)
	})

	t.Run("extracts links from main/article with content priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<main>
	<article>
		<p>Check out <a href="/docs/related">related docs</a> for more.</p>
	</article>
</main>
</body>
</html>`

		s := goquery.NewBaseSelector()
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
	<a href="/privacy">Privacy Policy</a>
	<a href="/terms">Terms of Service</a>
</footer>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/privacy", links[0].URL)
		assert.Equal(t, locdoc.PriorityFooter, links[0].Priority)
		assert.Equal(t, "footer", links[0].Source)
	})

	t.Run("filters out external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/intro">Internal Link</a>
	<a href="https://external.com/page">External Link</a>
	<a href="https://example.com/docs/guide">Same Host Link</a>
</nav>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		// Only internal links should be returned
		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
		assert.Equal(t, "https://example.com/docs/guide", links[1].URL)
	})

	t.Run("deduplicates links keeping highest priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/guide">Guide</a>
</nav>
<footer>
	<a href="/docs/guide">Guide in Footer</a>
</footer>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		// Should keep the nav link (higher priority than footer)
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})
}
