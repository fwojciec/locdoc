# Building a CLI documentation crawler: tools and architecture guide

**Crawl4AI and Katana emerge as the top open-source options** for building a local CLI tool that recursively crawls documentation sites, handles JavaScript rendering, and prepares content for RAG pipelines. For Go developers, combining Katana (crawling) with go-trafilatura (content extraction) provides a robust foundation, while Python developers should consider Crawl4AI for its all-in-one approach with native LLM integration.

The key insight from this research: **LLM-powered extraction is often overkill for well-structured documentation sites**. Traditional tools like Trafilatura handle technical content excellently, and LLMs add the most value in adaptive crawling decisions and handling inconsistent site structures—not basic HTML-to-Markdown conversion.

## Web crawling tools compared by JavaScript rendering capability

Modern documentation sites built with Next.js, Docusaurus, and VitePress require JavaScript rendering to access content. This eliminates many traditional crawlers from consideration.

**Katana** (Go, 15K GitHub stars) stands out for CLI-first workflows with built-in headless Chrome support. A single command like `katana -u https://docs.example.com -headless -jc -d 5` handles JS-rendered navigation, respects scope boundaries, and outputs clean URL lists. Its `-cs` flag for scope control keeps crawls within `/docs/` paths without additional configuration.

**Crawlee** (Node.js/Python, 15K stars) offers the most mature Playwright integration through its `PlaywrightCrawler` class. The `enqueueLinks()` method automatically handles recursive crawling with intelligent deduplication. Memory usage runs **500-1000MB** with browser instances—acceptable for local developer machines.

**Colly** (Go, 25K stars) remains the most popular Go option but **lacks native JavaScript rendering**. Integrating headless browser support requires manually orchestrating chromedp and passing rendered HTML to Colly—significant additional complexity that makes Katana more practical for JS-heavy sites.

| Tool | JS Rendering | CLI Quality | Memory | Best For |
|------|-------------|-------------|--------|----------|
| Katana | Native headless | Excellent | 300-500MB | Ad-hoc CLI crawling |
| Crawlee | Native Playwright | Good | 500-1000MB | Production scrapers |
| Colly | Requires chromedp | Library only | 30-80MB | Static sites |
| Scrapy | Via plugin | Moderate | 100-300MB | Large-scale projects |

For the stated preference of Go, **Katana is the practical choice** despite Colly's higher star count. Katana handles the JavaScript rendering requirement out of the box while Colly would require substantial integration work.

## Content extraction: Trafilatura leads for technical documentation

Extracting clean content from documentation pages requires stripping navigation, sidebars, and footers while preserving code blocks, tables, and technical formatting.

**Trafilatura** (Python) consistently outperforms alternatives in benchmarks and handles technical documentation particularly well. Its explicit `handle_code_blocks()` function preserves `<pre>` and `<code>` elements correctly. The `output_format="markdown"` option produces RAG-ready content directly:

```python
from trafilatura import fetch_url, extract
result = extract(fetch_url(url), output_format="markdown", 
                 include_tables=True, include_formatting=True)
```

**go-trafilatura** provides a Go port with similar capabilities. The CLI supports direct Markdown output: `go-trafilatura -f markdown https://docs.example.com > doc.md`

**Readability.js** (Mozilla) powers Firefox Reader View and works well but outputs HTML rather than Markdown—requiring an additional conversion step through Turndown or html2text. For JavaScript environments, chain Readability with Turndown using `codeBlockStyle: 'fenced'` to preserve code formatting.

Documentation frameworks expose content through predictable selectors. When building custom extractors, target these patterns:

| Framework | Main Content Selector |
|-----------|----------------------|
| Docusaurus | `article.theme-doc-markdown`, `[role="main"]` |
| MkDocs Material | `.md-content__inner`, `article` |
| VitePress | `.vp-doc`, `main` |
| GitBook | `.page-body`, `.markdown-section` |
| ReadTheDocs | `.rst-content`, `.document` |

For reliable fallbacks, check in order: `main`, `[role="main"]`, `article`, `.content`, `.markdown-body`.

## LLM tools add value in specific scenarios

The explosion of LLM-powered crawling tools—Firecrawl, Jina Reader, Crawl4AI, ScrapeGraphAI—raises the question: where do LLMs genuinely help?

**Where LLMs add genuine value:**
- Schema-free extraction ("extract all API endpoints with their parameters")
- Adaptive crawling that knows when to stop based on content relevance
- Handling sites with inconsistent or frequently-changing layouts
- Semantic filtering to remove irrelevant content without CSS selectors

**Where traditional parsing is better:**
- Well-structured documentation with consistent templates
- High-volume crawling where token costs accumulate
- Speed-critical pipelines where LLM inference adds latency

**Crawl4AI** (Apache 2.0, 51K stars) offers the best balance: fully open-source, runs completely locally, and implements a **hybrid approach** using BM25 heuristics first with optional LLM refinement. It works with any LLM including local Ollama models:

```bash
pip install crawl4ai
crwl https://docs.example.com --depth 3 --output markdown
```

**Jina's ReaderLM-v2** deserves special mention—a 1.5B parameter model specifically trained for HTML-to-Markdown conversion that runs locally on consumer GPUs (RTX 3090/4090). This provides LLM-quality extraction without API costs for teams with available GPU resources.

**Firecrawl** offers polished APIs but its AGPL-3.0 license and reduced features in self-hosted mode make it less suitable for local CLI tools. The simpler "firecrawl-simple" fork strips it down for easier self-hosting but loses anti-bot capabilities.

## Link extraction should start with sitemaps

The research reveals a clear best practice: **sitemap-first with recursive fallback**.

Check multiple sitemap locations (`/sitemap.xml`, `/sitemap_index.xml`, plus `Sitemap:` directive in `robots.txt`). Handle sitemap indexes recursively since they often nest. Support `.xml.gz` compression. Most documentation sites maintain accurate sitemaps because they're used for SEO.

For sites without sitemaps or with JavaScript-rendered navigation:

```javascript
await page.goto(url, {waitUntil: 'networkidle0'});
const links = await page.$$eval('nav a', anchors => 
    anchors.map(a => a.href));
```

Key filtering rules to stay within documentation scope:
- Domain matching to prevent external crawling
- Path prefix filtering (only `/docs/*` paths)
- Strip fragment identifiers (`#section`) before deduplication
- Exclude patterns like `.pdf`, `/api/`, `/blog/`

## Content hashing strategy for meaningful change detection

**Hash normalized content, not raw HTML.** Raw HTML produces false positives from timestamp updates, minor CSS changes, and dynamic elements. The recommended approach:

1. Extract main content area using framework-specific selectors
2. Strip whitespace, normalize line endings
3. Remove dynamic elements (dates, "last updated" text)
4. Hash with **xxHash64**

xxHash64 runs **10x faster than SHA256** while providing excellent collision resistance for non-adversarial scenarios like change detection. Performance benchmarks show xxHash processing a 6.6GB file in 0.5 seconds versus 27 seconds for SHA256.

```python
import xxhash
import re

def content_hash(html_content):
    # Normalize before hashing
    content = re.sub(r'\d{4}-\d{2}-\d{2}', '', html_content)
    content = re.sub(r'Last updated:.*', '', content)
    content = re.sub(r'\s+', ' ', content).strip()
    return xxhash.xxh64(content.encode()).hexdigest()
```

For **incremental crawling**, combine content hashing with HTTP conditional requests. Store `ETag` and `Last-Modified` headers, then send `If-None-Match`/`If-Modified-Since` on subsequent requests. Servers return `304 Not Modified` without body transfer when content hasn't changed—reducing bandwidth significantly.

## Storage architecture optimized for RAG consumption

The research points to a clear pattern: **SQLite for metadata and state, path-mirrored Markdown files for content**.

```
output/
└── docs.example.com/
    ├── index.md
    ├── guide/
    │   ├── index.md
    │   └── getting-started.md
    └── _metadata/
        └── manifest.json
```

Path mirroring preserves the original site hierarchy, making content human-navigable and easy to correlate with source URLs. Each Markdown file includes YAML frontmatter with metadata:

```yaml
---
title: "Getting Started Guide"
source_url: "https://docs.example.com/guide/getting-started"
content_hash: "a1b2c3d4e5f6"
fetched_at: "2025-12-07T10:30:00Z"
---
```

SQLite handles crawler state tracking—pending URLs, content hashes, ETags, crawl timestamps. This enables resumable crawls and efficient change detection:

```sql
CREATE TABLE crawl_state (
    url TEXT PRIMARY KEY,
    content_hash TEXT,
    etag TEXT,
    last_modified TEXT,
    last_crawl TIMESTAMP
);
```

For RAG integration, **chunk Markdown by headers** (split on `#`, `##`, `###`) to preserve topic boundaries. Use 10-20% overlap between chunks to maintain context. The frontmatter metadata enables citation back to source URLs in RAG responses.

## Recommended architecture and tool selection

Based on the research, here's the recommended stack for each language preference:

**Go-preferred stack:**
- Crawling: Katana with `-headless -jc` for JS rendering
- Content extraction: go-trafilatura with Markdown output
- Hashing: `github.com/cespare/xxhash`
- Storage: SQLite via `go-sqlite3`

**Python stack (alternative):**
- Crawling: Crawl4AI for integrated LLM support, or Scrapy + scrapy-playwright
- Content extraction: Trafilatura
- Hashing: `xxhash` package
- Storage: Built-in SQLite

**Node.js stack (alternative):**
- Crawling: Crawlee with PlaywrightCrawler
- Content extraction: Readability.js + Turndown
- Hashing: `xxhash-wasm`
- Storage: better-sqlite3

The overall architecture follows this pipeline:

```
┌─────────────────────────────────────────────────┐
│  Link Discovery                                  │
│  ├── Sitemap parser (try first)                 │
│  ├── Recursive crawler (fallback)               │
│  └── Headless browser for JS navigation         │
├─────────────────────────────────────────────────┤
│  Change Detection                                │
│  ├── HTTP conditional requests (ETag)           │
│  ├── xxHash on normalized content               │
│  └── SQLite state persistence                    │
├─────────────────────────────────────────────────┤
│  Content Processing                              │
│  ├── Framework-specific selector extraction     │
│  ├── HTML → Markdown conversion                 │
│  └── Frontmatter metadata injection             │
├─────────────────────────────────────────────────┤
│  Storage                                         │
│  ├── Path-mirrored Markdown files               │
│  ├── SQLite metadata/state database             │
│  └── JSON manifest for RAG ingestion            │
└─────────────────────────────────────────────────┘
```

## Existing implementations worth studying

Several open-source projects implement variations of this architecture:

**docrawl** (Rust) provides production-ready documentation crawling with auto-detection for Docusaurus, MkDocs, and Sphinx. It outputs path-mirrored Markdown with YAML frontmatter.

**Crawl4AI** (Python) includes a RAG-focused MCP server that demonstrates the full pipeline from crawling through vector storage integration.

**markdown-crawler** (Python) offers multithreaded crawling optimized for LLM consumption with clean Markdown output.

These projects provide tested implementations of the patterns described above—reviewing their source code accelerates development significantly.

## Conclusion

Building a CLI documentation crawler in 2025 benefits from mature, specialized tools rather than general-purpose web scrapers. **Katana + go-trafilatura** provides the strongest Go-native option, while **Crawl4AI** offers the most complete Python solution with built-in LLM capabilities.

The key architectural decisions—sitemap-first discovery, xxHash for change detection, SQLite for state, path-mirrored Markdown output—reflect proven patterns from existing tools. LLMs add genuine value for adaptive crawling and inconsistent content but remain overkill for well-structured documentation sites where traditional extraction produces excellent results at lower cost.

For RAG applications specifically, preserving document structure through header-based chunking and including source URLs in metadata enables accurate retrieval with proper citation—the ultimate goal of the documentation crawling pipeline.
