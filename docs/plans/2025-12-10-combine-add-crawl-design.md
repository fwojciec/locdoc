# Combine Add and Crawl Commands

## Problem

The current CLI has separate `add` and `crawl` commands:
- `add <name> <url>` - Creates project record only
- `crawl [name]` - Crawls documentation for project(s)

This two-step flow adds friction for the common case (add a doc source and immediately crawl it). The separation was intended to support "update" workflows, but the update logic provides minimal value since we must fetch all pages anyway to detect changes.

## Design

Merge `add` and `crawl` into a single `add` command. Remove the diffing/update logic in favor of delete + recreate.

### Commands

```
locdoc add <name> <url>                            # Create + crawl (errors if exists)
locdoc add <name> <url> --force                    # Delete existing + recreate + crawl
locdoc add <name> <url> --preview                  # Show sitemap URLs, don't crawl or save
locdoc add <name> <url> --preview --filter "..."   # Show filtered URLs
locdoc add <name> <url> --filter "..."             # Store filter + crawl filtered subset

locdoc list                                        # Show projects
locdoc delete <name> --force                       # Remove project + docs
locdoc docs <name> [--full]                        # Show documents
locdoc ask <name> "question"                       # Query docs
```

### Removed

- `crawl` command (functionality absorbed into `add`)

### Flag Behaviors

| Flags | Behavior |
|-------|----------|
| (none) | Create project, crawl all URLs |
| `--force` | Delete existing project if present, then create + crawl |
| `--preview` | Show URLs from sitemap, don't create project or crawl |
| `--filter <pattern>` | Store pattern on project, crawl only matching URLs |
| `--preview --filter` | Show filtered URLs, don't create or crawl |
| `--force --filter` | Delete existing, create with filter, crawl filtered |

### Filter Pattern

- Glob-style pattern matched against URL paths
- Stored on project record for documentation (what subset was crawled)
- Examples: `*/api/*`, `*/guide/*`, `*/reference/*`

### Database Change

Add `filter` column to projects table schema:

```sql
CREATE TABLE projects (
    ...
    filter TEXT NOT NULL DEFAULT '',
    ...
);
```

Note: No migration needed - column added directly to schema since project not yet deployed.

### Why Delete + Recreate

The current `crawl` command diffs content by hash and only updates changed documents. This requires:
1. Fetching all pages (to compute hash)
2. Comparing against existing
3. Conditional insert/update logic

Since we fetch everything anyway, the diffing provides minimal benefit (saves some DB writes) while adding complexity. Delete + recreate is simpler and has the same network cost.

## Implementation Tasks

1. Add `filter` column to projects table schema
2. Update `CmdAdd` to accept `--force`, `--preview`, `--filter` flags
3. Implement preview mode (sitemap discovery + display, no crawl)
4. Implement filter matching logic
5. Move crawl logic into `CmdAdd`
6. Update `--force` to delete existing project first
7. Remove `CmdCrawl` and `crawl` command
8. Update help text and usage
9. Update tests
