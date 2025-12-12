---
description: Validate, close beads issue, and create PR for current task
allowed-tools: Bash(bd:*), Bash(git:*), Bash(gh:*), Bash(make:*)
---

## Current State

Branch: !`git branch --show-current`
Git status: !`git status --porcelain`
Beads uncommitted: !`git status --porcelain .beads/`

## Your Workflow

### 1. Final Validation

Run `make validate` (the full validation suite).

If any issues arise:
- Fix them systematically
- Re-run validation
- Do not proceed until validation passes cleanly

### 2. Commit Outstanding Work

Ensure all implementation work is committed:
- [ ] No uncommitted code changes
- [ ] No temporary files or debug artifacts
- [ ] All commits have meaningful messages

### 3. Close Beads Issue

Extract the task ID from the current branch name (format: `locdoc-XXX`).

1. Close the issue: `bd update <task-id> -s closed`
2. Commit beads change immediately:
   ```bash
   git add .beads/ && git commit -m "Close <task-id>"
   ```

This ensures beads state is committed BEFORE PR creation, so it's not left behind if something fails.

### 4. Verify Clean State

Before creating PR, verify:
- [ ] `git status --porcelain .beads/` shows nothing
- [ ] `git status --porcelain` shows nothing
- [ ] All work is committed

### 5. Create Pull Request

**MANDATORY**: Use the `superpowers:finishing-a-development-branch` skill to guide PR creation.

For any branch with code changes:
- Push the branch to origin
- Use `gh pr create` with a clear title and description
- Include a summary of changes and test plan in the PR body

### 6. Final Verification

After PR creation:
- [ ] Branch is pushed to origin
- [ ] PR is created and URL is shared with user
- [ ] `git status` is completely clean
- [ ] `.beads/` has no uncommitted changes
- [ ] Beads issue shows as `closed` in `bd show <task-id>`

Report the PR URL to the user.
