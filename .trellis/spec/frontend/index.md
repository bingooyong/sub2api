# Frontend Development Guidelines

> Best practices for frontend development in this project.

---

## Overview

This directory contains guidelines for frontend development. Fill in each file with your project's specific conventions.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Module organization and file layout | To fill |
| [Component Guidelines](./component-guidelines.md) | Component patterns, props, composition | To fill |
| [Hook Guidelines](./hook-guidelines.md) | Custom hooks, data fetching patterns | To fill |
| [State Management](./state-management.md) | Local state, global state, server state | To fill |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, forbidden patterns | To fill |
| [Type Safety](./type-safety.md) | Type patterns, validation | To fill |
| [Build Toolchain](./build-toolchain.md) | pnpm 11 settings migration, frozen lockfile, build script allowlist, dist artifact | Documented |

---

## Quality Check

For any change to dependencies, overrides, or the build pipeline, follow
[Build Toolchain](./build-toolchain.md). The key constraints: pnpm 11 reads
`pnpm.*` settings from `pnpm-workspace.yaml` (not `package.json`), the override
set must stay byte-equivalent to what `pnpm-lock.yaml` records, and
`backend/internal/web/dist` must be produced from the same source snapshot as
the server binary.

## How to Fill These Guidelines

For each guideline file:

1. Document your project's **actual conventions** (not ideals)
2. Include **code examples** from your codebase
3. List **forbidden patterns** and why
4. Add **common mistakes** your team has made

The goal is to help AI assistants and new team members understand how YOUR project works.

---

**Language**: All documentation should be written in **English**.
