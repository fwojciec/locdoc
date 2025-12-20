# Adaptive Rendering Design

**Date:** 2025-01-20
**Status:** Ready for implementation

## Problem

locdoc currently uses Rod (headless Chrome) for all page fetching. This is slow and resource-intensive. Research shows 70-80% of documentation sites can be fetched with HTTP-only, which completes in milliseconds vs seconds for browser rendering.

## Solution

Probe the first URL of a crawl to determine the rendering strategy for the entire site. Use HTTP-only fetching when possible, fall back to Rod when JavaScript rendering is required.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Detection strategy | Probe-first | One probe, then use chosen fetcher for entire crawl |
| Detection location | Extend `goquery.Detector` | Reuse existing framework detection; avoid concept-named packages |
| Decision ownership | `crawl.Crawler` | Crawler probes on entry to `CrawlProject()`/`DiscoverURLs()` |
| Framework detection | Once per site | Determined during probe, not per-page |
| Probe timing | On crawl entry | Before any content fetching begins |
| Probe failure | Fall back to Rod | "It just works" behavior |
| Manual override | None initially | YAGNI; add flags later if needed |
| Preview mode | Same as full crawl | Consistent behavior, accurate preview |

## Detection Flow

```
First URL
    │
    ▼
HTTP Fetch (10s timeout, retries)
    │
    ▼
Detect Framework
    ├─ Known JS-required (Docsify, GitBook) → Use Rod
    ├─ Known HTTP-only (Sphinx, MkDocs, ...) → Use HTTP
    └─ Unknown → Fetch with Rod, compare content
                  ├─ >50% more content → Use Rod
                  └─ Similar content → Use HTTP
    │
    ▼
HTTP failed? → Fall back to Rod
    │
    ▼
Use chosen fetcher for entire crawl
```

## Components

### 1. Detection Logic (`goquery/detector.go`)

Extend existing `Detector` with rendering requirement knowledge:

```go
func (d *Detector) RequiresJS(framework string) (requires bool, known bool) {
    // Returns (true, true) for Docsify, GitBook
    // Returns (false, true) for Sphinx, MkDocs, Hugo, Docusaurus, VitePress, Nextra, ReadTheDocs
    // Returns (false, false) for unknown - caller should compare content
}
```

**Framework → Rendering Mapping:**

| Framework | Requires JS |
|-----------|-------------|
| Docsify | Yes |
| GitBook | Yes |
| Sphinx | No |
| MkDocs | No |
| Hugo | No |
| Docusaurus | No |
| VitePress | No |
| Nextra | No |
| ReadTheDocs | No |

### 2. HTTP Fetcher (`http/fetcher.go`)

New simple HTTP-based fetcher:

```go
type Fetcher struct {
    client  *http.Client
    timeout time.Duration
}

func NewFetcher(opts ...Option) *Fetcher  // Default 10s timeout

func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error)

func (f *Fetcher) Close() error  // No-op, satisfies interface
```

**Characteristics:**
- 10s timeout (consistent with Rod)
- Uses existing retry infrastructure from `crawl/retry.go`
- Implements `locdoc.Fetcher` interface

### 3. Content Comparison (`crawl/compare.go`)

For unknown frameworks, compare HTTP vs Rod content:

```go
func contentDiffers(httpHTML, rodHTML string, extractor locdoc.Extractor) bool {
    httpContent, err1 := extractor.Extract(httpHTML)
    rodContent, err2 := extractor.Extract(rodHTML)

    if err1 != nil || err2 != nil {
        return true  // Extraction failed, assume JS needed
    }

    httpLen := len(httpContent.Markdown)
    rodLen := len(rodContent.Markdown)

    if rodLen == 0 {
        return false  // Both empty or Rod failed, use HTTP
    }

    // >50% difference indicates JS-dependent content
    ratio := float64(rodLen-httpLen) / float64(rodLen)
    return ratio > 0.5
}
```

### 4. Crawler Changes (`crawl/crawl.go`)

Crawler receives both fetchers, probes and chooses at runtime:

```go
type Crawler struct {
    // Existing fields...

    // Both fetchers injected
    HTTPFetcher locdoc.Fetcher
    RodFetcher  locdoc.Fetcher

    // Detection (existing, reused)
    Detector    *goquery.Detector
    Extractor   locdoc.Extractor
}

func (c *Crawler) CrawlProject(ctx context.Context, ...) error {
    // 1. Fetch first URL with HTTP
    httpHTML, httpErr := c.HTTPFetcher.Fetch(ctx, firstURL)

    // 2. Detect framework
    framework := c.Detector.Detect(httpHTML)
    requires, known := c.Detector.RequiresJS(framework.Name)

    // 3. Choose fetcher
    var fetcher locdoc.Fetcher
    if httpErr != nil {
        fetcher = c.RodFetcher  // HTTP failed
    } else if known {
        if requires {
            fetcher = c.RodFetcher
        } else {
            fetcher = c.HTTPFetcher
        }
    } else {
        // Unknown: compare content
        rodHTML, _ := c.RodFetcher.Fetch(ctx, firstURL)
        if contentDiffers(httpHTML, rodHTML, c.Extractor) {
            fetcher = c.RodFetcher
        } else {
            fetcher = c.HTTPFetcher
        }
    }

    // 4. Continue with chosen fetcher...
}
```

Same logic applies to `DiscoverURLs()` for preview mode.

### 5. Wiring (`cmd/locdoc/main.go`)

```go
// Create both fetchers
httpFetcher := http.NewFetcher(http.WithFetchTimeout(cli.Add.Timeout))
rodFetcher, err := rod.NewFetcher(rod.WithFetchTimeout(cli.Add.Timeout))

// Crawler chooses at runtime
deps.Crawler = &crawl.Crawler{
    HTTPFetcher: httpFetcher,
    RodFetcher:  rodFetcher,
    Detector:    detector,
    Extractor:   extractor,
    // ...
}
```

## Testing Strategy

| Component | Approach |
|-----------|----------|
| HTTP Fetcher | `httptest.Server`, verify timeout behavior |
| `RequiresJS()` | Unit tests for known frameworks, unknown returns `known: false` |
| Content comparison | Mock extractor, test 50% threshold, edge cases |
| Crawler probe | Mock fetchers, test all decision branches |
| Integration | Skip for now (slow, flaky) |

## Files Changed

| File | Change |
|------|--------|
| `goquery/detector.go` | Add `RequiresJS()` method |
| `goquery/detector_test.go` | Tests for `RequiresJS()` |
| `http/fetcher.go` | New HTTP fetcher implementation |
| `http/fetcher_test.go` | HTTP fetcher tests |
| `crawl/compare.go` | New content comparison function |
| `crawl/compare_test.go` | Content comparison tests |
| `crawl/crawl.go` | Add probe logic, accept both fetchers |
| `crawl/crawl_test.go` | Probe decision tests |
| `crawl/discover.go` | Add probe logic for preview mode |
| `crawl/discover_test.go` | Preview mode probe tests |
| `cmd/locdoc/main.go` | Wire both fetchers |
| `mock/fetcher.go` | May need updates if not already flexible |
