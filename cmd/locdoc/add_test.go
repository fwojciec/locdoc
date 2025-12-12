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
				p.ID = "proj-123" // Simulate ID assignment
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

		// Verify project was created
		require.NotNil(t, createdProject)
		assert.Equal(t, "testdocs", createdProject.Name)
		assert.Equal(t, "https://example.com/docs", createdProject.SourceURL)

		// Verify document was saved
		require.NotNil(t, savedDoc)
		assert.Equal(t, "proj-123", savedDoc.ProjectID)
		assert.Equal(t, "https://example.com/docs/page1", savedDoc.SourceURL)
		assert.Equal(t, "Test Page", savedDoc.Title)
		assert.Equal(t, "Test content", savedDoc.Content)

		// Verify output includes project creation message
		assert.Contains(t, stdout.String(), "testdocs")
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
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
					"https://example.com/docs/page3",
				}, nil
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
			// No Crawler - preview mode doesn't need it
		}

		cmd := &main.AddCmd{
			Name:    "testdocs",
			URL:     "https://example.com/docs",
			Preview: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		// Project should NOT be created in preview mode
		assert.False(t, projectCreated, "project should not be created in preview mode")

		// Output should list all discovered URLs
		output := stdout.String()
		assert.Contains(t, output, "https://example.com/docs/page1")
		assert.Contains(t, output, "https://example.com/docs/page2")
		assert.Contains(t, output, "https://example.com/docs/page3")
	})

	t.Run("force mode deletes existing project first", func(t *testing.T) {
		t.Parallel()

		existingProject := &locdoc.Project{
			ID:        "existing-123",
			Name:      "testdocs",
			SourceURL: "https://old.example.com/docs",
		}

		var deletedID string
		var createdProject *locdoc.Project

		projects := &mock.ProjectService{
			FindProjectsFn: func(_ context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				// Return existing project when searching by name
				if filter.Name != nil && *filter.Name == "testdocs" {
					return []*locdoc.Project{existingProject}, nil
				}
				return nil, nil
			},
			DeleteProjectFn: func(_ context.Context, id string) error {
				deletedID = id
				return nil
			},
			CreateProjectFn: func(_ context.Context, p *locdoc.Project) error {
				p.ID = "new-456"
				createdProject = p
				return nil
			},
		}

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{}, nil
			},
		}

		crawler := &crawl.Crawler{
			Sitemaps:    sitemaps,
			Fetcher:     &mock.Fetcher{},
			Extractor:   &mock.Extractor{},
			Converter:   &mock.Converter{},
			Documents:   &mock.DocumentService{},
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
			Name:  "testdocs",
			URL:   "https://new.example.com/docs",
			Force: true,
		}

		err := cmd.Run(deps)

		require.NoError(t, err)

		// Existing project should be deleted
		assert.Equal(t, "existing-123", deletedID, "existing project should be deleted")

		// New project should be created
		require.NotNil(t, createdProject)
		assert.Equal(t, "testdocs", createdProject.Name)
		assert.Equal(t, "https://new.example.com/docs", createdProject.SourceURL)
	})

	t.Run("stores filter patterns in project", func(t *testing.T) {
		t.Parallel()

		var createdProject *locdoc.Project

		projects := &mock.ProjectService{
			CreateProjectFn: func(_ context.Context, p *locdoc.Project) error {
				p.ID = "proj-123"
				createdProject = p
				return nil
			},
		}

		sitemaps := &mock.SitemapService{
			DiscoverURLsFn: func(_ context.Context, _ string, _ *locdoc.URLFilter) ([]string, error) {
				return []string{}, nil
			},
		}

		crawler := &crawl.Crawler{
			Sitemaps:    sitemaps,
			Fetcher:     &mock.Fetcher{},
			Extractor:   &mock.Extractor{},
			Converter:   &mock.Converter{},
			Documents:   &mock.DocumentService{},
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
			Name:   "testdocs",
			URL:    "https://example.com/docs",
			Filter: []string{"api", "guide"},
		}

		err := cmd.Run(deps)

		require.NoError(t, err)
		require.NotNil(t, createdProject)
		assert.Equal(t, "api\nguide", createdProject.Filter)
	})

	t.Run("returns error for invalid filter regex", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		deps := &main.Dependencies{
			Ctx:    context.Background(),
			Stdout: stdout,
			Stderr: stderr,
		}

		cmd := &main.AddCmd{
			Name:   "testdocs",
			URL:    "https://example.com/docs",
			Filter: []string{"[invalid"},
		}

		err := cmd.Run(deps)

		require.Error(t, err)
		assert.Contains(t, stderr.String(), "invalid filter pattern")
	})
}
