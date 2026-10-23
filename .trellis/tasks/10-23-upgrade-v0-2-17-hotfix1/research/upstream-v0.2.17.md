# Upstream v0.2.17 Evidence

- Local base: `cbe2803a4a435a18864aa9141a022017ba3fcb18`
- Tag object: `6d6c4e701a463dcf83c261e8428e93423d6c0b3e`
- Peeled commit: `060d0160417782cd962a0c7f460f6cd4565d4916`
- Delta from `v0.2.14`: 273 commits, 535 files, +39,989/-2,567
- Text conflicts: 7
- Shared modified paths: 30

## Production Error

Commit `a4ea8059b7023ebf774dc1a9e78604712e0ff35f`, first released in `v0.2.15`, matches:

`The encrypted content ... could not be verified. Reason: Encrypted content could not be decrypted or parsed.`

It recognizes `thinking_signature_invalid`, strips rejected encrypted reasoning, retries once, and records invalid lineage while preserving visible history and tool output.

## Release Compatibility

- `v0.2.15`: platform catalog, Cline, Command Code, encrypted reasoning recovery, protocol/billing fixes.
- `v0.2.16`: Gemini explicit cache, Claude Haiku 5.5, scheduling fixes.
- `v0.2.17`: Gemini 4 Argon, minimum output tokens, API-key web search, payment/refund fixes, Responses `created_at`, Codex workspace import isolation.
- Breaking behavior: GPT-5.5 is retired for ChatGPT OAuth/SetupToken subscription accounts; API-key accounts remain supported.

## Text Conflicts

1. `backend/cmd/server/VERSION`
2. `frontend/src/api/__tests__/settings.authSourceDefaults.spec.ts`
3. `frontend/src/api/admin/settings.ts`
4. `frontend/src/components/admin/account/AccountTableFilters.vue`
5. `frontend/src/components/admin/account/ScheduledTestsPanel.vue`
6. `frontend/src/components/keys/__tests__/UseKeyModal.spec.ts`
7. `frontend/src/views/admin/SettingsView.vue`
