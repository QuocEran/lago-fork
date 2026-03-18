---
agent: agent
description: Plan review and update
argument-hint: 1. <plan-id>; 2. <optional> Specific changes need to be reviewed and updated
---

# Plan Review and Update

You are tasked with reviewing and updating the implementation plan for the Beads epics and issues to ensure it is accurate and up-to-date.

## Step 1: Load Current Plan

Load the current implementation plan for the specified plan ID or from current chat session if no ID is provided.

## Step 2: Review the plan

The plan should include:

- A overview of the issue and its goals
- High-level architectural approach to implementation with brief summary of how the issue will be solved and mermaid diagrams.
- A breakdown of the identified tasks with clear descriptions
  - Brief summary
  - (Optional) Mermaid diagrams if helpful
- Dependencies between tasks
  - Any important notes or considerations for each task
- List tasks (todo) and excution order based on dependencies

Review the plan for accuracy and completeness. Pay special attention to any specific changes that were mentioned in the argument hint.

## Step 3: Update the plan

If any inaccuracies or outdated information are found during the review, update the plan accordingly. Make sure to:

- Correct any inaccuracies
- Add any missing information
- Update any outdated information
- Ensure the plan is clear and actionable for workers

Update the plan into .github/plans/ directory with the same plan ID or create a new plan if no ID was provided.
