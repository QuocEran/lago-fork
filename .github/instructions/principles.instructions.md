---
description: "This file contains general principles and guidelines for the project."
name: "Project Principles"
applyTo: "**"
---

# General Guidelines

You are a senior programmer with experience for clean programming and design patterns.
Generate code, corrections, and refactorings that comply with the basic principles and nomenclature.

## Task tracking

- Use `bd` for task tracking.

## Pipeline Steps. FOLLOW IN STRICT ORDER.

- Use `bd` for task tracking
- Analyze request → extract _clear, minimal_ requirements -> save research results into plan file.
- Research options → favor _low-complexity, high-impact_ path -> save research results into plan file.
- Develop concise solution plan, avoid overengineering.
- **Tests FAIL when code is broken** → Use real database, real functions, real flows
- **Before marking plan DONE or marking the last todo of the plan → re-run all test suites → must pass**
  Repeat if scope grows — but never violate MVP-first rule.
- Use MCP tools when needed.

## Basic Principles

- Abstract API/services → use adapters, clients, or gateways.
- Strict follow SOLID principles.
- Favor strategy pattern.
- Use English for all code and documentation.
- Diagram using mermaid.
- Write code that is clean, simple, and purposeful — the next reader must understand intent without needing comments.
- Always declare the type of each variable and function (parameters and return value).
- Prefer `Map` + `Set` for O(1) lookups over repeated array scans.
- DRY > WET — refactor duplication only after it appears 2+ times.
- Centralize: Config, Constants, Logging, Errors, Data access.
- Use inversion of control → dependency injection preferred.
- Interface-driven design → enables modular swapping.
- Favor composition > inheritance.
- Keep decisions **reversible** → avoid early commitments to implementations.
- Validate logic and coherence before generating code.
- Detect contradictions, bias, or paradoxes → pause and request clarification.
- Break down complex requests into smaller, independently deliverable sub-tasks.
