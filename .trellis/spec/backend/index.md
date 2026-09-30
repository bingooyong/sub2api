# Backend Development Guidelines

> Best practices for backend development in this project.

---

## Overview

This directory contains guidelines for backend development. Fill in each file with your project's specific conventions.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Module organization and file layout | To fill |
| [Database Guidelines](./database-guidelines.md) | ORM patterns, queries, migrations | To fill |
| [Error Handling](./error-handling.md) | Error types, handling strategies | To fill |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, forbidden patterns | To fill |
| [Logging Guidelines](./logging-guidelines.md) | Structured logging, log levels | To fill |
| [Codex Forwarding](./codex-forwarding.md) | Client metadata, socket reuse, and native tool correlation | Documented |
| [Admin Account List](./account-admin-list.md) | Account table filters, plan_tier vs type, page-size defaults | Documented |
| [Account Quality Operations](./account-quality-operations.md) | Quality probes, judge outcomes, isolation, restore, and durable audit contracts | Documented |

## Quality Check

For Codex forwarding changes, follow the assertion points and validation commands
in [Codex Forwarding](./codex-forwarding.md). Preserve distinct call and item ID
semantics and verify metadata at the final HTTP or WebSocket boundary.

For admin account list filter or shared table page-size changes, follow
[Admin Account List](./account-admin-list.md). Keep `type` and `plan_tier`
separate and keep plan aliases platform-scoped.

---

## How to Fill These Guidelines

For each guideline file:

1. Document your project's **actual conventions** (not ideals)
2. Include **code examples** from your codebase
3. List **forbidden patterns** and why
4. Add **common mistakes** your team has made

The goal is to help AI assistants and new team members understand how YOUR project works.

---

**Language**: All documentation should be written in **English**.
