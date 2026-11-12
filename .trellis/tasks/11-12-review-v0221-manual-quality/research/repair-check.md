# Independent integrated repair review

## Review boundary and status

Review of the task-owned F1–F6 repairs against the approved PRD/design and curated specs. The reviewer did not implement these repairs. Frontend and backend final source repairs have been reviewed. One additional F4 finding was reported and repaired by the backend owner, then independently re-reviewed. Final source, affected checks and browser evidence have passed independent review. Verdict: **APPROVE**. Root still owns the final embedded-server build/version and commit/archive/journal completion gates.

## Frontend findings

No unresolved confirmed frontend logic defect remains in the reviewed snapshot. One F1 follow-up coverage gap was reported and repaired by the frontend owner. No product-code edits by this reviewer.

- F1: ManualQualityPanel, IQTestModal and PelicanRecordsDashboard all use the shared normalized document with an empty iframe sandbox and no-referrer. DOMPurify removes active tags/events; final filtering removes navigational href/src values while retaining internal SVG artwork references and image data. Visual SVG animation attributes are restricted; href/onload mutation is rejected. Generated scripts are not run. Raw output remains inspectable in all consumers, with bilingual capability help.
- F2: Manual probe HTTP401 handling uses the real shared refresh coordinator once, preserves the frozen JSON body/admin header, waits with an independent AbortSignal wrapper, rechecks cancellation and stored session/token identity before retry, and never reacts to upstream SSE errors as login refresh triggers.
- F3: HTTP423 code/metadata is parsed at the pre-stream boundary. The existing admin-compliance-required event reaches App.vue and requireAcknowledgement in the existing store. ManualQualityError preserves domain HTTP classification plus status/apiCode/metadata; no automatic 423 retry occurs.
- F6: Stable panel linkage, expanded state, nextTick initial focus and opener focus restoration preserve the mounted component, close-cancels-batch behavior and completed results. Tests exercise actual mounted view/panel focus.
- Shared utility review found no DOMPurify global hooks/config that would interfere with this helper. Queue generation guards, cancellation and result-state plumbing are unchanged.

## Browser evidence review

Initial production-CSP and clean no-parent-policy nine-fixture runs passed, but a later fresh run reproduced an intermittent CSS background request at iframe append despite final meta CSP. Root correctly reopened F1 rather than accepting earlier passing evidence. The frontend owner added conservative pre-srcdoc removal of resource-bearing CSS/import/image functions, escaped/comment-obfuscated CSS, and resource attributes; ordinary styles/keyframes and strict local SVG paint references remain supported.

Independent follow-up review found that the first CSS pass iterated only descendants, leaving root `<html style="background:url(...)">` unchecked. The owner included `documentElement` itself and added an html-style regression. Independent source re-review confirms root and descendant attributes now receive the same policy. Backslashes/comments, @import, image()/image-set() and non-fragment url() are rejected, including style blocks, inline styles, SVG presentation/animation attributes; srcset/background/poster/xml:base are forbidden. No further confirmed gap remains. Root owns the required fresh browser/UI smoke after this follow-up; previous browser success is not treated as final evidence. Root's final 11-fixture Chromium run after include-root filtering passed with zero requests/errors, unchanged srcdoc frames, and retained gradient/background/CSS animation; browser/UI details remain owned by root's evidence record.

## Verification reviewed so far

- `git diff --check`: passed in reviewer session.
- Final `/tmp/review-frontend-focused-final.log`: 7 files / 51 tests passed after the CSS follow-up, covering helper, manual client, shared tokenRefresh, manual panel, IQ modal, records and disclosure view. Explicit final helper rerun `/tmp/review-frontend-css-final.log`: 11/11 passed, including root HTML attributes.
- `/tmp/review-frontend-lint.log`: full configured ESLint completed successfully, as confirmed by implementer. Reviewer did not launch redundant checks during shared-host swapping.
- Frontend full suite before CSS follow-up passed 393 files / 3,158 tests. Final follow-up ESLint and vue-tsc pass (owner confirms exit 0); the final affected 51-test suite and explicit 11-test helper rerun prove the narrow follow-up. Final production build passed, exit 0, including locale completeness 3/3, `vue-tsc -b` and Vite (`/tmp/review-frontend-build.log`); existing mixed-import/chunk-size warnings only.
- Backend `/tmp/sub2api-backend-repair-final.log`: service passed (11.231s), handler/admin passed (16.731s), including final adopted-version expiry regression. Full `go vet -p 1 ./...` passed without diagnostics (`/tmp/sub2api-backend-repair-vet.log`). Full configured complete-report golangci-lint passed, `0 issues.` (`/tmp/sub2api-backend-repair-lint.log`). Final embedded build/version remain root completion gates.
- Production UI log `output/playwright/review-repair-ui.log` inspected: one HTTP401 refresh and exactly one retry using rotated token; success/success/cancelled cards; close/reopen retained 3; linked expanded state and panel/open-close focus; 390px viewport/document width; real compliance dialog received revision metadata after HTTP423. There were 3 compliance requests for the 3-account batch, not automatic retry of one request.
- Final helper production/no-parent-policy eleven-fixture logs inspected: zero unexpected requests/errors, every frame remained srcdoc, gradient/background and changing CSS animation preserved. Final production UI rerun also measured the scrolled bicycle SVG at 270×202 with four circles and the expected background; root visually inspected the direct iframe screenshot. Final browser/server were closed. Evidence: `browser-validation.md`, `review-repair-ui-final.log`.
- This reviewer inspected persisted commands/logs and code; costly test/lint/typecheck commands ran in the designated implementation lanes to avoid concurrent Go compilation/shared-host swapping. No redundant reviewer test run is claimed.
- No database integration rerun: no repository/schema/migration changes. No live GPT/account call, production deployment or remote push occurred.

## Backend review

Reviewed the stable initial patch in account_test_service.go, openai_token_provider.go, pelican_test_service.go and both transport/lifecycle regression files. Mapped provider-qualified classification is applied before dispatch; custom text aliases remain supported. Agent Identity bypass and ordinary/dependency-free connectivity compatibility are explicit. OAuthRefreshAPI → OpenAITokenRefresher → RefreshAccountToken/persistAccountCredentials persists credentials/version and related lifecycle snapshots, without introducing forbidden OpenAI health writes.

### Confirmed review finding repaired by owner

- **Medium, within F4:** `backend/internal/service/openai_token_provider.go:236-261`. The probe expiry guard checks the initial/refreshed account before `CheckTokenVersion`. When that later call adopts a higher-version `latestAccount`, only its access token is copied; its expiration is not checked. An unexpired snapshot plus a higher-version expired non-PAT repository account can therefore dispatch a known expired token, contrary to AC4.
- **Requested repair:** validate the effective latest credential account expiry before using its token, probe-only, preserving PAT and normal forwarding policy. Add regression using distinct snapshot/repository account objects; expired latest credentials must fail before upstream dispatch, and valid latest credentials must still work.
- Root and backend owner were informed. The owner added a probe-only adopted-account expiry check plus `TestManualPelicanOAuthVersionCheckValidatesAdoptedCredentialExpiry` with distinct snapshot/repository objects and expired/fresh/PAT cases. Independent source re-review confirms the gap is closed, valid latest tokens still work, and PAT/non-probe behavior is preserved. No reviewer source edits during the owner's compile; final targeted verification passed, including the adopted-version cases.

No additional confirmed backend repair finding remains after this source re-review. Missing-refresh account disabling/cache deletion is gated away from Pelican context; refresh failures return without old-token fallback; post-refresh expired and cancelled paths fail without dispatch. Normal non-probe behavior is deliberately unchanged. Production gateway/provider injection is present; dependency-free ordinary test paths remain explicit. Tests exercise cache, successful renewal/persistence, missing refresh, failures/cancellation, lock contention, absent refresh wiring, Agent Identity and normal connectivity.



## Final independent verdict

**APPROVE — no unresolved confirmed in-scope finding.** F1–F6 are repaired, and the two additional review gaps (root HTML resource filtering and adopted-version token expiry) were closed by their owners and independently re-reviewed. Reviewer product-code changes: none; reviewer-owned report only.

The final preview capability is intentionally passive: ordinary SVG/styles/CSS animation and raw output remain, while generated scripts, external resources and resource-bearing/obfuscated CSS are omitted. Browser fixture success does not establish real account intelligence. Backend constructor/ordinary connectivity fallback is intentional compatibility; production uses injected gateway/provider. Full frontend suite preceded the narrow CSS filter follow-up; final affected 51 tests, helper 11 tests, complete lint/typecheck/build and expanded browser checks cover that final change. Embedded binary build/version and repository bookkeeping must pass before the root reports task completion.
