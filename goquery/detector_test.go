package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/stretchr/testify/assert"
)

// Ensure Detector implements locdoc.FrameworkDetector at compile time.
var _ locdoc.FrameworkDetector = (*goquery.Detector)(nil)

func TestDetector_Detect(t *testing.T) {
	t.Parallel()

	// Docusaurus tests - uses __docusaurus skip link and data-theme attribute
	t.Run("detects Docusaurus from __docusaurus_skipToContent_fallback element", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en" data-theme="light" data-rh="lang,dir,data-theme,data-announcement-bar-initially-dismissed">
<head><title>Docusaurus Docs</title></head>
<body>
<a id="__docusaurus_skipToContent_fallback" class="skipToContent_fXgn" href="#__docusaurus_skipToContent_fallback">Skip to main content</a>
<div class="theme-doc-sidebar-container">
	<nav class="menu">
		<ul><li><a href="/docs/intro">Introduction</a></li></ul>
	</nav>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkDocusaurus, framework)
	})

	t.Run("detects Docusaurus from theme-doc-sidebar-container class", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Docusaurus</title></head>
<body>
<div class="theme-doc-sidebar-container">
	<nav class="menu"><ul><li><a href="/docs">Docs</a></li></ul></nav>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkDocusaurus, framework)
	})

	// MkDocs Material tests - uses data-md-color-* attributes
	t.Run("detects MkDocs from data-md-color-scheme attribute", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en">
<head><title>MkDocs Material</title></head>
<body data-md-color-scheme="default" data-md-color-primary="indigo" data-md-color-accent="indigo">
<nav class="md-nav md-nav--primary">
	<ul class="md-nav__list">
		<li><a href="/getting-started/">Getting Started</a></li>
	</ul>
</nav>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkMkDocs, framework)
	})

	t.Run("detects MkDocs from data-md-component attribute", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>MkDocs</title></head>
<body>
<div data-md-component="navigation">
	<nav>Navigation</nav>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkMkDocs, framework)
	})

	// Sphinx tests - uses meta generator tag
	t.Run("detects Sphinx from meta generator tag", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head>
	<title>Sphinx Docs</title>
	<meta name="generator" content="Sphinx 7.2.6">
</head>
<body>
<div class="document">
	<div class="bodywrapper">Content</div>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkSphinx, framework)
	})

	t.Run("detects Sphinx from toctree-wrapper class", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Sphinx Docs</title></head>
<body>
<div class="toctree-wrapper compound">
	<ul>
		<li><a href="intro.html">Introduction</a></li>
	</ul>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkSphinx, framework)
	})

	t.Run("detects Sphinx from wy-nav-side class (ReadTheDocs theme)", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>ReadTheDocs</title></head>
<body>
<nav class="wy-nav-side">
	<div class="wy-menu-vertical">
		<ul><li><a href="#">Docs</a></li></ul>
	</div>
</nav>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkSphinx, framework)
	})

	// VitePress tests - uses VPContent element and VPDoc class
	t.Run("detects VitePress from VPContent element", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en-US" class="dark">
<head><title>VitePress</title></head>
<body>
<a id="VPContent" tabindex="-1"></a>
<div class="VPDoc">
	<div class="VPDocAsideOutline">
		<nav>Table of Contents</nav>
	</div>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkVitePress, framework)
	})

	t.Run("detects VitePress from VPDoc class", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>VitePress</title></head>
<body>
<div class="VPDoc">
	<main>Documentation content</main>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkVitePress, framework)
	})

	// VuePress tests - uses theme-default-content and vuepress color scheme
	t.Run("detects VuePress from theme-default-content class", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en-US">
<head><title>VuePress Docs</title></head>
<body>
<div class="theme-default-content">
	<h1>Getting Started</h1>
	<p>Documentation content here.</p>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkVuePress, framework)
	})

	t.Run("detects VuePress from sidebar-links class", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>VuePress</title></head>
<body>
<aside class="sidebar">
	<ul class="sidebar-links">
		<li><a href="/guide/">Guide</a></li>
	</ul>
</aside>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkVuePress, framework)
	})

	// GitBook tests - uses meta generator tag and gitbook-specific classes
	t.Run("detects GitBook from meta generator tag", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en" class="circular-corners theme-clean tint sidebar-default">
<head>
	<title>GitBook</title>
	<meta name="generator" content="GitBook">
</head>
<body>
<div id="site-header">Header</div>
<div id="site-section">Content</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkGitBook, framework)
	})

	t.Run("detects GitBook from gitbook html classes", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html class="circular-corners theme-clean tint">
<head><title>GitBook</title></head>
<body>
<main>Content</main>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkGitBook, framework)
	})

	t.Run("detects GitBook from data-testid space.sidebar", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>GitBook</title></head>
<body>
<div data-testid="space.sidebar">
	<nav>Sidebar content</nav>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkGitBook, framework)
	})

	// Nextra tests - uses nextra-navbar and nextra-prefixed classes
	t.Run("detects Nextra from nextra-navbar class", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html lang="en" dir="ltr">
<head><title>Nextra Docs</title></head>
<body>
<nav class="nextra-navbar">
	<ul><li><a href="/docs">Docs</a></li></ul>
</nav>
<div class="nextra-banner">Announcement</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkNextra, framework)
	})

	t.Run("detects Nextra from nextra-toc class", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Nextra</title></head>
<body>
<div class="nextra-toc">
	<nav>Table of Contents</nav>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkNextra, framework)
	})

	// Priority order tests
	t.Run("meta generator takes priority over CSS class markers", func(t *testing.T) {
		t.Parallel()

		// HTML with Sphinx meta generator AND Docusaurus CSS classes
		// Should return Sphinx because meta generator is checked first
		html := `<!DOCTYPE html>
<html>
<head>
	<title>Conflicting Markers</title>
	<meta name="generator" content="Sphinx 7.2.6">
</head>
<body>
<div class="theme-doc-sidebar-container">
	<nav class="menu"><ul><li><a href="/docs">Docs</a></li></ul></nav>
</div>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkSphinx, framework)
	})

	t.Run("meta generator takes priority over multiple framework markers", func(t *testing.T) {
		t.Parallel()

		// HTML with GitBook meta generator AND MkDocs AND Nextra CSS classes
		html := `<!DOCTYPE html>
<html>
<head>
	<title>Multiple Conflicting Markers</title>
	<meta name="generator" content="GitBook">
</head>
<body data-md-color-scheme="default">
<nav class="nextra-navbar">
	<ul><li><a href="/docs">Docs</a></li></ul>
</nav>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkGitBook, framework)
	})

	// Edge cases
	t.Run("returns FrameworkUnknown for generic HTML", func(t *testing.T) {
		t.Parallel()

		html := `<!DOCTYPE html>
<html>
<head><title>Generic Site</title></head>
<body>
<nav>
	<ul><li><a href="/about">About</a></li></ul>
</nav>
<main>
	<article>Some content</article>
</main>
</body>
</html>`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		assert.Equal(t, locdoc.FrameworkUnknown, framework)
	})

	t.Run("returns FrameworkUnknown for empty HTML", func(t *testing.T) {
		t.Parallel()

		d := goquery.NewDetector()
		framework := d.Detect("")

		assert.Equal(t, locdoc.FrameworkUnknown, framework)
	})

	t.Run("returns FrameworkUnknown for malformed HTML", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><div class="incomplete`

		d := goquery.NewDetector()
		framework := d.Detect(html)

		// goquery is lenient with malformed HTML, should still return Unknown
		assert.Equal(t, locdoc.FrameworkUnknown, framework)
	})
}
