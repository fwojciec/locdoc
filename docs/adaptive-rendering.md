# Adaptive Rendering Detection for Go Web Scrapers

**Bottom line:** A Go-based documentation scraper can achieve 70-80% of requests with lightweight HTTP fetching by implementing a tiered detection system. The most reliable pre-fetch signal is detecting the documentation platform itself—**Docsify and modern GitBook require JavaScript**, while Sphinx, MkDocs, Hugo, Docusaurus, VitePress, and Nextra serve complete HTML. For unknown sites, combine empty root container detection, `<noscript>` warning analysis, and DOM element counting to achieve high-confidence decisions before escalating to headless browser rendering.

No existing Go library implements automatic JS detection—you must build custom logic using the patterns below, combining Rod for browser automation with standard HTTP clients.

---

## Detection signals that actually work

The detection strategy splits into three phases: **pre-request platform identification**, **post-fetch HTML analysis**, and **fallback content comparison**.

### Platform-specific detection (highest ROI for documentation scrapers)

Documentation sites follow predictable patterns that enable deterministic decisions without content analysis:

| Platform | Detection Pattern | Rendering Required |
|----------|------------------|-------------------|
| **Docsify** | `window.$docsify` in source, `docsify.min.js`, body contains "Loading..." | JavaScript |
| **GitBook** | `*.gitbook.io` domain, "Powered by GitBook" footer | JavaScript |
| **Sphinx** | `<meta name="generator" content="Sphinx">` | HTTP-only |
| **MkDocs** | `search_index.json`, `.md-content` classes | HTTP-only |
| **Hugo** | `<meta name="generator" content="Hugo">` | HTTP-only |
| **Docusaurus** | `data-docusaurus-root-container`, `/assets/js/main.*.js` | HTTP-only |
| **VitePress** | `.VPContent`, `.VPDoc` classes, `@vitepress` imports | HTTP-only |
| **Nextra** | `/_next/`, `__NEXT_DATA__` JSON block | HTTP-only |
| **ReadTheDocs** | `*.readthedocs.io` domain, `.rst-versions` class | HTTP-only |

**Docsify is the critical outlier**—it renders 100% client-side with an empty shell containing only "Loading..." text. Modern GitBook also requires browser rendering despite its popularity. Critically, modern SSG frameworks like Docusaurus, VitePress, and Nextra **pre-render complete HTML** at build time for SEO, making HTTP-only scraping fully viable despite their React/Vue foundations.

### Post-fetch HTML heuristics

When platform detection fails, analyze the raw HTML response using these signals ranked by reliability:

**Very high reliability signals:**
- **Empty framework containers**: `<div id="root"></div>`, `<div id="__next"></div>`, `<div id="__nuxt"></div>`, `<div id="app"></div>` with whitespace-only content
- **Framework hydration markers**: `<script id="__NEXT_DATA__">` with empty container indicates SSR content should exist; empty `__NEXT_DATA__` or missing props suggests CSR
- **Angular root**: `<app-root></app-root>` empty tag pattern

**High reliability signals:**
- **`<noscript>` warnings**: Text containing "enable JavaScript", "JavaScript required", or "requires JavaScript" (note: Google Tag Manager iframes in noscript are false positives)
- **Minimal text extraction**: Raw HTML text content < **500 characters** combined with multiple script bundle references
- **SSR indicator attributes**: `data-server-rendered="true"` on Vue root elements indicates complete HTML available

**Medium reliability signals:**
- **DOM element count**: Fewer than **50 meaningful elements** (excluding script/style tags) in body suggests shell page
- **Script-heavy structure**: More than 5 bundled JS files (`chunk.*.js`, `bundle.js`) with minimal semantic HTML
- **Content element absence**: No `<p>`, `<h1-h6>`, `<article>`, or `<main>` tags with text content

```go
func needsBrowserRendering(html string) bool {
    // Priority 1: Empty SPA containers
    emptyContainers := []string{
        `<div id="root"></div>`,
        `<div id="app"></div>`,
        `<div id="__next"></div>`,
        `<div id="__nuxt"></div>`,
    }
    for _, container := range emptyContainers {
        if strings.Contains(html, container) {
            return true
        }
    }
    
    // Priority 2: JavaScript requirement warnings
    if strings.Contains(html, "<noscript>") {
        lowerHTML := strings.ToLower(html)
        jsWarnings := []string{"enable javascript", "javascript required", "requires javascript"}
        for _, warning := range jsWarnings {
            if strings.Contains(lowerHTML, warning) {
                return true
            }
        }
    }
    
    // Priority 3: Minimal content check
    doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
    bodyText := strings.TrimSpace(doc.Find("body").Text())
    if len(bodyText) < 500 && strings.Count(html, "<script") > 3 {
        return true
    }
    
    return false
}
```

### Content comparison thresholds (last resort)

When heuristics are inconclusive, compare HTTP response against browser-rendered output. Empirical thresholds from production systems:

- **Size ratio**: If `(rendered_size - raw_size) / raw_size > 2.0` (200% increase), content is JS-dependent
- **Text extraction difference**: If browser-rendered text is **>50% longer** than HTTP-fetched text
- **Link count difference**: If rendered DOM contains **>1.5x more links** than raw HTML

These comparisons are expensive—they require rendering every uncertain page—so reserve them for building per-site profiles rather than per-page decisions.

---

## Site-level vs page-level detection architecture

The industry-standard approach is **"detect per-site, verify periodically"** which balances accuracy with performance.

### Per-site probing strategy

Sample **3-5 representative pages** per domain: homepage, a listing/index page, and a content/detail page. This coverage catches sites with mixed rendering requirements. Store results in a site profile:

```go
type SiteRenderingProfile struct {
    Domain          string
    RequiresJS      bool
    DetectedAt      time.Time
    ConfidenceScore float64  // 0.0-1.0 based on probe consistency
    SampleSize      int
    TTL             time.Duration
}
```

**Recommended TTLs:**
| Decision Type | Cache Duration | Rationale |
|---------------|----------------|-----------|
| "Site requires JS" | 24-48 hours | Rendering approaches rarely change |
| "Site is HTTP-only" | 12-24 hours | More conservative for false negatives |
| Failure state | 30-60 minutes | Allow recovery from transient issues |

### Hybrid verification pattern

The optimal architecture combines cached site decisions with per-page fallback detection:

1. Check site profile cache → if valid and high-confidence, use cached decision
2. If profile indicates HTTP-only → fetch via HTTP, apply content heuristics
3. If heuristics suggest incomplete content → escalate to browser, update site profile
4. Periodically re-probe sites (every 24-48 hours or after N consecutive failures)

For large sites with **mixed content** (e.g., `/blog/` static, `/app/` dynamic), extend profiles with URL pattern matching:

```go
var urlPatternOverrides = map[*regexp.Regexp]bool{
    regexp.MustCompile(`/api/`):    false, // Always HTTP
    regexp.MustCompile(`/static/`): false,
    regexp.MustCompile(`/app/`):    true,  // Always browser
    regexp.MustCompile(`/dashboard/`): true,
}
```

---

## Production tiered fetching architecture

The canonical pattern is **sequential HTTP-first with conditional escalation**—parallel attempts waste resources since HTTP completes in milliseconds while browser rendering takes seconds.

### Architecture diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    ADAPTIVE SCRAPER ARCHITECTURE                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  URL Queue ──▶ Site Profile Cache (24-48hr TTL)                │
│                        │                                        │
│                        ▼                                        │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   DECISION ENGINE                         │  │
│  │  if siteProfile.RequiresJS && siteProfile.IsValid()       │  │
│  │      → Browser Worker Pool                                │  │
│  │  else                                                     │  │
│  │      → HTTP Workers + Content Validator                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                     │                                           │
│         ┌──────────┴──────────┐                                │
│         ▼                     ▼                                │
│  ┌──────────────┐     ┌──────────────────┐                    │
│  │ HTTP WORKERS │     │ BROWSER WORKERS  │                    │
│  │ 10-50 conc.  │     │ Rod BrowserPool  │                    │
│  │ 15-20s timeout│    │ 3-5 instances    │                    │
│  └──────────────┘     │ 30-60s timeout   │                    │
│         │             └──────────────────┘                    │
│         ▼                       ▲                              │
│  ┌──────────────┐              │                              │
│  │   CONTENT    │──Fallback────┘                              │
│  │  VALIDATOR   │                                              │
│  │ Check size,  │                                              │
│  │ JS indicators│                                              │
│  └──────────────┘                                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Timeout recommendations

| Operation | Recommended | Maximum |
|-----------|-------------|---------|
| HTTP connect | 10s | 15s |
| HTTP total response | 15-20s | 30s |
| Browser navigation | 30s | 60s |
| Browser content wait | 5-10s | 15s |
| Total operation | 60s | 90s |

Commercial scraping services (ScrapingBee, Scrapfly) recommend **minimum 60-second total timeouts** to allow retry mechanisms to function properly. For complex JS scenarios, Scrapfly allows up to 25 seconds of rendering wait time.

### Worker pool sizing

HTTP workers are cheap—run **10-50 concurrent** HTTP fetchers with connection pooling. Browser workers are expensive in memory (**200-500MB RAM per instance**)—maintain **3-5 browser instances** in a pool using Rod's built-in `BrowserPool`:

```go
func NewAdaptiveScraper(browserPoolSize int) *AdaptiveScraper {
    return &AdaptiveScraper{
        httpClient: &http.Client{
            Timeout: 20 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
            },
        },
        browserPool: rod.NewBrowserPool(browserPoolSize),
    }
}
```

---

## Go library recommendations

### Rod vs chromedp comparison

**Rod is recommended** for new Go projects requiring headless browser automation:

| Feature | Rod | chromedp |
|---------|-----|----------|
| Performance | Faster (decode-on-demand) | Slower (full JSON decode) |
| Memory | Lower | Higher (fixed buffers) |
| API | Fluent, simple | DSL-like, verbose |
| High-level helpers | `WaitStable`, `WaitRequestIdle` | Fewer built-ins |
| Browser management | Auto-downloads if needed | Uses system browser |
| Thread safety | Built-in | Requires careful handling |
| iframe/shadow DOM | Excellent | Problematic |
| Community | ~6.5k GitHub stars | ~11.5k GitHub stars |
| Zombie processes | Handles cleanup | Can leave orphans |

**Key Rod features for adaptive scraping:**
- `page.MustWaitStable()` — waits for DOM mutations to settle
- `page.MustWaitRequestIdle()` — waits for network activity to complete
- `rod.NewBrowserPool(n)` — built-in concurrent browser management

### Recommended stack

| Use Case | Library |
|----------|---------|
| HTTP scraping | Colly or net/http + goquery |
| Browser automation | Rod |
| HTML parsing | goquery |
| HTTP client | net/http (standard) or resty |

**Critical finding**: No Go library implements automatic JavaScript detection. You must implement custom detection logic using the heuristics described above.

---

## How existing tools solve this problem

### Crawlee's AdaptivePlaywrightCrawler (gold standard)

Apify's Crawlee library provides the **only open-source implementation of true adaptive rendering detection**. Its algorithm:

1. For each URL, probabilistically run both HTTP and browser fetches (controlled by `renderingTypeDetectionRatio`, default 10%)
2. Compare extracted data from both methods using a custom `resultComparator`
3. If results match → use faster HTTP for subsequent requests to that domain
4. If results differ → use browser rendering
5. Results are persisted across sessions for learning

```javascript
const crawler = new AdaptivePlaywrightCrawler({
    renderingTypeDetectionRatio: 0.1,
    resultComparator: (resultA, resultB) => {
        return resultA.push_data_calls === resultB.push_data_calls;
    },
    async requestHandler({ querySelector }) {
        const $content = await querySelector('.content');
        await pushData({ text: $content.text() });
    },
});
```

### Commercial service approaches

**ScrapingBee** defaults to JavaScript rendering (5 credits) and offers `render_js=false` for HTTP-only (1 credit)—no auto-detection, user chooses per request.

**Zyte API** provides the most sophisticated commercial solution with AI-driven automatic optimization that "picks the minimum required toolset, reducing cost and latency."

**Browserless** and **Bright Data** focus on managed browser infrastructure assuming you've already determined JS is needed.

### Open-source library gaps

| Library | JS Detection | JS Rendering |
|---------|--------------|--------------|
| trafilatura (Python) | None | None (recommends Playwright) |
| newspaper3k (Python) | None | None |
| Readability.js | `isProbablyReaderable()` for content quality | Works on pre-rendered DOM |
| Scrapy | None | Via scrapy-playwright middleware |
| Colly (Go) | None | None |

Most tools expect users to know upfront whether JS rendering is needed, or implement a manual fallback chain.

---

## Documentation-site-specific recommendations

### Decision matrix

| Platform | HTTP-Only | Browser Required | Detection Priority |
|----------|-----------|------------------|-------------------|
| Sphinx | ✅ | | Check meta generator tag |
| MkDocs | ✅ | | Check for `search_index.json` |
| Hugo | ✅ | | Check meta generator tag |
| ReadTheDocs | ✅ | | Check domain pattern |
| Docusaurus | ✅ | | Check `data-docusaurus` attributes |
| VitePress | ✅ | | Check `.VPContent` classes |
| Nextra | ✅ | | Check `__NEXT_DATA__` script |
| **Docsify** | | ✅ | Check `window.$docsify` |
| **GitBook** | | ✅ | Check domain/footer |

For a documentation-focused scraper, implement platform detection **first** before falling back to generic heuristics. This provides deterministic, high-confidence decisions for the majority of documentation sites.

```go
func detectDocPlatform(html, url string) (platform string, needsJS bool) {
    // JS-required platforms
    if strings.Contains(html, "docsify") || strings.Contains(html, "window.$docsify") {
        return "docsify", true
    }
    if strings.Contains(url, "gitbook.io") || strings.Contains(html, "Powered by GitBook") {
        return "gitbook", true
    }
    
    // HTTP-only platforms
    if strings.Contains(html, `name="generator" content="Sphinx`) {
        return "sphinx", false
    }
    if strings.Contains(html, "search_index.json") || strings.Contains(html, "md-content") {
        return "mkdocs", false
    }
    if strings.Contains(html, `name="generator" content="Hugo`) {
        return "hugo", false
    }
    if strings.Contains(html, "data-docusaurus") {
        return "docusaurus", false
    }
    if strings.Contains(html, "VPContent") || strings.Contains(html, "vitepress") {
        return "vitepress", false
    }
    if strings.Contains(html, "__NEXT_DATA__") {
        return "nextjs", false  // Nextra and similar
    }
    if strings.Contains(url, "readthedocs") {
        return "readthedocs", false
    }
    
    return "unknown", false  // Default to HTTP-only, let content validation catch failures
}
```

---

## Recommended implementation strategy

### Complete decision flow

1. **Extract domain** → check site profile cache
2. **If cached profile valid** → use cached rendering decision
3. **If no cache** → fetch with HTTP, run platform detection
4. **If platform detected** → apply deterministic rendering choice, cache result
5. **If platform unknown** → apply HTML heuristics (empty containers, noscript warnings, DOM count)
6. **If heuristics confident** → cache decision, proceed with appropriate method
7. **If uncertain** → render with browser, compare content, cache learned result
8. **On content validation failure** → escalate to browser fallback, update profile

### Key thresholds summary

- Empty root container + bundle.js → **Likely JS-dependent**
- Text content < 500 chars + >3 script tags → **Likely JS-dependent**
- DOM elements < 50 in body → **Likely JS-dependent**
- `<noscript>` with "enable JavaScript" → **Likely JS-dependent**
- Size ratio (rendered/raw) > 2.0 → **Confirmed JS-dependent**
- Site profile cache TTL → **24-48 hours**
- HTTP timeout → **15-20 seconds**
- Browser timeout → **30-60 seconds**

### Cost-benefit analysis

HTTP requests complete in **50-200ms** vs **2-10 seconds** for browser rendering. Browser rendering consumes **5x more resources** (CPU, memory, proxy bandwidth). Production systems report **70-80% of pages** can be scraped with HTTP-only when using adaptive detection, delivering substantial performance gains over browser-first approaches.
