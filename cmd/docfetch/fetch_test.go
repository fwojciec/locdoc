package main_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/docfetch"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchCmd_Run_Preview(t *testing.T) {
	t.Parallel()

	t.Run("shows URLs from sitemap without saving", func(t *testing.T) {
		t.Parallel()

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Sitemaps: sitemaps,
		}

		cmd := &main.FetchCmd{
			URL:     "https://example.com/docs",
			Preview: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "https://example.com/docs/page1")
		assert.Contains(t, output, "https://example.com/docs/page2")
	})

	t.Run("falls back to recursive discovery when sitemap empty", func(t *testing.T) {
		t.Parallel()

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{}, nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				return `<html><body></body></html>`, nil
			},
		}

		prober := &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, true
			},
		}

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Test", ContentHTML: "<p>Test</p>"}, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						if baseURL == "https://example.com/docs/" {
							return []locdoc.DiscoveredLink{
								{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
							}, nil
						}
						return nil, nil
					},
					NameFn: func() string { return "test" },
				}
			},
		}

		rateLimiter := &mock.DomainLimiter{
			WaitFn: func(_ context.Context, _ string) error {
				return nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Sitemaps: sitemaps,
			Discoverer: &crawl.Discoverer{
				HTTPFetcher:   fetcher,
				RodFetcher:    fetcher,
				Prober:        prober,
				Extractor:     extractor,
				LinkSelectors: linkSelectors,
				RateLimiter:   rateLimiter,
				Concurrency:   1,
				RetryDelays:   []time.Duration{0},
			},
		}

		cmd := &main.FetchCmd{
			URL:     "https://example.com/docs/",
			Preview: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "https://example.com/docs/")
		assert.Contains(t, output, "https://example.com/docs/page1")
	})
}

func TestFetchCmd_Run_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("creates directory and writes documents", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
				}, nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				return "<html><body>Test content for " + url + "</body></html>", nil
			},
		}

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{
					Title:       "Test Page",
					ContentHTML: "<p>Test content</p>",
				}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "Test content", nil
			},
		}

		prober := &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, true
			},
		}

		crawler := &crawl.Crawler{
			Discoverer: &crawl.Discoverer{
				HTTPFetcher: fetcher,
				RodFetcher:  fetcher,
				Prober:      prober,
				Extractor:   extractor,
				Concurrency: 1,
				RetryDelays: []time.Duration{0},
			},
			Sitemaps:  sitemaps,
			Converter: converter,
			// Documents will be set by FetchCmd.Run
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Sitemaps: sitemaps,
			Crawler:  crawler,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
			Path: tmpDir,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		// Verify directory structure was created
		docsDir := filepath.Join(tmpDir, "testdocs")
		_, err = os.Stat(docsDir)
		require.NoError(t, err, "docs directory should exist")

		// Verify markdown files were created
		page1 := filepath.Join(docsDir, "docs", "page1.md")
		_, err = os.Stat(page1)
		require.NoError(t, err, "page1.md should exist")

		page2 := filepath.Join(docsDir, "docs", "page2.md")
		_, err = os.Stat(page2)
		require.NoError(t, err, "page2.md should exist")

		// Verify file content has frontmatter
		content, err := os.ReadFile(page1)
		require.NoError(t, err)
		assert.Contains(t, string(content), "---")
		assert.Contains(t, string(content), "source: https://example.com/docs/page1")
	})

	t.Run("shows progress during crawl", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
				}, nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return "<html><body>Test</body></html>", nil
			},
		}

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Test", ContentHTML: "<p>Test</p>"}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "Test", nil
			},
		}

		prober := &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, true
			},
		}

		crawler := &crawl.Crawler{
			Discoverer: &crawl.Discoverer{
				HTTPFetcher: fetcher,
				RodFetcher:  fetcher,
				Prober:      prober,
				Extractor:   extractor,
				Concurrency: 1,
				RetryDelays: []time.Duration{0},
			},
			Sitemaps:  sitemaps,
			Converter: converter,
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Sitemaps: sitemaps,
			Crawler:  crawler,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
			Path: tmpDir,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		output := stdout.String()
		// Should show completion message
		assert.Contains(t, output, "Saved 2 pages")
	})

	t.Run("atomic update replaces existing directory on success", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		// Create existing directory with old content
		existingDir := filepath.Join(tmpDir, "testdocs")
		require.NoError(t, os.MkdirAll(existingDir, 0755))
		oldFile := filepath.Join(existingDir, "old.md")
		require.NoError(t, os.WriteFile(oldFile, []byte("old content"), 0644))

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{"https://example.com/docs/new"}, nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return "<html><body>New content</body></html>", nil
			},
		}

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "New", ContentHTML: "<p>New</p>"}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "New content", nil
			},
		}

		prober := &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, true
			},
		}

		crawler := &crawl.Crawler{
			Discoverer: &crawl.Discoverer{
				HTTPFetcher: fetcher,
				RodFetcher:  fetcher,
				Prober:      prober,
				Extractor:   extractor,
				Concurrency: 1,
				RetryDelays: []time.Duration{0},
			},
			Sitemaps:  sitemaps,
			Converter: converter,
		}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   &bytes.Buffer{},
			Stderr:   &bytes.Buffer{},
			Sitemaps: sitemaps,
			Crawler:  crawler,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
			Path: tmpDir,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		// Old file should be gone
		_, err = os.Stat(oldFile)
		assert.True(t, os.IsNotExist(err), "old file should be deleted")

		// New file should exist
		newFile := filepath.Join(existingDir, "docs", "new.md")
		_, err = os.Stat(newFile)
		require.NoError(t, err, "new file should exist")

		// Temp directory should be cleaned up
		tmpDirPattern := filepath.Join(tmpDir, "testdocs.tmp")
		_, err = os.Stat(tmpDirPattern)
		assert.True(t, os.IsNotExist(err), "temp directory should be cleaned up")
	})

	t.Run("atomic update preserves original on failure", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		// Create existing directory with content we want to preserve
		existingDir := filepath.Join(tmpDir, "testdocs")
		require.NoError(t, os.MkdirAll(existingDir, 0755))
		preservedFile := filepath.Join(existingDir, "preserved.md")
		require.NoError(t, os.WriteFile(preservedFile, []byte("original content"), 0644))

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{"https://example.com/docs/page"}, nil
			},
		}

		// Fetcher that always fails
		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return "", locdoc.Errorf(locdoc.EINTERNAL, "simulated fetch failure")
			},
		}

		prober := &mock.Prober{
			DetectFn: func(_ string) locdoc.Framework {
				return locdoc.FrameworkUnknown
			},
			RequiresJSFn: func(_ locdoc.Framework) (bool, bool) {
				return false, true
			},
		}

		extractor := &mock.Extractor{
			ExtractFn: func(_ string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Test", ContentHTML: "<p>Test</p>"}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(_ string) (string, error) {
				return "Test", nil
			},
		}

		crawler := &crawl.Crawler{
			Discoverer: &crawl.Discoverer{
				HTTPFetcher: fetcher,
				RodFetcher:  fetcher,
				Prober:      prober,
				Extractor:   extractor,
				Concurrency: 1,
				RetryDelays: []time.Duration{0}, // No retries for fast test
			},
			Sitemaps:  sitemaps,
			Converter: converter,
		}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   &bytes.Buffer{},
			Stderr:   &bytes.Buffer{},
			Sitemaps: sitemaps,
			Crawler:  crawler,
		}

		cmd := &main.FetchCmd{
			URL:  "https://example.com/docs",
			Name: "testdocs",
			Path: tmpDir,
		}

		// Run should succeed even if pages fail (we continue on individual failures)
		err := cmd.Run(deps)
		require.NoError(t, err)

		// Original file should still exist
		content, err := os.ReadFile(preservedFile)
		require.NoError(t, err)
		assert.Equal(t, "original content", string(content), "original content should be preserved")

		// Temp directory should be cleaned up
		tmpDirPattern := filepath.Join(tmpDir, "testdocs.tmp")
		_, err = os.Stat(tmpDirPattern)
		assert.True(t, os.IsNotExist(err), "temp directory should be cleaned up")
	})
}
