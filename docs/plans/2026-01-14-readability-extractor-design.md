# Replace trafilatura with go-readability

## Problem

The trafilatura extractor strips `<pre>` elements during extraction, causing code blocks to be completely missing from the output. This is a fundamental limitation - trafilatura is designed for article text extraction, not documentation where code blocks are critical.

Example: extracting from nx.dev documentation produces markdown with headings and prose, but all command examples are missing.

## Solution

Replace go-trafilatura with go-readability for content extraction. Testing confirmed go-readability preserves code blocks correctly.

| Extractor | Preserves `<pre>` | Code in output |
|-----------|-------------------|----------------|
| go-trafilatura | No | Missing |
| go-readability | Yes | Present |

## Design Decisions

**Package naming**: `readability/` - follows Ben Johnson pattern of naming packages after the dependency they wrap.

**Interface unchanged**: The `locdoc.Extractor` interface stays as-is. Only the implementation changes.

**URL parameter**: go-readability accepts a URL for resolving relative links. We pass `nil` - we're extracting content, not resolving links. This hides unused functionality behind our simpler interface.

**No post-processing**: go-readability output has minor formatting quirks (extra blank lines in code blocks). We accept this - LLMs consuming the output handle it fine.

## Package Structure

```
locdoc/
├── extractor.go              # Interface (unchanged)
├── readability/              # NEW
│   ├── extractor.go
│   └── extractor_test.go
├── trafilatura/              # DELETE after migration
└── cmd/
    ├── locdoc/main.go        # Update import
    └── docfetch/main.go      # Update import
```

## Test Specifications

Tests tell the story of how the extractor works.

### Contract Basics

```go
// TestExtractor_RejectsEmptyInput
// Empty input is invalid - nothing to extract
// - assert.Equal(t, locdoc.EINVALID, locdoc.ErrorCode(err))
```

### Metadata Extraction

```go
// TestExtractor_ExtractsTitle
// Title comes from page metadata for downstream use
// - assert.Equal(t, "Page Title", result.Title)
```

### Boilerplate Removal (primary purpose)

```go
// TestExtractor_RemovesNavigation
// Navigation is boilerplate, not content
// - assert.NotContains(t, result.ContentHTML, "nav link text")

// TestExtractor_RemovesFooter
// Footer is boilerplate, not content
// - assert.NotContains(t, result.ContentHTML, "footer text")

// TestExtractor_RemovesSidebar
// Sidebars are boilerplate, not content
// - assert.NotContains(t, result.ContentHTML, "sidebar text")

// TestExtractor_KeepsMainArticleContent
// Article body is the content we want
// - assert.Contains(t, result.ContentHTML, "article paragraph text")
```

### Structure Preservation

```go
// TestExtractor_PreservesHeadings
// Heading hierarchy matters for document structure
// - assert.Contains(t, result.ContentHTML, "<h1")
// - assert.Contains(t, result.ContentHTML, "<h2")

// TestExtractor_PreservesParagraphs
// Basic text structure
// - assert.Contains(t, result.ContentHTML, "<p")

// TestExtractor_PreservesLists
// Lists are content structure
// - assert.Contains(t, result.ContentHTML, "<ul")
// - assert.Contains(t, result.ContentHTML, "<li")

// TestExtractor_PreservesTables
// Tables contain structured data
// - assert.Contains(t, result.ContentHTML, "<table")

// TestExtractor_PreservesLinks
// Links are part of content
// - assert.Contains(t, result.ContentHTML, "<a")

// TestExtractor_PreservesInlineCode
// Inline code like `variable` is content
// - assert.Contains(t, result.ContentHTML, "<code")
```

### Code Blocks (edge case: syntax-highlighted code)

```go
// TestExtractor_PreservesSimpleCodeBlocks
// Basic <pre><code>text</code></pre> must survive extraction
// - assert.Contains(t, result.ContentHTML, "<pre")
// - assert.Contains(t, result.ContentHTML, "code text")

// TestExtractor_PreservesCodeBlocksWithNestedSpans
// Syntax highlighters wrap code in <span> elements for coloring
// The text content must be extracted even from:
//   <pre><code><div class="line"><span>nx</span> <span>g</span></div></code></pre>
// - assert.Contains(t, result.ContentHTML, "<pre")
// - assert.Contains(t, result.ContentHTML, "nx g")

// TestExtractor_PreservesCodeBlocksInWrapperDivs
// Documentation sites wrap code in complex structures:
//   <div class="expressive-code"><figure><pre>...</pre></figure></div>
// The code must survive even inside these wrappers
// - assert.Contains(t, result.ContentHTML, "<pre")
// - assert.Contains(t, result.ContentHTML, "command text")

// TestExtractor_PreservesLanguageHints
// data-language or class="language-x" enables syntax highlighting in markdown
// - assert.Contains(t, result.ContentHTML, "bash") // however it's preserved
```

## Implementation Tickets

1. **Add readability extractor** - Create `readability/` package implementing `locdoc.Extractor` with full test coverage per specifications above.

2. **Switch to readability and remove trafilatura** - Update CLI imports, delete `trafilatura/` package, clean up go.mod.
