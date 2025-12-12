package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/gemini"
	"github.com/fwojciec/locdoc/htmltomarkdown"
	lochttp "github.com/fwojciec/locdoc/http"
	"github.com/fwojciec/locdoc/rod"
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
		return fmt.Errorf("failed to open database at %q: %w. Set LOCDOC_DB to use a different path", m.DBPath, err)
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
	if cmd == "add" && !cli.Add.Preview {
		fetcher, err := rod.NewFetcher()
		if err != nil {
			return fmt.Errorf("failed to start browser: %w. Chrome or Chromium must be installed", err)
		}
		defer fetcher.Close()

		tokenCounter, err := gemini.NewTokenCounter(defaultTokenizerModel)
		if err != nil {
			return fmt.Errorf("failed to create token counter: %w", err)
		}

		deps.Crawler = &crawl.Crawler{
			Sitemaps:     deps.Sitemaps,
			Fetcher:      fetcher,
			Extractor:    trafilatura.NewExtractor(),
			Converter:    htmltomarkdown.NewConverter(),
			Documents:    m.DocumentService,
			TokenCounter: tokenCounter,
			Concurrency:  cli.Add.Concurrency,
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
			return fmt.Errorf("failed to connect to Gemini API: %w. Check your GEMINI_API_KEY", err)
		}

		deps.Asker = gemini.NewAsker(client, m.DocumentService)
	}

	return kongCtx.Run(deps)
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
