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
| [Database Guidelines](./database-guidelines.md) | Migration identity, ordering, upgrade tests, additive migrations, platform whitelist ownership, and database conventions | Documented |
| [Error Handling](./error-handling.md) | Error types, handling strategies | To fill |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, forbidden patterns | To fill |
| [Logging Guidelines](./logging-guidelines.md) | Structured logging, log levels | To fill |
| [Codex Forwarding](./codex-forwarding.md) | Client metadata, socket reuse, and native tool correlation | Documented |
| [Admin Account List](./account-admin-list.md) | Account table filters, plan_tier vs type, page-size defaults | Documented |
| [Account Quality Operations](./account-quality-operations.md) | Quality probes, judge outcomes, isolation, restore, and durable audit contracts | Documented |
| [OpenAI Encrypted Reasoning Recovery](./openai-encrypted-reasoning-recovery.md) | Detecting upstream invalid encrypted content, self-healing retry, HTTP/WS parity | Documented |
| [Codex Outbound Identity Customization](./codex-identity-customization.md) | Preserving caller identity under disable_codex_identity_enforcement, upstream CLI normalization gating | Documented |

## Quality Check

For Codex forwarding changes, follow the assertion points and validation commands
in [Codex Forwarding](./codex-forwarding.md). Preserve distinct call and item ID
semantics and verify metadata at the final HTTP or WebSocket boundary.

For admin account list filter or shared table page-size changes, follow
[Admin Account List](./account-admin-list.md). Keep `type` and `plan_tier`
separate and keep plan aliases platform-scoped.

For any migration that adds tables or columns, or that touches the platform
whitelist, follow the migration scenarios in
[Database Guidelines](./database-guidelines.md). In particular: the platform
CHECK constraints dropped by `242_drop_platform_check_constraints.sql` must
not be re-added, the platform catalog in `backend/internal/domain/platforms.go`
is the single source of truth for valid platforms, and `groups.i18n` is
display-only while `name` remains the identity column.

For any change to OpenAI upstream error classification, retry, or failover
behavior, follow
[OpenAI Encrypted Reasoning Recovery](./openai-encrypted-reasoning-recovery.md).
The key constraints: the invalid-encrypted-content verdict must not depend on
`error.code` (production sends `code: null`), the HTTP and WSv2 paths must
share one verdict function, and the HTTP retry must preserve
`previous_response_id`.

For any change to outbound Codex identity headers (User-Agent / originator /
version), the `disable_codex_identity_enforcement` switch, or the default Codex
originator, follow
[Codex Outbound Identity Customization](./codex-identity-customization.md).
The key constraint: upstream's TUI-to-CLI normalization must stay gated on
`codexIdentityEnforcement`, otherwise the fork silently stops preserving the
caller's real identity.

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
