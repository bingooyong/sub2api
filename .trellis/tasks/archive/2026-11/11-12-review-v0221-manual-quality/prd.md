# Review and fix v0.2.21 manual quality integration

## Goal

Repair all confirmed defects in the completed v0.2.21 upgrade and manual multi-account quality detection so operators can compare account responses reliably and safely, while retaining the fork's fingerprint and unbranded GPT request fixes.

## Background and review coverage

Branch: `v0.2.21-hotfix1`; review baseline: `808c8c733`. Prior commits are upstream merge `81231fa12`, scheduled restoration `61dbfc2b4`, lint compatibility `6d96a1ee8`, and manual feature `f5109cf14`. Prior task: `.trellis/tasks/archive/2026-11/11-12-upgrade-v0221-manual-quality`. The user requested “review and fix all” and approved creation of this review task.

Independent backend, frontend and upgrade reviews cover task-owned changes, prompt/effort delivery, transport/credential/proxy consistency, selection/queue snapshots, SSE termination, cancellation/stale results, close/reopen, scheduled isolation/restoration and migrations. Six findings below are confirmed; remaining inspected boundaries have no confirmed findings. Production-CSP Chromium reproduction confirms meta refresh and SVG-link navigation using intercepted dummy requests. Prior passing tests are baseline evidence, not verification of these repairs. Detailed evidence lives in `research/`.

## Requirements and confirmed findings

| ID | Severity and evidence | Required observable outcome | Acceptance |
| --- | --- | --- | --- |
| F1 | Medium: `frontend/src/utils/pelicanHtml.ts:20`; `frontend/src/components/admin/account/ManualQualityPanel.vue:66`; shared consumers `IQTestModal.vue:144`, `PelicanRecordsDashboard.vue:20`. Meta refresh/SVG links navigate the preview; scripts are allowed by its own sandbox/policy. | All three previews show passive HTML/SVG/CSS, retain original output, and cannot execute generated JavaScript or initiate navigation. Operator help states this capability. | AC2 |
| F2 | Medium: `frontend/src/api/admin/manualQuality.ts:25-45`; normal refresh at `frontend/src/api/client.ts:170-195`. An expired admin access token causes false per-account failures. | Refreshable admin sessions continue with one coordinated pre-stream HTTP401 retry, respecting cancellation and session replacement. Upstream SSE errors never trigger auth replay. | AC3 |
| F3 | Medium: same manual client; shared handling at `frontend/src/api/client.ts:153-167`; listener at `frontend/src/App.vue:68-71`; enforcement at `backend/internal/server/middleware/admin_compliance.go:12-47` and protected route in `backend/internal/server/routes/admin.go:362-400`. | HTTP423 with `ADMIN_COMPLIANCE_ACK_REQUIRED` opens the existing confirmation flow with backend metadata, while preserving the probe error and avoiding automatic retry. | AC3 |
| F4 | Medium: `backend/internal/service/account_test_service.go:861-868`; provider missing-refresh branch `openai_token_provider.go:157-169`, mutation helper `:273-306`; fallback `refresh_policy.go:40-45`; production wiring `wire.go:259-286`. Snapshot access tokens bypass cache/refresh; naive provider reuse could alter health. | OAuth probes obtain current cached/refreshed credentials. Expired credentials without refresh or failed/cancelled refresh fail without stale dispatch or health/isolation changes. Normal forwarding and Agent Identity remain compatible. | AC4 |
| F5 | Medium: `backend/internal/service/pelican_test_service.go:75-84`; related classification `openai_codex_models_service.go:730-748`, `composite_platform.go:130-140`. Known transcription/TTS/moderation and provider-qualified mapped targets bypass rejection. | Reject known non-text targets before upstream dispatch, including mapped/provider-qualified names, while accepting custom text aliases. | AC5 |
| F6 | Low: `frontend/src/views/admin/AccountQualityView.vue:6-8`; close control `ManualQualityPanel.vue:8`. Hidden focused controls and absent disclosure state impair keyboard access. | Opener exposes linked expanded state, opening moves focus into the panel and closing restores it to the opener; retained results and cancellation remain intact. | AC6 |

Preservation requirements inherited from the review request:

- R1: Cover all task-owned functional changes and relevant upstream integration boundaries with evidence, avoiding unrelated style churn.
- R2: Retain fingerprint seeds/identities, caller identity opt-out, HTTP/WS tool correlation and gateway-generated unbranded GPT requests. Preserve caller-authored content and internal hash namespaces.
- R3: Retain actual prompt/effort, account/model/proxy validation and cancellation. Successful credential renewal may persist credentials/token version/cache and related scheduler snapshots/outbox; probes must not write health/error/401/429, extra/quota, groups, schedulability or isolation state.
- R4: Retain cross-page selection, bounded queues, immutable batch settings, terminal/error detection, raw partial output and mounted results; remove active preview execution consistently.
- R5: Preserve scheduled isolation/restoration, ownership/leases/audit and published full-filename migration identities.
- R6: Use narrow fixes, meaningful regressions and an independent repair review; report fresh evidence and validation limitations.

## Acceptance criteria

- AC1 (R1/R6): Backend/frontend/upgrade reports record severity, trigger, anchors and test gaps. All F1–F6 repairs pass independent review, or a finding is rejected with concrete evidence.
- AC2 (F1/R4): Under the production parent CSP, representative script/meta/HTML/SVG navigation payloads generate no requests or preview navigation; ordinary SVG/style/CSS animation remains visible and raw output remains intact. All three consumers share this policy.
- AC3 (F2/F3/R3): Tests cover shared concurrent 401 refresh, a single retry limit, refresh failure, abort while refreshing, logout/session replacement, and 423 event metadata/error propagation without retry or SSE replay.
- AC4 (F4/R2/R3): Tests cover fresh cached tokens, successful refresh/persistence, expired/no-refresh, failed/cancelled refresh and zero forbidden probe mutations; ordinary non-probe behavior, connectivity and Agent Identity tests pass.
- AC5 (F5/R3): Table-driven mapped/provider-qualified non-text cases, including gpt-4o(-mini)-transcribe, gpt-4o-mini-tts and omni-/text-moderation families, make zero upstream requests. Custom/provider-qualified text aliases remain accepted.
- AC6 (F6/R4): Keyboard tests verify expanded state, panel linkage, initial focus and focus restoration; Stop/close/unmount/reopen preserve cancellation and result contracts.
- AC7 (R2/R5/R6): Fresh affected tests, configured lint/vet, frontend typecheck/locale/build and embedded-server version checks pass. Identity/branding and scheduled-operation preservation evidence is recorded; simulated checks are distinguished from live checks.
- AC8 (R6): Scoped code/spec/task changes are committed, only this task is archived, journal recorded, and working tree is clean.

## Scope boundaries and compatibility

No new dependencies, unrelated redesign, production deployment, remote push, real GPT calls, configured-account group/scheduling changes, or automatic intelligence verdicts. Preview compatibility is explicitly passive: SVG, styles and CSS animation are supported; generated JavaScript is inspectable in raw output but is not executed. The current production parent nonce policy already blocks inline generated scripts. No database/schema change is planned.
