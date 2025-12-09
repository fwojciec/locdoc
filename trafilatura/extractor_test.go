package trafilatura_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/trafilatura"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure Extractor implements locdoc.Extractor at compile time.
var _ locdoc.Extractor = (*trafilatura.Extractor)(nil)

func TestExtractor_Extract(t *testing.T) {
	t.Parallel()

	t.Run("extracts title from meta tags", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head>
<title>Getting Started - My Docs</title>
<meta property="og:title" content="Getting Started Guide">
</head>
<body>
<nav>Navigation here</nav>
<main>
<h1>Getting Started</h1>
<p>This is the main content of the documentation page.</p>
</main>
<footer>Footer content</footer>
</body>
</html>`

		ext := trafilatura.NewExtractor()
		result, err := ext.Extract(html)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Title)
	})

	t.Run("extracts main content", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav><a href="/">Home</a><a href="/docs">Docs</a></nav>
<article>
<h1>Documentation</h1>
<p>This is important documentation content that should be extracted.</p>
<pre><code>func main() { fmt.Println("Hello") }</code></pre>
</article>
<aside>Sidebar content</aside>
<footer>Copyright 2024</footer>
</body>
</html>`

		ext := trafilatura.NewExtractor()
		result, err := ext.Extract(html)

		require.NoError(t, err)
		assert.Contains(t, result.ContentHTML, "important documentation content")
		assert.Contains(t, result.ContentHTML, "func main()")
	})

	t.Run("removes navigation boilerplate", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav class="main-nav">
<ul>
<li><a href="/">Home</a></li>
<li><a href="/about">About</a></li>
<li><a href="/docs">Documentation</a></li>
</ul>
</nav>
<main>
<h1>Main Content</h1>
<p>This paragraph contains the actual content we want.</p>
</main>
</body>
</html>`

		ext := trafilatura.NewExtractor()
		result, err := ext.Extract(html)

		require.NoError(t, err)
		assert.Contains(t, result.ContentHTML, "actual content we want")
		assert.NotContains(t, result.ContentHTML, "main-nav")
	})

	t.Run("removes footer boilerplate", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<article>
<h1>Article Title</h1>
<p>Article body with substantive content for readers.</p>
</article>
<footer>
<p>Copyright 2024 Example Corp</p>
<nav>Privacy | Terms | Contact</nav>
</footer>
</body>
</html>`

		ext := trafilatura.NewExtractor()
		result, err := ext.Extract(html)

		require.NoError(t, err)
		assert.Contains(t, result.ContentHTML, "substantive content")
		assert.NotContains(t, result.ContentHTML, "Copyright 2024 Example Corp")
	})

	t.Run("handles Docusaurus-style documentation", func(t *testing.T) {
		t.Parallel()

		// Simplified Docusaurus structure
		html := `<!DOCTYPE html>
<html>
<head>
<title>Introduction | My Project</title>
<meta property="og:title" content="Introduction">
</head>
<body>
<nav class="navbar">
<a href="/">My Project</a>
<a href="/docs">Docs</a>
<a href="/blog">Blog</a>
</nav>
<div class="sidebar">
<ul>
<li><a href="/docs/intro">Introduction</a></li>
<li><a href="/docs/install">Installation</a></li>
</ul>
</div>
<main class="docMainContainer">
<article>
<h1>Introduction</h1>
<p>Welcome to the documentation. This guide will help you get started.</p>
<h2>Prerequisites</h2>
<p>Before you begin, make sure you have Node.js installed.</p>
</article>
</main>
<footer class="footer">
<p>Built with Docusaurus</p>
</footer>
</body>
</html>`

		ext := trafilatura.NewExtractor()
		result, err := ext.Extract(html)

		require.NoError(t, err)
		assert.Contains(t, result.ContentHTML, "Welcome to the documentation")
		assert.Contains(t, result.ContentHTML, "Prerequisites")
	})

	t.Run("handles MkDocs-style documentation", func(t *testing.T) {
		t.Parallel()

		// Simplified MkDocs structure
		html := `<!DOCTYPE html>
<html>
<head>
<title>Home - MkDocs Project</title>
</head>
<body>
<header>
<nav class="md-header">
<a href=".">MkDocs Project</a>
</nav>
</header>
<nav class="md-nav" data-md-level="0">
<ul>
<li><a href=".">Home</a></li>
<li><a href="getting-started/">Getting Started</a></li>
</ul>
</nav>
<main>
<article class="md-content">
<h1>Welcome to MkDocs</h1>
<p>For full documentation visit mkdocs.org.</p>
<h2>Commands</h2>
<ul>
<li><code>mkdocs new [dir-name]</code> - Create a new project.</li>
<li><code>mkdocs serve</code> - Start the live-reloading docs server.</li>
</ul>
</article>
</main>
<footer class="md-footer">
<p>Made with MkDocs</p>
</footer>
</body>
</html>`

		ext := trafilatura.NewExtractor()
		result, err := ext.Extract(html)

		require.NoError(t, err)
		assert.Contains(t, result.ContentHTML, "Welcome to MkDocs")
		assert.Contains(t, result.ContentHTML, "mkdocs new")
	})

	t.Run("preserves code blocks", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Code Example</title></head>
<body>
<article>
<h1>Code Examples</h1>
<p>Here is a code example:</p>
<pre><code class="language-go">package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
</code></pre>
<p>And here is inline code: <code>go run main.go</code></p>
</article>
</body>
</html>`

		ext := trafilatura.NewExtractor()
		result, err := ext.Extract(html)

		require.NoError(t, err)
		assert.Contains(t, result.ContentHTML, "fmt.Println")
		// HTML rendering encodes quotes as &#34;
		assert.Contains(t, result.ContentHTML, "Hello, World!")
	})

	t.Run("returns error for empty input", func(t *testing.T) {
		t.Parallel()

		ext := trafilatura.NewExtractor()
		_, err := ext.Extract("")

		require.Error(t, err)
	})

	t.Run("handles minimal valid HTML", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><p>Simple content</p></body></html>`

		ext := trafilatura.NewExtractor()
		result, err := ext.Extract(html)

		require.NoError(t, err)
		assert.Contains(t, result.ContentHTML, "Simple content")
	})
}
