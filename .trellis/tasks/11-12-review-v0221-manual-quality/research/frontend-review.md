# Frontend review: v0.2.21 manual quality detection

## Scope and spec compliance

Reviewed `ManualQualityPanel.vue`, `manualQuality.ts`, the
`AccountQualityView.vue` disclosure, `pelicanHtml.ts`, related component/client
tests, translations, the older `IQTestModal`/records consumers, the shared auth
client, account-list types and the application's effective CSP.

The main product flow is present: cross-page selection is retained in a
reactive `Map`; settings and account labels are snapshotted per batch; the
worker queue stays within 1–8; Stop cancels active/queued runs and generation
guards reject stale callbacks; completed cards survive close/reopen; the SSE
client requires a successful terminal event and nonempty output; raw partial
output remains visible on failure. I found no spec violation in those paths.

## Confirmed findings

### [MEDIUM] Preview policy permits active model code and does not enforce the documented no-network boundary

- **Evidence:** `frontend/src/utils/pelicanHtml.ts:20` allows all inline scripts
  (`script-src 'unsafe-inline'`). `frontend/src/components/admin/account/ManualQualityPanel.vue:66`
  gives the opaque iframe `sandbox="allow-scripts"`; the same helper/attribute
  pair is reused by `IQTestModal.vue:144` and `PelicanRecordsDashboard.vue:20`.
  The preview CSP blocks fetch/XHR/subresources, frames, forms and objects, but
  contains no effective document-navigation control. `default-src` and
  `connect-src` do not govern an iframe navigating itself. Omitting
  `allow-top-navigation` protects the parent tab, not the iframe's own browsing
  context. The application CSP in `backend/internal/config/config.go:38`
  explicitly allows `frame-src 'self'` plus captcha/payment origins, so same-site
  (and those allowlisted) navigations remain eligible at the parent policy.
- **Trigger:** a model returns `<script>location.href='/some/path'</script>` or
  `<meta http-equiv="refresh" content="0;url=/some/path">`. It can initiate a
  same-site document request and replace the drawing. Inline script can also
  run unbounded work/timers. Because `AccountQualityView.vue:8` retains the
  component with `v-show`, completed iframe code remains mounted after the
  panel is closed.
- **Impact:** this contradicts the spec's “deny-network CSP” and “safe generated
  markup” contract. The sandbox still protects parent DOM/storage and blocks
  top-level navigation/popups, so this is not parent-page XSS; it is a contained
  navigation/resource-exhaustion boundary failure.
- **Test gap:** `pelicanHtml.spec.ts` verifies CSP ordering and the earlier
  browser review verified a blocked `fetch`, but neither exercises
  `location.href`, meta refresh, hidden-frame lifetime or a nonterminating
  script. `ManualQualityPanel.spec.ts:67-70` asserts the unsafe policy as the
  expected contract.
- **Fix:** treat model HTML as a passive preview. Remove `allow-scripts`, set
  `script-src 'none'`, and normalize/remove active navigation constructs before
  assigning `srcdoc` (at minimum scripts/event handlers, meta refresh,
  navigational `href`/`xlink:href`, base/link/preconnect-style elements). Do not
  claim that adding another fetch/connect directive blocks document navigation.
  Preserve SVG/SMIL/CSS animation. Add a real Chromium regression that observes
  zero requests and unchanged iframe content for script navigation and meta
  refresh, plus unit coverage for the normalized document. Apply the shared
  helper/iframe contract consistently to all three consumers.

### [MEDIUM] The streaming client bypasses the application's token-refresh contract

- **Evidence:** `frontend/src/api/admin/manualQuality.ts:25-45` reads the current
  access token once, calls raw `fetch`, and turns every 401 into a terminal HTTP
  error. Normal admin requests use `apiClient`; its response interceptor at
  `frontend/src/api/client.ts:170-195` coordinates `refreshAuthTokens` and retries
  once with the rotated access token. The account search immediately above the
  probe uses that shared client, so a session can be valid and refreshable while
  the manual stream reports every account as failed.
- **Trigger:** the access token expires after account selection or during a
  multi-account batch while a valid refresh token is present. Each probe that
  starts with the stale token returns 401; no coordinated refresh/retry occurs.
- **Impact:** valid admin sessions produce false per-account failures, which is
  especially misleading in an account-quality comparison.
- **Test gap:** `admin.manualQuality.spec.ts:51-55` covers a generic HTTP 400 only;
  there is no stale-token 401, coordinated refresh, retry-once, refresh failure
  or cancellation-during-refresh case.
- **Fix:** put streaming fetches behind a shared authenticated-fetch helper, or
  add one narrowly guarded 401 retry using `refreshAuthTokens({ failedAccessToken })`.
  Keep the original AbortSignal authoritative, never retry twice, preserve
  cross-tab/session-change safeguards, and test parallel 401 deduplication plus
  cancellation before the retry. Do not silently convert refresh failure into
  successful probe output.

### [MEDIUM] HTTP423 compliance confirmation is bypassed by the streaming client

- **Evidence:** `frontend/src/api/admin/manualQuality.ts:25-45` converts non-2xx responses into ordinary HTTP errors; unlike `frontend/src/api/client.ts:153-167`, it never dispatches `admin-compliance-required` with response metadata. `frontend/src/App.vue:68-71` relies on that event to open the confirmation dialog. `backend/internal/server/middleware/admin_compliance.go:12-47` protects the Pelican route at `backend/internal/server/routes/admin.go:362-400`.
- **Trigger:** compliance revision/state changes after the page/store has initialized, then the operator starts a probe.
- **Impact:** cards report HTTP423 while the required confirmation dialog never opens.
- **Fix/test:** preserve the existing event and metadata contract, retain the failed probe error, and do not retry HTTP423 automatically. Test event metadata, preserved error fields and one request only; share the pre-stream boundary with the 401 repair without confusing upstream SSE errors with administrator authentication.

### [LOW] The inline disclosure loses keyboard focus when closed and exposes no disclosure state

- **Evidence:** the opener at `frontend/src/views/admin/AccountQualityView.vue:6`
  has no `aria-expanded`/`aria-controls`. The child Close button at
  `ManualQualityPanel.vue:8` emits `close`; the parent immediately changes the
  retained component to `display:none` at `AccountQualityView.vue:8`, without
  restoring focus. Closing by keyboard therefore hides the focused control and
  leaves users without a predictable visible focus target. Opening leaves focus
  on the trigger, but the next tab stops are the other header actions before
  reaching the newly exposed panel.
- **Test gap:** no `AccountQualityView` interaction test covers activeElement,
  expanded state or control linkage.
- **Fix:** give the panel a stable ID; add `aria-expanded` and `aria-controls` to
  the opener; keep a trigger ref and restore focus after close. On open, either
  focus the panel heading/Close control after `nextTick` or use a disclosure DOM
  order that makes the panel the next tab destination. Add a keyboard-focused
  component test.

## Reviewed and rejected as findings

- Vue's wrapped `Map` mutations track `size`/`has`, so direct `set/delete/clear`
  does update selection and Start state.
- The `batch` generation guard plus per-account AbortControllers prevents old
  callbacks/finalizers from overwriting a newer batch. Controllers keyed by
  account ID are safe because selection IDs are unique.
- The SSE parser correctly handles split UTF-8, multiline data fields, CRLF,
  final unterminated data, explicit failure/error events, truncation and empty
  terminal success for this backend's LF-delimited `TestEvent` contract.
- User/model output rendered in Vue `<pre>` is escaped; the issue above is the
  intentionally active iframe, not raw-text DOM injection.
- Server filtering already requests `platform=openai`; backend validation
  remains authoritative for unsupported account/model modes, and independent
  cards expose those errors without stopping the batch.

## Browser confirmation

Root reproduced meta-refresh navigation to same-site and CSP-allowlisted dummy hosts, plus clicked SVG-link navigation, in real Chromium under the exact production parent CSP. Every request was intercepted. Parent nonce CSP blocked generated inline JavaScript. See `research/csp-browser.md` for method, limitations and artifacts; the repair must cover navigation without scripts as well as script removal.

## Static-analysis status

No LSP/AST diagnostics tool is exposed to this reviewer, and the assigned lane
explicitly prohibited npm/test execution. Prior final-source typecheck/lint/test
evidence exists in the archived task, but fixes require fresh diagnostics and
the focused regressions described above before approval.

## Verdict

**REQUEST CHANGES** — three medium correctness/security-boundary findings and one
low accessibility finding are confirmed. No critical/high parent-page XSS or
account-state mutation was found in this frontend slice.

Follow-up reviewer reported additional passing baseline checks (focused frontend 19 tests, typecheck/lint/locale/build, backend focused and integration/vet/lint), but complete command logs were not persisted. These are informational baseline reports only; repair validation must be fresh and recorded.
