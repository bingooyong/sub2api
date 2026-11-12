# Release integration and identity review

Baseline: local fork `8bb5def3d` through review baseline `808c8c733`; pinned upstream `10ba8f9b4dc2e6023d4a9879ab77b21841babd10`. Verdict: APPROVE, zero confirmed findings in this slice.

## Reviewed preservation evidence

- Core `openai_codex_identity.go`, `openai_codex_fingerprint.go`, `openai_codex_account_identity.go` and `openai_codex_transform.go` are blob-identical to the pre-upgrade fork.
- Final HTTP/passthrough/Messages/alpha-search builders retain identity-restoration ordering. WS acquisition retains handshake compatibility, and first/later frames retain staged fingerprint convergence.
- Upstream request patch/memory changes preserve map replay, immutable input, reservation release and explicit budget failure. Encrypted-content recovery follows final body reconstruction without overwriting staged IDs.
- Gateway-generated inference paths have no newly confirmed plaintext branding leak. Remaining product-name literals are stable internal hash namespaces or local/client-facing values. Serialized HTTP/WS capture regressions preserve caller-authored content and reject generated branding.
- Imported migration files are upstream-identical and coexist with local full-filename migration identities, including duplicate 249 prefixes. Fresh upgrade/idempotency coverage is representative; no published migration was edited.
- Resolved admin-key scope conflict keeps the fork seventh query parameter and upstream scope filtering.
- Dependency manifests/lock match upstream; go.sum retains upstream lines plus google/subcommands. pnpm 10/11 overrides match. Version is `0.2.21-hotfix1`.

## Static diagnostics

```bash
cd backend
gopls check internal/service/openai_gateway_forward.go internal/service/openai_gateway_passthrough.go internal/service/openai_gateway_messages.go internal/service/openai_gateway_chat_completions.go internal/service/openai_ws_forwarder_ingress.go internal/service/openai_ws_pool.go internal/service/openai_ws_v2_passthrough_adapter.go internal/service/openai_gateway_request_body.go internal/repository/custom_admin_key_scope.go internal/config/config.go
```

Reviewer reports exit 0 and no type errors. Six behavior-neutral tautological non-nil analyzer diagnostics predate the integration: openai_gateway_messages.go:706:57; openai_gateway_chat_completions.go:775:57; openai_gateway_request_body.go:1652:13, 1660:13, 1687:32, 1736:13. They are outside defect repairs for this task. No tests/build/live calls or code changes in this lane; the prior archived validation remains execution evidence.

The read-only reviewer could not persist files directly; this record was saved by the root coordinator from its delivered evidence.
