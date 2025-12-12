package main_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/mock"
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
}
