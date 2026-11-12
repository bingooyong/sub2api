# Frontend repair implementation

## Boundary

Own only frontend preview/helper/streaming/view/locale code and their regressions; backend and specs belong to separate lanes. Reuse existing DOMPurify and token-refresh coordination. Do not add dependencies, alter fingerprints, issue real probes, or commit from this lane.

## Implemented changes

- F1: one passive HTML/SVG sanitizer shared by manual panel, IQ modal and records dashboard; empty iframe sandbox, script-src none, removed active navigation/resource elements and external URL attributes, visual-only SVG animations and internal artwork references. A browser follow-up reproduced intermittent CSS background preloading in srcdoc despite its meta CSP; resource-bearing/escaped/comment-obfuscated CSS is now omitted before rendering (style elements and all root/descendant attributes), retaining plain CSS/keyframes and fragment-only url(#id). srcset/background/poster/xml:base resource attributes are forbidden. Original output remains unmodified; records expose raw text for script-driven drawings. Shared bilingual help explains supported previews.
- F2: raw probe fetch now performs at most one pre-stream administrator HTTP401 refresh/retry using refreshAuthTokens. The frozen body and admin headers are preserved. Abort does not cancel the shared refresh; cancellation and session replacement prevent replay. Upstream SSE failures do not trigger a login retry.
- F3: HTTP423 with ADMIN_COMPLIANCE_ACK_REQUIRED emits the existing admin-compliance-required event with metadata, keeps the failed probe, and never retries automatically. ManualQualityError retains its domain code `http`, plus status/apiCode/metadata for the backend error contract.
- F6: linked aria-expanded/controls, stable region ID, initial focus after nextTick and opener focus restoration; panel remains mounted across close/reopen.

## Validation

All frontend repair/check work is complete; no frontend validation processes remain. No commit was made by this implementation lane.

- New regression-only red pass: 3 failures proved missing script removal/navigation normalization and HTTP423 metadata behavior before fixes (`/tmp/review-frontend-red.log`).
- Initial focused pass: 6 files / 41 tests, including existing shared token-refresh tests (`/tmp/review-frontend-focused.log`). Focused pass after additional session-change and records regressions: 7 files / 44 tests. Final focused pass after CSS/root follow-up: **7 files / 51 tests passed**, exit 0 (`/tmp/review-frontend-focused-final.log`), followed by explicit final helper **11/11 tests passed** (`/tmp/review-frontend-css-final.log`).
- Initial full-suite (2 workers), lint and typecheck jobs were interrupted due severe shared-host swapping before completion. These are infrastructure interruptions, not passing checks. Preserved logs: `/tmp/review-frontend-{full,lint,typecheck}-interrupted.log`. Only this lane's verified process IDs were stopped; unrelated processes were untouched.
- Final validation is serialized: focused 7-file suite with one worker, lint, typecheck, full suite with one worker, then build.
- Final focused command: `pnpm --dir frontend exec vitest run src/utils/__tests__/pelicanHtml.spec.ts src/api/__tests__/admin.manualQuality.spec.ts src/api/__tests__/tokenRefresh.spec.ts src/components/admin/account/__tests__/ManualQualityPanel.spec.ts src/components/admin/account/__tests__/IQTestModal.spec.ts src/components/admin/account/__tests__/PelicanRecordsDashboard.spec.ts src/views/admin/__tests__/AccountQualityView.manual.spec.ts --maxWorkers=1 --minWorkers=1`; `/tmp/review-frontend-focused-final.log`.
- Root owns authoritative production-CSP Chromium, no-parent-CSP diagnostic and production UI smoke. Initial 9-fixture passes were superseded by an intermittent iframe CSS request reproduced in a fresh run; do not treat those initial passes as final proof. Root's final current-helper **11-fixture** production-CSP browser run passed with zero probe requests/errors, all previews remaining srcdoc, and normal gradient/background/CSS animation retained (`output/playwright/review-repair-browser-complete.log`).
- `pnpm --dir frontend lint:check`: passed again after final CSS/root source, exit 0; `/tmp/review-frontend-lint.log`.
- First completed typecheck found DOMPurify RETURN_DOM typing as Node. Added explicit HTMLHtmlElement/head runtime guards rather than casts or assertions. Final affected tests, lint and typecheck all passed afterward. `pnpm --dir frontend typecheck`: exit 0; `/tmp/review-frontend-typecheck.log`.
- Full suite: `pnpm --dir frontend test:run --maxWorkers=1 --minWorkers=1` passed, **393 files / 3,158 tests**, exit 0 (`/tmp/review-frontend-full.log`). This preceded the CSS resource filter; final affected 51-test pass, full lint/typecheck and browser follow-up validate that subsequent narrow repair. Final production build passed afterward.

- `pnpm --dir frontend build`: **passed, exit 0**, including locale completeness 3/3, `vue-tsc -b`, and Vite (34.96s); `/tmp/review-frontend-build.log`. Existing mixed-import/chunk-size warnings only. Verified production `index.html` and each referenced `/assets/` file exists. Root owns final production-UI smoke and embedded-server build.
- `git diff --check`: passed. No dependency/lockfile changes, snapshot artifacts, real GPT calls or frontend commits.

## Files changed

- `frontend/src/utils/pelicanHtml.ts`, `frontend/src/utils/__tests__/pelicanHtml.spec.ts`
- `frontend/src/api/admin/manualQuality.ts`, `frontend/src/api/__tests__/admin.manualQuality.spec.ts`
- `frontend/src/components/admin/account/{ManualQualityPanel,IQTestModal,PelicanRecordsDashboard}.vue` and matching three component tests
- `frontend/src/views/admin/AccountQualityView.vue`, new `frontend/src/views/admin/__tests__/AccountQualityView.manual.spec.ts`
- `frontend/src/i18n/locales/{en,zh}/qualityOps.ts`

## Limits and compatibility

- Passive previews intentionally omit generated JavaScript, external resources, and entire resource-bearing/escaped/comment-obfuscated CSS elements or offending attributes. Ordinary embedded CSS, SVG, CSS animations and internal fragment references remain. Original output remains inspectable unchanged.
- Full suite was run before the narrow CSS preload follow-up; final affected regressions, full lint/types/build and expanded browser validation cover that follow-up. No live-account quality result is inferred from fixtures.
- Independent frontend reviewer found and verified the root-HTML-attribute repair; no further confirmed frontend source defect remains. Root coordinates final integrated approval and records browser/UI evidence.
