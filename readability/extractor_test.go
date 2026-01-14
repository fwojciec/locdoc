package readability_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/readability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor_RejectsEmptyInput(t *testing.T) {
	t.Parallel()

	ext := readability.NewExtractor()
	_, err := ext.Extract("")

	require.Error(t, err)
	assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
}

func TestExtractor_ExtractsTitle(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Page Title</title></head>
<body><article><p>Content</p></article></body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Equal(t, "Page Title", result.Title)
}

func TestExtractor_RemovesNavigation(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav><a href="/home">Home Nav Link</a><a href="/about">About Nav Link</a></nav>
<article><p>This is the main article content that should be preserved in the output.</p></article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.NotContains(t, result.ContentHTML, "Home Nav Link")
	assert.NotContains(t, result.ContentHTML, "About Nav Link")
}

func TestExtractor_RemovesFooter(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article><p>This is the main article content that should be preserved in the output.</p></article>
<footer><p>Footer copyright text 2024</p></footer>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.NotContains(t, result.ContentHTML, "Footer copyright text")
}

func TestExtractor_RemovesSidebar(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<aside class="sidebar"><p>Sidebar navigation content</p></aside>
<article><p>This is the main article content that should be preserved in the output.</p></article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.NotContains(t, result.ContentHTML, "Sidebar navigation content")
}

func TestExtractor_KeepsMainArticleContent(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav><a href="/home">Home</a></nav>
<article><p>This is the important article paragraph text that must be kept.</p></article>
<footer><p>Footer</p></footer>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "important article paragraph text")
}

func TestExtractor_PreservesHeadings(t *testing.T) {
	t.Parallel()

	// Note: go-readability may demote h1 to h2, but heading text is preserved
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<h1>Main Heading</h1>
<p>Some intro text here.</p>
<h2>Subheading Level Two</h2>
<p>More content under the subheading.</p>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "Main Heading")
	assert.Contains(t, result.ContentHTML, "Subheading Level Two")
	assert.Contains(t, result.ContentHTML, "<h2")
}

func TestExtractor_PreservesParagraphs(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>First paragraph of content.</p>
<p>Second paragraph of content.</p>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "<p")
}

func TestExtractor_PreservesLists(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>Here is a list:</p>
<ul>
<li>First item</li>
<li>Second item</li>
</ul>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "<ul")
	assert.Contains(t, result.ContentHTML, "<li")
}

func TestExtractor_PreservesTables(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>Here is a data table:</p>
<table>
<tr><th>Name</th><th>Value</th></tr>
<tr><td>Foo</td><td>123</td></tr>
</table>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "<table")
}

func TestExtractor_PreservesLinks(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>Check out <a href="https://example.com">this link</a> for more info.</p>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "<a")
}

func TestExtractor_PreservesInlineCode(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>Use the <code>myVariable</code> to store the value.</p>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "<code")
}

func TestExtractor_PreservesSimpleCodeBlocks(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>Here is a code example:</p>
<pre><code>npm install my-package</code></pre>
<p>That's all you need.</p>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "<pre")
	assert.Contains(t, result.ContentHTML, "npm install my-package")
}

func TestExtractor_PreservesCodeBlocksWithNestedSpans(t *testing.T) {
	t.Parallel()

	// Syntax highlighters wrap code in span elements for coloring
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>Run this command:</p>
<pre><code><div class="line"><span class="token">nx</span> <span class="token">generate</span></div></code></pre>
<p>This generates a new component.</p>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "<pre")
	assert.Contains(t, result.ContentHTML, "nx")
	assert.Contains(t, result.ContentHTML, "generate")
}

func TestExtractor_PreservesCodeBlocksInWrapperDivs(t *testing.T) {
	t.Parallel()

	// Documentation sites wrap code in complex structures
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>Install the CLI:</p>
<div class="expressive-code">
<figure>
<pre><code>npm install -g @nx/cli</code></pre>
</figure>
</div>
<p>Now you can use nx commands.</p>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	assert.Contains(t, result.ContentHTML, "<pre")
	assert.Contains(t, result.ContentHTML, "npm install -g @nx/cli")
}

func TestExtractor_PreservesLanguageHints(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<p>Example bash command:</p>
<pre data-language="bash"><code class="language-bash">echo "hello"</code></pre>
<p>That prints hello.</p>
</article>
</body>
</html>`

	ext := readability.NewExtractor()
	result, err := ext.Extract(html)

	require.NoError(t, err)
	// Language hints should be preserved in some form
	assert.Contains(t, result.ContentHTML, "bash")
}
