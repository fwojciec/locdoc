# locdoc

A local CLI tool for storing and querying documentation. Crawls documentation sites, converts them to markdown, and stores everything in a local SQLite database.

## Why

AI coding assistants work well when they have access to library documentation. This tool provides a simple way to:

1. Crawl documentation sites and store them locally as markdown
2. Query stored docs through a CLI that agents can use directly
3. Avoid bloating agent context with MCP tool definitions

The interface is intentionally minimal - a few straightforward commands that any agent can call without needing schema definitions or protocol negotiation.

## Requirements

- Go 1.21+ (for installation from source)
- Google Chrome (only needed for JavaScript-rendered sites)
- Gemini API key (for the `ask` command)

## Installation

```bash
go install github.com/fwojciec/locdoc/cmd/locdoc@latest
```

## Features

### Intelligent Crawling

- **Automatic discovery** - Uses sitemap.xml when available, falls back to recursive link extraction
- **Adaptive rendering** - Probes sites to detect if JavaScript rendering is needed; uses fast HTTP fetching for static sites
- **Framework detection** - Recognizes common documentation frameworks for better link extraction
- **Robust fetching** - Retry with exponential backoff, configurable timeouts

### Content Extraction

- Removes navigation, footers, sidebars automatically (go-trafilatura)
- Preserves document structure: headers, lists, code blocks, tables
- JavaScript rendering via headless Chrome when needed

### Local Storage

- SQLite database (`~/.locdoc/locdoc.db`)
- No cloud dependencies for crawling

### LLM Q&A

- Natural language queries via Gemini Flash
- Retrieval-focused prompts optimized for accuracy

## Usage

### Add a documentation project

```bash
locdoc add <name> <url> [flags]
```

This discovers pages via sitemap (or recursive crawling), fetches each page, extracts main content, converts to markdown, and stores in SQLite.

**Flags:**

| Flag | Description |
|------|-------------|
| `--preview` | Show discovered URLs without crawling |
| `--force` | Delete existing project first (for re-crawling) |
| `--filter` | URL path prefix filter (can be repeated) |
| `-c, --concurrency N` | Concurrent fetch limit (default: 3) |
| `--timeout` | Per-page fetch timeout |
| `--debug` | Debug output in preview mode |

**Examples:**

```bash
# Crawl a documentation site
locdoc add htmx https://htmx.org/

# Preview what will be crawled
locdoc add htmx https://htmx.org/ --preview

# Re-crawl an existing project
locdoc add htmx https://htmx.org/ --force

# Filter to specific sections
locdoc add htmx https://htmx.org/ --filter /docs/ --filter /examples/

# Limit concurrent fetches (useful for rate-limited sites)
locdoc add htmx https://htmx.org/ -c 2
```

### List registered projects

```bash
locdoc list
```

### View stored documents

```bash
# List document titles and URLs
locdoc docs htmx

# Output full markdown content (for piping to agents)
locdoc docs htmx --full
```

### Ask questions about documentation

```bash
locdoc ask htmx "How do I trigger a request on page load?"
```

### Delete a project

```bash
locdoc delete htmx --force
```

## Configuration

| Variable | Purpose | Default |
|----------|---------|---------|
| `LOCDOC_DB` | Database path | `~/.locdoc/locdoc.db` |
| `GEMINI_API_KEY` | Required for `ask` command | - |

## Limitations

- **No GitHub/git support** - Cannot crawl README files or wikis from repositories
- **No incremental updates** - Re-crawling fetches all pages (use `--force`)
- **Single LLM provider** - Currently only supports Google Gemini
- **No semantic search** - All documents sent to LLM (works well for small-medium doc sites)

## Status

This project is written primarily by LLMs (Claude). Our goal is high-quality software, but this is alpha - expect bugs and edge cases we haven't covered yet. Contributions and bug reports welcome.

## License

MIT
