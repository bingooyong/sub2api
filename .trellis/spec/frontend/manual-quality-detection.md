# Manual Quality Detection

## 1. Scope / Trigger

Read before changing the manual comparison entry in `AccountQualityView`,
`ManualQualityPanel`, the `manualQuality` stream client, or HTML/SVG previews.

## 2. Signatures

`runManualQualityProbe(accountID, request, signal, onText): Promise<void>`
accepts a readonly `{model_id, prompt, reasoning_effort}` request and calls
`POST /admin/accounts/:id/pelican-test` with the existing admin UI/auth headers.
The panel emits `close`; its page keeps the mounted component with `v-show`
so completed results remain available after reopening.

## 3. Contracts

- Search OpenAI accounts and retain selection snapshots across pagination.
  Scheduled-plan membership does not disable manual selection.
- Freeze model/prompt/effort at batch start. A shared queue limits active
  requests to 1–8; do not start one fetch per selected account without a limit.
- Each card has `pending|running|success|error|cancelled` state, account name,
  actual submitted model/effort, duration, raw response/error and optional
  preview. Success means the response completed, not that intelligence passed.
- Reuse canonical bicycle/candy prompts; candy answer 21 is a reference,
  not an automatic downgrade decision. Custom prompts remain editable.
- The stream client owns incremental UTF-8/SSE decoding, requires explicit
  successful `test_complete` plus nonempty output, rejects malformed/truncated
  streams, and cancels/releases its reader in all terminal paths.
- Stop/close/unmount abort active requests, cancel queued work and advance a
  batch generation so late callbacks cannot update a newer run. Completed
  cards survive Stop; a new batch replaces the previous cards.
- Preview with `extractPelicanHtml`, an opaque-origin iframe `sandbox=""`,
  and `referrerpolicy="no-referrer"` in ManualQualityPanel, IQTestModal and
  PelicanRecordsDashboard. Never add script or same-origin permissions.
  Reuse DOMPurify for passive HTML/SVG normalization: remove scripts/event
  handlers, meta refresh, navigation URLs, base/link/resource hints, nested
  browsing contexts and forms. Preserve ordinary artwork, styles, CSS
  animation, safe visual SVG animations and internal artwork references.
  Reject externally loading or obfuscated CSS resources and external resource
  attributes before assigning srcdoc; keep plain styles/keyframes and fragment
  references. A final CSP alone proved insufficient to guarantee zero requests
  for a generated CSS background fixture in Chromium. Raw output preserves
  externally dependent styles and generated JavaScript for inspection; scripts
  are never executed.
- Install the trusted CSP before model markup, with `script-src 'none'`,
  denied subresource/network access and no forms/frames/objects. CSP
  `default-src`/`connect-src` does not prevent document navigation: remove
  navigation constructs as well. Do not rely on unsupported `navigate-to`.
- Before opening a manual SSE stream, an HTTP401 may trigger one retry through
  the existing `refreshAuthTokens({ failedAccessToken })` coordinator. Preserve
  body/admin headers; abort promptly while awaiting refresh without cancelling
  other callers' refresh. Do not replay across logout/session replacement or
  retry upstream SSE errors. Never add another refresh implementation.
- HTTP423 plus `ADMIN_COMPLIANCE_ACK_REQUIRED` dispatches the existing
  `admin-compliance-required` event with metadata, preserves the HTTP/API error
  information, and is not retried automatically. The existing App/store owns
  the confirmation dialog.
- The opener exposes `aria-expanded`/`aria-controls` linked to a stable panel
  ID. Opening moves focus into the panel after rendering; closing restores
  opener focus while retaining completed results and cancelling pending work.
- Show token/quota values only if supplied by a real upstream usage contract.
  Current manual SSE has no usage contract; do not invent screenshot counters.

## 4. Validation & Error Matrix

| Condition | UI result |
| --- | --- |
| No selected account / empty model or prompt / invalid concurrency | Start unavailable |
| Account search finishes after a newer search | Ignore stale result |
| One request fails | Its card fails; remaining queue continues |
| Plain text returned for bicycle question | Show raw answer and missing-preview hint |
| Stop while active/queued | Unfinished cards cancelled; completed cards retained |
| Terminal success but empty output / truncated stream | Error card |
| Close and reopen panel | Requests stopped; existing cards remain; focus returns to opener |
| Refreshable administrator HTTP401 before SSE | Shared refresh, at most one retry |
| Abort/logout/session replacement during refresh | No stale replay; shared refresh remains available to other callers |
| HTTP423 compliance requirement | Dispatch confirmation event with metadata, retain error, no automatic retry |
| Generated scripts/meta refresh/HTML or SVG links | Passive preview, no execution/navigation/network requests |

## 5. Good / Base / Bad Cases

- Good: shared question and options reach each selected account, with raw
  text available beside isolated generated previews.
- Base: 401/unsupported-model errors are visible per account and are not
  labelled as an intelligence failure.
- Bad: equating valid HTML with intelligence passing, leaving aborted cards
  running, or treating a fetch-blocking CSP as protection against iframe
  navigation.

## 6. Tests Required

`pnpm exec vitest run src/api/__tests__/admin.manualQuality.spec.ts
src/components/admin/account/__tests__/ManualQualityPanel.spec.ts` covers
split Unicode chunks, SSE failures, queue bounds, immutable batch settings,
pagination selection, Stop/stale callbacks, close/unmount and preview policy.
Include `pelicanHtml.spec.ts`, `admin.manualQuality.spec.ts`, and
`AccountQualityView.manual.spec.ts` for passive normalization, 401 concurrent
refresh/retry/cancellation/session cases, 423 metadata/error/no-retry handling,
and disclosure/focus. Run lint, typecheck, locale completeness and production
build. Chromium must exercise meta/script/HTML/SVG navigation and CSS artwork
under the actual backend parent CSP, observing requests and frame URLs rather
than only normalized strings. UI smoke covers desktop/mobile cards,
cancellation/reopen, focus, refresh and the compliance dialog using mock
responses; live model quality must not be claimed from fixture responses.

## 7. Wrong vs Correct

Wrong: `await Promise.all(selected.map(account => fetchProbe(account)))`.

Correct: freeze the request once, use a bounded worker queue, pass an
AbortSignal per active request, and guard all callbacks with its batch ID.

Wrong: permit inline scripts and assume `connect-src 'none'` prevents a
preview's meta refresh or clicked SVG link from navigating.

Correct: sanitize into a passive document before srcdoc assignment, remove
navigation constructs, use an empty sandbox and disable scripts in CSP.
