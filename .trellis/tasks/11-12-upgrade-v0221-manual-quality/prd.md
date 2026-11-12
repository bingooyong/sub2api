# Upgrade v0.2.21 and manual account quality detection

## Goal

Upgrade the customized fork to upstream v0.2.21 while preserving its request fingerprint and protocol fixes, and let administrators compare account reasoning and generated bicycle illustrations immediately from the Quality Operations page.

## Background and evidence

- Baseline: clean branch `v0.2.19-hotfix1`, HEAD `8bb5def3d`. Target release commit: `10ba8f9b4dc2e6023d4a9879ab77b21841babd10` (`v0.2.21`). The upstream delta touches 607 files; merge preview reports text conflicts in `.gitignore`, `backend/cmd/server/VERSION`, and `backend/internal/repository/custom_admin_key_scope.go`.
- `frontend/src/views/admin/AccountQualityView.vue:1` only exposes scheduled quality plans. A plan's existing Run action can isolate/restore accounts and is unsuitable for ad hoc visual comparison.
- Existing `IQTestModal.vue` is not imported by production code. `backend/internal/service/pelican_test_service.go:29` stores custom options, but `account_test_service.go:893` still builds the default `hi` payload. Existing helper tests do not cover actual upstream transport.
- `account_test_service.go:960`, `:978`, and `:981` update extras, rate-limit state, and authentication errors during ordinary connection probes; manual comparisons must avoid those quality-unrelated mutations.
- The requested cockpit checkout has no corresponding implementation. The supplied screenshot is the visual reference. Detailed evidence: `research/cockpit-manual.md` and `research/upgrade-identity.md`.

## Requirements

### R1 — Stable-release upgrade and preservation

Merge pinned upstream v0.2.21 into a new `v0.2.21-hotfix1` branch using the fork's existing release convention. Retain caller-identity opt-out, account UA precedence, WS handshake-identity partitioning, native tool/call ID correlation, account/session isolation, encrypted-content recovery, quality operations, account filters, and existing migration identities. Preserve upstream additions, including dependencies shipped in that release, without adding discretionary dependencies.

### R2 — GPT outbound identity

Gateway-authored outgoing GPT request headers, instructions, prompts, tool metadata, and literal request identifiers must not expose the product name `sub2api`. Verify final serialized HTTP and WS boundaries. Keep user-authored content and explicit administrator overrides intact; keep internal hash inputs, local namespaces, logs, module paths, and downstream-only metadata when they are not sent to GPT. Do not change stable hash salts merely to rename internal strings.

### R3 — Manual comparison entry and configuration

Expose Manual Detection directly in Quality Operations without requiring a saved or enabled schedule. Support searching and selecting multiple OpenAI accounts, including accounts already covered by scheduled rules, a shared model ID, editable question, reasoning effort, and bounded concurrent requests (1–8). Offer the existing pelican-riding-bicycle HTML/SVG and candy reasoning question presets, with editable text allowing the requested partridge variant or other probes. Snapshot request settings per batch so all selected accounts receive the same question and options. Ordinary connectivity checks retain their existing default behavior.

### R4 — Actual probe execution and account-state boundary

Send the selected prompt and reasoning options through actual supported OpenAI OAuth Responses, API-key Responses, and API-key Chat Completions transports. Keep credential resolution, account model mapping, proxy, TLS and identity overrides consistent with existing transport rules. Unsupported account/protocol modes must return an explicit error instead of silently running an unrelated connectivity question. Manual comparisons must not create scheduled plans, change group membership/scheduling, apply isolation/restore actions, or persist probe-induced health/error/rate-limit state. Scheduled quality workflows continue using their own explicit mutation contracts.

### R5 — Results, cancellation and honest interpretation

Render account-labelled comparison cards with pending/running/completed/error/cancelled status, elapsed time, raw answer/error, and sandboxed HTML/SVG preview. Allow stopping the batch and cancelling queued and in-flight requests; close/unmount must release requests and prevent late results from overwriting a newer batch. Preserve completed results for inspection during the current page session. Successful transport or valid HTML is a completed probe, not an automatic declaration that the account has normal intelligence. Bicycle outputs are for human comparison; show the candy reference answer separately. Display usage only when actual upstream data is available; never fabricate token or quota values from the screenshot.

## Acceptance criteria

- AC1 / R1: upstream target is an ancestor of the final branch; version and binary smoke report the intended v0.2.21 fork build; a conflict/resolution record documents retained local behavior and migrations.
- AC2 / R1–R2: captured final HTTP/WS requests retain supported client identity/tool ID behavior and contain no gateway-generated plaintext product branding; upstream identity opt-out and existing fingerprint suites pass.
- AC3 / R3: an administrator can select multiple accounts in Quality Operations and start a bicycle or candy comparison immediately without saving a schedule. All requests use batch-snapshotted model/question/reasoning.
- AC4 / R4: transport-capture tests prove the custom prompt and effort reach OAuth/API-key Responses and Chat Completions; ordinary connectivity still sends its original default. Invalid input is rejected before upstream dispatch.
- AC5 / R4: success, 401, 429, transport errors, and cancellation in manual mode do not mutate health/runtime/group/scheduling state; existing scheduled judge/isolation/restore behavior remains covered.
- AC6 / R5: cards update independently, one failed account does not stop others, concurrency never exceeds its bound, and Stop/close/unmount end requests with terminal cancellation states. Completed cards remain inspectable after Stop.
- AC7 / R5: generated markup remains confined to CSP-restricted sandboxed frames, errors and partial responses remain visible, and UI never equates a generated picture with a definitive quality verdict.
- AC8: targeted backend/frontend regressions, frontend lint/typecheck/i18n/build, backend tests/vet/build, and independent Trellis review pass; any unavailable integration/live-account checks are explicitly recorded.

## Out of scope

Production deployment, remote push/release publication, modifying real account groups or scheduling during manual comparisons, inventing an automatic image-quality scoring model, new database tables for ad hoc batches, redesigning scheduled rule semantics, reproducing quota counters without upstream usage, and searching unavailable cockpit history.

## Acceptance verification

| Criterion | Status / evidence |
| --- | --- |
| AC1 | Passed: pinned release ancestor, preserved merge parents, version/binary `0.2.21-hotfix1`, resolution matrix |
| AC2 | Passed: core identity files unchanged, full unit suites and final HTTP/WS capture/fingerprint regressions |
| AC3 | Passed: component/client tests and desktop/mobile mock-browser manual batches |
| AC4 | Passed: actual OAuth/API-key Responses/Chat payload capture and input-validation regressions |
| AC5 | Passed: manual non-mutation regressions plus fresh-database scheduled ownership/isolation/restore/audit checks |
| AC6 | Passed: bounded queue, settings snapshot, stale/stop/close/unmount tests and browser cancellation/reopen smoke |
| AC7 | Passed: CSP-first sandbox regressions, raw/error output and actual production preview rendering |
| AC8 | Passed: frontend full tests/lint/typecheck/build, backend full unit/vet/lint, fresh migration tests, embedded build/version and independent review |

Exact commands, final-source rechecks and limitations are in `research/validation.md`. No live GPT account quality claim, production deployment or remote push is included.
