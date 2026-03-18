---
agent: agent
description: Review, proofread, and refine filed Beads epics and issues
argument-hint: specific epic or issue IDs to focus on
---

# Review and Refine Beads Issues

You are tasked with thoroughly reviewing, proofreading, and polishing the filed Beads epics and issues to ensure workers have a smooth implementation experience.

## Step 1: Load Current Issues

First, get the current state:

```bash
bd list --json
bd ready --json
```

If specific IDs were provided, focus on those. Otherwise, review all issues.

## Step 2: Systematic Review Checklist

For EACH issue, verify:

### Clarity

- [ ] Title is action-oriented and specific
- [ ] Description is clear and unambiguous
- [ ] A developer unfamiliar with the codebase could understand the task
- [ ] No jargon without explanation

### Completeness

- [ ] Acceptance criteria are defined and testable
- [ ] Technical implementation hints are provided where helpful
- [ ] Relevant file paths or modules are mentioned
- [ ] Edge cases and error handling are considered

### Dependencies

- [ ] All blocking dependencies are linked
- [ ] No circular dependencies exist
- [ ] Dependencies are minimal (not over-constrained)
- [ ] Ready issues exist for parallel work

### Scope

- [ ] Issue is appropriately sized (not too large)
- [ ] Large issues are broken into subtasks
- [ ] No duplicate or overlapping issues

### Priority

- [ ] Priority reflects actual importance
- [ ] Critical path items are prioritized correctly
- [ ] Dependencies and priorities align

## Step 3: Common Issues to Fix

Watch for and correct:

1. **Vague titles**: "Fix bug" → "Fix null pointer in UserService.getProfile when user not found"
2. **Missing context**: Add relevant file paths, function names, or module references
3. **Implicit knowledge**: Make assumptions explicit
4. **Missing acceptance criteria**: Add "Done when..." statements
5. **Over-coupling**: Break dependencies that aren't strictly necessary
6. **Under-specified**: Add technical notes for complex tasks
7. **Duplicate work**: Merge or link related issues
8. **Missing dependencies**: Link issues that should be sequenced
9. **Wrong priorities**: Adjust based on critical path analysis
10. **Typos and grammar**: Fix for professionalism

## Step 4: Update Issues

Use `bd update` to fix issues:

```bash
bd update <id> --title "Improved title" --json
bd update <id> --priority <new-priority> --json
bd update <id> --description "New description" --json
bd update <id> --acceptance "Acceptance criteria" --json
```

Manage dependencies separately with `bd dep`:

```bash
bd dep add <issue-id> <dependency-id> --json    # Add dependency
bd dep remove <issue-id> <dependency-id> --json # Remove dependency
bd dep tree <issue-id> --json                   # View dependency tree
bd dep cycles --json                            # Check for circular deps
```

For major rewrites, close and recreate:

```bash
bd close <id> --reason "Replaced by <new-id>" --json
bd create "Better title" -t <type> -p <priority> --deps <dep-id> --json
```
