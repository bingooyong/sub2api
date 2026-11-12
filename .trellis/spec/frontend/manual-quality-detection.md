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
- Preview with `extractPelicanHtml`, an opaque-origin iframe
  `sandbox="allow-scripts"`, and `referrerpolicy="no-referrer"`. Never add
  `allow-same-origin`. The generated document must place the deny-network CSP
  before executable/user markup, including malformed model-produced heads.
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
| Close and reopen panel | Requests stopped; existing cards remain |

## 5. Good / Base / Bad Cases

- Good: shared question and options reach each selected account, with raw
  text available beside isolated generated previews.
- Base: 401/unsupported-model errors are visible per account and are not
  labelled as an intelligence failure.
- Bad: equating valid HTML with intelligence passing, leaving aborted cards
  running, or allowing a pre-head script to execute before the preview CSP.

## 6. Tests Required

`pnpm exec vitest run src/api/__tests__/admin.manualQuality.spec.ts
src/components/admin/account/__tests__/ManualQualityPanel.spec.ts` covers
split Unicode chunks, SSE failures, queue bounds, immutable batch settings,
pagination selection, Stop/stale callbacks, close/unmount and preview policy.
Include preview helper regression tests when changing document normalization.
Run lint, typecheck, locale completeness and the production build. Browser
smoke should exercise desktop/mobile cards and cancellation using mock
responses; live model quality must not be claimed from fixture responses.

## 7. Wrong vs Correct

Wrong: `await Promise.all(selected.map(account => fetchProbe(account)))`.

Correct: freeze the request once, use a bounded worker queue, pass an
AbortSignal per active request, and guard all callbacks with its batch ID.
