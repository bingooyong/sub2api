# Audit Codex turn-state article against gateway behavior

## Goal

Help diagnose and contain suspected upstream account-level degradation while
preserving legitimate Codex protocol behavior. The task must distinguish an
upstream decision to serve a lower-capability model from a gateway model
mapping, and provide evidence that can guide account rotation or isolation.

## Background and confirmed facts

- The source article is the local Obsidian note
  `292 State 注入 — Codex 不降智、不 Overload 的底层原理与实现.md`.
- The article describes extracting a `current_turn_state` value from a claimed
  292 response, injecting it into later requests, and renewing it after a
  claimed 312 response or TTL threshold.
- The article does not provide an official protocol specification or a
  reproducible test fixture for those status codes, and the proposed flow is
  intended to bypass undocumented upstream capacity/account controls.
- The repository already handles the documented `x-codex-turn-state` response
  header in `backend/internal/service/openai_codex_turn_state.go`.
- Existing code relays the header, records the account provenance by API-key /
  client-session seed, strips a known cross-account echo during failover, and
  covers these behaviors with unit tests in
  `backend/internal/service/openai_codex_turn_state_test.go`.
- Existing passthrough and streaming paths explicitly clear stale turn-state
  headers before relaying a current upstream value.
- No repository mapping found so far automatically maps `gpt-6-astra` to
  `gpt-5.6-luna`; the gateway's explicit model mapping, compact mapping, and
  response-model observation are separate from upstream internal routing.

## Requirements

1. Trace all current `x-codex-turn-state` ingress, storage, relay, and egress
   paths relevant to HTTP, streaming, compact, and WebSocket handling.
2. Compare the implementation and tests with the article's 292/312 claims,
   clearly separating observed repository behavior from article assertions and
   unresolved upstream behavior.
3. Add or specify safe observability for upstream status codes, response model,
   service tier, request ID, and account identity so a suspected downgrade can
   be proven without exposing tokens or undocumented state values.
4. Recommend account-level containment that relies on explicit upstream error
   or overload signals and preserves normal failover semantics.
5. Preserve account isolation and transparent protocol forwarding. Do not
   implement extraction, persistence, replay, renewal, or injection of
   undocumented `current_turn_state` values for the purpose of bypassing
   upstream scheduling or degradation controls.

## Acceptance Criteria

- [ ] A written audit names the relevant implementation and test files and
      describes the current behavior with source anchors.
- [ ] Each material article claim is labeled as repository-confirmed,
      repository-unsupported, or externally unverified.
- [ ] The audit documents the behavior on missing, expired, stale, and
      cross-account turn-state values.
- [ ] A supported diagnostic path can correlate account, requested/sent/
      response model, service tier, status code, and upstream request ID.
- [ ] Any containment change has observable acceptance criteria and targeted
      validation; if no safe code change is justified, the audit says so.
- [ ] No undocumented upstream state is collected, persisted, replayed,
      renewed, or injected as part of this task.

## Scope boundary

The requested 292/312 state-extraction and injection workaround is excluded:
it would operationalize a bypass of an upstream provider's undocumented
capacity and account-routing controls. The safe deliverable is evidence and
containment through documented request/response behavior, account isolation,
and ordinary failover.
