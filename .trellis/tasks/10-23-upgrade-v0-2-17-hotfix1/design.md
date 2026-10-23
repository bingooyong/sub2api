# Technical Design

## Integration

Create `v0.2.17-hotfix1` from `cbe2803a4a435a18864aa9141a022017ba3fcb18`, retain a local backup ref, and merge upstream commit `060d0160417782cd962a0c7f460f6cd4565d4916` with `--no-ff`. Record tag object `6d6c4e701a463dcf83c261e8428e93423d6c0b3e` and verify the merge's second parent.

## Conflict Policy

Seven textual conflicts are known: `VERSION`, platform-quota settings test/API/view, account filters, scheduled-test panel, and Codex config-generation tests. Combine local and upstream behavior instead of choosing one parent wholesale.

Thirty paths changed on both sides. Review configuration, account handlers/services, Codex transforms/gateways/WebSocket, Wire, deployment config, frontend types/locales/settings, and account management as a cross-layer contract.

## Local Capability Preservation

Release gates include stable Codex identity/fingerprint/session/tool-call/WebSocket behavior; platform-scoped type and plan-tier filters; page size 50; Pelican scheduled tests; account quality probe/judge/isolate/restore/history; and local migrations `243`, `247`, `248`, and `249`.

Upstream platform-catalog refactoring may replace hard-coded lists. Local filters and quality operations must integrate with it without being deleted or bypassed.

## Encrypted Reasoning Recovery

The server-160 error matches upstream commit `a4ea8059b`. Recognize old `invalid_encrypted_content` and the precise `thinking_signature_invalid` message, strip only rejected encrypted reasoning, retain summary/messages/tool results, retry once, and persist invalid-content lineage.

## Migrations

Do not renumber or edit published SQL. The runner keys on full filename, so upstream and local `243_*` files coexist. Test empty DB, a representative `v0.2.14-hotfix1` DB receiving upstream `242–245` after local higher migrations, and idempotent restart.

## Compatibility

Retain upstream GPT-5.5 subscription retirement and document it. This task builds a candidate; production deployment, canary, and PITR operations are separate.
