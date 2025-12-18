# Modern Web Crawling Techniques for Documentation URL Discovery

When sitemap.xml is unavailable or incomplete, documentation crawlers need robust fallback strategies combining **BFS traversal with navigation-aware link extraction**, **Bloom filter deduplication**, and **framework-specific DOM selectors**. For a Go CLI tool targeting <10k pages, the optimal architecture uses Colly as the HTTP layer with rod for JavaScript rendering, implements per-domain rate limiting via `golang.org/x/time/rate`, and stores crawl state in BoltDB for incremental crawling.

## Recursive link crawling: Mercator architecture and traversal strategies

The **Mercator crawler architecture** (Heydon & Najork, 1999) remains the foundational blueprint for URL frontier management. It uses a dual-queue system: **front queues** implement priority-based URL scheduling, while **back queues** enforce per-host politeness by maintaining separate queues for each domain. The key invariant is that each back queue contains URLs from only one host, with a recommended ratio of **3× more back queues than crawler threads**.

For documentation sites, **BFS (breadth-first search)** outperforms DFS because it discovers high-value pages early—research by Najork & Wiener found that BFS captures pages with high PageRank first since important pages have many inbound links discovered at shallow depths. However, pure BFS misses navigation structure. The recommended hybrid approach:

1. Parse sitemap.xml first for authoritative URL list with priorities
2. Apply BFS within documentation hierarchy
3. Boost priority for URLs found in navigation elements (sidebar, TOC)
4. Deprioritize inline content links and footer links

**OPIC (Online Page Importance Computation)** by Abiteboul et al. offers an alternative to computing PageRank-like scores during crawling without requiring the full link graph. It maintains "cash" and "history" values per page, distributing cash to linked pages when visited, converging to importance scores useful for prioritization.

### Data structures for the frontier

For <10k URLs, in-memory structures suffice. Go's `container/heap` implements priority queues efficiently. A practical frontier combines:

```go
type Frontier struct {
    queue    *PriorityQueue       // container/heap implementation
    seen     *bloom.BloomFilter   // github.com/bits-and-blooms/bloom/v3
    perHost  map[string]*rate.Limiter
}
```

**Bloom filters** provide memory-efficient deduplication with configurable false-positive rates. For 10,000 URLs at 1% FPR, you need only **~12KB** (95,851 bits with 7 hash functions). The formula for false positive rate is `FPR ≈ (1 - e^(-kn/m))^k`. The bits-and-blooms/bloom library handles sizing automatically:

```go
filter := bloom.NewWithEstimates(10000, 0.01) // 10K URLs, 1% FPR
filter.Add([]byte(normalizedURL))
if filter.Test([]byte(url)) { /* likely seen */ }
```

**Cuckoo filters** offer better space efficiency when FPR < 3% and support deletion, useful if you need to remove URLs from the seen set. The trade-off is slightly slower lookups (2 cache line accesses vs 1 for Bloom).

## URL normalization and deduplication strategies

URL normalization prevents crawling semantically identical pages. **RFC 3986-compliant normalization** includes:

- Lowercase scheme and host: `HTTP://EXAMPLE.COM` → `http://example.com`
- Remove dot segments: `/foo/./bar/../baz` → `/foo/baz`
- Decode unreserved percent-encoding: `%7E` → `~`
- Remove fragments: `#section1` stripped (never sent to server)
- Sort query parameters: `?b=2&a=1` → `?a=1&b=2`

The Go library `github.com/PuerkitoBio/purell` handles RFC 3986 normalization. For canonical URL detection, check in order of strength: **301/302 redirects** (strongest), **rel="canonical" link element**, and sitemap inclusion (weak signal).

For near-duplicate detection, **SimHash** (used by Google's crawler) generates 64-bit fingerprints where similar documents have similar fingerprints. Documents with Hamming distance ≤ 3 bits are near-duplicates. The library `github.com/mfonda/simhash` provides Go implementation:

```go
oldHash := simhash.Simhash(simhash.NewWordFeatureSet([]byte(oldContent)))
newHash := simhash.Simhash(simhash.NewWordFeatureSet([]byte(newContent)))
distance := simhash.Compare(oldHash, newHash) // re-index if > 3
```

## Scope limiting and boundary detection

Effective scope limiting combines URL patterns, domain rules, and content heuristics. Common documentation URL patterns:

| Pattern Type | Examples |
|-------------|----------|
| Path-based | `/docs/*`, `/documentation/*`, `/api/*`, `/reference/*` |
| Subdomain | `docs.example.com`, `developer.example.com` |
| Versioned | `/docs/v1/*`, `/docs/latest/*`, `/docs/2.0/*` |
| Localized | `/en/docs/*`, `/docs/en-us/*` |

Colly's URL filtering provides the cleanest Go implementation:

```go
c := colly.NewCollector(
    colly.URLFilters(regexp.MustCompile(`https://example\.com/docs(/.*)?$`)),
    colly.AllowedDomains("docs.example.com", "example.com"),
    colly.DisallowedURLFilters(regexp.MustCompile(`/blog/|/pricing/|/about/`)),
)
```

**Boundary detection heuristics** classify links leaving documentation: blog indicators (`/blog/`, `/news/`), marketing pages (`/pricing/`, `/features/`), and external domains. Position-based link classification assigns higher priority to links in `<nav>`, `<aside>`, and elements with `role="navigation"` versus footer or body links.

For robots.txt parsing, `github.com/jimsmart/grobotstxt` is a direct port of Google's official C++ parser. The library `github.com/temoto/robotstxt` is the most popular alternative with Crawl-delay support:

```go
robots, _ := robotstxt.FromString(content)
group := robots.FindGroup("MyBot")
allowed := group.Test("/docs/page")
crawlDelay := group.CrawlDelay
```

## Navigation-aware extraction for documentation frameworks

Rather than extracting all links, targeting navigation elements yields higher-quality URL discovery. Universal selectors that work across frameworks:

```css
nav, aside, [role="navigation"], [role="complementary"]
.sidebar, .toc, .menu, .docs-nav, .docs-sidebar
.table-of-contents, .on-this-page
```

### Framework-specific DOM patterns

**Docusaurus** (React-based): `.theme-doc-sidebar-container`, `.menu__list`, `.menu__link`, `nav.menu`

**MkDocs Material**: `nav.md-nav`, `.md-nav__link`, `[data-md-component="navigation"]`, `.md-sidebar--primary`

**Sphinx RTD Theme**: `.wy-nav-side`, `.wy-menu-vertical`, `.toctree-l1`, `.toctree-l2`, `a.reference.internal`

**VuePress/VitePress**: `.VPSidebar`, `#VPSidebarNav`, `.sidebar-item`, `.sidebar-link`

**GitBook**: Modern GitBook uses React with hashed classes; prefer `[data-testid="page.desktopTableOfContents"]`, `[data-testid="space.sidebar"]`

**Nextra**: `nav.nextra-sidebar`, `.nextra-sidebar-container`, `.nextra-toc`

### Link prioritization algorithm

Score links by DOM position and context:

```javascript
const priorities = {
    'nav.sidebar a, aside a, .sidebar a': 100,
    '.toc a, .table-of-contents a': 90,
    '[role="navigation"] a': 85,
    'main a, article a, .content a': 50,
    'footer a': 20
};
// Boost for nav-specific classes, penalize external links
```

## Detecting JavaScript hydration completion

Modern documentation frameworks (Docusaurus, VuePress, Nextra) use client-side hydration. The naive approach of waiting for `DOMContentLoaded` fails—you need framework-specific detection or generic DOM stability monitoring.

**Generic MutationObserver approach** (works across frameworks):

```javascript
function waitForDOMStable(stabilityThreshold = 500) {
    return new Promise(resolve => {
        let lastMutationTime = Date.now();
        const observer = new MutationObserver(() => {
            lastMutationTime = Date.now();
        });
        observer.observe(document.body, { childList: true, subtree: true });
        const check = setInterval(() => {
            if (Date.now() - lastMutationTime > stabilityThreshold) {
                clearInterval(check);
                observer.disconnect();
                resolve();
            }
        }, 100);
    });
}
```

**Rod wait patterns for Go**:

```go
// Combined strategy for documentation pages
func waitForDocsPage(page *rod.Page) error {
    page.MustWaitLoad()
    wait := page.MustWaitRequestIdle()
    wait()
    
    // Wait for navigation element to appear
    page.Timeout(5*time.Second).MustElement(
        "nav, aside, .sidebar, [role='navigation'], .menu",
    ).MustWaitVisible()
    
    page.MustElement("body").MustWaitStable()
    return nil
}
```

For **React hydration detection**, check `document.getElementById('root')._reactRootContainer` or `document.getElementById('__next')`. For **Vue**, the `data-server-rendered` attribute is removed after hydration.

## Politeness: rate limiting and exponential backoff

Documentation sites typically tolerate **1-5 second delays** between requests. The standard formula for exponential backoff with jitter:

```
delay = baseDelay × 2^attempt × (1 + jitter × random())
```

With `baseDelay = 1s`, `maxDelay = 30s`, and `jitter = 0.5`, this produces delays of approximately 1s, 2s, 4s, 8s, 16s, capped at 30s.

**Token bucket rate limiting** via `golang.org/x/time/rate`:

```go
limiter := rate.NewLimiter(rate.Every(time.Second), 3) // 1/sec, burst 3
if err := limiter.Wait(ctx); err != nil { return err }
```

For per-domain limiting, maintain a map of limiters:

```go
type DomainLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.Mutex
}

func (d *DomainLimiter) Wait(ctx context.Context, domain string) error {
    d.mu.Lock()
    limiter, ok := d.limiters[domain]
    if !ok {
        limiter = rate.NewLimiter(rate.Every(2*time.Second), 2)
        d.limiters[domain] = limiter
    }
    d.mu.Unlock()
    return limiter.Wait(ctx)
}
```

Handle **429 responses** by respecting `Retry-After` headers (seconds or HTTP-date) and `X-RateLimit-*` headers. Circuit breakers (`github.com/sony/gobreaker`) prevent cascading failures after **5+ consecutive errors**.

| Setting | Recommended Value |
|---------|------------------|
| Default delay | 1-2 seconds |
| Per-domain concurrency | 2-5 requests |
| Total concurrency | 10-50 requests |
| Backoff max | 30 seconds |
| Circuit breaker threshold | 5 failures |

## Incremental crawling with HTTP caching and change detection

For delta crawls, leverage HTTP conditional requests:

```go
func fetchWithCache(url string, cached *CachedResponse) (*http.Response, error) {
    req, _ := http.NewRequest("GET", url, nil)
    if cached.ETag != "" {
        req.Header.Set("If-None-Match", cached.ETag)
    }
    if cached.LastModified != "" {
        req.Header.Set("If-Modified-Since", cached.LastModified)
    }
    resp, _ := http.DefaultClient.Do(req)
    if resp.StatusCode == 304 {
        return nil, nil // Content unchanged
    }
    // Store new ETag/Last-Modified for next crawl
    return resp, nil
}
```

**Sitemap `<lastmod>`** is the most reliable field—Google and Bing actively use it. Compare stored timestamps to identify changed pages:

```go
func findChangedURLs(old, new []SitemapEntry) []string {
    oldMap := make(map[string]time.Time)
    for _, e := range old { oldMap[e.Loc] = e.LastMod }
    var changed []string
    for _, e := range new {
        if t, ok := oldMap[e.Loc]; !ok || e.LastMod.After(t) {
            changed = append(changed, e.Loc)
        }
    }
    return changed
}
```

Note: `<changefreq>` and `<priority>` are **largely ignored** by search engines due to widespread misconfiguration.

For content fingerprinting, **xxHash** (`github.com/cespare/xxhash/v2`) provides fast hashing for change detection. Store hashes in BoltDB (`go.etcd.io/bbolt`) for persistence between runs:

```go
type PageRecord struct {
    URL          string    `json:"url"`
    ContentHash  uint64    `json:"content_hash"`
    ETag         string    `json:"etag"`
    LastModified string    `json:"last_modified"`
    LastCrawled  time.Time `json:"last_crawled"`
}
```

## Open-source implementations and Go ecosystem

### Primary Go crawlers to study

**Colly** (`github.com/gocolly/colly`) is the most mature Go crawler with **23k+ stars**. It provides event-driven callbacks (`OnHTML`, `OnRequest`), built-in visited tracking, rate limiting via `LimitRule`, and goquery integration. Key pattern:

```go
c := colly.NewCollector(colly.Async(true))
c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2, Delay: 2*time.Second})
c.OnHTML("a[href]", func(e *colly.HTMLElement) {
    link := e.Request.AbsoluteURL(e.Attr("href"))
    e.Request.Visit(link)
})
```

**Katana** (`github.com/projectdiscovery/katana`) offers hybrid headless/standard crawling using go-rod, with BFS/DFS strategies, automatic form filling, and excellent scope management.

### Essential Go libraries

| Purpose | Library | Import Path |
|---------|---------|-------------|
| HTML parsing | goquery | `github.com/PuerkitoBio/goquery` |
| Bloom filter | bits-and-blooms | `github.com/bits-and-blooms/bloom/v3` |
| Rate limiting | x/time/rate | `golang.org/x/time/rate` |
| robots.txt | grobotstxt | `github.com/jimsmart/grobotstxt` |
| URL normalization | purell | `github.com/PuerkitoBio/purell` |
| Embedded DB | bbolt | `go.etcd.io/bbolt` |
| Browser automation | rod | `github.com/go-rod/rod` |
| Fast hashing | xxhash | `github.com/cespare/xxhash/v2` |
| SimHash | simhash | `github.com/mfonda/simhash` |
| Circuit breaker | gobreaker | `github.com/sony/gobreaker` |

### Cross-language references

**Scrapy** (Python) has excellent scheduler and dupefilter architecture—study `scrapy/core/scheduler.py` and `scrapy/dupefilters.py` for RFPDupeFilter implementation. **Crawlee** (Node.js, `github.com/apify/crawlee`) offers the best request queue design with persistent storage and batch operations. **Trafilatura** (`github.com/adbar/trafilatura`) achieves **0.958 F1 score** in content extraction benchmarks; a Go port exists at `github.com/markusmobius/go-trafilatura`.

## Conclusion

Building robust URL discovery for documentation sites without sitemaps requires combining multiple techniques. Start with **Colly for HTTP crawling** and **rod for JavaScript rendering**, implement **BFS with navigation-aware prioritization**, use **Bloom filters for deduplication** at ~10 bits per URL, and target **framework-specific selectors** for navigation extraction. The key insight is that documentation sites have hierarchical structure that navigation-aware crawling exploits—prioritizing sidebar and TOC links over inline content links dramatically improves discovery efficiency. For incremental crawling, **conditional HTTP requests** with ETag/Last-Modified and **sitemap lastmod tracking** minimize redundant work, while **SimHash** (Hamming distance > 3 bits) detects meaningful content changes worth re-indexing.
