# Backend review

Baseline: `808c8c733`; read-only review of prior task functional changes and token/model boundaries. Verdict: REQUEST CHANGES, two medium findings.

## B1 — Known non-text models reach text transports

- Anchor: `backend/internal/service/pelican_test_service.go:75-84`.
- Trigger: request or mapped target uses `gpt-4o-transcribe`, `gpt-4o-mini-transcribe`, `gpt-4o-mini-tts`, `omni-moderation-*`, or `text-moderation-*`.
- Existing predicate omits these known non-text families and dispatches Responses/Chat Completions instead of rejecting unsupported modes.
- Related evidence: `openai_codex_models_service.go:730-748` excludes audio/realtime/transcribe/tts; `composite_platform.go:130-140` classifies embedding/moderation/image/tts/whisper; existing manifest tests filter non-text families.
- Repair: normalize provider-qualified mapped targets and conservatively deny known non-text families. Preserve arbitrary custom text model names; no GPT-only allowlist. Add table-driven zero-upstream-request negatives and custom/provider-qualified text positives.

## B2 — OAuth probes bypass valid credential resolution

- Anchor: `backend/internal/service/account_test_service.go:861-868` directly reads snapshot access_token.
- Normal forwarding uses injected `OpenAIGatewayService.GetAccessToken`, including cache/version/refresh handling. A stale snapshot therefore yields false authentication failures during quality detection.
- Production wiring sets the gateway on AccountTestService in `service/wire.go:259-286`.
- Do not reuse the provider without an observation boundary: `openai_token_provider.go:157-169` calls `disableAccountMissingRefreshToken` for an expired token without refresh_token; that helper at `:273-306` blocks runtime scheduling, sets a permanent account error and deletes the token cache.
- Existing provider refresh policy may fall back to the old token on refresh failure. Probes must return the credential failure instead of dispatching known stale credentials.
- Repair: Pelican requests resolve through the existing gateway/provider with their observational context. Guard missing-refresh health mutation and old-token fallback for this context; ordinary gateway behavior stays intact. Agent Identity signing remains its existing separate path. Keep explicit constructor/test compatibility where dependencies are absent.
- Successful OAuth refresh may persist credentials, token version, cache and scheduler snapshot/outbox. These are authentication lifecycle writes, distinct from forbidden probe health/error/429/extra/group/scheduling/isolation writes.
- Regressions: fresh cached token, successful refresh/persistence, expired/no-refresh, refresh failure, normal-request mutation behavior, Agent Identity/default connectivity compatibility and existing no-health tests.

## Review evidence and limits

The reviewer reports no additional account-state, scheduled restore, identity or migration finding in the assigned slice. Core identity/fingerprint hashes remain unchanged. Reviewer-reported baseline gopls (15 Go files), vue-tsc, focused ESLint, frontend 19 tests and backend manual/branding/identity/fingerprint selections passed; their exact full Go command and logs were not persisted, so these are informational baseline evidence rather than the final fix gate. Root will run fresh named checks after implementation. No live account/API calls or product edits occurred in this review.
