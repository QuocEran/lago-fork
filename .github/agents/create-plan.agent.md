---
name: create-plan
description: Create detail implementation plan for beads epics/issues
tools: [vscode, execute, read, agent, edit, search, web, browser, todo]
---

# Create Implementation Plan from Beads Epics/Issues

You are tasked with creating a detailed implementation plan for the Beads epics and issues to guide workers through development.

## Step 1: Load Current Issues

First, get the current state:

```bash
bd list --json
bd ready --json
```

If specific IDs were provided, focus on those. Otherwise, review all issues and pick the first one ready for implementation (based on order, priority, and status).

## Step 2: Analyze Issues and Extract Tasks

For EACH issue, identify and extract the following:

- Key technical tasks required for implementation
- Dependencies between tasks
- Required files/modules to modify
- Potential edge cases or complexities
- Any missing information that would be needed for implementation

## Step 3: Create Implementation Plan

- Using /plan command, create a structured implementation plan. The plan should include:
  - A overview of the issue and its goals
  - High-level architectural approach to implementation with brief summary of how the issue will be solved and mermaid diagrams.
  - A breakdown of the identified tasks with clear descriptions
    - Brief summary
    - (Optional) Mermaid diagrams if helpful
  - Dependencies between tasks
    - Any important notes or considerations for each task
  - List tasks (todo) and excution order based on dependencies

- Save the plan in a markdown file named `lago-fork-<issue-id>.md` in the `.github/plans/` directory.

## Step 4: Implement the Plan

- Implement the plan by executing the tasks in order.
- After complete the plan, summarize the implementation to the plan file (in `.github/plans/` directory).

## Step 5: Update Issue Status

- Review the bd issue and update its status to reflect the current state of implementation (e.g., in-progress, review, done).
- Add any relevant comments or notes to the issue for context.
- Reminder: run `bd dolt push` and `git push` before ending the session.

## Step 6: Handoff to Next Agent

- Run `/compact` the session context and end the session
- Include any relevant information or context in the handoff prompt.
- Stop the session and ask me what next issue id to work on.
