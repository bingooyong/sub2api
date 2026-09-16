# Audit Codex request compatibility and hardcoded metadata

## Goal

Audit hardcoded metadata in the gateway's Codex request paths and fix observable
protocol compatibility defects while preserving caller intent and tenant isolation.

## Requirements

- Inventory request headers, default instructions, bridge text, tool identifiers,
  account/session identifiers, and relevant transport/configuration behavior.
- Distinguish plaintext sent upstream from internal inputs to one-way hashes.
- Observe the installed official Codex CLI against a local dummy endpoint, recording
  only sanitized protocol metadata without using provider credentials.
- Make the existing identity-enforcement opt-out preserve supplied client metadata
  consistently across Responses HTTP, compact, passthrough, Messages, alpha-search,
  and WebSocket forwarding requests for accounts using the Codex protocol.
- Preserve valid native Responses tool-call identifiers and call/output correlation.
- Keep credential/session isolation, explicit bridge markers, legacy enforcement
  defaults, and configured account overrides stable.
- Do not implement client impersonation, anti-detection, or account-ban evasion.

## Acceptance Criteria

- [x] A repository audit explains the relevant hardcoded values and their effects.
- [x] Local capture evidence states the CLI version, platform, mode, and limitations.
- [x] With no explicit account UA override or ForceCodexCLI, opt-out forwarding
  preserves supplied valid User-Agent, originator, and version values without
  synthesizing missing Codex identities or elevating versions; WebSocket reuse
  respects those three fields. Account/session identifiers remain separately scoped.
- [x] Distinct valid call IDs remain distinct, oversized IDs remain bounded, and
  item references retain their established item-ID semantics.
- [x] Targeted regression tests, package checks, and required static validation pass,
  or an external validation limitation is documented precisely.
- [x] Existing unrelated workspace changes and global Codex configuration are preserved.

## Notes

- The workspace already contains unrelated Trellis/editor files and a modified
  `.gitattributes`; these do not belong to the implementation diff.
- Matching an HTTP user-agent cannot establish transport identity or prove anything
  about upstream enforcement decisions. No production capture or provider probe is
  needed for this compatibility work.
