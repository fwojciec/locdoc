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

## Quality-First Development

**Feedback Loops**: TDD → Systematic Validation → Continuous Integration

**Standard Practice**:
```
bd ready → Pick task → Test (should fail) → Implement → make validate → Land
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

## Essential Commands

```bash
make validate     # Complete quality gate - run before completing any task
make help         # Show all available targets
bd ready          # Show tasks with no blockers
bd list           # Show all tasks
```

## Beads Task Tracking

This project uses [beads](https://github.com/steveyegge/beads) for task tracking (not GitHub Issues).

**Essential Commands**:
```bash
bd ready              # Show tasks with no blockers (start here)
bd list               # Show all tasks
bd show <id>          # Show task details
bd create "title" --description "..."  # Create new task (ALWAYS include description)
bd update <id> -s closed  # Mark task complete
bd dep add <id> <blocker-id> --type blocks  # Add dependency
```

**Always Include Descriptions**: Every issue must have a description following the template in "Writing Issues" below. Use `--description` flag or update immediately after creation.

**Task IDs**: Use `locdoc-XXXX` format (e.g., `locdoc-hw3`).

**Discovering Work**:
```bash
bd ready --json       # Machine-readable ready tasks
```

**Branch Workflow**:
```bash
bd ready              # Find next task
git checkout -b locdoc-XXXX  # Create branch named after task
# ... work on task ...
bd update locdoc-XXXX -s closed  # Mark complete when done
```

**Sync**: Git hooks handle syncing `.beads/` changes with your commits automatically.

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

## Skills

### Project-Specific

**`go-standard-package-layout`** - Use when:
- Creating new packages or files
- Deciding where new code belongs
- Naming packages or files
- Tempted to create concept-named packages (e.g., `fetcher/`, `processor/`)
- Writing new mocks or adding methods to existing mocks in `mock/`

### Development Workflows (Superpowers)

**`superpowers:test-driven-development`** - Use when:
- Starting work on any issue
- Implementing any feature or bugfix
- Write test first, watch it fail, then implement

**`superpowers:systematic-debugging`** - Use when:
- Encountering any bug or unexpected behavior
- Before proposing fixes - understand root cause first

**`superpowers:finishing-a-development-branch`** - Use when:
- Implementation complete and tests pass
- Ready to create PR or merge

**`superpowers:receiving-code-review`** - Use when:
- Addressing PR feedback
- Before implementing suggestions - verify they're technically sound

**`superpowers:verification-before-completion`** - Use when:
- About to claim work is complete
- Before committing or creating PRs
- Evidence before assertions

## Reference Documentation

- [docs/extracting-documentation-links.md](docs/extracting-documentation-links.md) - Crawling research (Katana, go-trafilatura)
- [docs/local-rag.md](docs/local-rag.md) - RAG implementation research (sqlite-vec, embeddings)
- `.claude/commands/` - Specialized workflows

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

**Placement**: Add tests alongside implementation. Use `mock/` package for isolating components.

## Branch Naming

Use beads IDs directly: `git checkout -b locdoc-hw3`

## Linting

golangci-lint enforces:
- No global state (`gochecknoglobals`) - per Ben Johnson pattern
- Separate test packages (`testpackage`)
- Error checking (`errcheck`) - all errors must be handled
