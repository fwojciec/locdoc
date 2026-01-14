# docfetch Test Suite Refactor Design

## Problem

The current test suite for `cmd/docfetch/` has two issues:

1. **Narrative structure**: Tests are organized by method names and implementation details rather than behaviors and user stories
2. **Mock explosion**: Testing `FetchCmd` requires mocking 8+ interfaces, a design smell indicating leaky abstractions

Following Steve Freeman's philosophy from GOOS, tests should serve as living documentation that tells the story of what the system does. Heavy mocking requirements suggest the code knows too much about its collaborators' internals.

## Goals

1. **Refactor toward cleaner interfaces** that hide implementation complexity
2. **Transform tests into behavioral specifications** that read as stories
3. **Reduce mock burden** to 3 core interfaces instead of 8+
4. **Full pipeline coverage** with synthetic minimal fixtures

---

## New Architecture: Three Simple Interfaces

The current design exposes too many internals. Instead, we think in terms of **what docfetch does**:

```
URL → [Discover URLs] → [Fetch Each] → [Save Files]
```

Three stages, three interfaces.

**Location**: All domain types and interfaces go in the root `locdoc` package (`page.go`), following Ben Johnson's Standard Package Layout. Mocks go in `mock/page.go`.

```go
// page.go - in root locdoc package

// Page represents a fetched documentation page.
type Page struct {
    URL     string
    Title   string
    Content string // Markdown
}

// FetchProgress reports progress during page fetching.
type FetchProgress struct {
    URL       string
    Completed int
    Total     int
    Error     error
}

// FetchProgressFunc is called as pages are processed.
type FetchProgressFunc func(FetchProgress)

// URLSource discovers documentation URLs from a site.
// Implementations hide the complexity of sitemap vs recursive discovery.
type URLSource interface {
    Discover(ctx context.Context, sourceURL string) ([]string, error)
}

// PageFetcher retrieves and converts documentation pages.
// Implementations hide HTTP vs browser selection, retry logic,
// content extraction, and markdown conversion.
type PageFetcher interface {
    FetchAll(ctx context.Context, urls []string, progress FetchProgressFunc) ([]*Page, error)
}

// PageStore persists pages to storage with atomic semantics.
// Save writes to a temporary location; Commit makes changes permanent;
// Abort discards pending changes.
type PageStore interface {
    Save(ctx context.Context, page *Page) error
    Commit() error
    Abort() error
}
```

### Benefits

1. **3 mocks instead of 8+** - tests become trivial to set up
2. **Clear responsibilities** - each interface does one thing
3. **Hidden complexity** - rate limiting, probing, retry are implementation details
4. **Atomic semantics built-in** - PageStore handles temp directory lifecycle

### Key Design Decision: Move Probing Up

The HTTP-vs-Rod decision happens **before** creating ConcurrentFetcher, not inside it:

```go
// In main.go - wiring layer
func (m *Main) Run(ctx context.Context, args []string, ...) error {
    // Create both fetchers
    rodFetcher, _ := rod.NewFetcher(...)
    httpFetcher := http.NewFetcher(...)

    // Probe ONCE to decide which to use
    fetcher := probeFetcher(ctx, sourceURL, httpFetcher, rodFetcher, prober)

    // ConcurrentFetcher only needs ONE fetcher - decision already made
    pageFetcher := NewConcurrentFetcher(fetcher, extractor, converter)

    cmd := &FetchCmd{Source: source, Fetcher: pageFetcher, Store: store}
}
```

This means ConcurrentFetcher is fetcher-agnostic:

```go
// Simple: just one fetcher, pre-selected
type ConcurrentFetcher struct {
    fetcher   locdoc.Fetcher    // HTTP or Rod - doesn't care which
    extractor locdoc.Extractor
    converter locdoc.Converter
}
```

**Testing benefit**: ConcurrentFetcher tests mock a single `Fetcher` interface. No Rod, no HTTP client needed. The probing logic becomes a separate, independently testable function.

### Test Comparison

**Before (8+ mocks):**
```go
deps := &main.Dependencies{
    Sitemaps: mockSitemap,
    Discoverer: &crawl.Discoverer{
        HTTPFetcher:   mockHTTP,
        RodFetcher:    mockRod,
        Prober:        mockProber,
        Extractor:     mockExtractor,
        LinkSelectors: mockRegistry,
        RateLimiter:   mockLimiter,
    },
    Crawler: crawler,
}
```

**After (3 mocks):**
```go
cmd := &FetchCmd{
    Source:  mockSource,   // returns URLs
    Fetcher: mockFetcher,  // returns Pages
    Store:   mockStore,    // records saves
}
```

---

## Testing Layers

The refactor creates clear testing layers, each with appropriate mock granularity:

### Layer 1: Orchestration (FetchCmd)
```go
func TestFetch_SavesAllPages(t *testing.T) {
    // 3 simple mocks - no browser, no HTTP, no extraction
    cmd := &FetchCmd{
        Source:  &mockSource{urls: []string{"a", "b"}},
        Fetcher: &mockFetcher{pages: []*Page{{URL: "a"}, {URL: "b"}}},
        Store:   &mockStore{},
    }
    // ...
}
```

### Layer 2: Pipeline (ConcurrentFetcher)
```go
func TestConcurrentFetcher_ProcessesPages(t *testing.T) {
    // Mock at natural boundaries - still no real browser
    fetcher := &mockFetcher{html: "<html>...</html>"}
    extractor := &mockExtractor{...}
    converter := &mockConverter{...}

    cf := NewConcurrentFetcher(fetcher, extractor, converter)
    // ...
}
```

### Layer 3: Decision Logic (Probing)
```go
func TestProbeFetcher_ChoosesRodForJSSites(t *testing.T) {
    // Tests the probe logic without real Chrome
    httpFetcher := &mockFetcher{html: "<html>Loading...</html>"}
    rodFetcher := &mockFetcher{html: "<html><article>Full content</article></html>"}

    chosen := probeFetcher(ctx, url, httpFetcher, rodFetcher, prober)
    assert.Same(t, rodFetcher, chosen)
}
```

### Layer 4: Integration (Real Dependencies)
```go
//go:build integration

func TestRealFetch_DocusaurusSite(t *testing.T) {
    // Uses real Rod, real HTTP - runs in CI only
}
```

---

## Current vs Target Structure

### Current Interfaces (Too Granular)

| Role | Interface | Responsibility |
|------|-----------|----------------|
| URL Provider | `SitemapService` | Discovers URLs from sitemaps |
| Recursive Explorer | `Discoverer` | Finds URLs by following links |
| Page Fetcher | `Fetcher` | Retrieves HTML content |
| Content Extractor | `Extractor` | Removes boilerplate |
| Format Converter | `Converter` | HTML to Markdown |
| Document Store | `DocumentWriter` | Persists documents |
| Framework Detector | `Prober` | Identifies frameworks |
| Rate Controller | `DomainLimiter` | Request pacing |
| Link Finder | `LinkSelectorRegistry` | Link extraction |

### Target Interfaces (Coarse-Grained)

| Role | Interface | What It Hides |
|------|-----------|---------------|
| URL Discovery | `URLSource` | Sitemap vs recursive, framework detection |
| Page Processing | `PageFetcher` | HTTP vs browser, retry, extraction, conversion |
| Page Storage | `PageStore` | Filesystem, atomic updates, frontmatter |

---

## Implementation Tasks

### Phase 1: Define Core Types and Interfaces

**Task 1.1: Define domain types and interfaces** ✅
- Add `Page` struct, `FetchProgress` struct, `FetchProgressFunc` type
- Add `URLSource`, `PageFetcher`, `PageStore` interfaces
- Location: `page.go` (root locdoc package)
- Mocks: `mock/page.go`

**Task 1.2: Create CompositeSource implementation**
- Create `CompositeSource` that tries sitemap then falls back to recursive
- Wire up existing `SitemapService` + `crawl.Discoverer` behind URLSource interface
- Location: `cmd/docfetch/` (wiring layer, implements `locdoc.URLSource`)
- Test: `cmd/docfetch/source_test.go`

**Task 1.3: Create ConcurrentFetcher implementation**
- Create `ConcurrentFetcher` that wraps existing crawl logic
- Hide: retry, extraction, conversion (probing happens in main.go)
- Location: `cmd/docfetch/` (wiring layer, implements `locdoc.PageFetcher`)
- Test: `cmd/docfetch/fetcher_test.go`

**Task 1.4: Create FileStore implementation**
- Create `FileStore` that wraps `fs.Writer` + temp directory logic
- Location: `fs/pagestore.go` (depends on fs package, implements `locdoc.PageStore`)
- Test: `fs/pagestore_test.go`

### Phase 2: Refactor FetchCmd

**Task 2.1: Rewrite FetchCmd to use new interfaces**
- Replace `Dependencies` struct with URLSource, PageFetcher, PageStore
- Simplify `Run` to: discover → fetch → save loop
- Move wiring logic to `main.go`
- Location: `cmd/docfetch/fetch.go`
- Test: `cmd/docfetch/fetch_test.go`

**Task 2.2: Update main.go wiring**
- Create concrete implementations
- Wire them together
- Keep CLI parsing as-is
- Location: `cmd/docfetch/main.go`

### Phase 3: Rewrite Tests as Stories

**Task 3.1: CLI tests as stories**
- Rename to behavioral names
- Add Given-When-Then comments
- Location: `cmd/docfetch/cli_test.go` (renamed from `main_test.go`)

**Task 3.2: Preview tests as stories**
- Separate preview tests into own file
- Focus on URLSource behavior
- Location: `cmd/docfetch/preview_test.go`

**Task 3.3: Fetch tests as stories**
- Orchestration and atomic update behavior
- Uses only 3 mocks
- Location: `cmd/docfetch/fetch_test.go`

### Phase 4: Supporting Module Tests (Lower Layers)

**Task 4.1: URLSource implementation tests**
- Test sitemap → recursive fallback logic
- Synthetic minimal HTML fixtures
- Location: `cmd/docfetch/source_test.go`

**Task 4.2: PageFetcher implementation tests**
- Test retry, extraction, conversion pipeline
- Mock at Fetcher/Extractor level (internal to implementation)
- Location: `cmd/docfetch/fetcher_test.go`

**Task 4.3: PageStore implementation tests**
- Test atomic update semantics
- Test path handling, frontmatter
- Location: `cmd/docfetch/store_test.go`

---

## Test Specifications

### 1. CLI Behavior (`cli_test.go`)

Story: **"A user invokes docfetch from the command line"**

```go
// Story: CLI Help and Discovery
// A user should be able to discover how to use the tool

func TestCLI_ShowsHelpWhenAsked(t *testing.T) {
    // Given the user runs docfetch with --help
    // When the command executes
    // Then usage information is displayed
    // And no error occurs
}

func TestCLI_ShowsHelpWhenNoArgumentsProvided(t *testing.T) {
    // Given the user runs docfetch with no arguments
    // When the command executes
    // Then usage information is displayed
    // And an error indicates arguments are required
}

// Story: CLI Validation
// Invalid inputs should be rejected with clear messages

func TestCLI_RequiresURLForAllOperations(t *testing.T) {
    // Given the user runs docfetch --preview without a URL
    // When the command executes
    // Then an error indicates URL is required
}

func TestCLI_RequiresNameForFetchMode(t *testing.T) {
    // Given the user runs docfetch with URL but no name (fetch mode)
    // When the command executes
    // Then an error indicates name is required for fetch mode
}

func TestCLI_AllowsPreviewWithoutName(t *testing.T) {
    // Given the user runs docfetch --preview with a URL
    // When the command executes
    // Then preview proceeds (name not required)
}
```

### 2. Preview Behavior (`preview_test.go`)

Story: **"A user wants to see what would be fetched before committing"**

```go
// Story: Previewing Documentation Sites
// Preview mode shows URLs without downloading content

func TestPreview_ShowsURLsFromSource(t *testing.T) {
    // Given a URLSource that returns URLs
    source := &mockSource{urls: []string{
        "https://example.com/docs/intro",
        "https://example.com/docs/api",
    }}

    // When I preview the site
    cmd := &FetchCmd{Source: source, Preview: true}
    stdout := &bytes.Buffer{}
    err := cmd.Run(ctx, stdout, stderr)

    // Then all URLs are listed to stdout
    require.NoError(t, err)
    assert.Contains(t, stdout.String(), "docs/intro")
    assert.Contains(t, stdout.String(), "docs/api")
}

func TestPreview_ReportsDiscoveryErrors(t *testing.T) {
    // Given a URLSource that fails
    source := &mockSource{err: errors.New("connection refused")}

    // When I preview the site
    cmd := &FetchCmd{Source: source, Preview: true}
    err := cmd.Run(ctx, stdout, stderr)

    // Then the error is returned
    assert.Error(t, err)
}
```

### 3. Fetch Behavior (`fetch_test.go`)

Story: **"A user fetches documentation for local use"**

Note: These tests use only **3 mocks** (Source, Fetcher, Store).

```go
// Story: Fetching Documentation
// Fetch mode downloads and converts documentation to markdown

func TestFetch_SavesAllDiscoveredPages(t *testing.T) {
    // Given URLs are discovered
    source := &mockSource{urls: []string{"a", "b", "c"}}
    // And pages are fetched successfully
    fetcher := &mockFetcher{pages: []*Page{
        {URL: "a", Title: "Page A", Content: "# A"},
        {URL: "b", Title: "Page B", Content: "# B"},
        {URL: "c", Title: "Page C", Content: "# C"},
    }}
    // And a store to save them
    store := &mockStore{}

    // When I fetch the documentation
    cmd := &FetchCmd{Source: source, Fetcher: fetcher, Store: store}
    err := cmd.Run(ctx)

    // Then all pages are saved
    require.NoError(t, err)
    assert.Len(t, store.saved, 3)
    // And the store is committed
    assert.True(t, store.committed)
}

func TestFetch_ReportsProgressViaCallback(t *testing.T) {
    // Given a fetcher that reports progress
    var events []ProgressEvent
    fetcher := &mockFetcher{
        pages: []*Page{{URL: "a"}, {URL: "b"}},
        progressFn: func(e ProgressEvent) {
            events = append(events, e)
        },
    }

    // When I fetch
    cmd := &FetchCmd{
        Source:  &mockSource{urls: []string{"a", "b"}},
        Fetcher: fetcher,
        Store:   &mockStore{},
    }
    err := cmd.Run(ctx)

    // Then progress is reported for each page
    require.NoError(t, err)
    assert.Len(t, events, 2)
}

func TestFetch_ContinuesOnPageFailures(t *testing.T) {
    // Given some pages fail
    fetcher := &mockFetcher{
        pages: []*Page{{URL: "a"}, {URL: "c"}}, // b failed
        errors: map[string]error{"b": errors.New("timeout")},
    }

    // When I fetch
    cmd := &FetchCmd{
        Source:  &mockSource{urls: []string{"a", "b", "c"}},
        Fetcher: fetcher,
        Store:   &mockStore{},
    }
    err := cmd.Run(ctx)

    // Then successful pages are still saved
    require.NoError(t, err)
    // And failures are reported via progress (not tested here)
}
```

### 4. Atomic Update Behavior (`fetch_test.go`)

Story: **"Updates should be safe and recoverable"**

```go
// Story: Atomic Updates
// The store handles atomic semantics; FetchCmd just calls Commit/Abort

func TestFetch_CommitsStoreOnSuccess(t *testing.T) {
    // Given a successful fetch
    store := &mockStore{}
    cmd := &FetchCmd{
        Source:  &mockSource{urls: []string{"a"}},
        Fetcher: &mockFetcher{pages: []*Page{{URL: "a"}}},
        Store:   store,
    }

    // When fetch completes
    err := cmd.Run(ctx)

    // Then store is committed
    require.NoError(t, err)
    assert.True(t, store.committed)
    assert.False(t, store.aborted)
}

func TestFetch_AbortsStoreWhenNoPagesSaved(t *testing.T) {
    // Given all pages fail to fetch
    store := &mockStore{}
    cmd := &FetchCmd{
        Source:  &mockSource{urls: []string{"a"}},
        Fetcher: &mockFetcher{err: errors.New("all failed")},
        Store:   store,
    }

    // When fetch completes
    _ = cmd.Run(ctx)

    // Then store is aborted (preserves existing content)
    assert.False(t, store.committed)
    assert.True(t, store.aborted)
}

func TestFetch_AbortsStoreOnDiscoveryFailure(t *testing.T) {
    // Given discovery fails
    store := &mockStore{}
    cmd := &FetchCmd{
        Source:  &mockSource{err: errors.New("no sitemap")},
        Fetcher: &mockFetcher{},
        Store:   store,
    }

    // When fetch fails early
    _ = cmd.Run(ctx)

    // Then store is aborted
    assert.True(t, store.aborted)
}
```

---

## Supporting Module Test Stories

These tests are for the **implementations** of the three core interfaces. They have more internal knowledge but still follow the story format.

### 5. URLSource Implementation (`source_test.go`)

Story: **"URL discovery tries sitemap first, then recursive crawling"**

```go
// Story: Composite URL Discovery
// The source tries multiple strategies to find documentation URLs

func TestCompositeSource_UsesSitemapWhenAvailable(t *testing.T) {
    // Given a sitemap service returns URLs
    sitemap := &mockSitemap{urls: []string{"a", "b", "c"}}
    source := NewCompositeSource(sitemap, nil)

    // When I discover URLs
    urls, err := source.Discover(ctx, "https://example.com")

    // Then sitemap URLs are returned
    require.NoError(t, err)
    assert.Equal(t, []string{"a", "b", "c"}, urls)
}

func TestCompositeSource_FallsBackToRecursiveWhenSitemapEmpty(t *testing.T) {
    // Given sitemap returns no URLs
    sitemap := &mockSitemap{urls: []string{}}
    // And recursive discoverer finds some
    recursive := &mockRecursive{urls: []string{"x", "y"}}
    source := NewCompositeSource(sitemap, recursive)

    // When I discover URLs
    urls, err := source.Discover(ctx, "https://example.com")

    // Then recursive URLs are returned
    require.NoError(t, err)
    assert.Equal(t, []string{"x", "y"}, urls)
}

func TestCompositeSource_ReturnsEmptyWhenBothFail(t *testing.T) {
    // Given both discovery methods find nothing
    source := NewCompositeSource(
        &mockSitemap{urls: []string{}},
        &mockRecursive{urls: []string{}},
    )

    // When I discover URLs
    urls, err := source.Discover(ctx, "https://example.com")

    // Then empty list is returned (not an error)
    require.NoError(t, err)
    assert.Empty(t, urls)
}
```

### 6. PageFetcher Implementation (`fetcher_test.go`)

Story: **"Pages are fetched, extracted, and converted to markdown"**

```go
// Story: Page Processing Pipeline
// The fetcher handles the full fetch → extract → convert pipeline

func TestConcurrentFetcher_ProcessesAllURLs(t *testing.T) {
    // Given an HTML fetcher and extractor
    httpFetcher := &mockHTTPFetcher{html: "<html><body>Hello</body></html>"}
    extractor := &mockExtractor{result: &ExtractResult{Title: "Hi", ContentHTML: "<p>Hello</p>"}}
    converter := &mockConverter{markdown: "Hello"}

    fetcher := NewConcurrentFetcher(httpFetcher, extractor, converter)

    // When I fetch multiple URLs
    pages, err := fetcher.FetchAll(ctx, []string{"a", "b"}, nil)

    // Then all pages are processed
    require.NoError(t, err)
    assert.Len(t, pages, 2)
    assert.Equal(t, "Hello", pages[0].Content)
}

func TestConcurrentFetcher_ReportsProgressPerPage(t *testing.T) {
    // Given a fetcher with progress callback
    var events []ProgressEvent
    fetcher := NewConcurrentFetcher(/* ... */)

    // When I fetch with progress
    _, _ = fetcher.FetchAll(ctx, []string{"a", "b", "c"}, func(e ProgressEvent) {
        events = append(events, e)
    })

    // Then progress is reported for each URL
    assert.Len(t, events, 3)
}

func TestConcurrentFetcher_ContinuesOnIndividualFailures(t *testing.T) {
    // Given one URL will fail
    httpFetcher := &mockHTTPFetcher{
        responses: map[string]string{"a": "<html>A</html>", "c": "<html>C</html>"},
        errors:    map[string]error{"b": errors.New("timeout")},
    }

    // When I fetch all URLs
    pages, err := NewConcurrentFetcher(httpFetcher, extractor, converter).
        FetchAll(ctx, []string{"a", "b", "c"}, nil)

    // Then successful pages are returned
    require.NoError(t, err)
    assert.Len(t, pages, 2) // a and c
}
```

### 7. PageStore Implementation (`store_test.go`)

Story: **"Pages are saved atomically with proper file structure"**

```go
// Story: Atomic File Storage
// The store uses temp directory for atomic updates

func TestFileStore_SavesPageAsMarkdown(t *testing.T) {
    // Given a store targeting a temp directory
    store := NewFileStore(t.TempDir(), "output")

    // When I save a page
    err := store.Save(ctx, &Page{
        URL:     "https://example.com/docs/api",
        Title:   "API Reference",
        Content: "# API\n\nWelcome to the API.",
    })

    // Then no error (file in temp dir)
    require.NoError(t, err)
}

func TestFileStore_CommitMovesFromTempToFinal(t *testing.T) {
    // Given a store with saved pages
    base := t.TempDir()
    store := NewFileStore(base, "output")
    _ = store.Save(ctx, &Page{URL: "https://x.com/a", Title: "A", Content: "# A"})

    // When I commit
    err := store.Commit()

    // Then final directory exists with content
    require.NoError(t, err)
    _, err = os.Stat(filepath.Join(base, "output", "a.md"))
    require.NoError(t, err)
    // And temp directory is gone
    _, err = os.Stat(filepath.Join(base, "output.tmp"))
    assert.True(t, os.IsNotExist(err))
}

func TestFileStore_AbortCleansUpTempDirectory(t *testing.T) {
    // Given a store with saved pages
    base := t.TempDir()
    store := NewFileStore(base, "output")
    _ = store.Save(ctx, &Page{URL: "https://x.com/a", Title: "A", Content: "# A"})

    // When I abort
    err := store.Abort()

    // Then temp directory is cleaned up
    require.NoError(t, err)
    _, err = os.Stat(filepath.Join(base, "output.tmp"))
    assert.True(t, os.IsNotExist(err))
    // And final directory doesn't exist
    _, err = os.Stat(filepath.Join(base, "output"))
    assert.True(t, os.IsNotExist(err))
}

func TestFileStore_IncludesFrontmatter(t *testing.T) {
    // Given a page with metadata
    base := t.TempDir()
    store := NewFileStore(base, "output")
    _ = store.Save(ctx, &Page{
        URL:     "https://example.com/intro",
        Title:   "Introduction",
        Content: "# Welcome",
    })
    _ = store.Commit()

    // When I read the file
    content, _ := os.ReadFile(filepath.Join(base, "output", "intro.md"))

    // Then it has YAML frontmatter
    assert.Contains(t, string(content), "---")
    assert.Contains(t, string(content), "source: https://example.com/intro")
    assert.Contains(t, string(content), "title: Introduction")
}

func TestFileStore_PreservesURLPathStructure(t *testing.T) {
    // Given pages with nested paths
    base := t.TempDir()
    store := NewFileStore(base, "output")
    _ = store.Save(ctx, &Page{URL: "https://x.com/docs/api/users"})
    _ = store.Commit()

    // Then nested directories are created
    _, err := os.Stat(filepath.Join(base, "output", "docs", "api", "users.md"))
    require.NoError(t, err)
}
```

---

## Target File Structure

After refactoring (following Ben Johnson's Standard Package Layout):

```
locdoc/
├── page.go              # Domain types: Page, FetchProgress, FetchProgressFunc
│                        # Interfaces: URLSource, PageFetcher, PageStore
├── mock/
│   └── page.go          # Mocks for URLSource, PageFetcher, PageStore
├── fs/
│   ├── pagestore.go     # FileStore implementation (locdoc.PageStore)
│   └── pagestore_test.go
└── cmd/docfetch/
    ├── main.go          # Entry point, wiring, probing logic
    ├── cli.go           # CLI parsing (FetchCmd struct)
    ├── fetch.go         # FetchCmd.Run orchestration
    ├── source.go        # CompositeSource (locdoc.URLSource)
    ├── fetcher.go       # ConcurrentFetcher (locdoc.PageFetcher)
    ├── source_test.go   # Story: URL discovery strategies
    ├── fetcher_test.go  # Story: Page processing pipeline
    ├── fetch_test.go    # Story: Fetch orchestration (3 mocks)
    ├── preview_test.go  # Story: Preview behavior
    └── cli_test.go      # Story: CLI validation
```

**Architecture notes:**
- Domain types and interfaces in root (`page.go`) - can be imported anywhere
- FileStore in `fs/` - depends on filesystem operations
- CompositeSource/ConcurrentFetcher in `cmd/docfetch/` - docfetch-specific wiring
- Tests use `mock.URLSource`, `mock.PageFetcher`, `mock.PageStore`

Dependencies used by implementations:
- `locdoc.SitemapService` → inside CompositeSource
- `crawl.Discoverer` → inside CompositeSource
- `locdoc.Fetcher` → inside ConcurrentFetcher
- `locdoc.Extractor` → inside ConcurrentFetcher
- `locdoc.Converter` → inside ConcurrentFetcher
- `fs.Writer` → inside FileStore

---

## Summary

This refactor accomplishes:

1. **Cleaner API**: FetchCmd depends on 3 interfaces instead of 8+
2. **Better testability**: Simple mocks, clear responsibilities
3. **Story-driven tests**: Tests read as specifications
4. **Preserved functionality**: All existing behavior is maintained
5. **Preparation for independence**: docfetch can eventually stand alone
