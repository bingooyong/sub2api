# Technical Design

## Merge Strategy

Use one `--no-ff` merge of pinned upstream `v0.2.19` into a new `v0.2.19-hotfix1` branch from current `v0.2.18-hotfix1`. This preserves the complete relationship between scoped admin keys, billing changes, migrations, and their tests. Cherry-picking only individual fixes risks missing generated code, routes, schema, or related security checks.

Known textual conflicts are `.gitignore` and `backend/cmd/server/VERSION`. There are 23 shared paths requiring semantic review, including Codex transforms/gateways, account/admin services, Wire, config, frontend types/settings, and admin APIs.

## Stable-release Change Tiers

### Included in the complete release merge

- Scoped admin API keys: hashed secrets, operation/resource policy, expiration, IP allowlist, revoke/audit semantics, middleware and routes.
- Billing/data correctness: per-image billing, Gemini fileData image counts, DeepSeek cache-hit tokens/pricing, image-tier usage retention, model prices.
- Security/robustness: multipart upload hard limit, pagination overflow validation, admin guard 400 behavior, dependency security update, malformed tool-call rejection, web-search dimension preservation.
- Compatibility: GPT-6.1 Terra, OpenCode DeepSeek role normalization, Gemini model/cache retry behavior, Grok catalog consistency.

### Merge unless conflict requires local adaptation

- Grok model diagnostics and model catalog observability.
- Admin-key UI, documentation, partner assets, and other operator-facing surfaces.

### Do not blindly overwrite

- Local encrypted-content recovery and sticky session logic.
- Local `session_hash` diagnostic logging.
- Codex fingerprint/identity/WebSocket and account isolation.
- Pelican, quality operations/history, account plan-tier filters, and local migration files.

## Version and Migration Contracts

Set `backend/cmd/server/VERSION` to `0.2.19-hotfix1`; the upstream value is not the hotfix value. Keep upstream `247_scoped_admin_api_keys.sql` and `248_group_image_count_billing.sql` unchanged even if local filenames use the same numbers; migration identity is the complete filename.

## Rollback Boundary

Before push, the backup ref is the rollback point. Production rollback must preserve the new admin-key security and billing correctness fixes; use a security-preserving artifact or disable affected endpoints before reverting.

## Post-tag security backports

Audit pinned main `d22e0724d6a28435bcf21280333e89883d069a5e` separately from the stable release. Apply isolated dependency vulnerability fixes, Bedrock EventStream allocation bounds, and malformed-key/client-disconnect authentication handling with regression tests. See `research/latest-main-triage.md` for exact commits and deferred changes. Preserve local Codex contracts when evaluating larger WebSocket/Responses changes.
