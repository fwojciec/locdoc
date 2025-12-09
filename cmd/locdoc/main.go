package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/htmltomarkdown"
	lochttp "github.com/fwojciec/locdoc/http"
	"github.com/fwojciec/locdoc/rod"
	"github.com/fwojciec/locdoc/sqlite"
	"github.com/fwojciec/locdoc/trafilatura"
)

func main() {
	ctx := context.Background()

	m := NewMain()

	if err := m.Run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the program.
type Main struct {
	// Database path. Set before calling Run().
	DBPath string

	// SQLite database used by SQLite service implementations.
	DB *sqlite.DB

	// Services for end-to-end testing.
	ProjectService  locdoc.ProjectService
	DocumentService locdoc.DocumentService
}

// NewMain returns a new instance of Main with defaults.
func NewMain() *Main {
	return &Main{
		DBPath: defaultDBPath(),
	}
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	if m.DB != nil {
		return m.DB.Close()
	}
	return nil
}

// Run executes the CLI with the given arguments.
func (m *Main) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		return m.usage(stderr)
	}

	// Open database
	m.DB = sqlite.NewDB(m.DBPath)
	if err := m.DB.Open(); err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer m.Close()

	// Wire services
	m.ProjectService = sqlite.NewProjectService(m.DB)
	m.DocumentService = sqlite.NewDocumentService(m.DB)

	// Dispatch command
	cmd, cmdArgs := args[0], args[1:]
	switch cmd {
	case "add":
		return m.runAdd(ctx, cmdArgs, stdout, stderr)
	case "list":
		return m.runList(ctx, stdout, stderr)
	case "delete":
		return m.runDelete(ctx, cmdArgs, stdout, stderr)
	case "crawl":
		return m.runCrawl(ctx, cmdArgs, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n", cmd)
		return m.usage(stderr)
	}
}

func (m *Main) usage(w io.Writer) error {
	fmt.Fprintln(w, "usage: locdoc <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  add <name> <url>       Register a documentation project")
	fmt.Fprintln(w, "  list                   List all registered projects")
	fmt.Fprintln(w, "  delete <name> --force  Delete a project and its documents")
	fmt.Fprintln(w, "  crawl [name]           Crawl documentation for all or one project")
	return fmt.Errorf("invalid usage")
}

func (m *Main) runAdd(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	code := CmdAdd(ctx, args, stdout, stderr, m.ProjectService)
	if code != 0 {
		return fmt.Errorf("add command failed")
	}
	return nil
}

func (m *Main) runList(ctx context.Context, stdout, stderr io.Writer) error {
	code := CmdList(ctx, stdout, stderr, m.ProjectService)
	if code != 0 {
		return fmt.Errorf("list command failed")
	}
	return nil
}

func (m *Main) runDelete(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	code := CmdDelete(ctx, args, stdout, stderr, m.ProjectService)
	if code != 0 {
		return fmt.Errorf("delete command failed")
	}
	return nil
}

func (m *Main) runCrawl(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	// Wire crawl dependencies
	sitemapSvc := lochttp.NewSitemapService(nil)
	fetcher, err := rod.NewFetcher()
	if err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}
	defer fetcher.Close()
	extractor := trafilatura.NewExtractor()
	converter := htmltomarkdown.NewConverter()

	code := CmdCrawl(ctx, args, stdout, stderr, m.ProjectService, m.DocumentService, sitemapSvc, fetcher, extractor, converter)
	if code != 0 {
		return fmt.Errorf("crawl command failed")
	}
	return nil
}

func defaultDBPath() string {
	if path := os.Getenv("LOCDOC_DB"); path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "locdoc.db"
	}
	dir := filepath.Join(home, ".locdoc")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "locdoc.db")
}

// CmdAdd handles the "add" command to register a new project.
func CmdAdd(ctx context.Context, args []string, stdout, stderr io.Writer, projects locdoc.ProjectService) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "usage: locdoc add <name> <url>")
		return 1
	}

	name, url := args[0], args[1]

	project := &locdoc.Project{
		Name:      name,
		SourceURL: url,
	}

	if err := projects.CreateProject(ctx, project); err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}

	fmt.Fprintf(stdout, "Added project %q (%s)\n", name, project.ID)
	return 0
}

// CmdList handles the "list" command to show all registered projects.
func CmdList(ctx context.Context, stdout, stderr io.Writer, projects locdoc.ProjectService) int {
	list, err := projects.FindProjects(ctx, locdoc.ProjectFilter{})
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}

	if len(list) == 0 {
		fmt.Fprintln(stdout, "No projects. Use 'locdoc add <name> <url>' to add one.")
		return 0
	}

	for _, p := range list {
		id := p.ID
		if len(id) > 8 {
			id = id[:8]
		}
		fmt.Fprintf(stdout, "%s  %s  %s\n", id, p.Name, p.SourceURL)
	}
	return 0
}

// CmdCrawl handles the "crawl" command to crawl documentation for projects.
func CmdCrawl(
	ctx context.Context,
	args []string,
	stdout, stderr io.Writer,
	projects locdoc.ProjectService,
	documents locdoc.DocumentService,
	sitemap locdoc.SitemapService,
	fetcher locdoc.Fetcher,
	extractor locdoc.Extractor,
	converter locdoc.Converter,
) int {

	// Determine which projects to crawl
	var projectList []*locdoc.Project
	if len(args) > 0 {
		// Crawl specific project by name
		name := args[0]
		list, err := projects.FindProjects(ctx, locdoc.ProjectFilter{Name: &name})
		if err != nil {
			fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
			return 1
		}
		if len(list) == 0 {
			fmt.Fprintf(stderr, "error: project %q not found\n", name)
			return 1
		}
		projectList = list
	} else {
		// Crawl all projects
		var err error
		projectList, err = projects.FindProjects(ctx, locdoc.ProjectFilter{})
		if err != nil {
			fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
			return 1
		}
	}

	if len(projectList) == 0 {
		fmt.Fprintln(stderr, "No projects to crawl. Use 'locdoc add' first.")
		return 1
	}

	// Crawl each project
	var hasError bool
	for _, project := range projectList {
		fmt.Fprintf(stdout, "Crawling %s (%s)...\n", project.Name, project.SourceURL)

		if err := crawlProject(ctx, project, stdout, stderr,
			sitemap, fetcher, extractor, converter, documents); err != nil {
			fmt.Fprintf(stderr, "error crawling %s: %v\n", project.Name, err)
			hasError = true
		}
	}

	if hasError {
		return 1
	}
	return 0
}

func crawlProject(
	ctx context.Context,
	project *locdoc.Project,
	stdout, stderr io.Writer,
	sitemap locdoc.SitemapService,
	fetcher locdoc.Fetcher,
	extractor locdoc.Extractor,
	converter locdoc.Converter,
	documents locdoc.DocumentService,
) error {
	// Discover URLs from sitemap
	urls, err := sitemap.DiscoverURLs(ctx, project.SourceURL, nil)
	if err != nil {
		return fmt.Errorf("sitemap discovery: %w", err)
	}

	fmt.Fprintf(stdout, "  Found %d URLs\n", len(urls))

	// Process each URL
	for i, url := range urls {
		fmt.Fprintf(stdout, "  [%d/%d] %s\n", i+1, len(urls), url)

		// Fetch HTML
		html, err := fetcher.Fetch(ctx, url)
		if err != nil {
			fmt.Fprintf(stderr, "    skip (fetch failed): %v\n", err)
			continue
		}

		// Extract main content
		result, err := extractor.Extract(html)
		if err != nil {
			fmt.Fprintf(stderr, "    skip (extract failed): %v\n", err)
			continue
		}

		// Convert to markdown
		markdown, err := converter.Convert(result.ContentHTML)
		if err != nil {
			fmt.Fprintf(stderr, "    skip (convert failed): %v\n", err)
			continue
		}

		// Compute content hash
		hash := computeHash(markdown)

		// Check if document already exists with same hash
		existing, _ := findDocumentByURL(ctx, documents, project.ID, url)
		if existing != nil && existing.ContentHash == hash {
			fmt.Fprintln(stdout, "    unchanged")
			continue
		}

		// Create or update document
		doc := &locdoc.Document{
			ProjectID:   project.ID,
			SourceURL:   url,
			Title:       result.Title,
			Content:     markdown,
			ContentHash: hash,
		}

		if existing != nil {
			// Update existing
			if _, err := documents.UpdateDocument(ctx, existing.ID, locdoc.DocumentUpdate{
				Title:       &doc.Title,
				Content:     &doc.Content,
				ContentHash: &doc.ContentHash,
			}); err != nil {
				fmt.Fprintf(stderr, "    error updating: %v\n", err)
				continue
			}
			fmt.Fprintln(stdout, "    updated")
		} else {
			// Create new
			if err := documents.CreateDocument(ctx, doc); err != nil {
				fmt.Fprintf(stderr, "    error creating: %v\n", err)
				continue
			}
			fmt.Fprintln(stdout, "    saved")
		}
	}

	return nil
}

func findDocumentByURL(ctx context.Context, docs locdoc.DocumentService, projectID, url string) (*locdoc.Document, error) {
	list, err := docs.FindDocuments(ctx, locdoc.DocumentFilter{
		ProjectID: &projectID,
		SourceURL: &url,
	})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}

func computeHash(content string) string {
	h := xxhash.Sum64String(content)
	return fmt.Sprintf("%x", h)
}

// CmdDelete handles the "delete" command to remove a project.
func CmdDelete(ctx context.Context, args []string, stdout, stderr io.Writer, projects locdoc.ProjectService) int {
	var name string
	var force bool

	// Parse arguments - allow --force in any position
	for _, arg := range args {
		if arg == "--force" {
			force = true
		} else if name == "" {
			name = arg
		}
	}

	if name == "" {
		fmt.Fprintln(stderr, "usage: locdoc delete <name> --force")
		return 1
	}

	// Find project by name
	list, err := projects.FindProjects(ctx, locdoc.ProjectFilter{Name: &name})
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}
	if len(list) == 0 {
		fmt.Fprintf(stderr, "error: project %q not found\n", name)
		return 1
	}

	project := list[0]

	// Require --force flag
	if !force {
		fmt.Fprintf(stderr, "error: use --force to confirm deletion of project %q\n", name)
		return 1
	}

	// Delete project
	if err := projects.DeleteProject(ctx, project.ID); err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}

	fmt.Fprintf(stdout, "Deleted project %q\n", name)
	return 0
}
