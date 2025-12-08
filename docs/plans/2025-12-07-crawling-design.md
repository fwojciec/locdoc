# Documentation Crawler Design

Pure Go documentation crawler for locdoc MVP.

## Architecture Overview

**Package structure (Ben Johnson pattern):**

```
locdoc/
├── [existing domain types and storage interfaces]
├── http/          # Sitemap parsing (net/http + etree)
├── rod/           # Browser rendering (go-rod/rod)
├── trafilatura/   # Content extraction (go-trafilatura)
├── htmltomd/      # HTML→Markdown (html-to-markdown)
├── sqlite/        # Storage (ncruces/go-sqlite3)
└── cmd/locdoc/    # CLI wiring
```

**Dependencies (all pure Go, no CGO):**

| Package | Purpose | CGO |
|---------|---------|-----|
| `github.com/ncruces/go-sqlite3` | Pure Go SQLite (WASM) | No |
| `github.com/asg017/sqlite-vec-go-bindings/ncruces` | Vector search (future) | No |
| `github.com/go-rod/rod` | Browser automation | No |
| `github.com/markusmobius/go-trafilatura` | Content extraction | No |
| `github.com/JohannesKaufmann/html-to-markdown` | Markdown conversion | No |
| `github.com/beevik/etree` | XML/sitemap parsing | No |

**Runtime requirement:** Chrome/Chromium must be installed.

**Interfaces:** Defined during implementation, following Ben Johnson pattern (root package, compile-time verification).

## Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. DISCOVER                                                     │
│    http.SitemapParser                                           │
│    ├── Fetch robots.txt, look for Sitemap: directives          │
│    ├── Fall back to /sitemap.xml                               │
│    ├── Parse XML, extract <loc> elements                       │
│    ├── Handle sitemap indexes recursively                      │
│    └── Filter URLs (regex include/exclude)                     │
│                                     ↓                          │
│                              []string (URLs)                    │
├─────────────────────────────────────────────────────────────────┤
│ 2. FETCH                                                        │
│    rod.Fetcher                                                  │
│    ├── Launch Chrome (or connect to existing)                  │
│    ├── Navigate to URL                                         │
│    ├── Wait for JS to render                                   │
│    └── Return rendered HTML                                    │
│                                     ↓                          │
│                              string (raw HTML)                  │
├─────────────────────────────────────────────────────────────────┤
│ 3. EXTRACT                                                      │
│    trafilatura.Extractor                                        │
│    ├── Remove boilerplate (nav, footer, ads)                   │
│    ├── Identify main content area                              │
│    └── Return clean HTML + metadata (title)                    │
│                                     ↓                          │
│                              (title, cleanHTML)                 │
├─────────────────────────────────────────────────────────────────┤
│ 4. CONVERT                                                      │
│    htmltomd.Converter                                           │
│    ├── Convert HTML to Markdown                                │
│    ├── Preserve code blocks, tables, links                     │
│    └── Return Markdown string                                  │
│                                     ↓                          │
│                              string (Markdown)                  │
├─────────────────────────────────────────────────────────────────┤
│ 5. STORE                                                        │
│    sqlite.DocumentService                                       │
│    ├── Hash content (xxHash for change detection)              │
│    ├── Create/update Document record                           │
│    └── Persist to SQLite                                       │
└─────────────────────────────────────────────────────────────────┘
```

## MVP Scope

**In scope:**

| Feature | Notes |
|---------|-------|
| `locdoc add <name> <url>` | Register a project with sitemap URL |
| `locdoc crawl [project]` | Crawl all/one project, store docs |
| `locdoc list` | List registered projects |
| Sitemap-only discovery | No recursive link following |
| Chrome required | User must have Chrome installed |
| SQLite storage | Single file, no server |

**Out of scope (future):**

| Feature | Rationale |
|---------|-----------|
| Recursive link crawling | Sitemap covers 90% of doc sites |
| Bundled Chromium | Adds ~150MB, complexity |
| Vector search / RAG | Separate epic after crawling works |
| Chunking for embeddings | Depends on RAG epic |
| Watch mode / incremental | Nice-to-have, not MVP |
| MCP server | Integration layer, after core works |

**Success criteria:**

1. `locdoc add go-docs https://go.dev/doc/` registers project
2. `locdoc crawl go-docs` fetches all pages from sitemap, stores as Markdown
3. Documents queryable via `sqlite3` directly (no search UI yet)
4. `make validate` passes
5. No CGO - builds with `CGO_ENABLED=0`

## Implementation Order

```
cmd/locdoc (CLI)
    ↓ uses all
┌───────────────────────────────────────────┐
│  sqlite/     ←── stores output from...    │
│      ↑                                    │
│  htmltomd/   ←── converts output from...  │
│      ↑                                    │
│  trafilatura/ ←── extracts from...        │
│      ↑                                    │
│  rod/        ←── fetches URLs from...     │
│      ↑                                    │
│  http/       ←── discovers URLs           │
└───────────────────────────────────────────┘
```

| Phase | Package | Rationale |
|-------|---------|-----------|
| 1 | `sqlite/` | Foundation - need storage before anything else |
| 2 | `http/` | Sitemap parsing - can test independently with real URLs |
| 3 | `rod/` | Browser fetching - can test with real pages |
| 4 | `trafilatura/` | Content extraction - takes HTML from rod |
| 5 | `htmltomd/` | Markdown conversion - takes clean HTML |
| 6 | `cmd/locdoc/` | Wire everything together |

## Package Requirements

### `sqlite/`

- Implement `ProjectService` and `DocumentService` (interfaces already defined)
- Use `ncruces/go-sqlite3` with `database/sql` driver
- Auto-create schema on first run
- Store document content as Markdown text
- Content hash (xxHash) for change detection

### `http/`

- Discover sitemap URL from robots.txt or fallback to `/sitemap.xml`
- Parse sitemap XML, handle sitemap indexes recursively
- Filter URLs by pattern (include/exclude)
- Return list of discovered URLs
- Respect context cancellation

### `rod/`

- Launch Chrome or connect to existing instance
- Navigate to URL, wait for JS to render
- Return rendered HTML as string
- Handle timeouts gracefully
- Clean up browser resources on shutdown

### `trafilatura/`

- Accept raw HTML, return clean HTML + title
- Remove boilerplate (nav, footer, sidebar, ads)
- Preserve main content structure
- Use go-trafilatura library API

### `htmltomd/`

- Accept clean HTML, return Markdown string
- Preserve code blocks with language hints
- Preserve tables, links, headings
- Use html-to-markdown library

### `cmd/locdoc/`

- `add <name> <url>` - register project
- `list` - show projects
- `crawl [name]` - run pipeline for all/one project
- Wire all packages together
- Exit codes for success/failure

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Chrome not installed | Medium | Blocks crawling | Clear error message with install instructions |
| WASM performance | Low | Slower than native | Acceptable for CLI tool |
| Sitemap missing/incomplete | Medium | Partial crawl | MVP accepts this; future: recursive fallback |
| Rate limiting by target sites | Medium | Failed requests | Add configurable delay; respect robots.txt |
| rod browser resource leaks | Low | Memory issues | Use rod's launcher management; cleanup on exit |

## Open Questions (not blocking MVP)

| Question | When to decide |
|----------|----------------|
| Chunking strategy for RAG | RAG epic |
| Embedding model choice | RAG epic |
| Incremental crawl (hash-based) | Post-MVP enhancement |
| URL filtering UX | During http/ implementation |
| Browser timeout/retry policy | During rod/ implementation |

## Research References

- [docs/extracting-documentation-links.md](../extracting-documentation-links.md) - Crawling research
- [docs/local-rag.md](../local-rag.md) - RAG implementation research
