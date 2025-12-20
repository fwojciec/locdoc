---
description: Fetch PR comments, evaluate them, and respond inline
allowed-tools: Bash(gh:*), Bash(git:*), Bash(make:*)
---

## Current Context

Branch: !`git branch --show-current`
Git status: !`git status --short`

## Your Workflow

### 1. Fetch PR Comments

Get repo and PR info:
```bash
# Get owner/repo (for API calls)
REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')

# Get PR number
PR_NUM=$(gh pr view --json number -q '.number')
```

Fetch all comments (both review comments and inline/code comments):
```bash
# General PR comments
gh pr view --comments

# Inline code review comments (note: {owner}/{repo} auto-substitutes)
gh api repos/{owner}/{repo}/pulls/$PR_NUM/comments
```

### 2. Present Summary

Provide a brief summary of all comments to the user:
- Group by type (general feedback vs. specific code comments)
- Note the reviewer and their main concerns
- Highlight any blocking vs. non-blocking feedback

**Do not pause for user feedback** - proceed directly to evaluation.

### 3. Evaluate Each Comment

**MANDATORY**: Use the `superpowers:receiving-code-review` skill for evaluation.

For each comment:
1. Read the relevant code in context
2. Consider the project's coding standards (see CLAUDE.md)
3. Evaluate technical merit objectively
4. Determine if the suggestion improves the code

Do NOT implement changes blindly. Only make changes that:
- Are technically sound
- Align with project standards
- Actually improve the code

If a suggestion is incorrect or would make the code worse, do not implement it.

### 4. Implement Valuable Changes

For changes you decide to make:
1. Implement the change
2. Run `make validate` to ensure nothing breaks
3. Commit with a clear message referencing the feedback

### 5. Respond Inline

For EVERY inline code review comment, reply using the `/replies` endpoint:
```bash
# Note: {owner}/{repo} auto-substitutes, but $PR_NUM and $COMMENT_ID must be real values
gh api repos/{owner}/{repo}/pulls/$PR_NUM/comments/$COMMENT_ID/replies \
  -f body="Your response"
```

**Important**: The `$COMMENT_ID` must be the numeric `id` field from the comment JSON (e.g., `2637288064`), not a placeholder.

For general PR comments (not inline code comments):
```bash
gh pr comment $PR_NUM --body "Your response"
```

Response format for each comment:
- **If implemented**: "Done - [brief description of change]"
- **If partially implemented**: "Partially addressed - [what was done and why]"
- **If not implemented**: "Not changing - [technical rationale]"

Be professional and constructive. Explain reasoning when declining suggestions.

### 6. Push Updates

After all changes are made:
1. Push the updated branch
2. Summarize actions taken for the user
