# Concurrent Recursive Crawling

## Problem

Recursive crawling (fallback when sitemap returns no URLs) processes URLs sequentially. With 100+ pages, this is slow. The frontier often has multiple URLs queued that could be fetched in parallel.

## Goals

- Speed up recursive crawling via bounded concurrency
- Reuse existing `Concurrency` field (default 10)
- Maintain current rate limiting behavior (shared per-domain limit)
- Keep same scope filtering, max URL limit, and error handling

## Design

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Coordinator                             │
│  ┌──────────┐    ┌─────────────┐    ┌──────────────────┐   │
│  │ Frontier │───▶│  Dispatcher │───▶│  Work Channel    │   │
│  │ (queue)  │    │  (loop)     │    │  (bounded)       │   │
│  └──────────┘    └─────────────┘    └────────┬─────────┘   │
│       ▲                                      │              │
│       │                                      ▼              │
│       │         ┌──────────────────────────────────────┐   │
│       │         │           Worker Pool                 │   │
│       │         │  ┌────────┐ ┌────────┐ ┌────────┐    │   │
│       │         │  │Worker 1│ │Worker 2│ │Worker N│    │   │
│       │         │  └───┬────┘ └───┬────┘ └───┬────┘    │   │
│       │         └──────┼──────────┼──────────┼─────────┘   │
│       │                └──────────┼──────────┘              │
│       │                           ▼                         │
│  ┌────┴───────────────────────────────────────────────┐    │
│  │              Results Channel                        │    │
│  │  (discovered URLs, content, errors)                 │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Components

**Coordinator (main goroutine):**
- Pops URLs from frontier, sends to work channel
- Receives results from workers
- Pushes discovered URLs to frontier (after scope filtering)
- Tracks pending count for termination detection
- Accumulates stats (saved, failed, bytes, tokens)

**Worker pool (N goroutines where N = Concurrency):**
- Receives URLs from work channel
- Waits on rate limiter
- Fetches, extracts links, extracts content, converts
- Sends result back via results channel

**Termination condition:**
- Frontier empty AND pending == 0

### Channels

```go
workCh   chan locdoc.DiscoveredLink  // buffered, size = concurrency
resultCh chan crawlResult             // unbuffered
```

### Extended crawlResult

```go
type crawlResult struct {
    position   int
    url        string
    title      string
    markdown   string
    hash       string
    err        error
    discovered []locdoc.DiscoveredLink  // NEW: links found on this page
}
```

### Coordinator Loop (pseudocode)

```go
for {
    select {
    case <-ctx.Done():
        // Graceful shutdown
        close(workCh)
        drainWithTimeout(resultCh, &pending, 5*time.Second)
        return &result, ctx.Err()

    case result := <-resultCh:
        pending--
        for _, link := range result.discovered {
            if inScope(link) {
                frontier.Push(link)
            }
        }
        accumulate(&result, crawlResult)

    case workCh <- nextLink:
        pending++
        processedCount++
        nextLink = nil
    }

    if nextLink == nil && processedCount < maxRecursiveCrawlURLs {
        nextLink, ok = frontier.Pop()
        if !ok && pending == 0 {
            break  // Done
        }
    }
}
close(workCh)
drainRemaining(resultCh, &pending)
```

### Rate Limiting

Workers call `RateLimiter.Wait(ctx, host)` before fetching. Multiple workers share the same per-domain limiter, so total throughput stays within configured limit regardless of concurrency level.

### Scope Filtering

Coordinator filters discovered URLs before pushing to frontier:
1. Same host as source URL
2. Path starts with source URL's path prefix
3. Matches URL filter regex (if configured)

Frontier handles deduplication via Bloom filter.

### Context Cancellation

1. Context canceled → workers return from blocking calls
2. Coordinator closes workCh → workers exit range loop
3. Coordinator drains resultCh with timeout → collects partial stats
4. Return accumulated results (partial crawl is useful)

## Changes Required

| File | Change |
|------|--------|
| `crawl/crawl.go` | Replace sequential loop with coordinator pattern |
| `crawl/crawl.go` | Extend `crawlResult` with `discovered` field |
| `crawl/crawl.go` | Add/refactor URL processing for workers |
| `crawl/crawl_test.go` | Add concurrent crawling tests |

## No Changes Needed

- `Frontier` - already thread-safe
- `DomainLimiter` - already thread-safe
- `Crawler` struct - reuses `Concurrency` field
- Domain types - no new interfaces

## Testing

- Concurrent workers process URLs in parallel
- Rate limiter still enforces per-domain limits
- Termination correct when frontier drains
- Context cancellation returns partial results
- Max URL limit still enforced
- Scope filtering still works
