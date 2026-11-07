# v0.2.19-hotfix1 Resolution Matrix

## Merge evidence

- Local base: `c2c75842d6cdebbe8422286dd19afea035baf45a`
- Backup: `backup/v0.2.18-hotfix1-before-v0.2.19`
- Upstream annotated tag object: `e60cefeaf78f2fbfa95aa076029e28b40d0a5527`
- Peeled release commit: `dece26153621959ccd7635814f4b2d08304e681a`
- Merge: `git merge --no-ff dece26153621959ccd7635814f4b2d08304e681a`
- Hotfix version: `0.2.19-hotfix1`
- Text conflicts: `.gitignore`, `backend/cmd/server/VERSION`
- Shared paths since merge base: 23

## Conflict decisions

| Path | Decision |
| --- | --- |
| `.gitignore` | Keep local Trellis, handoff, audit, and generated-directory rules; add upstream `AGENTS.md` and `docs/admin-api-keys.md` exceptions. |
| `backend/cmd/server/VERSION` | Set deliberately to `0.2.19-hotfix1`; do not retain upstream release-only value. |

## Shared-path review

| Area | Paths | Decision |
| --- | --- | --- |
| Config/generated | `backend/cmd/server/wire_gen.go`, `backend/go.sum`, `backend/internal/config/config.go`, `backend/internal/service/wire.go` | Keep upstream scoped-admin-key wiring and Gemini retry config; regenerate and verify no drift. |
| Admin/account | `backend/internal/handler/admin/account_handler.go`, `backend/internal/repository/account_repo.go`, `backend/internal/service/admin_account.go`, `backend/internal/service/admin_service.go`, `backend/internal/server/routes/admin.go` | Preserve local plan-tier/account filters and quality operations; retain upstream Grok diagnostics and scoped admin API-key access. |
| Codex/OpenAI | `backend/internal/service/openai_codex_transform.go`, `openai_gateway_chat_completions.go`, `openai_gateway_messages.go` | Keep local identity/fingerprint/session/tool-ID/WebSocket and encrypted-content recovery; add GPT-6.1 Terra and upstream model-audit behavior. |
| API contracts | `backend/internal/server/api_contract_test.go`, `frontend/src/api/admin/accounts.ts`, `frontend/src/types/index.ts` | Reconcile new admin-key/Grok contracts with existing local account and quality types. |
| Frontend | `frontend/src/components/keys/__tests__/UseKeyModal.spec.ts`, admin account locales/index files, `frontend/src/views/admin/SettingsView.vue` | Keep local settings/filter behavior and add upstream admin-key/platform UI strings and settings. |

## Stable-release changes retained

- Scoped admin API keys: hashed secret, scope policy, expiry, IP allowlist, revoke/audit.
- Per-image billing and Gemini `fileData` image accounting.
- DeepSeek cache-hit billing/pricing and image-tier token retention.
- Pagination overflow and multipart upload hard limits.
- Admin guard 400 behavior, dependency security update, and web-search dimension preservation.
- GPT-6.1 Terra, OpenCode DeepSeek role compatibility, Gemini retry/model-list fixes, Grok catalog consistency.

## Local capability checks

- `session_hash`, sticky session, encrypted-content recovery, Codex fingerprint/identity/WebSocket remain present.
- Account plan-tier/page-size filters, Pelican scheduled tests, account isolation, quality judgment/restore/history remain present.
- Upstream `247_scoped_admin_api_keys.sql` and `248_group_image_count_billing.sql` retain their complete filenames alongside local migration history.

## Compatibility notes

- Gemini default model lists remove deprecated 2.x/2.5 and 3-pro-preview models; deployments that require them need explicit mappings/whitelists.
- Migration 246 changes Gemini/Antigravity image-generation defaults; composite groups require explicit operator review.
