---
description: "Use when handling requests that reference prompts, instruction files, or process constraints. Enforces instruction-first execution and minimal clarification for ambiguous scope."
name: "Instruction-First Execution"
applyTo: "**"
---

# Instruction-First Execution

- Before executing any request, identify and read all applicable instruction sources for the current context.
- If a request references an attached prompt or instruction file, read those files first, then execute.
- Treat this as a hard rule for all workspace tasks, including planning, implementation, and validation.
- If no clear persistent rule can be extracted from the conversation, ask targeted clarification about:
  - Scope (workspace-wide or file-specific)
  - Technology or file types affected
  - Hard rule versus preference
- Keep proposed instructions concise, single-purpose, and actionable.
- When creating a new instructions file, avoid duplicate overlap with existing instruction files.
- If constraints conflict, follow higher-priority instruction sources and call out unresolved ambiguity before proceeding.
