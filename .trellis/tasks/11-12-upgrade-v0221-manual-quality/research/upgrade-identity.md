# Research: v0.2.21 upgrade and outgoing GPT identity preservation

- Query: Which fork identity/fingerprint/tool-correlation fixes must survive v0.2.21, and does gateway-generated outgoing GPT traffic contain plaintext `sub2api`?
- Scope: internal source audit plus parent-provided upstream diff
- Date: 2026-11-12
- Baseline: parent reports current HEAD `8bb5def3d`; upstream v0.2.21 pinned to `10ba8f9b4dc2e6023d4a9879ab77b21841babd10`.

## Findings

### Files and preservation contracts

| Source | Responsibility / evidence |
| --- | --- |
| `backend/internal/service/openai_codex_identity.go:162` | TUI-to-CLI normalization remains gated by `codexIdentityEnforcement`; disabling enforcement must preserve caller identity. |
| `backend/internal/service/openai_codex_identity.go:268` | `preserveCodexClientIdentityHeaders` restores all valid original User-Agent/originator/version values, preserves absent fields, skips ordinary API-key/synthetic requests, and honors explicit account UA / ForceCodexCLI precedence. |
| `backend/internal/service/openai_gateway_forward.go:1561` | Main HTTP builder restores caller identity after legacy enforcement. Must remain before the intentional OpenCode/Command Code UA helper and applicable account overrides. |
| `backend/internal/service/openai_gateway_passthrough.go:721` | Raw passthrough has its own final identity restoration. |
| `backend/internal/service/openai_gateway_messages.go:390` | Messages outer forwarding restores identity after inner builder modifications. |
| `backend/internal/service/openai_alpha_search.go:286`, `:427` | Both search request builders restore identity. |
| `backend/internal/service/openai_ws_forwarder_payload.go:182` | WS handshake preserves caller identity. |
| `backend/internal/service/openai_ws_pool.go:2367` | Reuse compatibility includes quoted full multi-value UA/originator/version when enforcement is disabled; absent/empty and value boundaries remain distinct. |
| `backend/internal/service/openai_ws_pool.go:1151`, `:2158`, `:2339` | Compatibility participates in acquisition, connection creation, and request equivalence, rather than only a standalone helper. |
| `backend/internal/service/openai_gateway_forward.go:509`, `:519` | Native Responses transforms explicitly opt into `PreserveToolCallIDs: true`. |
| `backend/internal/service/openai_codex_transform.go:332`, `:1612`, `:1664`, `:1684` | Preserve flag reaches input filtering and cross-turn/item-reference mapping. Call IDs remain independent of item-ID rules. |
| `backend/internal/service/openai_codex_transform.go:131` | Oversized IDs retain stable SHA-256 compression namespace and existing pairing. |
| `backend/internal/service/openai_ws_forwarder_ingress.go:100`, `:339` | Native WS stages account-specific fingerprint IDs once, then applies them to normalized frames. |
| `backend/internal/service/openai_ws_v2_passthrough_adapter.go:771`, `:1030` | V2 adapter applies convergence to first and later frames. Preserve account scoping before convergence and per-frame original body-session capture. |
| `backend/internal/service/openai_codex_fingerprint.go:254`, `:262`, `:272` | Stable installation/session/thread derivation namespaces must remain unchanged. |
| `backend/internal/service/openai_codex_account_identity.go:100` | API-key/account isolation namespace must remain unchanged. |

### Plaintext gateway branding audit

No newly identified gateway-generated plaintext `sub2api` marker was found in the inspected outgoing GPT inference header/body construction. This is source evidence, not a live traffic capture or an exhaustive proof for arbitrary administrator-configured overrides.

Existing generated compatibility markers are already neutral:

- `openai_codex_transform.go:148` uses `<codex-image-generation-bridge>`; `:150` uses `<codex-spark-image-unsupported>`.
- `openai_messages_todo_guard.go:11` uses `<claude-code-todo-guard>`.
- `openai_codex_tool_names.go:15` uses `python__codex` for reserved Python tool compatibility.
- `openai_gateway_service.go:72` and `:92` define forwarding header allowlists. Local admin/diagnostic headers are not blindly forwarded.

Remaining matching literals were classified by their actual use:

| Literal location | Actual use | Action |
| --- | --- | --- |
| `openai_codex_fingerprint.go:254/262/272`, `openai_codex_account_identity.go:100`, `openai_codex_transform.go:131`, `openai_compact_probe.go:163/165` | Hash/UUID derivation input; only derived values go upstream | Preserve byte-for-byte to avoid changing stable identities. |
| `openai_codex_models_service.go:331/458/472/489/503` | Client-facing model manifest description, rather than outgoing inference instructions | Outside strict upstream-emission change; do not mistake these descriptions for evidence of an upstream leak. |
| `openai_live_attestation.go:76` | Local error when no DeviceCheck provider exists; `openai_live.go:141` checks this before upstream creation | No successful outgoing request contains this error reason. |
| `openai_live_attestation.go:29` | AES key derivation namespace | Preserve for decryptability. |
| `openai_first_output_timeout.go:54` | Local temporary filename | Not upstream traffic. |
| `openai_gateway_grok_cache.go:19` | Grok client opt-in header, consumed locally with whitelist forwarding | Not GPT request branding. |
| `account_test_service.go:92` | Default Grok TTS probe text | Not GPT/pelican traffic. |
| `openai_images.go:51` | Local environment-variable name | Not upstream traffic. |

Do not globally replace user-authored `sub2api` strings in messages, tool results, or caller identity; that changes user intent and breaks the existing preservation contract. Likewise, administrator-configured custom UA/headers/instruction templates remain explicit user input.

Recommended implementation addition: outbound-boundary regression tests using neutral client input, asserting case-insensitive absence of `sub2api` in the actual serialized request headers/body and WS frames, while preserving a separate case proving caller-authored text is untouched. Cover manual probe payloads as well as bridge-generated markers. Existing marker tests mostly assert the desired marker is present; they do not establish a comprehensive forbidden-brand boundary.

### v0.2.21 upstream impact

Read `.trellis/tasks/11-12-upgrade-v0221-manual-quality/research/upstream-openai-diff.md (reproduction recipe for the original patch)`, supplied by the parent. Parent also reports merge-tree text conflicts only in `.gitignore`, `backend/cmd/server/VERSION`, and `backend/internal/repository/custom_admin_key_scope.go`.

- The supplied diff does not modify `openai_codex_identity.go`, `openai_codex_fingerprint.go`, `openai_codex_account_identity.go`, `openai_codex_transform.go`, or the fork's identity-restoration call sites.
- `openai_gateway_forward.go` gains request-body memory scopes and `flushPatches`; raw patches can fall back to decoded maps. Re-run actual forwarded-body fingerprint/call-ID tests because this changes patch application, even without textual conflicts.
- `openai_gateway_request_body.go` gains `ojson` batching with bounded shape/path rules and replay fallback. Preserve exact unrelated request content and existing metadata patches during this migration.
- `openai_ws_pool.go` adds first-response hosted-search state (`responseCreateSent`, `initialWebSearch`) without modifying handshake identity key logic. Test both identity isolation and new hosted-search connection rules together.
- Model metadata expands GPT-6.1-Terra capabilities/effort behavior. Manual test UI should allow an editable model and valid reasoning options; avoid inferring account degradation from a model/effort validation or transport failure.
- `account_test_service.go` changes Anthropic URL construction and image probe fields; supplied patch does not remove dedicated pelican prompt options.

## Targeted validation plan

Commands below are proposed for implementation verification, not executed by this research agent:

```bash
cd backend
go test ./internal/service -run 'IdentityOptOut|CodexIdentity|PassthroughFingerprint|WSFingerprint|CodexFingerprint|FingerprintIDs|ApplyCodexFingerprint|ResolveCodexFingerprint|StageCodexFingerprint|BuildUpstreamRequestNamespacesCodexIdentity|ApplyCodexOAuthTransform|FilterCodexInput' -count=1
go test ./internal/service -run 'OpenAIWSConnPool|WebSearch|RequestView|RequestBodyMemory|Pelican' -count=1
go test ./internal/pkg/openai -count=1
```

Key existing regression files:

- `openai_gateway_service_identity_test.go:16/91/122/140/162/175`: complete/malformed/missing metadata, API-key/synthetic parity, bridge omission, explicit overrides.
- `openai_ws_pool_test.go:812`: actual Acquire rejects changed or multi-value handshake identity and distinguishes missing versus explicit empty values.
- `openai_ws_v2_fingerprint_test.go:29`: actual native ingress and adapter writes across first/later response.create and session.update, all modes, custom cache, API-key guard.
- `openai_ws_v2_fingerprint_test.go:125`: frame-local snapshot bookkeeping and account guard.
- `openai_oauth_passthrough_test.go:2127/2187/2249/2293`: HTTP transformed/raw metadata parity; compact and Messages cache-key boundaries.
- `openai_codex_transform_test.go:193/229/338/370`: opaque native ID preservation and bounded deterministic pairing.
- `openai_codex_function_call_id_test.go:128/141/165`: native item references, item-ID precedence, cross-turn preservation.

After targeted checks, run full backend unit/integration/lint checks required by project specs. Test global identity switches serially and restore values via cleanup.

## Related specs

- `.trellis/spec/backend/codex-forwarding.md` (identity metadata, native IDs, WS parity, gateway-generated marker policy).
- `.trellis/spec/backend/codex-identity-customization.md` (enforcement-off behavior and normalization gate).
- `.trellis/spec/backend/account-quality-operations.md` (unknown/error must not mutate account; scheduled operations remain separate).

## Caveats / Not Found

- Research role prohibits git operations. Upstream impact relies on the parent-provided patch and pinned revision rather than independently executing git diff; parent owns merge and complete conflict analysis.
- No live GPT calls, packet captures, tests, or database mutations were performed.
- Scope inspected service `openai*.go`, account probe code, OpenAI/APICOMPat packages, bundled service prompts, relevant specs/tests, and the supplied patch. User/config/plugin-generated values cannot be promised brand-free without changing their contracts.
