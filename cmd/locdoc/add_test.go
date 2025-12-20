package main_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/goquery"
	lochttp "github.com/fwojciec/locdoc/http"
	"github.com/fwojciec/locdoc/mock"
	"github.com/fwojciec/locdoc/rod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("creates project and crawls documents", func(t *testing.T) {
		t.Parallel()

		var createdProject *locdoc.Project
		var savedDoc *locdoc.Document

		projects := &mock.ProjectService{
			CreateProjectFn: func(_ context.Context, p *locdoc.Project) error {
				p.ID = "proj-123"
				createdProject = p
				return nil
			},
		}

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{"https://example.com/docs/page1"}, nil
			},
		}

		documents := &mock.DocumentService{
			CreateDocumentFn: func(_ context.Context, doc *locdoc.Document) error {
				savedDoc = doc
				return nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, _ string) (string, error) {
				return "<html><body>Test content</body></html>", nil
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

		tokenCounter := &mock.TokenCounter{
			CountTokensFn: func(_ context.Context, text string) (int, error) {
				return len(text) / 4, nil
			},
		}

		crawler := &crawl.Crawler{
			Sitemaps:     sitemaps,
			Fetcher:      fetcher,
			Extractor:    extractor,
			Converter:    converter,
			Documents:    documents,
			TokenCounter: tokenCounter,
			Concurrency:  1,
			RetryDelays:  []time.Duration{0},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
			Sitemaps: sitemaps,
			Crawler:  crawler,
		}

		cmd := &main.AddCmd{
			Name:        "testdocs",
			URL:         "https://example.com/docs",
			Concurrency: 10,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		require.NotNil(t, createdProject)
		assert.Equal(t, "testdocs", createdProject.Name)
		require.NotNil(t, savedDoc)
		assert.Equal(t, "proj-123", savedDoc.ProjectID)
	})

	t.Run("preview mode shows URLs without creating project", func(t *testing.T) {
		t.Parallel()

		var projectCreated bool

		projects := &mock.ProjectService{
			CreateProjectFn: func(_ context.Context, _ *locdoc.Project) error {
				projectCreated = true
				return nil
			},
		}

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{"https://example.com/docs/page1"}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
			Sitemaps: sitemaps,
		}

		cmd := &main.AddCmd{
			Name:    "testdocs",
			URL:     "https://example.com/docs",
			Preview: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		assert.False(t, projectCreated)
		assert.Contains(t, stdout.String(), "https://example.com/docs/page1")
	})

	t.Run("invalid filter pattern shows helpful error", func(t *testing.T) {
		t.Parallel()

		stderr := &bytes.Buffer{}
		deps := &main.Dependencies{
			Ctx:    context.Background(),
			Stdout: &bytes.Buffer{},
			Stderr: stderr,
		}

		cmd := &main.AddCmd{
			Name:   "testdocs",
			URL:    "https://example.com/docs",
			Filter: []string{"[invalid"},
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		errMsg := stderr.String()
		assert.Contains(t, errMsg, "[invalid")
		// Error should mention regex and give an example of valid patterns
		assert.Contains(t, errMsg, "regex")
		assert.Contains(t, errMsg, "Example", "error should include example patterns")
	})

	t.Run("shows live progress as URLs complete", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			CreateProjectFn: func(_ context.Context, p *locdoc.Project) error {
				p.ID = "proj-123"
				return nil
			},
		}

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
					"https://example.com/docs/page3",
				}, nil
			},
		}

		documents := &mock.DocumentService{
			CreateDocumentFn: func(_ context.Context, _ *locdoc.Document) error {
				return nil
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

		crawler := &crawl.Crawler{
			Sitemaps:    sitemaps,
			Fetcher:     fetcher,
			Extractor:   extractor,
			Converter:   converter,
			Documents:   documents,
			Concurrency: 1,
			RetryDelays: []time.Duration{0},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
			Sitemaps: sitemaps,
			Crawler:  crawler,
		}

		cmd := &main.AddCmd{
			Name: "testdocs",
			URL:  "https://example.com/docs",
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		output := stdout.String()
		// Progress should use carriage return for in-place updates
		assert.Contains(t, output, "\r", "progress should use carriage return for in-place updates")
		// Progress should show [N/M] format
		assert.Contains(t, output, "/3]", "progress should show total count")
	})

	t.Run("shows progress without total for recursive crawling", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			CreateProjectFn: func(_ context.Context, p *locdoc.Project) error {
				p.ID = "proj-123"
				return nil
			},
		}

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{}, nil // No sitemap, triggers recursive crawl
			},
		}

		documents := &mock.DocumentService{
			CreateDocumentFn: func(_ context.Context, _ *locdoc.Document) error {
				return nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				if url == "https://example.com/docs/" {
					return `<html><body><nav><a href="/docs/page1">Page 1</a></nav><p>Content</p></body></html>`, nil
				}
				return `<html><body><p>Page content</p></body></html>`, nil
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

		crawler := &crawl.Crawler{
			Sitemaps:      sitemaps,
			Fetcher:       fetcher,
			Extractor:     extractor,
			Converter:     converter,
			Documents:     documents,
			LinkSelectors: linkSelectors,
			RateLimiter:   rateLimiter,
			Concurrency:   1,
			RetryDelays:   []time.Duration{0},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
			Sitemaps: sitemaps,
			Crawler:  crawler,
		}

		cmd := &main.AddCmd{
			Name: "testdocs",
			URL:  "https://example.com/docs/",
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		output := stdout.String()
		// For recursive crawling (unknown total), should show [N] format, not [N/0]
		assert.Contains(t, output, "[1]", "progress should show count without total")
		assert.NotContains(t, output, "/0]", "progress should NOT show '/0]' for unknown total")
	})

	t.Run("preview mode falls back to recursive discovery when sitemap unavailable", func(t *testing.T) {
		t.Parallel()

		var projectCreated bool

		projects := &mock.ProjectService{
			CreateProjectFn: func(_ context.Context, _ *locdoc.Project) error {
				projectCreated = true
				return nil
			},
		}

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{}, nil // Empty sitemap, should trigger recursive discovery
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				if url == "https://example.com/docs/" {
					return `<html><body><nav><a href="/docs/page1">Page 1</a><a href="/docs/page2">Page 2</a></nav></body></html>`, nil
				}
				if url == "https://example.com/docs/page1" {
					return `<html><body><nav><a href="/docs/page3">Page 3</a></nav></body></html>`, nil
				}
				return `<html><body></body></html>`, nil
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						if baseURL == "https://example.com/docs/" {
							return []locdoc.DiscoveredLink{
								{URL: "https://example.com/docs/page1", Priority: locdoc.PriorityNavigation},
								{URL: "https://example.com/docs/page2", Priority: locdoc.PriorityNavigation},
							}, nil
						}
						if baseURL == "https://example.com/docs/page1" {
							return []locdoc.DiscoveredLink{
								{URL: "https://example.com/docs/page3", Priority: locdoc.PriorityNavigation},
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
			Ctx:           context.Background(),
			Stdout:        stdout,
			Stderr:        stderr,
			Projects:      projects,
			Sitemaps:      sitemaps,
			Fetcher:       fetcher,
			LinkSelectors: linkSelectors,
			RateLimiter:   rateLimiter,
		}

		cmd := &main.AddCmd{
			Name:    "testdocs",
			URL:     "https://example.com/docs/",
			Preview: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		assert.False(t, projectCreated, "preview mode should not create project")

		output := stdout.String()
		// Should discover URLs recursively
		assert.Contains(t, output, "https://example.com/docs/")
		assert.Contains(t, output, "https://example.com/docs/page1")
		assert.Contains(t, output, "https://example.com/docs/page2")
		assert.Contains(t, output, "https://example.com/docs/page3")
	})

	t.Run("debug mode logs progress to stderr", func(t *testing.T) {
		t.Parallel()

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{}, nil // Empty sitemap triggers recursive discovery
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				return `<html><body><nav><a href="/docs/page1">Page 1</a></nav></body></html>`, nil
			},
		}

		detector := &mock.FrameworkDetector{
			DetectFn: func(html string) locdoc.Framework {
				return locdoc.FrameworkDocusaurus
			},
		}

		linkSelectors := &mock.LinkSelectorRegistry{
			GetForHTMLFn: func(html string) locdoc.LinkSelector {
				return &mock.LinkSelector{
					ExtractLinksFn: func(html string, baseURL string) ([]locdoc.DiscoveredLink, error) {
						return nil, nil // No links to follow
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

		// Create logger writing to stderr (like main.go does when --debug is set)
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		// Wrap services with logging decorators (simulating main.go wiring when Debug=true)
		loggingSitemaps := lochttp.NewLoggingSitemapService(sitemaps, logger)
		loggingFetcher := rod.NewLoggingFetcher(fetcher, logger)
		loggingRegistry := goquery.NewLoggingRegistry(linkSelectors, detector, logger)

		deps := &main.Dependencies{
			Ctx:           context.Background(),
			Stdout:        stdout,
			Stderr:        stderr,
			Sitemaps:      loggingSitemaps,
			Fetcher:       loggingFetcher,
			LinkSelectors: loggingRegistry,
			RateLimiter:   rateLimiter,
		}

		cmd := &main.AddCmd{
			Name:    "testdocs",
			URL:     "https://example.com/docs/",
			Preview: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		// Verify debug logs appear in stderr
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "sitemap discovery", "should log sitemap discovery")
		assert.Contains(t, stderrOutput, "fetch", "should log page fetches")
		assert.Contains(t, stderrOutput, "framework detection", "should log framework detection")
		assert.Contains(t, stderrOutput, "duration=", "should log timing information")
	})

	t.Run("without debug mode stderr remains quiet", func(t *testing.T) {
		t.Parallel()

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{"https://example.com/docs/page1"}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// No logging decorators - simulating Debug=false
		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Sitemaps: sitemaps,
		}

		cmd := &main.AddCmd{
			Name:    "testdocs",
			URL:     "https://example.com/docs",
			Preview: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		// Stderr should be empty (no debug logs)
		assert.Empty(t, stderr.String(), "stderr should be empty without debug mode")
	})

	t.Run("prints failures on separate lines to stderr", func(t *testing.T) {
		t.Parallel()

		projects := &mock.ProjectService{
			CreateProjectFn: func(_ context.Context, p *locdoc.Project) error {
				p.ID = "proj-123"
				return nil
			},
		}

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/failing",
					"https://example.com/docs/page3",
				}, nil
			},
		}

		documents := &mock.DocumentService{
			CreateDocumentFn: func(_ context.Context, _ *locdoc.Document) error {
				return nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(_ context.Context, url string) (string, error) {
				if url == "https://example.com/docs/failing" {
					return "", locdoc.Errorf(locdoc.ENOTFOUND, "connection timeout")
				}
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

		crawler := &crawl.Crawler{
			Sitemaps:    sitemaps,
			Fetcher:     fetcher,
			Extractor:   extractor,
			Converter:   converter,
			Documents:   documents,
			Concurrency: 1,
			RetryDelays: []time.Duration{0},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:      context.Background(),
			Stdout:   stdout,
			Stderr:   stderr,
			Projects: projects,
			Sitemaps: sitemaps,
			Crawler:  crawler,
		}

		cmd := &main.AddCmd{
			Name: "testdocs",
			URL:  "https://example.com/docs",
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		// Failures should print to stderr on separate lines
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "failing", "stderr should contain the failing URL")
		assert.Contains(t, stderrOutput, "\n", "failures should be on separate lines")

		// Summary should show correct count (2 saved, not 3)
		stdoutOutput := stdout.String()
		assert.Contains(t, stdoutOutput, "Saved 2 pages", "summary should show 2 saved pages")
	})
}
