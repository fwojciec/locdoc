package main_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testContext returns a background context for tests.
func testContext() context.Context {
	return context.Background()
}

func TestCmdAdd(t *testing.T) {
	t.Parallel()

	t.Run("creates project successfully", func(t *testing.T) {
		t.Parallel()

		var createdProject *locdoc.Project
		projectSvc := &mock.ProjectService{
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				p.ID = "test-id-123"
				createdProject = p
				return nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs"}, stdout, stderr, projectSvc, nil, nil)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "Added project")
		assert.Contains(t, stdout.String(), "myproject")
		assert.Empty(t, stderr.String())
		require.NotNil(t, createdProject)
		assert.Equal(t, "myproject", createdProject.Name)
		assert.Equal(t, "https://example.com/docs", createdProject.SourceURL)
	})

	t.Run("returns error for missing arguments", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"onlyname"}, stdout, stderr, nil, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "usage:")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error for no arguments", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{}, stdout, stderr, nil, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "usage:")
	})

	t.Run("returns error when create fails", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				return locdoc.Errorf(locdoc.ECONFLICT, "project already exists")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"existing", "https://example.com"}, stdout, stderr, projectSvc, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})

	t.Run("preview shows URLs without creating project", func(t *testing.T) {
		t.Parallel()

		createCalled := false
		projectSvc := &mock.ProjectService{
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				createCalled = true
				return nil
			},
		}

		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				assert.Equal(t, "https://example.com/docs", baseURL)
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
					"https://example.com/docs/page3",
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs", "--preview"}, stdout, stderr, projectSvc, sitemapSvc, nil)

		assert.Equal(t, 0, code)
		assert.False(t, createCalled, "CreateProject should not be called in preview mode")
		assert.Contains(t, stdout.String(), "https://example.com/docs/page1")
		assert.Contains(t, stdout.String(), "https://example.com/docs/page2")
		assert.Contains(t, stdout.String(), "https://example.com/docs/page3")
		assert.Empty(t, stderr.String())
	})

	t.Run("preview returns error when sitemap discovery fails", func(t *testing.T) {
		t.Parallel()

		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				return nil, fmt.Errorf("failed to fetch sitemap")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs", "--preview"}, stdout, stderr, nil, sitemapSvc, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Contains(t, stderr.String(), "failed to fetch sitemap")
		assert.Empty(t, stdout.String())
	})

	t.Run("preview returns error when sitemap service is nil", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs", "--preview"}, stdout, stderr, nil, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})

	t.Run("preview with filter passes filter to sitemap service", func(t *testing.T) {
		t.Parallel()

		var receivedFilter *locdoc.URLFilter
		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				receivedFilter = filter
				return []string{
					"https://example.com/docs/api/one",
					"https://example.com/docs/api/two",
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{
			"myproject", "https://example.com/docs",
			"--preview",
			"--filter", "/api/",
		}, stdout, stderr, nil, sitemapSvc, nil)

		assert.Equal(t, 0, code)
		require.NotNil(t, receivedFilter)
		require.Len(t, receivedFilter.Include, 1)
		assert.Equal(t, "/api/", receivedFilter.Include[0].String())
	})

	t.Run("preview with multiple filters passes all to sitemap service", func(t *testing.T) {
		t.Parallel()

		var receivedFilter *locdoc.URLFilter
		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				receivedFilter = filter
				return []string{"https://example.com/docs/page"}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{
			"myproject", "https://example.com/docs",
			"--preview",
			"--filter", "docs",
			"--filter", "blog",
		}, stdout, stderr, nil, sitemapSvc, nil)

		assert.Equal(t, 0, code)
		require.NotNil(t, receivedFilter)
		require.Len(t, receivedFilter.Include, 2)
		assert.Equal(t, "docs", receivedFilter.Include[0].String())
		assert.Equal(t, "blog", receivedFilter.Include[1].String())
	})

	t.Run("stores filter on project creation", func(t *testing.T) {
		t.Parallel()

		var createdProject *locdoc.Project
		projectSvc := &mock.ProjectService{
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				p.ID = "test-id-123"
				createdProject = p
				return nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{
			"myproject", "https://example.com/docs",
			"--filter", "api",
			"--filter", "docs",
		}, stdout, stderr, projectSvc, nil, nil)

		assert.Equal(t, 0, code)
		require.NotNil(t, createdProject)
		assert.Equal(t, "api\ndocs", createdProject.Filter)
	})

	t.Run("returns error for invalid filter regex", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{
			"myproject", "https://example.com/docs",
			"--filter", "[invalid",
		}, stdout, stderr, nil, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "invalid")
	})

	t.Run("creates project and crawls documents", func(t *testing.T) {
		t.Parallel()

		var createdProject *locdoc.Project
		projectSvc := &mock.ProjectService{
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				p.ID = "test-id-123"
				createdProject = p
				return nil
			},
		}

		var createdDocs []*locdoc.Document
		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return nil, nil // No existing docs
			},
			CreateDocumentFn: func(ctx context.Context, doc *locdoc.Document) error {
				doc.ID = "doc-" + doc.SourceURL
				createdDocs = append(createdDocs, doc)
				return nil
			},
		}

		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				return []string{
					"https://example.com/docs/page1",
					"https://example.com/docs/page2",
				}, nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body><h1>Test</h1><p>Content</p></body></html>", nil
			},
			CloseFn: func() error { return nil },
		}

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Test Page", ContentHTML: "<p>Content</p>"}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(html string) (string, error) {
				return "# Content\n\nSome text", nil
			},
		}

		crawlDeps := &main.CrawlDeps{
			Documents:    documentSvc,
			Fetcher:      fetcher,
			Extractor:    extractor,
			Converter:    converter,
			TokenCounter: nil,
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs"}, stdout, stderr, projectSvc, sitemapSvc, crawlDeps)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "Added project")
		assert.Contains(t, stdout.String(), "Saved 2 pages")
		assert.Empty(t, stderr.String())
		require.NotNil(t, createdProject)
		require.Len(t, createdDocs, 2)
		assert.Equal(t, "test-id-123", createdDocs[0].ProjectID)
	})

	t.Run("applies filter during crawl", func(t *testing.T) {
		t.Parallel()

		var createdProject *locdoc.Project
		projectSvc := &mock.ProjectService{
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				p.ID = "test-id-123"
				createdProject = p
				return nil
			},
		}

		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return nil, nil
			},
			CreateDocumentFn: func(ctx context.Context, doc *locdoc.Document) error {
				doc.ID = "doc-" + doc.SourceURL
				return nil
			},
		}

		var receivedFilter *locdoc.URLFilter
		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				receivedFilter = filter
				return []string{"https://example.com/docs/api/one"}, nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body><p>Content</p></body></html>", nil
			},
			CloseFn: func() error { return nil },
		}

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Test", ContentHTML: "<p>Content</p>"}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(html string) (string, error) {
				return "Content", nil
			},
		}

		crawlDeps := &main.CrawlDeps{
			Documents:    documentSvc,
			Fetcher:      fetcher,
			Extractor:    extractor,
			Converter:    converter,
			TokenCounter: nil,
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{
			"myproject", "https://example.com/docs",
			"--filter", "/api/",
		}, stdout, stderr, projectSvc, sitemapSvc, crawlDeps)

		assert.Equal(t, 0, code)
		assert.Empty(t, stderr.String())
		require.NotNil(t, createdProject)
		assert.Equal(t, "/api/", createdProject.Filter)
		// Verify filter was passed to sitemap discovery during crawl
		require.NotNil(t, receivedFilter, "filter should be passed to DiscoverURLs during crawl")
		require.Len(t, receivedFilter.Include, 1)
		assert.Equal(t, "/api/", receivedFilter.Include[0].String())
	})

	t.Run("with --force deletes existing project before creating", func(t *testing.T) {
		t.Parallel()

		var deletedID string
		var createdProject *locdoc.Project
		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "myproject" {
					return []*locdoc.Project{
						{ID: "existing-id", Name: "myproject", SourceURL: "https://old.com"},
					}, nil
				}
				return nil, nil
			},
			DeleteProjectFn: func(ctx context.Context, id string) error {
				deletedID = id
				return nil
			},
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				p.ID = "new-id-123"
				createdProject = p
				return nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs", "--force"}, stdout, stderr, projectSvc, nil, nil)

		assert.Equal(t, 0, code)
		assert.Equal(t, "existing-id", deletedID)
		require.NotNil(t, createdProject)
		assert.Equal(t, "myproject", createdProject.Name)
		assert.Equal(t, "https://example.com/docs", createdProject.SourceURL)
		assert.Contains(t, stdout.String(), "Added project")
		assert.Empty(t, stderr.String())
	})

	t.Run("without --force errors on existing project", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				return locdoc.Errorf(locdoc.ECONFLICT, "project already exists")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"existing", "https://example.com"}, stdout, stderr, projectSvc, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})

	t.Run("with --force succeeds when project does not exist", func(t *testing.T) {
		t.Parallel()

		var createdProject *locdoc.Project
		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				// No existing project
				return nil, nil
			},
			CreateProjectFn: func(ctx context.Context, p *locdoc.Project) error {
				p.ID = "new-id-123"
				createdProject = p
				return nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"newproject", "https://example.com/docs", "--force"}, stdout, stderr, projectSvc, nil, nil)

		assert.Equal(t, 0, code)
		require.NotNil(t, createdProject)
		assert.Equal(t, "newproject", createdProject.Name)
		assert.Contains(t, stdout.String(), "Added project")
		assert.Empty(t, stderr.String())
	})

	t.Run("with --force returns error when find projects fails", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return nil, locdoc.Errorf(locdoc.EINTERNAL, "database error")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs", "--force"}, stdout, stderr, projectSvc, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})

	t.Run("with --force returns error when delete fails", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{ID: "existing-id", Name: "myproject", SourceURL: "https://old.com"},
				}, nil
			},
			DeleteProjectFn: func(ctx context.Context, id string) error {
				return locdoc.Errorf(locdoc.EINTERNAL, "database error")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs", "--force"}, stdout, stderr, projectSvc, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})
}

func TestParseAddArgs(t *testing.T) {
	t.Parallel()

	t.Run("parses name and url", func(t *testing.T) {
		t.Parallel()

		opts, err := main.ParseAddArgs([]string{"myproject", "https://example.com/docs"})

		require.NoError(t, err)
		assert.Equal(t, "myproject", opts.Name)
		assert.Equal(t, "https://example.com/docs", opts.URL)
		assert.False(t, opts.Preview)
		assert.Empty(t, opts.Filters)
	})

	t.Run("parses --preview flag", func(t *testing.T) {
		t.Parallel()

		opts, err := main.ParseAddArgs([]string{"myproject", "https://example.com/docs", "--preview"})

		require.NoError(t, err)
		assert.Equal(t, "myproject", opts.Name)
		assert.Equal(t, "https://example.com/docs", opts.URL)
		assert.True(t, opts.Preview)
	})

	t.Run("parses --preview flag in any position", func(t *testing.T) {
		t.Parallel()

		opts, err := main.ParseAddArgs([]string{"--preview", "myproject", "https://example.com/docs"})

		require.NoError(t, err)
		assert.Equal(t, "myproject", opts.Name)
		assert.Equal(t, "https://example.com/docs", opts.URL)
		assert.True(t, opts.Preview)
	})

	t.Run("returns error for missing url", func(t *testing.T) {
		t.Parallel()

		_, err := main.ParseAddArgs([]string{"onlyname"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "usage:")
	})

	t.Run("returns error for no arguments", func(t *testing.T) {
		t.Parallel()

		_, err := main.ParseAddArgs([]string{})

		assert.Error(t, err)
	})

	t.Run("parses single --filter flag", func(t *testing.T) {
		t.Parallel()

		opts, err := main.ParseAddArgs([]string{"myproject", "https://example.com/docs", "--filter", "docs"})

		require.NoError(t, err)
		assert.Equal(t, "myproject", opts.Name)
		assert.Equal(t, "https://example.com/docs", opts.URL)
		assert.Equal(t, []string{"docs"}, opts.Filters)
	})

	t.Run("parses multiple --filter flags", func(t *testing.T) {
		t.Parallel()

		opts, err := main.ParseAddArgs([]string{
			"myproject", "https://example.com/docs",
			"--filter", "docs",
			"--filter", "blog",
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"docs", "blog"}, opts.Filters)
	})

	t.Run("parses --filter with --preview", func(t *testing.T) {
		t.Parallel()

		opts, err := main.ParseAddArgs([]string{
			"myproject", "https://example.com/docs",
			"--preview",
			"--filter", "api",
		})

		require.NoError(t, err)
		assert.True(t, opts.Preview)
		assert.Equal(t, []string{"api"}, opts.Filters)
	})

	t.Run("returns error for --filter without value", func(t *testing.T) {
		t.Parallel()

		_, err := main.ParseAddArgs([]string{"myproject", "https://example.com/docs", "--filter"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "--filter")
	})

	t.Run("returns error for extra positional arguments", func(t *testing.T) {
		t.Parallel()

		_, err := main.ParseAddArgs([]string{"myproject", "https://example.com/docs", "extraarg"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected")
	})

	t.Run("parses --force flag", func(t *testing.T) {
		t.Parallel()

		opts, err := main.ParseAddArgs([]string{"myproject", "https://example.com/docs", "--force"})

		require.NoError(t, err)
		assert.Equal(t, "myproject", opts.Name)
		assert.Equal(t, "https://example.com/docs", opts.URL)
		assert.True(t, opts.Force)
	})

	t.Run("parses --force flag in any position", func(t *testing.T) {
		t.Parallel()

		opts, err := main.ParseAddArgs([]string{"--force", "myproject", "https://example.com/docs"})

		require.NoError(t, err)
		assert.Equal(t, "myproject", opts.Name)
		assert.Equal(t, "https://example.com/docs", opts.URL)
		assert.True(t, opts.Force)
	})
}

func TestCmdList(t *testing.T) {
	t.Parallel()

	t.Run("lists projects", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{ID: "id-1", Name: "project-one", SourceURL: "https://one.com"},
					{ID: "id-2", Name: "project-two", SourceURL: "https://two.com"},
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdList(testContext(), stdout, stderr, projectSvc)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "project-one")
		assert.Contains(t, stdout.String(), "project-two")
		assert.Contains(t, stdout.String(), "https://one.com")
		assert.Empty(t, stderr.String())
	})

	t.Run("shows message when no projects", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdList(testContext(), stdout, stderr, projectSvc)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "No projects")
		assert.Empty(t, stderr.String())
	})

	t.Run("returns error when find fails", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return nil, locdoc.Errorf(locdoc.EINTERNAL, "database error")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdList(testContext(), stdout, stderr, projectSvc)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})
}

func TestCmdDelete(t *testing.T) {
	t.Parallel()

	t.Run("deletes project by name", func(t *testing.T) {
		t.Parallel()

		var deletedID string
		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "myproject" {
					return []*locdoc.Project{
						{ID: "proj-123", Name: "myproject", SourceURL: "https://example.com"},
					}, nil
				}
				return nil, nil
			},
			DeleteProjectFn: func(ctx context.Context, id string) error {
				deletedID = id
				return nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDelete(testContext(), []string{"myproject", "--force"}, stdout, stderr, projectSvc)

		assert.Equal(t, 0, code)
		assert.Equal(t, "proj-123", deletedID)
		assert.Contains(t, stdout.String(), "Deleted")
		assert.Contains(t, stdout.String(), "myproject")
		assert.Empty(t, stderr.String())
	})

	t.Run("returns error for missing name argument", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDelete(testContext(), []string{}, stdout, stderr, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "usage:")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDelete(testContext(), []string{"nonexistent", "--force"}, stdout, stderr, projectSvc)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "not found")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{ID: "proj-123", Name: "myproject", SourceURL: "https://example.com"},
				}, nil
			},
			DeleteProjectFn: func(ctx context.Context, id string) error {
				return locdoc.Errorf(locdoc.EINTERNAL, "database error")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDelete(testContext(), []string{"myproject", "--force"}, stdout, stderr, projectSvc)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})

	t.Run("requires force flag without confirmation", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{ID: "proj-123", Name: "myproject", SourceURL: "https://example.com"},
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// No --force flag
		code := main.CmdDelete(testContext(), []string{"myproject"}, stdout, stderr, projectSvc)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "--force")
		assert.Empty(t, stdout.String())
	})

	t.Run("allows force flag before project name", func(t *testing.T) {
		t.Parallel()

		var deletedID string
		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "myproject" {
					return []*locdoc.Project{
						{ID: "proj-123", Name: "myproject", SourceURL: "https://example.com"},
					}, nil
				}
				return nil, nil
			},
			DeleteProjectFn: func(ctx context.Context, id string) error {
				deletedID = id
				return nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// --force before project name
		code := main.CmdDelete(testContext(), []string{"--force", "myproject"}, stdout, stderr, projectSvc)

		assert.Equal(t, 0, code)
		assert.Equal(t, "proj-123", deletedID)
		assert.Contains(t, stdout.String(), "Deleted")
		assert.Empty(t, stderr.String())
	})
}

func TestCmdDocs(t *testing.T) {
	t.Parallel()

	t.Run("lists documents in summary mode", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "myproject" {
					return []*locdoc.Project{
						{ID: "proj-1", Name: "myproject", SourceURL: "https://example.com"},
					}, nil
				}
				return nil, nil
			},
		}

		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				if filter.ProjectID != nil && *filter.ProjectID == "proj-1" && filter.SortBy == "position" {
					return []*locdoc.Document{
						{ID: "doc-1", Title: "Getting Started", SourceURL: "https://example.com/docs/getting-started", Position: 0},
						{ID: "doc-2", Title: "Functions", SourceURL: "https://example.com/docs/functions", Position: 1},
					}, nil
				}
				return nil, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDocs(testContext(), []string{"myproject"}, stdout, stderr, projectSvc, documentSvc)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "Documents for myproject (2 total)")
		assert.Contains(t, stdout.String(), "1. Getting Started")
		assert.Contains(t, stdout.String(), "https://example.com/docs/getting-started")
		assert.Contains(t, stdout.String(), "2. Functions")
		assert.Contains(t, stdout.String(), "https://example.com/docs/functions")
		assert.Empty(t, stderr.String())
	})

	t.Run("shows full content with --full flag", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "myproject" {
					return []*locdoc.Project{
						{ID: "proj-1", Name: "myproject", SourceURL: "https://example.com"},
					}, nil
				}
				return nil, nil
			},
		}

		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				if filter.ProjectID != nil && *filter.ProjectID == "proj-1" && filter.SortBy == "position" {
					return []*locdoc.Document{
						{ID: "doc-1", Title: "Getting Started", SourceURL: "https://example.com/docs/getting-started", Content: "# Getting Started\n\nWelcome to the docs.", Position: 0},
						{ID: "doc-2", Title: "Functions", SourceURL: "https://example.com/docs/functions", Content: "# Functions\n\nHere are the functions.", Position: 1},
					}, nil
				}
				return nil, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDocs(testContext(), []string{"myproject", "--full"}, stdout, stderr, projectSvc, documentSvc)

		assert.Equal(t, 0, code)
		// Uses FormatDocuments output
		assert.Contains(t, stdout.String(), "## Document: Getting Started")
		assert.Contains(t, stdout.String(), "# Getting Started")
		assert.Contains(t, stdout.String(), "Welcome to the docs.")
		assert.Contains(t, stdout.String(), "## Document: Functions")
		assert.Contains(t, stdout.String(), "# Functions")
		assert.Contains(t, stdout.String(), "Here are the functions.")
		assert.Empty(t, stderr.String())
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDocs(testContext(), []string{"nonexistent"}, stdout, stderr, projectSvc, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), `project "nonexistent" not found`)
		assert.Contains(t, stderr.String(), "locdoc list")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error when project has no documents", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{ID: "proj-1", Name: "emptyproject", SourceURL: "https://example.com"},
				}, nil
			},
		}

		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return []*locdoc.Document{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDocs(testContext(), []string{"emptyproject"}, stdout, stderr, projectSvc, documentSvc)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), `project "emptyproject" has no documents`)
		assert.Contains(t, stderr.String(), "locdoc delete")
		assert.Contains(t, stderr.String(), "locdoc add")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error for missing name argument", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDocs(testContext(), []string{}, stdout, stderr, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "usage:")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error when find projects fails", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return nil, locdoc.Errorf(locdoc.EINTERNAL, "database error")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDocs(testContext(), []string{"myproject"}, stdout, stderr, projectSvc, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error when find documents fails", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{ID: "proj-1", Name: "myproject", SourceURL: "https://example.com"},
				}, nil
			},
		}

		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return nil, locdoc.Errorf(locdoc.EINTERNAL, "database error")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdDocs(testContext(), []string{"myproject"}, stdout, stderr, projectSvc, documentSvc)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})

	t.Run("allows --full flag before project name", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "myproject" {
					return []*locdoc.Project{
						{ID: "proj-1", Name: "myproject", SourceURL: "https://example.com"},
					}, nil
				}
				return nil, nil
			},
		}

		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return []*locdoc.Document{
					{ID: "doc-1", Title: "Test", Content: "Test content", Position: 0},
				}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// --full before project name
		code := main.CmdDocs(testContext(), []string{"--full", "myproject"}, stdout, stderr, projectSvc, documentSvc)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "## Document: Test")
		assert.Empty(t, stderr.String())
	})
}

func TestCmdAsk(t *testing.T) {
	t.Parallel()

	t.Run("asks question and returns answer", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == "myproject" {
					return []*locdoc.Project{
						{ID: "proj-1", Name: "myproject", SourceURL: "https://example.com"},
					}, nil
				}
				return nil, nil
			},
		}

		asker := &mock.Asker{
			AskFn: func(ctx context.Context, projectID, question string) (string, error) {
				assert.Equal(t, "proj-1", projectID)
				assert.Equal(t, "What is htmx?", question)
				return "htmx is a library for building web applications.", nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAsk(testContext(), []string{"myproject", "What is htmx?"}, stdout, stderr, projectSvc, asker)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "htmx is a library")
		assert.Empty(t, stderr.String())
	})

	t.Run("returns error for missing arguments", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAsk(testContext(), []string{"onlyproject"}, stdout, stderr, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "usage:")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error for no arguments", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAsk(testContext(), []string{}, stdout, stderr, nil, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "usage:")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAsk(testContext(), []string{"nonexistent", "question?"}, stdout, stderr, projectSvc, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), `project "nonexistent" not found`)
		assert.Contains(t, stderr.String(), "locdoc list")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error when asker fails", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{
					{ID: "proj-1", Name: "myproject", SourceURL: "https://example.com"},
				}, nil
			},
		}

		asker := &mock.Asker{
			AskFn: func(ctx context.Context, projectID, question string) (string, error) {
				return "", locdoc.Errorf(locdoc.ENOTFOUND, "no documents found")
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAsk(testContext(), []string{"myproject", "question?"}, stdout, stderr, projectSvc, asker)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})
}

func TestRun_HelpFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"--help flag", []string{"--help"}},
		{"-h flag", []string{"-h"}},
		{"help command", []string{"help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := main.NewMain()
			m.DBPath = filepath.Join(t.TempDir(), "test.db")

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			err := m.Run(testContext(), tt.args, stdout, stderr)

			require.NoError(t, err)
			// Usage should be printed to stdout (not stderr) when explicitly requested
			assert.Contains(t, stdout.String(), "usage: locdoc")
			assert.Contains(t, stdout.String(), "Commands:")
			assert.Empty(t, stderr.String())
		})
	}
}

func TestRun_NoArgs(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	m := main.NewMain()
	m.DBPath = filepath.Join(tmpDir, "test.db")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := m.Run(testContext(), []string{}, stdout, stderr)

	// No args should show usage to stderr and return error
	require.Error(t, err)
	assert.Contains(t, stderr.String(), "usage: locdoc")
}

func TestRun_HelpWithoutCreatingDB(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "should-not-exist.db")

	m := main.NewMain()
	m.DBPath = dbPath

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := m.Run(testContext(), []string{"--help"}, stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "usage: locdoc")
	assert.Empty(t, stderr.String())

	// Verify database file was NOT created
	_, statErr := os.Stat(dbPath)
	assert.True(t, os.IsNotExist(statErr), "database file should not be created for --help")
}
