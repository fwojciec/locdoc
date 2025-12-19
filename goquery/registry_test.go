package goquery_test

import (
	"testing"

	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns registered selector for framework", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{}
		fallback := &mock.LinkSelector{NameFn: func() string { return "fallback" }}
		docusaurus := &mock.LinkSelector{NameFn: func() string { return "docusaurus" }}

		registry := goquery.NewRegistry(detector, fallback)
		registry.Register(locdoc.FrameworkDocusaurus, docusaurus)

		got := registry.Get(locdoc.FrameworkDocusaurus)

		require.NotNil(t, got)
		assert.Equal(t, "docusaurus", got.Name())
	})

	t.Run("returns nil for unregistered framework", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{}
		fallback := &mock.LinkSelector{NameFn: func() string { return "fallback" }}

		registry := goquery.NewRegistry(detector, fallback)

		got := registry.Get(locdoc.FrameworkDocusaurus)

		assert.Nil(t, got)
	})
}

func TestRegistry_GetForHTML(t *testing.T) {
	t.Parallel()

	t.Run("returns selector for detected framework", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkDocusaurus
			},
		}
		fallback := &mock.LinkSelector{NameFn: func() string { return "fallback" }}
		docusaurus := &mock.LinkSelector{NameFn: func() string { return "docusaurus" }}

		registry := goquery.NewRegistry(detector, fallback)
		registry.Register(locdoc.FrameworkDocusaurus, docusaurus)

		got := registry.GetForHTML("<html>docusaurus</html>")

		require.NotNil(t, got)
		assert.Equal(t, "docusaurus", got.Name())
	})

	t.Run("returns fallback selector for unknown framework", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
		}
		fallback := &mock.LinkSelector{NameFn: func() string { return "generic" }}

		registry := goquery.NewRegistry(detector, fallback)

		got := registry.GetForHTML("<html>unknown</html>")

		require.NotNil(t, got)
		assert.Equal(t, "generic", got.Name())
	})

	t.Run("returns fallback when framework detected but no selector registered", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkSphinx
			},
		}
		fallback := &mock.LinkSelector{NameFn: func() string { return "generic" }}

		registry := goquery.NewRegistry(detector, fallback)
		// Sphinx detected but no selector registered for it

		got := registry.GetForHTML("<html>sphinx</html>")

		require.NotNil(t, got)
		assert.Equal(t, "generic", got.Name())
	})
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("registers selector for framework", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{}
		fallback := &mock.LinkSelector{NameFn: func() string { return "fallback" }}
		mkdocs := &mock.LinkSelector{NameFn: func() string { return "mkdocs" }}

		registry := goquery.NewRegistry(detector, fallback)
		registry.Register(locdoc.FrameworkMkDocs, mkdocs)

		got := registry.Get(locdoc.FrameworkMkDocs)

		require.NotNil(t, got)
		assert.Equal(t, "mkdocs", got.Name())
	})

	t.Run("overwrites existing selector for framework", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{}
		fallback := &mock.LinkSelector{NameFn: func() string { return "fallback" }}
		mkdocsV1 := &mock.LinkSelector{NameFn: func() string { return "mkdocs-v1" }}
		mkdocsV2 := &mock.LinkSelector{NameFn: func() string { return "mkdocs-v2" }}

		registry := goquery.NewRegistry(detector, fallback)
		registry.Register(locdoc.FrameworkMkDocs, mkdocsV1)
		registry.Register(locdoc.FrameworkMkDocs, mkdocsV2)

		got := registry.Get(locdoc.FrameworkMkDocs)

		require.NotNil(t, got)
		assert.Equal(t, "mkdocs-v2", got.Name())
	})
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no selectors registered", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{}
		fallback := &mock.LinkSelector{NameFn: func() string { return "fallback" }}

		registry := goquery.NewRegistry(detector, fallback)

		got := registry.List()

		assert.Empty(t, got)
	})

	t.Run("returns all registered frameworks", func(t *testing.T) {
		t.Parallel()

		detector := &mock.FrameworkDetector{}
		fallback := &mock.LinkSelector{NameFn: func() string { return "fallback" }}
		docusaurus := &mock.LinkSelector{NameFn: func() string { return "docusaurus" }}
		mkdocs := &mock.LinkSelector{NameFn: func() string { return "mkdocs" }}

		registry := goquery.NewRegistry(detector, fallback)
		registry.Register(locdoc.FrameworkDocusaurus, docusaurus)
		registry.Register(locdoc.FrameworkMkDocs, mkdocs)

		got := registry.List()

		assert.Len(t, got, 2)
		assert.Contains(t, got, locdoc.FrameworkDocusaurus)
		assert.Contains(t, got, locdoc.FrameworkMkDocs)
	})
}

func TestRegistry_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Compile-time check that Registry implements LinkSelectorRegistry
	var _ locdoc.LinkSelectorRegistry = (*goquery.Registry)(nil)
}
