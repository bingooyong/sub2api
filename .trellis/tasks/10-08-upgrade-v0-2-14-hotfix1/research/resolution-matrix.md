# v0.2.14-hotfix1 Resolution Matrix

## Pinned integration evidence

- Local base: `36b75566cd79e56513f954a2d212495054c04604`
- Backup ref: `backup/v0.2.10-hotfix1-before-v0.2.14`
- Upstream tag object: `1400a7b482974d98db5b284a8b2afbe3eaf9aaef`
- Upstream peeled commit: `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d`
- Merge commit: `32f15a78e1ad9d322b35c8ef1eea794b151a0dc3`
- Merge parents: `36b75566cd79e56513f954a2d212495054c04604 0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d`

The merge completed without textual conflicts. Every path below changed on both
sides since the merge base and was therefore reviewed against both merge parents.

## Shared-path decisions

| Path | Local behavior retained | Upstream behavior retained | Resolution evidence |
| --- | --- | --- | --- |
| `backend/internal/config/config.go` | Codex identity opt-out, fingerprint/session settings, and other local gateway settings | API-key creation limits and billing inflight reservation configuration/defaults/validation | `DisableCodexIdentityEnforcement` remains; `APIKeyCreate` and `InflightReservation` are wired with defaults and validation. |
| `backend/internal/handler/admin/account_handler.go` | `plan_tier` list/bulk/ETag flow and Pelican account tests | TypeSafe available-model response | Both `PelicanTest`/`PlanTier` and the `account.IsTypeSafe()` branch are present. |
| `backend/internal/server/api_contract_test.go` | Page-size 50, plan-tier repository signatures, and local API contract fixtures | TypeSafe quotas, recharge-bonus settings, and new API-key cache contract | Contract test compiles and passes in `go test ./...`. |
| `backend/internal/server/routes/admin.go` | Pelican and account-quality routes | Claude reset-credit redemption route | All routes remain registered under authenticated admin routes. |
| `backend/internal/service/account_service.go` | Extended account repository filters and local platform behavior | TypeSafe API-key-only validation and credential-test handling | Repository signatures still carry `planTier`; TypeSafe create/test branches are present. |
| `backend/internal/service/admin_account.go` | Plan-tier filtering for list, score pool, export/bulk snapshot paths | TypeSafe API-key-only create/update validation | Both sets of branches remain and targeted plan-tier tests pass. |
| `backend/internal/service/openai_codex_transform.go` | Opaque native call-ID preservation, bounded legacy references, neutral gateway markers | GPT-6.1 Sol aliases and canonical mapping | `PreserveCallIDs` path remains; `gpt-6.1-sol` is in both exact and prefix maps. |
| `backend/internal/service/openai_gateway_chat_completions.go` | Native Responses tool-call IDs and isolated prompt-cache/session behavior | GPT-6.1 compatibility validation | `PreserveToolCallIDs: isResponsesShape` and `validateGPT61SolCompatRequest` are both applied. |
| `backend/internal/service/openai_gateway_messages.go` | Messages identity restoration, session isolation, and call-ID preservation | GPT-6.1 compatibility validation | Final outbound request calls `preserveCodexClientIdentityHeaders`; GPT-6.1 validation runs after model resolution. |
| `backend/internal/service/wire.go` | Quality judge injection into scheduled tests | Claude reset-credit redemption provider | Generated Wire output now constructs `QualityJudgeService` and calls `ProvideScheduledTestRunnerService` with it; Claude provider remains configured. |
| `deploy/config.example.yaml` | Documented `disable_codex_identity_enforcement` compatibility behavior | API-key limits and inflight reservation settings | All three sections exist with explicit defaults and failure semantics. |
| `frontend/src/i18n/locales/en/admin/accounts.ts` | Plan-tier and Pelican/intelligence-test text | Account priority, TypeSafe, and Claude reset-credit redemption text | Locale completeness is covered by the frontend build/test gate. |
| `frontend/src/i18n/locales/zh/admin/accounts.ts` | Plan-tier and Pelican/intelligence-test text | Account priority, TypeSafe, and Claude reset-credit redemption text | Chinese keys mirror the English feature surface. |
| `frontend/src/types/index.ts` | Quality/Pelican plan and result contracts | TypeSafe platform union and account priority contracts | Both TypeSafe platform values and quality-operation types are present. |
| `frontend/src/views/admin/AccountsView.vue` | `plan_tier` request snapshots and client-side incremental matching | Priority cell behavior | `AccountPriorityCell` remains rendered and every filter snapshot forwards `plan_tier`. |
| `frontend/src/views/admin/SettingsView.vue` | Page-size default 50 and local gateway settings | TypeSafe quota rows and recharge-bonus editor | Six quota platforms render; recharge bonus is loaded, sanitized, and saved. |
| `frontend/src/views/admin/__tests__/SettingsView.spec.ts` | Page-size 50 and existing local settings regressions | TypeSafe quota and recharge-bonus regressions | The merged test suite contains both sets of assertions. |
| `backend/cmd/server/VERSION` | Hotfix suffix identifies this fork build | Upstream release base is advanced to 0.2.14 | Deliberately set to `0.2.14-hotfix1`; built binary logs that exact value. |

## Configuration decisions

- Keep upstream API-key limits enabled: at most 200 active keys per user and 60
  creations per user per hour. Setting either value to `0` remains the explicit
  operator escape hatch.
- Keep inflight reservation enabled with the upstream defaults. Redis or pricing
  failure remains fail-open, while priced concurrent requests reserve against the
  cached balance. `fail_closed_on_unpriced` remains `false`.
- Preserve async settlement for requests whose API key is deleted after admission;
  repository and billing tests cover the retained owner snapshot behavior.
- Password-reset tokens use the upstream hashed, atomic single-use flow. Existing
  pre-upgrade reset links are intentionally invalidated.
- Remote Codex model discovery retains the upstream `api_key_model_discovery`
  configuration fix while local account/API-key and manifest filtering remains in
  place.

## Migration decision

The two upstream `241_*` migrations and local `243/247/248/249` migrations keep
their published filenames and contents. The runner keys records by full filename,
not numeric prefix. `TestMigrationsRunner_AppliesLowerNumberedUpstreamMigrationsAfterLocalHigherMigrations`
models the existing hotfix database by retaining the four higher-numbered records,
removing both upstream 241 records, removing `payment_orders.bonus_amount`, and
restoring the platform constraints from migration 238 before running migrations
twice. It verifies the two late files apply, both platform constraints gain
`typesafe`, and the second application is idempotent.

## Generation finding

The merge retained the pre-quality-operations `wire_gen.go` call even though the
provider signature now requires `QualityJudgeService`. Regenerating Wire produced
the missing constructor and argument. A second generation produced the same hash.
`go.sum` now contains the Wire command's required transitive module checksums.
