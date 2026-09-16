# Design

## Observed gaps

The existing identity opt-out still changes client names and versions. HTTP header
allowlists and the WebSocket builder discard `version`, compact builders invent it,
and socket compatibility ignores the identity sent during the handshake. Separately,
the native Responses transform normalizes valid opaque call IDs: `call_abc` and
`fc_abc` can both become `fc_abc`.

## Boundaries

Repair the existing opt-out at the final outbound forwarding boundaries. Keep the
shared identity enforcer's behavior for synthetic callers. Identity metadata here
means User-Agent, originator, and version. Preserve
supplied syntactically valid values, including unknown clients and independently
declared versions, and do not manufacture absent Codex metadata in that mode.
The underlying HTTP library may still supply its own default User-Agent.
Apply the projection after identity enforcement and before applicable account header
overrides. Explicit account UA overrides, forced CLI mode, and synthetic probe,
authentication, and Live identities retain their existing settings/flows. Forwarded
Responses, compact, passthrough, Messages, both alpha-search builders, and WebSocket
handshakes use this contract for Codex-protocol accounts. Ordinary API-key forwarding
is unchanged. Include all effective identity header values in WebSocket connection
compatibility while opted out; distinguish missing and explicitly empty fields.

Enable the existing PreserveToolCallIDs option for native Responses. Use that same
option for cross-turn reference fallback, keeping actual item IDs independent of
call IDs and retaining deterministic compression over 64 bytes. Do not rename hash
domains: this would change stable identifiers without removing a plaintext field.

Legacy enforcement defaults, convergence modes, default instructions, image/Spark
bridge delimiters, reserved Python tool mapping, TLS configuration, and account/API
key isolation are audited rather than changed in this patch.

## Ownership

- Header lane: identity projection helper; HTTP service/header builders; Messages
  and alpha-search final header boundaries; passthrough and WS header/pool code;
  corresponding header and pool regression tests.
- Payload lane: Codex transform and transform/reference tests; a dedicated native
  Responses call-ID forwarding regression file.
- Leader: this plan, loopback capture, audit documentation, review and integration;
  update the existing WS native-item regression to retain call IDs independently.
- `openai_gateway_forward.go` belongs to the header lane; its native transform option
  is changed there on behalf of the payload lane to avoid shared-file editing.

## Validation

Capture only a controlled loopback request with a dummy provider key, no real API
traffic, and a canned text response. Record allowed headers and body shape, with
identifier values and credentials excluded. This demonstrates application-layer
behavior on this installed CLI/platform only; it is not a TLS or production trace.

Add regression cases before implementation and record their failures. Preserve the
existing enabled-enforcement tests; update opt-out expectations to their new stated
contract. Then run focused service/config/OpenAI package checks and
static analysis. No new dependencies or data migration is required.
