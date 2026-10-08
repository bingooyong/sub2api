# Upstream v0.2.14 Integration Evidence

- Base head: `36b75566cd79e56513f954a2d212495054c04604`
- Tag object: `1400a7b482974d98db5b284a8b2afbe3eaf9aaef`
- Peeled commit: `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d`
- Tag type: unsigned annotated tag
- Delta from `v0.2.10`: 70 commits, 252 files, 11,946 insertions, 770 deletions
- Merge-tree result: clean automatic merge tree; semantic review remains required

## Shared Paths Since Merge Base

1. `backend/internal/config/config.go`
2. `backend/internal/handler/admin/account_handler.go`
3. `backend/internal/server/api_contract_test.go`
4. `backend/internal/server/routes/admin.go`
5. `backend/internal/service/account_service.go`
6. `backend/internal/service/admin_account.go`
7. `backend/internal/service/openai_codex_transform.go`
8. `backend/internal/service/openai_gateway_chat_completions.go`
9. `backend/internal/service/openai_gateway_messages.go`
10. `backend/internal/service/wire.go`
11. `deploy/config.example.yaml`
12. `frontend/src/i18n/locales/en/admin/accounts.ts`
13. `frontend/src/i18n/locales/zh/admin/accounts.ts`
14. `frontend/src/types/index.ts`
15. `frontend/src/views/admin/AccountsView.vue`
16. `frontend/src/views/admin/SettingsView.vue`
17. `frontend/src/views/admin/__tests__/SettingsView.spec.ts`

Review `backend/cmd/server/VERSION` separately because upstream contains the stale value `0.2.13`.

## Release Drivers

- `v0.2.11`: GPT-6.1 Sol, remote Codex catalog, API-key limits, balance inflight reservation.
- `v0.2.12`: TypeSafe, recharge tiers, priority, reset-token hardening, order-verification limiting, error sanitization, schema changes.
- `v0.2.13`: TypeSafe account creation and deleted-key settlement fixes.
- `v0.2.14`: fresh-install admin hardening, EasyPay callback/replay fix, remote catalog configuration fix.
