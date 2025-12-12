---
description: Pick a ready beads task, create branch, and implement using TDD
allowed-tools: Bash(bd:*), Bash(git:*), Bash(make:*)
---

## Current State

Branch: !`git branch --show-current`
Uncommitted changes: !`git status --porcelain`
Beads uncommitted: !`git status --porcelain .beads/`

## In-Progress Work

!`bd list --status in_progress 2>/dev/null || echo "None"`

## Ready Tasks

!`bd ready`

## Your Workflow

### 1. Pre-flight Validation

Before proceeding, verify:
- [ ] Currently on `main` branch (if not, ask user before proceeding)
- [ ] No uncommitted changes in `.beads/` directory (if there are, commit and push them first)
- [ ] Working tree is clean (if not, ask user how to proceed)

If any checks fail, stop and resolve with the user before continuing.

### 2. Check for Abandoned Work

If there are issues with status `in_progress`:
- Show them to the user
- Ask: "Continue with existing in-progress work, or start fresh task?"
- If continuing: skip to step 4 with existing branch
- If starting fresh: ask if abandoned work should be reset to `open`

### 3. Task Selection

Present the ready tasks to the user with a brief recommendation based on:
- Task complexity and dependencies
- Logical ordering (foundational work before dependent work)

Use the AskUserQuestion tool to let the user choose which task to work on.

### 4. Branch Setup

Once user selects a task:
1. Create branch first: `git checkout -b <task-id>` (e.g., `git checkout -b locdoc-abc`)
2. Mark the task as in-progress: `bd update <task-id> -s in_progress`
3. Commit the beads change: `git add .beads/ && git commit -m "Start work on <task-id>"`
4. Show full task details: `bd show <task-id>`

**Note**: All commits happen on the feature branch, keeping main clean.

### 5. Implementation

**MANDATORY**: Use the `superpowers:test-driven-development` skill for implementation.

If the task involves any of these architectural decisions:
- Creating new packages or files
- Deciding where code belongs
- Adding new mocks or mock methods
- Package naming decisions

Then **ALSO** use the `go-standard-package-layout` skill for guidance.

Follow the RED-GREEN-REFACTOR cycle:
1. Write a failing test first
2. Implement minimal code to pass
3. Refactor if needed
4. Repeat

### 6. Progress Checkpointing

At major milestones during implementation, update beads notes:
```bash
bd update <task-id> --notes "COMPLETED: [what's done]
IN_PROGRESS: [current work]
NEXT: [immediate next step]
KEY_DECISIONS: [any important choices made]"
```

Commit beads changes with your code commits to keep them in sync.

### 7. Validation

After implementation is complete:
1. Run `make validate`
2. Address any issues that arise (linting, test failures, etc.)
3. Iterate until validation passes

Only proceed to `/finish-task` when `make validate` passes cleanly.
