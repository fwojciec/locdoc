---
description: Execute one iteration of the Ralph loop for an epic
---

## Epic: $ARGUMENTS

## Workflow

### 1. Pick Task

Find a ready task (no open blockers) within this epic:

```bash
# See all open tasks in epic
bd list --parent=$ARGUMENTS --status=open

# See which tasks are ready globally
bd ready
```

Pick a task that appears in both lists (open in epic AND ready). If no tasks are ready, create `.ralph-complete` and exit immediately.

### 2. Claim & Understand

- `bd update <id> -s in_progress`
- `bd show <id>` - read the full task description
- Read any referenced design docs or entrypoint files

### 3. Implement

Use the `superpowers:test-driven-development` skill. Follow RED-GREEN-REFACTOR strictly.

For architectural decisions (new packages, where code belongs, mocks), also use `go-standard-package-layout` skill.

### 4. Validate

```bash
make validate && make integration
```

Fix issues and retry until both pass.

### 5. Self-Review

Stage changes first so the reviewer can see the diff:
```bash
git add -A
```

Then use `superpowers:requesting-code-review` to get review from a subagent.

Use `superpowers:receiving-code-review` to evaluate feedback critically. Not all suggestions require action - apply technical judgment.

### 6. Reflect

Before closing, consider:
- Did implementation reveal anything about upcoming sibling tasks?
- Are their descriptions still accurate, or do they need updates?
- `bd update <sibling-id> --description "..."` if refinement needed

### 7. Complete

```bash
git add -A && git commit -m "Implement <id>: <summary>"
bd close <id>
git add .beads/ && git commit -m "Close <id>"
```

Exit cleanly. The loop will restart with the next task.
