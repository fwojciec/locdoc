package main_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/fwojciec/locdoc"
	main "github.com/fwojciec/locdoc/cmd/locdoc"
	"github.com/fwojciec/locdoc/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

		code := main.CmdAdd([]string{"myproject", "https://example.com/docs"}, stdout, stderr, projectSvc)

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

		code := main.CmdAdd([]string{"onlyname"}, stdout, stderr, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "usage:")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error for no arguments", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd([]string{}, stdout, stderr, nil)

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

		code := main.CmdAdd([]string{"existing", "https://example.com"}, stdout, stderr, projectSvc)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
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

		code := main.CmdList(stdout, stderr, projectSvc)

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

		code := main.CmdList(stdout, stderr, projectSvc)

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

		code := main.CmdList(stdout, stderr, projectSvc)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "error:")
		assert.Empty(t, stdout.String())
	})
}

func TestCmdCrawl(t *testing.T) {
	t.Parallel()

	t.Run("crawls single project by name", func(t *testing.T) {
		t.Parallel()

		projectName := "myproject"
		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name != nil && *filter.Name == projectName {
					return []*locdoc.Project{
						{ID: "proj-1", Name: projectName, SourceURL: "https://example.com"},
					}, nil
				}
				return nil, nil
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
				return []string{"https://example.com/page1", "https://example.com/page2"}, nil
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

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdCrawl(
			[]string{projectName},
			stdout, stderr,
			projectSvc, documentSvc,
			sitemapSvc, fetcher, extractor, converter,
		)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "Crawling")
		assert.Contains(t, stdout.String(), projectName)
		assert.Empty(t, stderr.String())
		assert.Len(t, createdDocs, 2)
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

		code := main.CmdCrawl(
			[]string{"nonexistent"},
			stdout, stderr,
			projectSvc, nil, nil, nil, nil, nil,
		)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "not found")
	})

	t.Run("crawls all projects when no name given", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				if filter.Name == nil {
					return []*locdoc.Project{
						{ID: "proj-1", Name: "project-one", SourceURL: "https://one.com"},
						{ID: "proj-2", Name: "project-two", SourceURL: "https://two.com"},
					}, nil
				}
				return nil, nil
			},
		}

		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				return nil, nil
			},
			CreateDocumentFn: func(ctx context.Context, doc *locdoc.Document) error {
				return nil
			},
		}

		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				return []string{baseURL + "/page"}, nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html></html>", nil
			},
			CloseFn: func() error { return nil },
		}

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Page", ContentHTML: "<p>text</p>"}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(html string) (string, error) {
				return "text", nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdCrawl(
			[]string{},
			stdout, stderr,
			projectSvc, documentSvc,
			sitemapSvc, fetcher, extractor, converter,
		)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "project-one")
		assert.Contains(t, stdout.String(), "project-two")
	})

	t.Run("returns error when no projects exist", func(t *testing.T) {
		t.Parallel()

		projectSvc := &mock.ProjectService{
			FindProjectsFn: func(ctx context.Context, filter locdoc.ProjectFilter) ([]*locdoc.Project, error) {
				return []*locdoc.Project{}, nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdCrawl(
			[]string{},
			stdout, stderr,
			projectSvc, nil, nil, nil, nil, nil,
		)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "No projects")
	})
}
