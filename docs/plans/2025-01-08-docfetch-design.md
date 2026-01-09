# docfetch Design

A single-purpose CLI that crawls documentation sites and saves them as markdown files on disk for direct exploration with Claude's file tools.

## Motivation

Current approach stuffs entire documentation into Gemini's 1M context window. This doesn't scale for large documentation sites (e.g., Salesforce docs exceed 1M tokens).

New approach: save docs as files on disk, let Claude use `tree`, `grep`, `glob`, `read` directly. Claude can iteratively search and reason about what to look for next - more powerful than one-shot RAG retrieval.

## CLI Interface

```
docfetch <url> <name> [path]   # fetch to [path]/[name]/ (path defaults to .)
docfetch --preview <url>       # preview what would be fetched
```

Examples:
```bash
docfetch https://react.dev/reference react
docfetch https://docs.anthropic.com anthropic ~/docs
docfetch --preview https://salesforce.com/docs
```

Navigation uses standard Unix tools:
- `tree react/` - table of contents
- `grep -r "useEffect" react/` - search content
- `rm -rf react/` - delete

## Output Structure

Directory structure mirrors URL paths:

```
react/
├── reference/
│   └── react/
│       ├── useState.md
│       └── useEffect.md
└── learn/
    ├── index.md
    └── installation.md
```

URL to path conversion:
- `https://react.dev/reference/react/useState` → `reference/react/useState.md`
- `https://react.dev/learn/` → `learn/index.md`

## File Format

Each markdown file includes YAML frontmatter:

```markdown
---
source: https://react.dev/reference/react/useState
title: useState
crawled: 2025-01-08
---

# useState

`useState` is a React Hook that lets you add a state variable...
```

## Behavior

### Atomic Updates

When fetching to an existing directory:
1. Crawl to temporary directory (`[name].tmp/`)
2. On success: delete original `[name]/`, rename temp to `[name]/`
3. On failure: keep original intact, clean up temp

### Error Handling

Continue on individual page failures, summarize at end:
```
Fetched 142 pages, 3 failed:
  - /api/deprecated: 404
  - /internal/auth: timeout
  - /legacy/v1: extraction failed
```

## Architecture

### Interface Change

Add narrow interface to root package:

```go
// document.go

// DocumentWriter writes documents to storage.
type DocumentWriter interface {
    CreateDocument(ctx context.Context, doc *Document) error
}

// DocumentService provides full document management.
type DocumentService interface {
    DocumentWriter  // Embed narrow interface
    FindDocumentByID(ctx context.Context, id string) (*Document, error)
    FindDocuments(ctx context.Context, filter DocumentFilter) ([]*Document, error)
    DeleteDocument(ctx context.Context, id string) error
    DeleteDocumentsByProject(ctx context.Context, projectID string) error
}
```

### Package Structure

```
locdoc/
├── document.go         # DocumentWriter + DocumentService interfaces
├── fs/                 # New package
│   └── writer.go       # Implements DocumentWriter
├── sqlite/
│   └── document.go     # Implements DocumentService (includes DocumentWriter)
├── crawl/
│   └── crawl.go        # Depends on DocumentWriter (changed from DocumentService)
└── cmd/
    ├── locdoc/         # Existing binary (unchanged)
    └── docfetch/       # New binary
        └── main.go
```

### fs Package

```go
// fs/writer.go
package fs

var _ locdoc.DocumentWriter = (*Writer)(nil)

type Writer struct {
    BaseDir string
}

func NewWriter(baseDir string) *Writer

func (w *Writer) CreateDocument(ctx context.Context, doc *locdoc.Document) error
// - Converts doc.SourceURL to relative file path
// - Creates parent directories
// - Writes markdown with YAML frontmatter
```

### Reused Components

No changes needed:
- `crawl/` - Discoverer, Crawler orchestration, rate limiting
- `rod/` - headless browser fetching
- `http/` - HTTP fetching, sitemap parsing
- `goquery/` - link extraction, framework detection
- `trafilatura/` - content extraction
- `htmltomarkdown/` - HTML to markdown conversion

Not used by docfetch:
- `sqlite/` - no database needed
- `gemini/` - no LLM queries
- Token counting

## Implementation Tasks

1. Add `DocumentWriter` interface to `document.go`
2. Embed `DocumentWriter` in `DocumentService`
3. Update `crawl.Crawler` to depend on `DocumentWriter`
4. Create `fs/writer.go` implementing `DocumentWriter`
5. Create `cmd/docfetch/main.go` with CLI wiring
6. Add mock for `DocumentWriter` in `mock/`

## Future Considerations

- Claude skill for navigating docfetch output directories
- Potential `--filter` flag for URL path filtering
- Progress output during crawl
