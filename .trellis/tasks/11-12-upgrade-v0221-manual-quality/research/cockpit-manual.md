# Research: Manual account intelligence probes and cockpit reference

- Query: Identify cockpit-tools screenshot-equivalent pelican/bicycle account comparison behavior and reusable sub2api manual quality components.
- Scope: internal, two local checkouts; no external API calls or dependency decisions.
- Date: 2026-11-12

## Findings

### Reference availability

The supplied `~/Code/01Code/github.com/bingooyong/cockpit-tools` checkout contains no implementation matching the screenshot. A repository-wide search (including hidden files, excluding dependencies/generated sidecars/.git) for `pelican`, `partridge`, `pigeon`, `bicycl`, `intelligen`, `降智`, `鹈鹕`, `鹧鸪` and `测试结果` found only an unrelated provider conversation-test title at `src/components/codex/CodexModelProviderManager.tsx:3829`. Do not describe a specific cockpit implementation as verified. The supplied screenshot is the usable UX reference: multiple account-labelled output cards, completed/error state, duration, optional usage metrics, and termination controls.

### Existing reusable sub2api frontend

- `frontend/src/utils/intelligenceTest.ts:1`: `IntelligenceQuestion = candy | pelican`, canonical candy question and SVG pelican-riding-bicycle prompt, and per-kind delivery contract. The reference bird is a pelican (鹈鹕); the user wrote 鹧鸪 but explicitly allows equivalent probes.
- `frontend/src/utils/pelicanHtml.ts:1`: extracts HTML/XML fenced output, full HTML or standalone SVG; wraps standalone SVG, injects a CSP disabling network, external frames, forms and object resources while allowing inline scripts/styles for animation. Reuse rather than inventing rendering.
- `frontend/src/components/admin/account/IQTestModal.vue:216`: existing single-account manual modal accepts an account, defaults to candy / gpt-6-astra / medium, allows 1–8 parallel runs, can preview scheduled output.
- `IQTestModal.vue:325`: fetches `POST /admin/accounts/:id/pelican-test` with admin bearer auth, `ADMIN_UI_REQUEST_HEADER`, model, composed prompt, reasoning effort and AbortSignal; parses `content`, `test_complete`, and `error` SSE events.
- `IQTestModal.vue:318,379,397,445`: owns AbortControllers per run, aborts on close/unmount, awaits all parallel runs and stores local history. Current abort leaves a run status as `running`, close button is disabled while running, and there is no explicit stop button; improve cancellation when reusing this flow.
- `IQTestModal.vue:143`: sandboxed iframe `sandbox="allow-scripts"`, `referrerpolicy="no-referrer"` with CSP-injected `srcdoc`. No `allow-same-origin`.
- `IQTestModal.vue:189,256`: per-account localStorage key `sub2api-pelican-test:<id>` and up to eight local records; this prefix is local browser storage, not upstream request data.
- `frontend/src/components/admin/account/PelicanRecordsDashboard.vue:13,54`: existing multi-account historical cards use the same sandboxed renderer, scheduled history plus current modal manual record; no batch/manual execution.
- **Critical integration gap:** repository search for `IQTestModal` finds only its own test import (`frontend/src/components/admin/account/__tests__/IQTestModal.spec.ts:3`), no production usage. The existing manual UI is orphaned.
- `frontend/src/views/admin/AccountQualityView.vue:1`: actual quality operations route currently renders only scheduled plan form/list/history. Account search/pagination and group loading already exist; scheduled run button requires a persisted enabled plan and therefore is not an ad hoc manual probe.
- `frontend/src/api/admin/accountQuality.ts:1`: lists/runs quality plans and lists durable operations. Do not reuse scheduled-plan run as manual test because it can apply isolation/restore actions.

### Existing backend and drift to fix

- `backend/internal/server/routes/admin.go:397`: admin-protected dedicated route `/:id/pelican-test` already exists.
- `backend/internal/handler/admin/account_handler.go:1283,1334`: `PelicanTestRequest` currently has unconstrained `model_id`, `prompt`, `reasoning_effort`; handler dispatches `TestPelicanAccountConnection`. Add bounded validation appropriate to the manual UI.
- `backend/internal/service/pelican_test_service.go:29`: dedicated entry puts prompt/reasoning into context then calls normal `TestAccountConnection`.
- `pelican_test_service.go:48,64`: helper functions build Claude and OpenAI payloads with the supplied prompt/reasoning while preserving the ordinary connectivity payload. Existing helper tests in `pelican_test_service_test.go:9` prove helpers, not transport integration.
- **Critical integration gap:** `pelicanTestOptionsFromContext`, `createPelicanClaudePayload`, and `createPelicanOpenAIPayload` have no consumers in the normal probe service. `backend/internal/service/account_test_service.go:893` still calls fixed `createOpenAITestPayload`; its definition at `:2785` sends literal `hi` and no reasoning. Claude paths at `:563,641` similarly call fixed `createTestPayload`. A visible UI alone would send the wrong test question for Responses/OAuth accounts.
- `account_test_service.go:873,2135`: API-key Chat Completions paths already pass `prompt`; reasoning is not wired there. `:2421` Gemini also accepts prompt. Scope UI to verified supported providers/protocols rather than implying all paths honor identical reasoning.
- `account_test_service.go:902,927,945`: normal OpenAI probe respects context cancellation, canonical Codex identity/account UA, account header overrides, credential model mapping and proxy. Keep these boundaries and existing fingerprint behavior.
- **Mutation caveat:** handler comment calls the probe isolated/non-mutating, but normal OpenAI test writes probe extras (`:960`), reconciles 429 (`:978`) and sets account error for 401 (`:981`). Reusing this path requires an explicit manual probe non-mutation guard if the feature promises read-only quality comparison.
- `backend/internal/service/pelican_scheduled.go:32`: scheduled and quality probes use the same `TestPelicanAccountConnection`; fixing custom-prompt integration also repairs these callers.
- `pelican_scheduled.go:73`: scheduled runner owns leases, judge/isolation/restore and durable operation finalization. Manual visual comparisons should not call this runner or apply quality actions.

### Recommended minimal feature boundaries

1. Expose a distinct manual-test section/button on the quality operations page, with selectable OpenAI accounts, shared model/prompt/reasoning and bounded concurrency; no scheduled-plan prerequisite.
2. Reuse canonical candy/pelican prompts, existing dedicated endpoint and CSP/sandbox renderer. Show one account-labelled card per run with terminal state, duration, text/error and visual output; never call a successful SVG a definitive intelligence verdict.
3. Support explicit stop and close/unmount abort, mark queued/in-flight runs cancelled, and prevent stale completions after a new batch/account selection.
4. Keep manual comparisons free of scheduled isolation/restore actions; do not add a database migration merely to display batch results. Existing local history pattern can be reused if needed.
5. Fix backend option consumption at the actual HTTP payload boundary and verify captured outbound request prompt/reasoning, default connectivity behavior and cancellation/non-mutation. Helper-only tests are insufficient given the current drift.
6. Keep browser/local namespaces and hash salts unchanged; inspect only gateway-authored plaintext at actual outbound GPT boundaries for project-brand leakage. Do not strip user-authored content or change stable identifier derivations.

### Suggested files and validation

- Feature: `frontend/src/views/admin/AccountQualityView.vue`, a focused manual comparison component under `components/admin/account/`, `utils/intelligenceTest.ts`, `utils/pelicanHtml.ts`, existing locale files, focused component tests.
- Probe repair: `backend/internal/service/pelican_test_service.go`, `account_test_service.go`, `backend/internal/handler/admin/account_handler.go`; add a capture-transport test for the endpoint-to-upstream integration and non-mutating errors.
- Existing test reference: `IQTestModal.spec.ts:71` verifies identical parallel request settings and sandbox/CSP; `:99` invalid HTML; `:113` immutable timing/settings; `:147` candy plain text.
- Related specs: `.trellis/spec/backend/account-quality-operations.md` (strict judge and non-mutation contracts), `.trellis/spec/backend/codex-identity-customization.md` (preserve caller identity when enforcement disabled), `.trellis/spec/backend/codex-forwarding.md`, `.trellis/spec/frontend/component-guidelines.md` and frontend build-toolchain guidelines.

## Caveats / Not Found

- No cockpit manual pelican implementation was present in the supplied checkout; screenshot behavior is the only verified reference.
- Findings refer to the inspected pre-upgrade files; upstream merge may move line numbers and alter integration points.
- No live account was called. This research does not establish model quality, token accounting availability or real cancellation behavior at providers.
- `sub2api` Go import/module paths and localStorage keys are not proof of outbound brand leakage; verify the serialized headers/body rather than renaming repository identifiers.
