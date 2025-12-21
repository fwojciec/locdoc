package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/gemini"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/fwojciec/locdoc/htmltomarkdown"
	lochttp "github.com/fwojciec/locdoc/http"
	"github.com/fwojciec/locdoc/rod"
	locslog "github.com/fwojciec/locdoc/slog"
	"github.com/fwojciec/locdoc/sqlite"
	"github.com/fwojciec/locdoc/trafilatura"
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
	// Initialize dependencies struct for Kong binding
	deps := &Dependencies{
		Ctx:    ctx,
		Stdout: stdout,
		Stderr: stderr,
	}

	// Create Kong parser with dependency binding
	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name("locdoc"),
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) {}), // Don't exit on help
		kong.Bind(deps),
	)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	// Handle help flags using Kong
	if len(args) == 0 {
		_, _ = parser.Parse([]string{"--help"})
		return fmt.Errorf("no command specified. Run 'locdoc --help' to see available commands")
	}

	cmd := args[0]
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		_, _ = parser.Parse([]string{"--help"})
		return nil
	}

	// Parse arguments first to know which command and its flags
	kongCtx, err := parser.Parse(args)
	if err != nil {
		return err
	}

	// Open database
	m.DB = sqlite.NewDB(m.DBPath)
	if err := m.DB.Open(); err != nil {
		fmt.Fprintf(stderr, "Hint: Set LOCDOC_DB to use a different database path\n")
		return fmt.Errorf("failed to open database at %q: %w", m.DBPath, err)
	}
	defer m.Close()

	// Wire core services into dependencies
	m.ProjectService = sqlite.NewProjectService(m.DB)
	m.DocumentService = sqlite.NewDocumentService(m.DB)
	deps.DB = m.DB
	deps.Projects = m.ProjectService
	deps.Documents = m.DocumentService
	deps.Sitemaps = lochttp.NewSitemapService(nil)

	// Wire command-specific dependencies based on command
	if cmd == "add" {
		rodFetcher, err := rod.NewFetcher(rod.WithFetchTimeout(cli.Add.Timeout))
		if err != nil {
			fmt.Fprintln(stderr, "Hint: Chrome or Chromium must be installed")
			return fmt.Errorf("failed to start browser: %w", err)
		}
		defer rodFetcher.Close()

		httpFetcher := lochttp.NewFetcher(lochttp.WithTimeout(cli.Add.Timeout))

		// Create link selector registry for recursive crawling fallback
		detector := goquery.NewDetector()
		fallbackSelector := goquery.NewGenericSelector()
		linkSelectors := goquery.NewRegistry(detector, fallbackSelector)
		registerFrameworkSelectors(linkSelectors)

		// Create rate limiter for recursive crawling (1 request per second per domain)
		rateLimiter := crawl.NewDomainLimiter(1.0)
		extractor := trafilatura.NewExtractor()

		// Use interfaces to allow wrapping with logging decorators
		var activeLinkSelectors locdoc.LinkSelectorRegistry = linkSelectors
		var activeRodFetcher locdoc.Fetcher = rodFetcher
		var activeHTTPFetcher locdoc.Fetcher = httpFetcher

		// Wrap services with logging decorators when debug is enabled
		if cli.Add.Debug {
			logger := slog.New(slog.NewTextHandler(stderr, nil))
			deps.Sitemaps = locslog.NewLoggingSitemapService(deps.Sitemaps, logger)
			activeRodFetcher = locslog.NewLoggingFetcher(rodFetcher, logger)
			activeHTTPFetcher = locslog.NewLoggingFetcher(httpFetcher, logger)
			activeLinkSelectors = locslog.NewLoggingRegistry(linkSelectors, detector, logger)
		}

		// Create Discoverer for URL discovery (preview mode and recursive crawl fallback)
		deps.Discoverer = &crawl.Discoverer{
			HTTPFetcher:   activeHTTPFetcher,
			RodFetcher:    activeRodFetcher,
			Prober:        detector,
			Extractor:     extractor,
			LinkSelectors: activeLinkSelectors,
			RateLimiter:   rateLimiter,
			Concurrency:   cli.Add.Concurrency,
		}

		// Create Crawler with core dependencies (used by both preview and full crawl)
		deps.Crawler = &crawl.Crawler{
			Sitemaps:      deps.Sitemaps,
			HTTPFetcher:   activeHTTPFetcher,
			RodFetcher:    activeRodFetcher,
			Prober:        detector,
			Extractor:     extractor,
			LinkSelectors: activeLinkSelectors,
			RateLimiter:   rateLimiter,
			Concurrency:   cli.Add.Concurrency,
		}

		// Add full crawl dependencies for non-preview mode
		if !cli.Add.Preview {
			tokenCounter, err := gemini.NewTokenCounter(tokenizerModel)
			if err != nil {
				return fmt.Errorf("failed to create token counter: %w", err)
			}

			deps.Crawler.Converter = htmltomarkdown.NewConverter()
			deps.Crawler.Documents = m.DocumentService
			deps.Crawler.TokenCounter = tokenCounter
		}
	}

	if cmd == "ask" {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(stderr, "GEMINI_API_KEY environment variable not set. Get an API key at https://aistudio.google.com/apikey")
			return fmt.Errorf("GEMINI_API_KEY not set. Get a key at https://aistudio.google.com/apikey")
		}

		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			fmt.Fprintln(stderr, "Hint: Check your GEMINI_API_KEY is valid")
			return fmt.Errorf("failed to connect to Gemini API: %w", err)
		}

		deps.Asker = gemini.NewAsker(client, m.DocumentService, defaultModel)
	}

	return kongCtx.Run(deps)
}

const defaultModel = "gemini-3-flash-preview"

// tokenizerModel is used for token counting. Using gemini-2.5-flash until
// gemini-3-flash-preview is supported by google.golang.org/genai/tokenizer.
// Track: locdoc-okw
const tokenizerModel = "gemini-2.5-flash"

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

// registerFrameworkSelectors registers all framework-specific link selectors with the registry.
func registerFrameworkSelectors(registry locdoc.LinkSelectorRegistry) {
	registry.Register(locdoc.FrameworkDocusaurus, goquery.NewDocusaurusSelector())
	registry.Register(locdoc.FrameworkMkDocs, goquery.NewMkDocsSelector())
	registry.Register(locdoc.FrameworkSphinx, goquery.NewSphinxSelector())
	registry.Register(locdoc.FrameworkVuePress, goquery.NewVuePressSelector())
	registry.Register(locdoc.FrameworkGitBook, goquery.NewGitBookSelector())
	registry.Register(locdoc.FrameworkNextra, goquery.NewNextraSelector())
}
