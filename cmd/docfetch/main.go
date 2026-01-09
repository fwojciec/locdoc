package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/fwojciec/locdoc"
	"github.com/fwojciec/locdoc/crawl"
	"github.com/fwojciec/locdoc/goquery"
	"github.com/fwojciec/locdoc/htmltomarkdown"
	lochttp "github.com/fwojciec/locdoc/http"
	"github.com/fwojciec/locdoc/rod"
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
type Main struct{}

// NewMain returns a new instance of Main with defaults.
func NewMain() *Main {
	return &Main{}
}

// Run executes the CLI with the given arguments.
func (m *Main) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name("docfetch"),
		kong.Description("Fetch documentation sites to local markdown files"),
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) {}),
	)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	// Handle no arguments
	if len(args) == 0 {
		_, _ = parser.Parse([]string{"--help"})
		return fmt.Errorf("no arguments provided")
	}

	// Handle help flags
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		_, _ = parser.Parse([]string{"--help"})
		return nil
	}

	_, err = parser.Parse(args)
	if err != nil {
		return err
	}

	// Validate: name is required unless in preview mode
	if !cli.Preview && cli.Name == "" {
		return fmt.Errorf("name is required when not in preview mode")
	}

	// Wire dependencies
	deps := &Dependencies{
		Ctx:    ctx,
		Stdout: stdout,
		Stderr: stderr,
	}

	// Create sitemap service
	deps.Sitemaps = lochttp.NewSitemapService(nil)

	// Create fetchers
	timeout := cli.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	rodFetcher, err := rod.NewFetcher(rod.WithFetchTimeout(timeout))
	if err != nil {
		fmt.Fprintln(stderr, "Hint: Chrome or Chromium must be installed")
		return fmt.Errorf("failed to start browser: %w", err)
	}
	defer rodFetcher.Close()

	httpFetcher := lochttp.NewFetcher(lochttp.WithTimeout(timeout))

	// Create link selector registry for recursive crawling fallback
	detector := goquery.NewDetector()
	fallbackSelector := goquery.NewGenericSelector()
	linkSelectors := goquery.NewRegistry(detector, fallbackSelector)
	registerFrameworkSelectors(linkSelectors)

	// Create rate limiter for recursive crawling (1 request per second per domain)
	rateLimiter := crawl.NewDomainLimiter(1.0)
	extractor := trafilatura.NewExtractor()

	concurrency := cli.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}

	// Create Discoverer for URL discovery (preview mode and recursive crawl fallback)
	deps.Discoverer = &crawl.Discoverer{
		HTTPFetcher:   httpFetcher,
		RodFetcher:    rodFetcher,
		Prober:        detector,
		Extractor:     extractor,
		LinkSelectors: linkSelectors,
		RateLimiter:   rateLimiter,
		Concurrency:   concurrency,
	}

	// Create Crawler for full crawl mode
	if !cli.Preview {
		deps.Crawler = &crawl.Crawler{
			Discoverer: deps.Discoverer,
			Sitemaps:   deps.Sitemaps,
			Converter:  htmltomarkdown.NewConverter(),
			// Documents will be set by FetchCmd.runFetch
		}
	}

	// Create and run the fetch command
	cmd := &FetchCmd{
		URL:         cli.URL,
		Name:        cli.Name,
		Path:        cli.Path,
		Preview:     cli.Preview,
		Concurrency: concurrency,
	}

	return cmd.Run(deps)
}

// CLI defines the command-line interface structure for Kong.
type CLI struct {
	Preview     bool          `short:"p" help:"Preview what would be fetched without saving"`
	Concurrency int           `short:"c" default:"3" help:"Concurrent fetch limit"`
	Timeout     time.Duration `short:"t" default:"10s" help:"Fetch timeout per page"`
	URL         string        `arg:"" required:"" help:"Documentation URL to fetch"`
	Name        string        `arg:"" optional:"" help:"Name for the output directory"`
	Path        string        `arg:"" optional:"" default:"." help:"Base path for output (default: current directory)"`
}

// registerFrameworkSelectors registers all framework-specific link selectors with the registry.
func registerFrameworkSelectors(registry *goquery.Registry) {
	registry.Register(locdoc.FrameworkDocusaurus, goquery.NewDocusaurusSelector())
	registry.Register(locdoc.FrameworkMkDocs, goquery.NewMkDocsSelector())
	registry.Register(locdoc.FrameworkSphinx, goquery.NewSphinxSelector())
	registry.Register(locdoc.FrameworkVuePress, goquery.NewVuePressSelector())
	registry.Register(locdoc.FrameworkGitBook, goquery.NewGitBookSelector())
	registry.Register(locdoc.FrameworkNextra, goquery.NewNextraSelector())
}
