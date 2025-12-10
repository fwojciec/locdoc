# Crawl Summary with Token Counting

## Problem

After making crawl concurrent, output dumps as a wall of text at the end. The verbose per-URL output isn't useful when it all appears at once. Need a concise summary with useful stats.

## Design Decisions

1. **Token counts computed on-demand** - Not stored in DB since different models have different tokenizers
2. **Model as const in main** - `gemini-2.5-flash` hardcoded for now, easy to make configurable later
3. **Simple interface** - `CountTokens(ctx, text) (int, error)`
4. **Count during crawl** - Accumulate totals in results loop while content is in memory
5. **Gemini package** - Token counter lives in `gemini/` alongside future LLM query functionality

## Interface

Root package (`locdoc.go` or new `token.go`):

```go
// TokenCounter counts tokens in text for a specific model.
type TokenCounter interface {
    CountTokens(ctx context.Context, text string) (int, error)
}
```

## Implementation

New `gemini/` package using `google.golang.org/genai/tokenizer`:

```go
// gemini/token.go
package gemini

import (
    "context"

    "github.com/fwojciec/locdoc"
    "google.golang.org/genai/tokenizer"
)

var _ locdoc.TokenCounter = (*TokenCounter)(nil)

type TokenCounter struct {
    tok *tokenizer.LocalTokenizer
}

func NewTokenCounter(model string) (*TokenCounter, error) {
    tok, err := tokenizer.NewLocalTokenizer(model)
    if err != nil {
        return nil, err
    }
    return &TokenCounter{tok: tok}, nil
}

func (tc *TokenCounter) CountTokens(ctx context.Context, text string) (int, error) {
    // Use tokenizer API to count tokens in text
}
```

## Wiring

In `cmd/locdoc/main.go`:

```go
const defaultModel = "gemini-2.5-flash"

type Main struct {
    // existing fields...
    TokenCounter locdoc.TokenCounter
}

func (m *Main) Run(ctx context.Context) error {
    // ... existing setup ...

    tokenCounter, err := gemini.NewTokenCounter(defaultModel)
    if err != nil {
        return fmt.Errorf("token counter: %w", err)
    }
    m.TokenCounter = tokenCounter
}
```

Pass `TokenCounter` to `CmdCrawl` and `crawlProject`.

## Output Format

Replace verbose per-URL output with summary:

```go
// Accumulate stats during results loop
var totalBytes int
var totalTokens int
var savedCount int

for i, result := range results {
    // ... existing save logic ...

    if result.err == nil {
        totalBytes += len(result.markdown)
        tokens, _ := tokenCounter.CountTokens(ctx, result.markdown)
        totalTokens += tokens
        savedCount++
    }
}

// Print summary
fmt.Fprintf(stdout, "  Saved %d pages (%s, ~%dk tokens)\n",
    savedCount, formatBytes(totalBytes), totalTokens/1000)
```

Example output:
```
Crawling htmx (https://htmx.org/)...
  Found 179 URLs
  Saved 179 pages (2.3 MB, ~580k tokens)
```

## Testing

Mock in `mock/token.go`:

```go
package mock

var _ locdoc.TokenCounter = (*TokenCounter)(nil)

type TokenCounter struct {
    CountTokensFn func(ctx context.Context, text string) (int, error)
}

func (tc *TokenCounter) CountTokens(ctx context.Context, text string) (int, error) {
    return tc.CountTokensFn(ctx, text)
}
```

## File Changes

- `locdoc.go` or `token.go` - Add `TokenCounter` interface
- `gemini/token.go` - New file, implementation
- `mock/token.go` - New file, mock
- `cmd/locdoc/main.go` - Add const, wire TokenCounter, update CmdCrawl signature
- `cmd/locdoc/main.go` - Update `crawlProject` to accumulate stats and print summary

## Future Extensions

- `docs` command could also show token counts per project
- Model could become configurable via flag or config file
- Could add `--verbose` flag to restore per-URL output if needed
