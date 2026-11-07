# Final Trellis review

Date: 2026-11-07

## Scope reviewed

- Merge-base review confirms exactly 23 shared paths between the local base and
  the peeled v0.2.19 release commit. The two textual conflicts remain resolved
  deliberately: local ignore rules plus upstream documentation exception, and
  `backend/cmd/server/VERSION=0.2.19-hotfix1`.
- Local Codex identity/fingerprint, native call/item ID correlation, WebSocket
  handshake/session isolation, sticky-session binding, `session_hash` context
  diagnostics, and code-null encrypted-content recovery remain present in the
  changed tree. Stable-release edits add GPT-6.1 Terra/model observation and do
  not replace those local paths.
- Scoped-admin reads/writes use an isolated principal and redacted account
  projection; account repository query calls preserve the seven-argument
  plan-tier filter contract with an empty tier for the scoped endpoint.
- Published migrations remain immutable and distinct by complete filename. The
  upgrade test covers same-number upstream 247/248 beside local 247/248/249,
  data retention, schema assertions, and idempotent restart. The old 241/242
  fixture cleanup reapplies 242 so a failed test cannot leave platform checks
  in the disposable database.
- Post-tag security backports are bounded to compression dependencies, Bedrock
  EventStream pre-allocation validation, malformed UTF-8/client-disconnect API
  key handling, and the frontend dependency security unit. Broader MCP,
  request-body timeout, resource permissions, and WS/Responses behavior changes
  remain deferred in `latest-main-triage.md`.

## Evidence

- Shared-path count: 23 (computed from `git merge-base` and both release diffs).
- `git diff --check`: passed.
- Scoped-admin repository tests: 13/13 passed against disposable PostgreSQL 18
  (`/tmp/v0219-repo-scoped-postgres.log`).
- Migration upgrade/idempotency scenarios: passed against isolated PostgreSQL 15
  (`/tmp/v0219-repo-migration-direct.log`, reproduction `/tmp/v0219-verify-migrations.py`).
- Streaming keepalive regression: five repeated runs passed.
- Frontend frozen pnpm 11 install passed with the explicit esbuild/vue-demi
  build allowlist; resolved versions are asserted in the lockfile.
- Full backend test was retried; the first full run was blocked by a transient
  EOF while downloading `github.com/klauspost/compress@v1.20.2` from
  proxy.golang.org. This is an environment/network gap, not a test assertion.

## Findings not fixed here

- A full final backend lint/test and frontend typecheck/test/build must be
  recorded by the main session after concurrent jobs are stopped; early parallel
  runs were resource-contended and must not be called green from their partial
  logs.
- Standard Docker PostgreSQL integration startup timed out; the direct isolated
  PostgreSQL migration evidence above is the available equivalent.
