# CLI Refactor Design

## Problem

The current `cmd/locdoc/main.go` is 850+ lines mixing CLI argument parsing, business logic orchestration, and output formatting. This results in:

- **1800+ lines of tests** in `cmd/locdoc/`, many testing argument parsing edge cases
- **Hand-rolled argument parsing** that's error-prone and verbose
- **Business logic coupled to CLI** making it hard to test crawling in isolation
- **Difficult maintenance** as new features (adaptive rendering detection) are added

## Goals

1. **Reduce test maintenance burden** - Eliminate argument parsing tests entirely
2. **Improve code organization** - Separate CLI concerns from business logic
3. **Enable crawl logic evolution** - Isolated, testable crawl package for future features
4. **Simplify cmd/locdoc/** - Thin command wrappers, clear structure

## Design

### Package Structure

```
locdoc/
├── locdoc.go                    # Domain types (unchanged)
├── crawl/                       # NEW: crawl orchestration
│   ├── crawl.go                 # Crawler struct + CrawlProject method
│   ├── crawl_test.go            # Business logic tests with mocks
│   ├── retry.go                 # Retry logic (moved from cmd/)
│   └── retry_test.go            # Retry tests (moved from cmd/)
├── sqlite/                      # (unchanged)
├── http/                        # (unchanged)
├── ...other deps...
└── cmd/locdoc/
    ├── main.go                  # Main struct, CLI struct, Dependencies, wiring
    ├── add.go                   # AddCmd struct + Run method
    ├── list.go                  # ListCmd struct + Run method
    ├── delete.go                # DeleteCmd struct + Run method
    ├── docs.go                  # DocsCmd struct + Run method
    ├── ask.go                   # AskCmd struct + Run method
    ├── main_test.go             # Top-level integration tests
    ├── add_test.go              # Add command integration tests
    └── ...
```

### Kong CLI Library

Replace hand-rolled parsing with Kong (zero external dependencies, struct-tag based):

```go
// cmd/locdoc/main.go
type CLI struct {
    Add    AddCmd    `cmd:"" help:"Add and crawl a documentation project"`
    List   ListCmd   `cmd:"" help:"List all registered projects"`
    Delete DeleteCmd `cmd:"" help:"Delete a project and its documents"`
    Docs   DocsCmd   `cmd:"" help:"List documents for a project"`
    Ask    AskCmd    `cmd:"" help:"Ask a question about project documentation"`
}

// cmd/locdoc/add.go
type AddCmd struct {
    Name        string   `arg:"" help:"Project name"`
    URL         string   `arg:"" help:"Documentation URL"`
    Preview     bool     `short:"p" help:"Show URLs without creating project"`
    Force       bool     `short:"f" help:"Delete existing project first"`
    Filter      []string `short:"F" help:"Filter URLs by regex (repeatable)"`
    Concurrency int      `short:"c" default:"10" help:"Concurrent fetch limit"`
}

func (a *AddCmd) Run(deps *Dependencies) error {
    // Thin wrapper - delegates to services/crawler
}
```

### Crawl Package API

```go
// crawl/crawl.go
package crawl

type Crawler struct {
    Sitemaps     locdoc.SitemapService
    Fetcher      locdoc.Fetcher
    Extractor    locdoc.Extractor
    Converter    locdoc.Converter
    Documents    locdoc.DocumentService
    TokenCounter locdoc.TokenCounter
    Concurrency  int
    RetryDelays  []time.Duration
}

type Result struct {
    Saved  int
    Failed int
    Bytes  int
    Tokens int
}

type ProgressEvent struct {
    Type      ProgressType
    Completed int
    Total     int
    URL       string
    Error     error
}

type ProgressType int

const (
    ProgressStarted ProgressType = iota
    ProgressCompleted
    ProgressFailed
    ProgressFinished
)

type ProgressFunc func(event ProgressEvent)

func (c *Crawler) CrawlProject(ctx context.Context, project *locdoc.Project, progress ProgressFunc) (*Result, error) {
    // Orchestrates: sitemap discovery -> concurrent fetch -> extract -> convert -> save
}
```

### Dependency Injection

Single Dependencies struct passed to all command Run methods:

```go
// cmd/locdoc/main.go
type Dependencies struct {
    Ctx          context.Context
    Stdout       io.Writer
    Stderr       io.Writer
    DB           *sqlite.DB
    Projects     locdoc.ProjectService
    Documents    locdoc.DocumentService
    Sitemaps     locdoc.SitemapService
    Crawler      *crawl.Crawler
    Asker        locdoc.Asker
}
```

### Progress Reporting

Callback-based, decoupled from stdout/stderr:

```go
// In cmd/locdoc/add.go
result, err := deps.Crawler.CrawlProject(ctx, project, func(e crawl.ProgressEvent) {
    switch e.Type {
    case crawl.ProgressCompleted:
        fmt.Fprintf(deps.Stdout, "\r[%d/%d] %s", e.Completed, e.Total, truncateURL(e.URL))
    case crawl.ProgressFailed:
        fmt.Fprintf(deps.Stderr, "  skip %s: %v\n", e.URL, e.Error)
    case crawl.ProgressFinished:
        fmt.Fprintf(deps.Stdout, "\r%-80s\r", "") // Clear progress line
    }
})
```

## Migration Strategy

### What Moves Where

| Current Location | New Location | Notes |
|------------------|--------------|-------|
| `ParseAddArgs()` | Deleted | Kong handles parsing |
| `CmdAdd/List/Delete/Docs/Ask` | Individual command files | Thin Run methods |
| `crawlProject()` | `crawl/crawl.go` | Core business logic |
| `processURL()` | `crawl/crawl.go` | Part of crawl orchestration |
| `FetchWithRetry()` | `crawl/retry.go` | Used by crawler |
| `CrawlDeps` struct | `crawl.Crawler` | Proper type with methods |
| Progress reporting | Callback in cmd/ | Decoupled from crawler |

### Test Impact

| Category | Before | After |
|----------|--------|-------|
| Arg parsing tests | ~50 cases | 0 (Kong's responsibility) |
| Crawl logic tests | Mixed with CLI | Isolated in `crawl/crawl_test.go` |
| Retry tests | `cmd/locdoc/retry_test.go` | `crawl/retry_test.go` |
| Integration tests | ~1800 lines | ~200-300 lines |
| Total test files | 2 | 8 (but much smaller each) |

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI library | Kong | Zero deps, testable without os.Args, struct tags |
| Crawl location | New `crawl/` package | Isolate evolving logic, enable unit testing |
| Crawl API | Struct with method | One crawler per CLI execution, can hold config |
| Progress | Callback function | Simple, flexible, easy to test |
| DI pattern | Single Dependencies struct | Simple, all commands share one type |
| File organization | One file per command | Better test organization, clear boundaries |

## Non-Goals

- No `CrawlService` interface in root package (CLI-only app, no need for abstraction)
- No breaking CLI interface changes (same commands, same flags)
- No new features in this refactor (pure restructuring)
