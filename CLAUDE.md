# CLAUDE.md

Strategic guidance for LLMs working with this codebase.

## Why This Codebase Exists

**Core Problem**: Developers need efficient access to library documentation when working with AI coding assistants. Current solutions require cloud services, lack local-first operation, or don't integrate well with CLI workflows.

**Solution**: A local CLI tool that crawls documentation sites, extracts content as markdown, indexes it for semantic search, and provides a CLI query interface for asking natural language questions about the documentation.

## Design Philosophy

- **Ben Johnson Standard Package Layout** - domain types in root, dependencies in subdirectories
- **Local-first** - all processing happens locally, no cloud dependencies for core functionality
- **CLI-native** - designed for terminal workflows and easy integration with AI coding assistants
- **Process over polish** - systematic validation results in quality rather than fixing issues post-hoc

## Workflows

Use slash commands for standard development workflows:

| Command | Purpose |
|---------|---------|
| `/start-task` | Pick a ready task, create branch, implement with TDD |
| `/finish-task` | Validate, close beads issue, create PR |
| `/address-pr-comments` | Fetch, evaluate, and respond to PR feedback |

**Quick reference**:
```bash
make validate     # Quality gate - run before completing any task
bd ready          # Show tasks with no blockers
bd show <id>      # Show task details
```

## Architecture Patterns

**Ben Johnson Pattern**:
- Root package: domain types and interfaces only (no external dependencies)
- Subdirectories: one per external dependency (sqlite/, crawler/, embedding/)
- `mock/`: manual mocks with function fields for testing
- `cmd/locdoc/`: wires everything together

**Data Flow**:
```
Documentation URL → Crawler → Extractor → Markdown → Embeddings → sqlite-vec → Query → LLM → Answer
```

## File Structure

```
locdoc/
├── locdoc.go               # Main domain types file
├── *.go                    # Other domain types and interfaces (pure, no deps)
├── error.go                # Application error codes
├── mock/                   # Manual mocks for testing
├── sqlite/                 # sqlite-vec storage implementation
├── katana/                 # Crawling implementation (wraps Katana)
├── trafilatura/            # Content extraction (wraps go-trafilatura)
├── ollama/                 # Embedding generation via Ollama
├── cmd/locdoc/             # CLI entry point
└── docs/                   # Research and workflow documentation
```

## Skills

### Task Tracking

**`bd-issue-tracking`** - Use for all beads operations. Covers:
- When to use bd vs TodoWrite
- Session start protocol
- Progress checkpointing and compaction survival
- Issue lifecycle and dependency management

### Architecture

**`go-standard-package-layout`** - Use when:
- Creating new packages or files
- Deciding where code belongs
- Naming packages or files
- Writing mocks in `mock/`

### Development (invoked automatically by `/start-task`)

- **`superpowers:test-driven-development`** - Write test first, watch it fail, implement
- **`superpowers:systematic-debugging`** - Understand root cause before fixing
- **`superpowers:verification-before-completion`** - Evidence before assertions

## Writing Issues

Issues should be easy to complete. Include three elements:

**Template**:
```
## Problem
[What needs to be fixed/added - high level description]

## Entrypoints
- [File or function where work starts]
- [Related files if known]

## Validation
- [ ] Specific testable outcome
- [ ] `make validate` passes
```

**Principles**:
- Write **what** needs doing, not **how**
- One issue = one PR
- Reference specific files to reduce discovery time

## Test Philosophy

**TDD is mandatory** - write failing tests first, then implement.

**Package Convention**:
- All tests MUST use external test packages: `package foo_test` (not `package foo`)
- This enforces testing through the public API only
- Linter (`testpackage`) will fail on tests in the same package

**Parallel Tests**:
- All tests MUST call `t.Parallel()` at the start of:
  - Every top-level test function
  - Every subtest (`t.Run` callback)
- Linter (`paralleltest`) will fail on missing parallel calls

**Example Pattern**:
```go
package sqlite_test  // External test package

func TestFoo(t *testing.T) {
    t.Parallel()  // Required

    t.Run("subtest", func(t *testing.T) {
        t.Parallel()  // Also required
        // test code...
    })
}
```

**Assertions**:
- Use `require` for setup (fails fast)
- Use `assert` for test assertions (continues on failure)
- Use `assert.Empty(t, slice)` not `assert.Len(t, slice, 0)`

## Linting

golangci-lint enforces:
- No global state (`gochecknoglobals`) - per Ben Johnson pattern
- Separate test packages (`testpackage`)
- Error checking (`errcheck`) - all errors must be handled

## Reference Documentation

- [docs/extracting-documentation-links.md](docs/extracting-documentation-links.md) - Crawling research
- [docs/local-rag.md](docs/local-rag.md) - RAG implementation research
- `.claude/commands/` - Workflow commands
