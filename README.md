# locdoc

A local CLI tool for storing and querying documentation. Crawls documentation sites, converts them to markdown, and stores everything in a local SQLite database.

## Why

AI coding assistants work well when they have access to library documentation. This tool provides a simple way to:

1. Crawl documentation sites and store them locally as markdown
2. Query stored docs through a CLI that agents can use directly
3. Avoid bloating agent context with MCP tool definitions

The interface is intentionally minimal - a few straightforward commands that any agent can call without needing schema definitions or protocol negotiation.

## Status

This project is written primarily by LLMs (Claude). Our goal is high-quality software, but this is alpha - expect bugs and edge cases we haven't covered yet. Contributions and bug reports welcome.

## Installation

```bash
go install github.com/fwojciec/locdoc/cmd/locdoc@latest
```

## Usage

### Register a documentation project

```bash
locdoc add htmx https://htmx.org/
```

### Crawl and store the documentation

```bash
locdoc crawl htmx
```

This discovers pages via sitemap, fetches each page, extracts main content, converts to markdown, and stores in SQLite.

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

Requires `GEMINI_API_KEY` environment variable.

```bash
locdoc ask htmx "How do I trigger a request on page load?"
```

### Delete a project

```bash
locdoc delete htmx --force
```

## Configuration

- **Database location**: `~/.locdoc/locdoc.db` by default, or set `LOCDOC_DB` environment variable
- **API key**: Set `GEMINI_API_KEY` for the `ask` command

## License

MIT
