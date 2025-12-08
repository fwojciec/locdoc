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

This project uses [beads](https://github.com/steveyegge/beads) for task tracking.

**Essential Commands**:
```bash
bd ready              # Show tasks with no blockers (start here)
bd list               # Show all tasks
bd show <id>          # Show task details
bd create "title"     # Create new task
bd update <id> -s closed  # Mark task complete
bd dep add <id> <blocker-id> --type blocks  # Add dependency
```

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

**Protected Branches**: Beads is configured with `protected_branches=true`. Issue metadata commits go to the `beads-metadata` branch (via git worktree), keeping main clean for PR-based workflow. Periodically merge metadata to main:
```bash
git fetch origin beads-metadata
git merge origin/beads-metadata --no-edit  # On main branch
```

## "Land This Plane" Procedure

When completing any task:

1. `make validate` - must pass
2. Clean up temporary files
3. `bd update <task-id> -s closed` - mark task complete
4. `git add -p && git commit` - atomic commit on `locdoc-XXXX` branch
5. `git push -u origin locdoc-XXXX` - push branch
6. Create PR via `gh pr create`
7. **STOP and wait for user** - pause to allow context window management
8. `bd ready` - check what's next (after user confirms to continue)

**Important**: After completing a task, always pause and wait for user confirmation before starting the next task. This allows the user to manage context, review changes, or provide feedback.

## Reference Documentation

- [docs/extracting-documentation-links.md](docs/extracting-documentation-links.md) - Crawling research (Katana, go-trafilatura)
- [docs/local-rag.md](docs/local-rag.md) - RAG implementation research (sqlite-vec, embeddings)
- [docs/ben-johnson-standard-package-layout.md](docs/ben-johnson-standard-package-layout.md) - Architecture reference
- `docs/workflow.md` - Beads workflow details (when created)
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

**Approach**:
- Use `_test` package suffix to enforce public API testing
- Use `require` for setup, `assert` for assertions
- Tests should pass with `go test -race ./...`

**Placement**: Add tests alongside implementation. Use `mock/` package for isolating components.

## Branch Naming

Use beads IDs directly: `git checkout -b locdoc-hw3`

## Linting

golangci-lint enforces:
- No global state (`gochecknoglobals`) - per Ben Johnson pattern
- Separate test packages (`testpackage`)
- Error checking (`errcheck`) - all errors must be handled
