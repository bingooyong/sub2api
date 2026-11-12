# Technical design

## Release integration

Keep one task because the manual transport repair and UI must be verified against the merged release together. Create a backup ref for baseline HEAD and a new `v0.2.21-hotfix1` branch. Merge pinned v0.2.21 with both parents preserved. Resolve the three previewed conflicts semantically: union ignore rules, fork VERSION, and admin-key scope additions from both sides. Review automatically merged identity/WS/forwarding paths against local specs. Never rename already published migrations; duplicate numeric prefixes are resolved by the existing full-filename migration identity contract.

## Manual data flow and boundaries

`Quality Operations -> manual comparison component -> typed probe client -> existing admin POST /accounts/:id/pelican-test -> AccountTestService -> credential/proxy/identity-aware upstream request -> SSE decoder -> per-account cards`.

The page owns whether the manual panel is open. A focused batch component owns account selection, shared configuration, per-run state and bounded queue. The client boundary owns SSE decoding and the typed result, including terminal failures, empty/truncated streams and cancellation. Reuse existing canonical prompts and `extractPelicanHtml`; do not duplicate the obsolete modal's parser in several consumers. Update or share its parser only when needed by the new production path, avoiding a broad cleanup of unrelated components.

Use an immutable batch configuration plus per-account snapshots. States are `pending`, `running`, `success`, `error`, `cancelled`; no state means the model passed a quality verdict. A worker queue limits active requests to 1–8 independently of selection count. Cancellation aborts active fetches and marks queued work cancelled. A batch generation token prevents stale callbacks from replacing new results. Close/unmount follows the same stop function. Results live in the component/page session; no new persistence schema is required.

## Backend options and non-mutation

Consume the existing pelican context options at the actual payload builders. Ordinary connection probes keep their default payload and state handling. Manual mode validates nonempty/size-bounded prompt and model and known effort, and rejects unsupported media/provider modes. Responses uses `reasoning.effort`; Chat Completions uses the protocol's supported effort field. Update the shared option plumbing carefully because scheduled pelican/quality calls also use this entry point.

Separate probe-induced health mutation from scheduled quality actions. Existing pelican options establish an observational probe context; guard probe extra writes, 401 permanent-error writes and 429 reconciliation on this context. Scheduled quality isolation/restore still occurs only in its existing runner/repository after a valid judge result and lease checks. Credential refresh/task registration needed for authentication is a distinct compatibility mechanism and must not be accidentally treated as a quality action; record and test any special account-mode boundary before exposing it.

Keep existing forwarding transport construction, proxy/TLS settings, credential/model mapping and account explicit headers. This is not a new general-purpose gateway endpoint. Endpoint validation and tests must reject models routed to unrelated image/audio/video probes rather than succeeding on a different test.

## Brand and fingerprint contracts

Audit gateway-authored plaintext at final OpenAI HTTP/WS boundaries. Do not scrub arbitrary user messages, explicit account overrides or opaque identifiers. Preserve deterministic UUID/hash namespace inputs so cache/session/tool linkage does not change. Add request-capture regression assertions for actual serialized headers/body and protect caller-identity opt-out, account overrides, WS pool key isolation and original tool call/reference IDs.

## Rendering and interpretation

Reuse `pelicanHtml.ts` to parse HTML/fenced SVG and insert its deny-network CSP. Every preview uses an opaque-origin iframe (`sandbox="allow-scripts"`, no allow-same-origin, `referrerpolicy="no-referrer"`). Raw output remains accessible. Show completion separately from human quality comparison; candy reference is informational unless a separately tested verdict is added. Use bilingual labels. Avoid fabricated token/quota fields when the existing SSE stream has no usage.

## Compatibility, risk and rollback

No new database migration is expected for manual detection. The upstream release brings migrations and dependency changes requiring frozen-install and migration review. Existing scheduled quality UI and automatic actions retain their configuration and durable audit behavior. Roll back the feature with its separate commit; roll back the release through the baseline backup ref. Do not deploy during this task.

## Verification-driven scheduled restore correction

The expanded fresh-database integration run exposed a pre-existing mismatch: both the scheduled restore INSERT and its fixture referenced `account_groups.allowed_models`, but neither published migrations nor the Ent schema define that column. Keep the actual four-column membership contract (`account_id`, `group_id`, `priority`, `created_at`), remove the fictional field from restoration, and strengthen the existing regression to assert original priority and timestamp. This repairs the required scheduled restore preservation path without inventing a schema feature or changing migration identities.
