# Crawl Robustness Improvements

**Date:** 2025-12-10
**Status:** Approved

## Problem

When crawling large documentation sites (e.g., nx.dev with 662 URLs), the `locdoc add` command has several limitations:

1. **No retry logic** - Transient failures (timeouts, network issues) cause permanent skips
2. **No progress visibility** - 10+ minute crawls show nothing until completion
3. **Fixed concurrency** - Hardcoded at 20, not tunable for different machines/sites
4. **Implicit timeout** - Relies on Rod defaults, no explicit control

Example failure output:
```
skip https://nx.dev/docs/guides/installation (fetch failed): navigation failed: net::ERR_TIMED_OUT
```
20 URLs failed with timeouts - unclear if transient or permanent, no way to retry without re-crawling everything.

## Design

### 1. Retry Logic

Add exponential backoff retry for fetch failures:

- **Max attempts:** 3 (1 initial + 2 retries)
- **Backoff:** 1s → 2s → 4s between attempts
- **Scope:** Fetch stage only (extract/convert failures are deterministic)

```
Attempt 1: fetch fails with timeout
  wait 1s
Attempt 2: fetch fails with timeout
  wait 2s
Attempt 3: fetch succeeds → continue to extract/convert
```

**Location:** `cmd/locdoc/main.go` in `processURL()`

### 2. Progress Reporting

Replace silent crawling with live progress updates:

```
Added project "nx" (19825135-...)
  Found 662 URLs
  [45/662] https://nx.dev/docs/guides/setup... (2 failed, 43 saved, ~32k tokens)
```

- Progress line updates in place using `\r` carriage return
- Failures print on separate lines (persist in scroll history)
- Stats include: failed count, saved count, token estimate

**Location:** `cmd/locdoc/main.go` in `crawlProject()`

### 3. Configurable Concurrency

Add `--concurrency` / `-c` flag:

```
locdoc add --concurrency 30 nx https://nx.dev/docs
```

- **Default:** 10 (lower than current 20 for safety)
- **Rationale:** 20 Chrome instances is memory-heavy; users can increase if needed

**Location:** `cmd/locdoc/main.go` in `ParseAddArgs()` and `crawlProject()`

### 4. Fetch Timeout

Set explicit 30-second timeout per page:

- Long enough for JS-heavy doc sites
- Short enough that stuck pages don't block retries
- With retry: worst case 90s total per URL (3 × 30s)

**Location:** `rod/fetcher.go` - add timeout to page navigation

## Implementation Order

These changes are independent and can be implemented in any order:

1. **Fetch timeout** - Smallest change, foundational for retry logic
2. **Retry logic** - Builds on timeout, biggest reliability win
3. **Configurable concurrency** - Simple flag addition
4. **Progress reporting** - Most code change, best UX improvement

## Not Included

- Persistent failure tracking / `locdoc retry` command (future work if needed)
- Configurable timeout flag (can add later if 30s proves wrong)
- Rate limiting (rely on concurrency + site's own throttling)
