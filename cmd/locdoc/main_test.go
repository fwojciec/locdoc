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

		code := main.CmdAdd(testContext(), []string{"myproject", "https://example.com/docs"}, stdout, stderr, projectSvc)

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

		code := main.CmdAdd(testContext(), []string{"onlyname"}, stdout, stderr, nil)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "usage:")
		assert.Empty(t, stdout.String())
	})

	t.Run("returns error for no arguments", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdAdd(testContext(), []string{}, stdout, stderr, nil)

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

		code := main.CmdAdd(testContext(), []string{"existing", "https://example.com"}, stdout, stderr, projectSvc)

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
			testContext(),
			[]string{projectName},
			stdout, stderr,
			projectSvc, documentSvc,
			sitemapSvc, fetcher, extractor, converter,
		)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "Crawling")
		assert.Contains(t, stdout.String(), projectName)
		assert.Empty(t, stderr.String())
		require.Len(t, createdDocs, 2)
		assert.Equal(t, 0, createdDocs[0].Position)
		assert.Equal(t, 1, createdDocs[1].Position)
	})

	t.Run("sets position when updating document content", func(t *testing.T) {
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

		var updatedPosition *int
		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				// Return existing document with different hash (content changed)
				return []*locdoc.Document{
					{ID: "doc-1", ProjectID: "proj-1", SourceURL: "https://example.com/page1", ContentHash: "old-hash", Position: 99},
				}, nil
			},
			UpdateDocumentFn: func(ctx context.Context, id string, update locdoc.DocumentUpdate) (*locdoc.Document, error) {
				updatedPosition = update.Position
				return &locdoc.Document{ID: id}, nil
			},
		}

		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				return []string{"https://example.com/page1"}, nil
			},
		}

		fetcher := &mock.Fetcher{
			FetchFn: func(ctx context.Context, url string) (string, error) {
				return "<html><body><p>New content</p></body></html>", nil
			},
			CloseFn: func() error { return nil },
		}

		extractor := &mock.Extractor{
			ExtractFn: func(html string) (*locdoc.ExtractResult, error) {
				return &locdoc.ExtractResult{Title: "Page", ContentHTML: "<p>New content</p>"}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(html string) (string, error) {
				return "New content", nil
			},
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdCrawl(
			testContext(),
			[]string{projectName},
			stdout, stderr,
			projectSvc, documentSvc,
			sitemapSvc, fetcher, extractor, converter,
		)

		assert.Equal(t, 0, code)
		require.NotNil(t, updatedPosition)
		assert.Equal(t, 0, *updatedPosition)
	})

	t.Run("updates position when content unchanged but position changed", func(t *testing.T) {
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

		// xxhash of "ef46db3751d8e999" is "ef46db3751d8e999" (we'll use this as content)
		// Actually, we track the hash dynamically by capturing what content is converted
		contentToReturn := "same content"
		var capturedHash string
		var updatedPosition *int
		documentSvc := &mock.DocumentService{
			FindDocumentsFn: func(ctx context.Context, filter locdoc.DocumentFilter) ([]*locdoc.Document, error) {
				// If we've captured the hash, return a doc with that hash but different position
				if capturedHash != "" {
					return []*locdoc.Document{
						{ID: "doc-1", ProjectID: "proj-1", SourceURL: "https://example.com/page1", ContentHash: capturedHash, Position: 5},
					}, nil
				}
				return nil, nil
			},
			UpdateDocumentFn: func(ctx context.Context, id string, update locdoc.DocumentUpdate) (*locdoc.Document, error) {
				updatedPosition = update.Position
				return &locdoc.Document{ID: id}, nil
			},
		}

		sitemapSvc := &mock.SitemapService{
			DiscoverURLsFn: func(ctx context.Context, baseURL string, filter *locdoc.URLFilter) ([]string, error) {
				return []string{"https://example.com/page1"}, nil
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
				return &locdoc.ExtractResult{Title: "Page", ContentHTML: "<p>same</p>"}, nil
			},
		}

		converter := &mock.Converter{
			ConvertFn: func(html string) (string, error) {
				return contentToReturn, nil
			},
		}

		// First, compute the hash that the code will compute
		capturedHash = main.ComputeHashForTest(contentToReturn)

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		code := main.CmdCrawl(
			testContext(),
			[]string{projectName},
			stdout, stderr,
			projectSvc, documentSvc,
			sitemapSvc, fetcher, extractor, converter,
		)

		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "position updated")
		require.NotNil(t, updatedPosition)
		assert.Equal(t, 0, *updatedPosition) // Position should be 0 (first in list)
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
			testContext(),
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
			testContext(),
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
			testContext(),
			[]string{},
			stdout, stderr,
			projectSvc, nil, nil, nil, nil, nil,
		)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), "No projects")
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
