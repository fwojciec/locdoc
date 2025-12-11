package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/gemini"
	"github.com/fwojciec/locdoc/htmltomarkdown"
	lochttp "github.com/fwojciec/locdoc/http"
	"github.com/fwojciec/locdoc/rod"
	"github.com/fwojciec/locdoc/sqlite"
	"github.com/fwojciec/locdoc/trafilatura"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genai"
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

	// Handle help flags before opening database
	cmd := args[0]
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		m.printUsage(stdout)
		return nil
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
	cmdArgs := args[1:]
	switch cmd {
	case "add":
		return m.runAdd(ctx, cmdArgs, stdout, stderr)
	case "list":
		return m.runList(ctx, stdout, stderr)
	case "delete":
		return m.runDelete(ctx, cmdArgs, stdout, stderr)
	case "docs":
		return m.runDocs(ctx, cmdArgs, stdout, stderr)
	case "ask":
		return m.runAsk(ctx, cmdArgs, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n", cmd)
		return m.usage(stderr)
	}
}

func (m *Main) printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: locdoc <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  add <name> <url>       Add and crawl a documentation project")
	fmt.Fprintln(w, "      --filter <regex>   Filter URLs by regex (can be repeated)")
	fmt.Fprintln(w, "      --preview          Show URLs without creating project")
	fmt.Fprintln(w, "      --force            Delete existing project first")
	fmt.Fprintln(w, "      -c, --concurrency N  Concurrent fetch limit (default: 10)")
	fmt.Fprintln(w, "  list                   List all registered projects")
	fmt.Fprintln(w, "  delete <name> --force  Delete a project and its documents")
	fmt.Fprintln(w, "  docs <name> [--full]   List documents for a project (--full for content)")
	fmt.Fprintln(w, "  ask <name> \"<question>\" Ask a question about project documentation")
}

func (m *Main) usage(w io.Writer) error {
	m.printUsage(w)
	return fmt.Errorf("invalid usage")
}

func (m *Main) runAdd(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	sitemapSvc := lochttp.NewSitemapService(nil)

	// Parse args early to check preview mode and get concurrency before
	// initializing expensive crawl dependencies
	opts, err := ParseAddArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return fmt.Errorf("add command failed")
	}

	var crawlDeps *CrawlDeps
	if !opts.Preview {
		// Wire crawl dependencies only when we'll actually crawl
		fetcher, err := rod.NewFetcher()
		if err != nil {
			return fmt.Errorf("failed to start browser: %w", err)
		}
		defer fetcher.Close()
		extractor := trafilatura.NewExtractor()
		converter := htmltomarkdown.NewConverter()

		tokenCounter, err := gemini.NewTokenCounter(defaultTokenizerModel)
		if err != nil {
			return fmt.Errorf("failed to create token counter: %w", err)
		}

		crawlDeps = &CrawlDeps{
			Documents:    m.DocumentService,
			Fetcher:      fetcher,
			Extractor:    extractor,
			Converter:    converter,
			TokenCounter: tokenCounter,
			Concurrency:  opts.Concurrency,
		}
	}

	code := CmdAdd(ctx, args, stdout, stderr, m.ProjectService, sitemapSvc, crawlDeps)
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

func (m *Main) runDocs(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	code := CmdDocs(ctx, args, stdout, stderr, m.ProjectService, m.DocumentService)
	if code != 0 {
		return fmt.Errorf("docs command failed")
	}
	return nil
}

func (m *Main) runAsk(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	// Check for API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(stderr, "GEMINI_API_KEY environment variable not set. Get an API key at https://aistudio.google.com/apikey")
		return fmt.Errorf("missing API key")
	}

	// Create Gemini client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Wire asker
	asker := gemini.NewAsker(client, m.DocumentService)

	code := CmdAsk(ctx, args, stdout, stderr, m.ProjectService, asker)
	if code != 0 {
		return fmt.Errorf("ask command failed")
	}
	return nil
}

const defaultTokenizerModel = "gemini-2.5-flash"

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

// AddOptions holds parsed arguments for the add command.
type AddOptions struct {
	Name        string
	URL         string
	Preview     bool
	Force       bool
	Filters     []string
	Concurrency int
}

// CrawlDeps holds dependencies for crawling documents.
type CrawlDeps struct {
	Documents    locdoc.DocumentService
	Fetcher      locdoc.Fetcher
	Extractor    locdoc.Extractor
	Converter    locdoc.Converter
	TokenCounter locdoc.TokenCounter
	Concurrency  int
}

// ParseAddArgs parses command-line arguments for the add command.
func ParseAddArgs(args []string) (*AddOptions, error) {
	opts := &AddOptions{
		Concurrency: 10, // default concurrency
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--preview":
			opts.Preview = true
		case "--force":
			opts.Force = true
		case "--filter":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--filter requires a pattern argument")
			}
			i++
			opts.Filters = append(opts.Filters, args[i])
		case "--concurrency", "-c":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--concurrency requires a number argument")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, fmt.Errorf("invalid concurrency value %q: must be a number", args[i])
			}
			if n < 1 {
				return nil, fmt.Errorf("invalid concurrency value %q: must be a positive integer", args[i])
			}
			opts.Concurrency = n
		default:
			if opts.Name == "" {
				opts.Name = arg
			} else if opts.URL == "" {
				opts.URL = arg
			} else {
				return nil, fmt.Errorf("unexpected argument: %q\nusage: locdoc add <name> <url> [--preview] [--force] [--filter <pattern>...]", arg)
			}
		}
	}

	if opts.Name == "" || opts.URL == "" {
		return nil, fmt.Errorf("usage: locdoc add <name> <url> [--preview] [--force] [--filter <pattern>...]")
	}

	return opts, nil
}

// CmdAdd handles the "add" command to register a new project and crawl it.
func CmdAdd(ctx context.Context, args []string, stdout, stderr io.Writer, projects locdoc.ProjectService, sitemaps locdoc.SitemapService, crawlDeps *CrawlDeps) int {
	opts, err := ParseAddArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	// Compile filters to URLFilter (validates regex patterns early)
	var urlFilter *locdoc.URLFilter
	if len(opts.Filters) > 0 {
		urlFilter = &locdoc.URLFilter{}
		for _, pattern := range opts.Filters {
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Fprintf(stderr, "error: invalid filter pattern %q: %v\n", pattern, err)
				return 1
			}
			urlFilter.Include = append(urlFilter.Include, re)
		}
	}

	// Preview mode: show URLs without creating project
	if opts.Preview {
		if sitemaps == nil {
			fmt.Fprintln(stderr, "error: preview mode requires sitemap service")
			return 1
		}
		urls, err := sitemaps.DiscoverURLs(ctx, opts.URL, urlFilter)
		if err != nil {
			fmt.Fprintf(stderr, "error: %s\n", err)
			return 1
		}
		for _, u := range urls {
			fmt.Fprintln(stdout, u)
		}
		return 0
	}

	// With --force, delete existing project first
	if opts.Force {
		existing, err := projects.FindProjects(ctx, locdoc.ProjectFilter{Name: &opts.Name})
		if err != nil {
			fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
			return 1
		}
		if len(existing) > 0 {
			if err := projects.DeleteProject(ctx, existing[0].ID); err != nil {
				fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
				return 1
			}
		}
	}

	project := &locdoc.Project{
		Name:      opts.Name,
		SourceURL: opts.URL,
		Filter:    strings.Join(opts.Filters, "\n"),
	}

	if err := projects.CreateProject(ctx, project); err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}

	fmt.Fprintf(stdout, "Added project %q (%s)\n", opts.Name, project.ID)

	// Crawl documents if crawl dependencies are provided
	if crawlDeps != nil && sitemaps != nil {
		if err := crawlProject(ctx, project, stdout, stderr,
			sitemaps, crawlDeps.Fetcher, crawlDeps.Extractor, crawlDeps.Converter,
			crawlDeps.Documents, crawlDeps.TokenCounter, crawlDeps.Concurrency); err != nil {
			fmt.Fprintf(stderr, "error crawling: %v\n", err)
			return 1
		}
	}

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

// crawlResult holds the result of fetching and processing a single URL.
type crawlResult struct {
	position int
	url      string
	title    string
	markdown string
	hash     string
	err      error
	errStage string // "fetch", "extract", or "convert"
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
	tokenCounter locdoc.TokenCounter,
	concurrency int,
) error {
	// Reconstruct URLFilter from project's stored filter patterns
	var urlFilter *locdoc.URLFilter
	if project.Filter != "" {
		urlFilter = &locdoc.URLFilter{}
		for _, pattern := range strings.Split(project.Filter, "\n") {
			if pattern == "" {
				continue
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid filter pattern %q: %w", pattern, err)
			}
			urlFilter.Include = append(urlFilter.Include, re)
		}
	}

	// Discover URLs from sitemap
	urls, err := sitemap.DiscoverURLs(ctx, project.SourceURL, urlFilter)
	if err != nil {
		return fmt.Errorf("sitemap discovery: %w", err)
	}

	fmt.Fprintf(stdout, "  Found %d URLs\n", len(urls))

	if len(urls) == 0 {
		return nil
	}

	// Fetch and process URLs concurrently with bounded concurrency
	if concurrency <= 0 {
		concurrency = 10 // default if not set
	}

	// Channel for results as they complete
	resultCh := make(chan crawlResult, len(urls))

	// Create a logger that writes retry messages to stderr
	logger := func(format string, args ...any) {
		fmt.Fprintf(stderr, format+"\n", args...)
	}

	// Start workers
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for i, url := range urls {
		i, url := i, url // capture loop variables
		g.Go(func() error {
			result := processURL(gctx, i, url, fetcher, extractor, converter, logger)
			resultCh <- result
			return nil // never return error to allow all goroutines to complete
		})
	}

	// Close channel when all workers done
	go func() {
		_ = g.Wait() // errors handled per-URL, never returned
		close(resultCh)
	}()

	// Collect results and show progress
	results := make([]crawlResult, len(urls))
	var completed, failed, saved int
	total := len(urls)

	for result := range resultCh {
		completed++
		results[result.position] = result

		if result.err != nil {
			failed++
			// Print failure on its own line (persists in scroll history)
			fmt.Fprintf(stderr, "  skip %s (%s failed): %v\n", result.url, result.errStage, result.err)
		} else {
			saved++
		}

		// Update progress line in place
		fmt.Fprintf(stdout, "\r  [%d/%d] %s (%d failed, %d saved)",
			completed, total, truncateURL(result.url, 40), failed, saved)
	}

	// Clear progress line and move to next line
	fmt.Fprintf(stdout, "\r%s\r", strings.Repeat(" ", 80))

	// Accumulate stats for summary
	var totalBytes int
	var totalTokens int

	// Save documents (in position order)
	for _, result := range results {
		if result.err != nil {
			continue
		}

		doc := &locdoc.Document{
			ProjectID:   project.ID,
			SourceURL:   result.url,
			Title:       result.title,
			Content:     result.markdown,
			ContentHash: result.hash,
			Position:    result.position,
		}

		if err := documents.CreateDocument(ctx, doc); err != nil {
			fmt.Fprintf(stderr, "  error creating %s: %v\n", result.url, err)
			saved--
			continue
		}

		// Accumulate stats
		totalBytes += len(result.markdown)
		if tokenCounter != nil {
			if tokens, err := tokenCounter.CountTokens(ctx, result.markdown); err == nil {
				totalTokens += tokens
			}
		}
	}

	// Print summary
	fmt.Fprintf(stdout, "  Saved %d pages (%s, %s)\n",
		saved, formatBytes(totalBytes), formatTokens(totalTokens))

	return nil
}

// truncateURL shortens a URL for display, keeping the end which is more informative.
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return "..." + url[len(url)-maxLen+3:]
}

// formatBytes formats bytes in human-readable form.
func formatBytes(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatTokens formats token count in human-readable form.
func formatTokens(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("~%d tokens", tokens)
	}
	return fmt.Sprintf("~%dk tokens", (tokens+500)/1000)
}

// processURL fetches and processes a single URL, returning the result.
// It uses retry logic with exponential backoff for transient fetch failures.
func processURL(
	ctx context.Context,
	position int,
	url string,
	fetcher locdoc.Fetcher,
	extractor locdoc.Extractor,
	converter locdoc.Converter,
	logger LogFunc,
) crawlResult {
	result := crawlResult{
		position: position,
		url:      url,
	}

	// Fetch HTML with retry logic for transient failures
	fetchFn := func(ctx context.Context, url string) (string, error) {
		return fetcher.Fetch(ctx, url)
	}
	html, err := FetchWithRetry(ctx, url, fetchFn, logger)
	if err != nil {
		result.err = err
		result.errStage = "fetch"
		return result
	}

	// Extract main content
	extracted, err := extractor.Extract(html)
	if err != nil {
		result.err = err
		result.errStage = "extract"
		return result
	}

	// Convert to markdown
	markdown, err := converter.Convert(extracted.ContentHTML)
	if err != nil {
		result.err = err
		result.errStage = "convert"
		return result
	}

	result.title = extracted.Title
	result.markdown = markdown
	result.hash = computeHash(markdown)

	return result
}

func computeHash(content string) string {
	h := xxhash.Sum64String(content)
	return fmt.Sprintf("%x", h)
}

// ComputeHashForTest is exported for testing purposes only.
func ComputeHashForTest(content string) string {
	return computeHash(content)
}

// CmdDocs handles the "docs" command to list documents for a project.
func CmdDocs(
	ctx context.Context,
	args []string,
	stdout, stderr io.Writer,
	projects locdoc.ProjectService,
	documents locdoc.DocumentService,
) int {
	var name string
	var full bool

	// Parse arguments - allow --full in any position
	for _, arg := range args {
		if arg == "--full" {
			full = true
		} else if name == "" {
			name = arg
		}
	}

	if name == "" {
		fmt.Fprintln(stderr, "usage: locdoc docs <name> [--full]")
		return 1
	}

	// Find project by name
	list, err := projects.FindProjects(ctx, locdoc.ProjectFilter{Name: &name})
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}
	if len(list) == 0 {
		fmt.Fprintf(stderr, "project %q not found. Use \"locdoc list\" to see available projects.\n", name)
		return 1
	}

	project := list[0]

	// Find documents for project sorted by position
	docs, err := documents.FindDocuments(ctx, locdoc.DocumentFilter{
		ProjectID: &project.ID,
		SortBy:    "position",
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}

	if len(docs) == 0 {
		fmt.Fprintf(stderr, "project %q has no documents. To re-add, first run \"locdoc delete %s --force\", then run \"locdoc add %s <url>\".\n", name, name, name)
		return 1
	}

	if full {
		// Print full formatted content (same as what ask sends to LLM)
		fmt.Fprintln(stdout, locdoc.FormatDocuments(docs))
		return 0
	}

	// Print summary listing
	fmt.Fprintf(stdout, "Documents for %s (%d total):\n\n", name, len(docs))
	for i, doc := range docs {
		title := doc.Title
		if title == "" {
			title = doc.SourceURL
		}
		fmt.Fprintf(stdout, "  %d. %s\n     %s\n", i+1, title, doc.SourceURL)
	}

	return 0
}

// CmdAsk handles the "ask" command to query project documentation.
func CmdAsk(
	ctx context.Context,
	args []string,
	stdout, stderr io.Writer,
	projects locdoc.ProjectService,
	asker locdoc.Asker,
) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: locdoc ask <project> \"<question>\"")
		return 1
	}

	name, question := args[0], args[1]

	// Find project by name
	list, err := projects.FindProjects(ctx, locdoc.ProjectFilter{Name: &name})
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}
	if len(list) == 0 {
		fmt.Fprintf(stderr, "project %q not found. Use \"locdoc list\" to see available projects.\n", name)
		return 1
	}

	project := list[0]

	// Ask the question
	answer, err := asker.Ask(ctx, project.ID, question)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", locdoc.ErrorMessage(err))
		return 1
	}

	fmt.Fprintln(stdout, answer)
	return 0
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
