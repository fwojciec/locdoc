package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseSelector_Name(t *testing.T) {
	t.Parallel()

	s := goquery.NewBaseSelector()
	assert.Equal(t, "base", s.Name())
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

	t.Run("deduplicates links updating to higher priority when found later", func(t *testing.T) {
		t.Parallel()

		// PriorityTOC (110) > PriorityNavigation (100)
		// aside is processed after nav, so this tests the update path
		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
	<a href="/docs/guide">Guide in Nav</a>
</nav>
<aside>
	<a href="/docs/guide">Guide in Sidebar</a>
</aside>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		// Should update to aside link (PriorityTOC > PriorityNavigation)
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
		assert.Equal(t, "sidebar", links[0].Source)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><nav><a href="/docs">Docs</a></nav></body></html>`

		s := goquery.NewBaseSelector()
		_, err := s.ExtractLinks(html, "://invalid-url")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks("", "https://example.com")

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("preserves fragments and query params in URLs", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/guide#section1">Section Link</a>
	<a href="/docs/search?q=test">Search Link</a>
</nav>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 2)

		assert.Equal(t, "https://example.com/docs/guide#section1", links[0].URL)
		assert.Equal(t, "https://example.com/docs/search?q=test", links[1].URL)
	})

	t.Run("skips non-HTTP scheme links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/intro">Real Link</a>
	<a href="javascript:void(0)">JS Link</a>
	<a href="mailto:test@example.com">Email Link</a>
	<a href="tel:+1234567890">Phone Link</a>
</nav>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
	})

	t.Run("handles anchor-only links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="#section1">Anchor Only</a>
	<a href="/docs/guide">Full Path</a>
</nav>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com/current/page")

		require.NoError(t, err)
		require.Len(t, links, 2)

		// Anchor-only resolves to current page with fragment
		assert.Equal(t, "https://example.com/current/page#section1", links[0].URL)
		assert.Equal(t, "https://example.com/docs/guide", links[1].URL)
	})

	t.Run("handles protocol-relative URLs", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="//example.com/docs/guide">Protocol Relative Same Host</a>
	<a href="//other.com/page">Protocol Relative External</a>
</nav>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		// Only same-host protocol-relative URL should be included
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
	})

	t.Run("filters subdomain links (exact host match)", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/intro">Same Host</a>
	<a href="https://docs.example.com/guide">Subdomain Link</a>
	<a href="https://api.example.com/reference">Another Subdomain</a>
</nav>
</body>
</html>`

		s := goquery.NewBaseSelector()
		links, err := s.ExtractLinks(html, "https://example.com")

		require.NoError(t, err)
		require.Len(t, links, 1)

		// Only exact host match, subdomains are filtered
		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
	})
}
