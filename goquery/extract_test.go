package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLinksWithConfigs(t *testing.T) {
	t.Parallel()

	t.Run("extracts links using provided selector configs", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/intro">Introduction</a>
	<a href="/docs/guide">Guide</a>
</nav>
<aside>
	<a href="/docs/section1">Section 1</a>
</aside>
</body>
</html>`

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
			{Selector: "aside a[href]", Priority: locdoc.PriorityTOC, Source: "sidebar"},
		}

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com", configs)

		require.NoError(t, err)
		require.Len(t, links, 3)

		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
		assert.Equal(t, "nav", links[0].Source)

		assert.Equal(t, "https://example.com/docs/guide", links[1].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[1].Priority)

		assert.Equal(t, "https://example.com/docs/section1", links[2].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[2].Priority)
		assert.Equal(t, "sidebar", links[2].Source)
	})

	t.Run("deduplicates links keeping highest priority", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/guide">Guide in Nav</a>
</nav>
<footer>
	<a href="/docs/guide">Guide in Footer</a>
</footer>
</body>
</html>`

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
			{Selector: "footer a[href]", Priority: locdoc.PriorityFooter, Source: "footer"},
		}

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com", configs)

		require.NoError(t, err)
		require.Len(t, links, 1)

		// Should keep the nav link (higher priority than footer)
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
		assert.Equal(t, locdoc.PriorityNavigation, links[0].Priority)
	})

	t.Run("updates to higher priority when found later", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/guide">Guide in Nav</a>
</nav>
<aside>
	<a href="/docs/guide">Guide in Sidebar</a>
</aside>
</body>
</html>`

		// Process nav first (lower priority), then aside (higher priority)
		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
			{Selector: "aside a[href]", Priority: locdoc.PriorityTOC, Source: "sidebar"},
		}

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com", configs)

		require.NoError(t, err)
		require.Len(t, links, 1)

		// Should update to aside link (PriorityTOC > PriorityNavigation)
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
		assert.Equal(t, locdoc.PriorityTOC, links[0].Priority)
		assert.Equal(t, "sidebar", links[0].Source)
	})

	t.Run("filters external links", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/intro">Internal Link</a>
	<a href="https://external.com/page">External Link</a>
</nav>
</body>
</html>`

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		}

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com", configs)

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
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
</nav>
</body>
</html>`

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		}

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com", configs)

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs/intro", links[0].URL)
	})

	t.Run("strips fragments from URLs", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/guide#section1">Section Link</a>
</nav>
</body>
</html>`

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		}

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com", configs)

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
	})

	t.Run("returns error for invalid base URL", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><nav><a href="/docs">Docs</a></nav></body></html>`

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		}

		_, err := goquery.ExtractLinksWithConfigs(html, "://invalid-url", configs)

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		t.Parallel()

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		}

		links, err := goquery.ExtractLinksWithConfigs("", "https://example.com", configs)

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("handles empty configs", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><nav><a href="/docs">Docs</a></nav></body></html>`

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com", nil)

		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("extracts link text", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<body>
<nav>
	<a href="/docs/intro">  Introduction  </a>
</nav>
</body>
</html>`

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		}

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com", configs)

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "Introduction", links[0].Text)
	})

	t.Run("filters self-referential anchor links", func(t *testing.T) {
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

		configs := []goquery.SelectorConfig{
			{Selector: "nav a[href]", Priority: locdoc.PriorityNavigation, Source: "nav"},
		}

		links, err := goquery.ExtractLinksWithConfigs(html, "https://example.com/current/page", configs)

		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "https://example.com/docs/guide", links[0].URL)
	})
}
