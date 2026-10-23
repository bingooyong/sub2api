# v0.2.17-hotfix1 Resolution Matrix

## Merge evidence

| Item | Evidence | Result |
| --- | --- | --- |
| Local base | `cbe2803a4a435a18864aa9141a022017ba3fcb18` | Preserved as first parent |
| Upstream target | `060d0160417782cd962a0c7f460f6cd4565d4916` (`v0.2.17`) | Preserved as second parent |
| Merge commit | `8934e084b` | `git show --format=raw` confirms both parents |
| Built version source | `backend/cmd/server/VERSION` | `0.2.17-hotfix1` |
| Former textual conflicts | VERSION; auth-source settings test/API; account filters; scheduled tests; key modal test; SettingsView | No conflict markers; local behavior and upstream settings retained |
| Shared paths | 30 paths changed by both parents relative to merge-base | Reviewed below; `git diff --check` clean |

The count of 30 uses the merge-base comparison (`0363b8cd`). Comparing the
resolved merge tree against each parent reports 29 because one shared settings
path is identical to the upstream side after conflict resolution.

## Shared-path review

| Path group | Paths | Resolution/evidence |
| --- | --- | --- |
| Version and generation | `backend/cmd/server/VERSION`; `backend/cmd/server/wire.go`; `backend/cmd/server/wire_gen.go` | Version explicitly set to hotfix suffix. `make generate` completed and produced no working-tree delta. |
| Dependency/config | `backend/go.sum`; `backend/internal/config/config.go`; `deploy/config.example.yaml` | Upstream dependencies and settings retained; local Codex/quality settings remain in config. |
| Account/admin API | `backend/internal/handler/admin/account_codex_import.go`; `backend/internal/handler/admin/account_handler.go`; `backend/internal/service/account_service.go`; `backend/internal/service/admin_account.go`; `backend/internal/service/admin_service.go` | Account import, type/plan-tier filtering, page-size behavior, scheduling, isolation, quality operations, and platform catalog paths remain present. Targeted handler/service tests pass. |
| Protocol transforms | `backend/internal/service/openai_codex_transform.go`; `backend/internal/service/openai_gateway_chat_completions.go`; `backend/internal/service/openai_gateway_forward.go`; `backend/internal/service/openai_gateway_messages.go`; `backend/internal/service/openai_gateway_passthrough.go`; `backend/internal/service/openai_ws_forwarder_ingress.go`; `backend/internal/service/openai_ws_v2_passthrough_adapter.go` | Local identity/fingerprint/session/tool-call/WebSocket contracts remain in the merged implementations. Encrypted-content recovery is present in `openai_gateway_forward.go` and covered by exact signature tests. |
| Contract tests | `backend/internal/pkg/apicompat/chatcompletions_responses_test.go`; `backend/internal/server/api_contract_test.go` | Upstream Responses compatibility and local API contracts retained; included in backend package test runs. |
| Frontend settings/account | `frontend/src/api/__tests__/settings.authSourceDefaults.spec.ts`; `frontend/src/api/admin/settings.ts`; `frontend/src/components/admin/account/AccountTableFilters.vue`; `frontend/src/components/admin/account/ScheduledTestsPanel.vue`; `frontend/src/components/keys/__tests__/UseKeyModal.spec.ts`; `frontend/src/components/user/profile/__tests__/ProfileIdentityBindingsSection.spec.ts`; `frontend/src/i18n/locales/en/admin/accounts.ts`; `frontend/src/i18n/locales/zh/admin/accounts.ts`; `frontend/src/types/index.ts`; `frontend/src/views/admin/SettingsView.vue` | Upstream platform/catalog settings and local account filters, Pelican controls, quality UI, types, and translations are present. Full frontend gates remain environment-dependent and are listed below. |

## Encrypted reasoning recovery

The merged HTTP path recognizes both the legacy `invalid_encrypted_content`
code and the production `thinking_signature_invalid` response when its message
contains the exact verification/decryption failure. It strips only encrypted
reasoning/compaction items, preserves summaries/messages/tool results, retries
once, and records encrypted-content lineage for later turns. These tests pass:

- `TestOpenAIGatewayService_Forward_HTTPThinkingSignatureRecoveryAndLineage`
- `TestOpenAIGatewayService_Forward_HTTPThinkingSignatureDoesNotRetryUnrelatedErrors`
- `TestOpenAIGatewayService_Forward_HTTPRetryRecoveryDoesNotDecodeBeforeError`
- `TestOpenAIGatewayService_Forward_HTTPRetryRecoveryDropsCompaction`

The supplied server-160 message is therefore covered without requiring a full
version upgrade to diagnose it.

## Local capability checks

- Codex identity/fingerprint/session/tool-call/WebSocket tests are present in
  `backend/internal/service` and the selected OpenAI service test run passed.
- Account type/plan-tier filters and page-size handling remain in the admin
  handler; admin handler targeted tests passed.
- Pelican scheduled tests and quality probe/judge/isolate/restore/history code
  and migrations remain present; service, handler, and migration targeted tests
  passed.
- Upstream platform catalog, Gemini, payment/security, model retirement, and
  Responses changes are present in the merged tree; relevant service/handler/
  migration test packages passed.

## Migration coexistence

Published SQL filenames were not renumbered or edited. The tree contains both
same-prefix files where required:

```text
241_add_payment_order_bonus_amount.sql
241_add_typesafe_platform.sql
242_drop_platform_check_constraints.sql
242_group_i18n.sql
243_gemini_cached_contents.sql
243_pelican_scheduled_tests.sql
244_gemini_cached_contents_channel_usage.sql
245_gemini_cached_contents_mutation_lease.sql
247_account_quality_ops.sql
248_account_quality_judgment.sql
249_account_quality_operation_history.sql
```

`schema_migrations.filename` is the primary key and the runner sorts and
records the complete filename, so numeric-prefix collisions do not overwrite
each other. Migration package tests passed. A live PostgreSQL upgrade from a
representative `v0.2.14-hotfix1` database and a second-start idempotency run
were not available in this environment and remain release-gate follow-ups.

## Verification record

Passed:

```text
make generate                         (backend; no generated diff)
go test ./internal/service ./internal/handler/admin ./migrations \
  -run 'Test.*(Codex|OpenAI|Account|Pelican|Quality|Plan|Platform|Migration|Gemini|EasyPay|Payment|ModelRetirement)' -count=1
go test ./internal/service -run 'TestOpenAIGatewayService_Forward_HTTP(ThinkingSignature|RetryRecovery)|TestOpenAI.*Encrypted|TestOpenAI.*Codex|TestOpenAI.*WS' -count=1
git diff --check
```

The selected backend run passed for `internal/service`, `internal/handler/admin`,
and `migrations`. The full backend test suite and production build also passed
in the main session; the binary reported `Sub2API 0.2.17-hotfix1`. Frontend
frozen install/lint/typecheck/tests/build could not run because Corepack stalled
downloading the repository-pinned pnpm 12.15.0 binary. Live PostgreSQL upgrade
tests were not available in this environment and remain release-gate follow-ups.
`.omc/` and unrelated task files are untracked and must not be included in a
commit.
