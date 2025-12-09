package htmltomarkdown_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/htmltomarkdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure Converter implements locdoc.Converter at compile time.
var _ locdoc.Converter = (*htmltomarkdown.Converter)(nil)

func TestConverter_Convert(t *testing.T) {
	t.Parallel()

	t.Run("converts basic paragraph", func(t *testing.T) {
		t.Parallel()

		html := `<p>Hello, world!</p>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "Hello, world!")
	})

	t.Run("converts headings", func(t *testing.T) {
		t.Parallel()

		html := `<h1>Title</h1><h2>Subtitle</h2><h3>Section</h3>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "# Title")
		assert.Contains(t, md, "## Subtitle")
		assert.Contains(t, md, "### Section")
	})

	t.Run("converts links", func(t *testing.T) {
		t.Parallel()

		html := `<p>Visit <a href="https://example.com">Example</a> for more info.</p>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "[Example](https://example.com)")
	})

	t.Run("converts unordered lists", func(t *testing.T) {
		t.Parallel()

		html := `<ul><li>First</li><li>Second</li><li>Third</li></ul>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "- First")
		assert.Contains(t, md, "- Second")
		assert.Contains(t, md, "- Third")
	})

	t.Run("converts ordered lists", func(t *testing.T) {
		t.Parallel()

		html := `<ol><li>First</li><li>Second</li><li>Third</li></ol>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "1. First")
		assert.Contains(t, md, "2. Second")
		assert.Contains(t, md, "3. Third")
	})

	t.Run("converts inline code", func(t *testing.T) {
		t.Parallel()

		html := `<p>Run <code>go build</code> to compile.</p>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "`go build`")
	})

	t.Run("converts code blocks with language hint", func(t *testing.T) {
		t.Parallel()

		html := `<pre><code class="language-go">package main

func main() {
    println("Hello")
}
</code></pre>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "```go")
		assert.Contains(t, md, "package main")
		assert.Contains(t, md, "```")
	})

	t.Run("converts code blocks without language hint", func(t *testing.T) {
		t.Parallel()

		html := `<pre><code>some code here</code></pre>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "```")
		assert.Contains(t, md, "some code here")
	})

	t.Run("converts tables", func(t *testing.T) {
		t.Parallel()

		html := `<table>
<thead><tr><th>Name</th><th>Age</th></tr></thead>
<tbody><tr><td>Alice</td><td>30</td></tr><tr><td>Bob</td><td>25</td></tr></tbody>
</table>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		// Table cells may have padding for alignment, so check for content
		assert.Contains(t, md, "Name")
		assert.Contains(t, md, "Age")
		assert.Contains(t, md, "Alice")
		assert.Contains(t, md, "Bob")
		assert.Contains(t, md, "|")
		assert.Contains(t, md, "---")
	})

	t.Run("converts bold and italic", func(t *testing.T) {
		t.Parallel()

		html := `<p><strong>Bold</strong> and <em>italic</em> text.</p>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "**Bold**")
		assert.Contains(t, md, "*italic*")
	})

	t.Run("converts blockquotes", func(t *testing.T) {
		t.Parallel()

		html := `<blockquote><p>This is a quote.</p></blockquote>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "> This is a quote.")
	})

	t.Run("returns error for empty input", func(t *testing.T) {
		t.Parallel()

		conv := htmltomarkdown.NewConverter()
		_, err := conv.Convert("")

		require.Error(t, err)
		assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
	})

	t.Run("handles complex documentation page", func(t *testing.T) {
		t.Parallel()

		html := `<div>
<h1>Getting Started</h1>
<p>Welcome to the documentation.</p>
<h2>Installation</h2>
<p>Run the following command:</p>
<pre><code class="language-bash">go get github.com/example/pkg</code></pre>
<h2>Usage</h2>
<p>Import the package:</p>
<pre><code class="language-go">import "github.com/example/pkg"</code></pre>
<p>Then call <code>pkg.New()</code> to create an instance.</p>
<h3>Configuration</h3>
<table>
<thead><tr><th>Option</th><th>Default</th><th>Description</th></tr></thead>
<tbody>
<tr><td>timeout</td><td>30s</td><td>Request timeout</td></tr>
<tr><td>retries</td><td>3</td><td>Number of retries</td></tr>
</tbody>
</table>
</div>`

		conv := htmltomarkdown.NewConverter()
		md, err := conv.Convert(html)

		require.NoError(t, err)
		assert.Contains(t, md, "# Getting Started")
		assert.Contains(t, md, "## Installation")
		assert.Contains(t, md, "```bash")
		assert.Contains(t, md, "go get github.com/example/pkg")
		assert.Contains(t, md, "```go")
		assert.Contains(t, md, "`pkg.New()`")
		// Table cells may have padding for alignment
		assert.Contains(t, md, "Option")
		assert.Contains(t, md, "Default")
		assert.Contains(t, md, "Description")
	})
}
