# Account Quality Operations

## 1. Scope / Trigger

This contract applies to opt-in `pelican_config.quality` scheduled plans and
the `/admin/account-quality-*` APIs. It covers database migrations, the
scheduler, account isolation/restore, judge selection, and the admin UI data
flow.

## 2. Signatures

- `POST /admin/scheduled-test-plans` and `PUT /admin/scheduled-test-plans/:id`
  accept `pelican_config.quality`.
- `GET /admin/account-quality-plans` returns quality plans; `POST
  /admin/account-quality-plans/:id/run` queues an enabled plan.
- `GET /admin/account-quality-results?before_id=<id>` returns cursor-paginated
  operation summaries.
- `account_quality_operations` is durable audit storage; generated result
  bodies remain in `scheduled_test_results` and may expire.

## 3. Contracts

Quality configuration contains `expected_answer`, `action` (`remove_groups` or
`disable_scheduling`), `remove_group_ids`, `auto_restore`, and an optional
judge `{group_id, model_id, prompt, retry_count}`. `retry_count=0` means the
backward-compatible default of 3; explicit values are 1–5.

Judge verdicts are strict JSON with `verdict` (`correct`, `incorrect`, or
`unknown`) and a bounded `reason`. Operation `outcome` is one of `probed`,
`judged`, `isolated`, `restored`, `skipped`, or `error`; `quality_action`
contains the concrete account action.

Operation history stores a snapshot and result IDs, but its returned
`pelican_config` has probe and judge prompts redacted. Plan-edit responses are
admin-only because they must return the configured prompts.

Group restoration uses the actual `account_groups` schema: `account_id`,
`group_id`, `priority`, and `created_at`. Preserve the original priority and
creation timestamp. There is no `allowed_models` column; do not assume an
unpublished schema extension or add one merely to satisfy an old fixture.

## 4. Validation & Error Matrix

| Condition | Behavior |
| --- | --- |
| Missing/oversized prompt, answer, judge model, or judge prompt | Reject plan |
| Retry count outside 0–5, parallel count outside 1–8 | Reject plan |
| Judge group unavailable, malformed output, timeout, auth error, or `unknown` | Record `probed`/`judged`; do not mutate account |
| Explicit `incorrect` | Apply configured isolation in one transaction |
| Paused/edited/deleted plan or expired lease | Record `skipped`; do not mutate account |
| Restore snapshot conflicts with live account/groups | Record `skipped` with `restore_conflict` |

## 5. Good / Base / Bad Cases

- Good: a compatible judge returns `correct`; all probes pass and no account
  state changes.
- Base: a judge returns `incorrect`; only the selected groups or scheduling
  flag changes, with a snapshot and outbox event.
- Bad: transport failure or `unknown`; no membership, scheduling, or restore
  state is changed.

## 6. Tests Required

- Parse strict verdict JSON, reject duplicate/unknown fields, and assert retry
  bounds and self-account exclusion.
- Assert lease contention, stale-run protection, isolation, restore conflicts,
  and non-mutating failure paths. Run `QualityActionsRestoreOwnershipAndStaleRuns`
  against a fresh migrated database and verify priority/created_at restoration;
  a test fixture must use the same columns as the published schema.
- Assert operation history pagination, prompt redaction, durable snapshots,
  and result IDs after result persistence.
- Assert handler/API validation, localized UI validation, typecheck, focused
  component tests, and production build.

## 7. Wrong vs Correct

### Wrong

Derive operation history only by grouping `scheduled_test_results`, or persist
the full probe/judge prompts in the dashboard response. Both couple audit data
to retention and expose operator-authored instructions.

### Correct

Write one durable `account_quality_operations` row per round, finalize it with
persisted result IDs, and redact prompt fields from the operation payload while
keeping prompt values available only to the authorized plan editor.

## Scenario: Immediate observational account comparisons

### 1. Scope / Trigger

Applies to the Quality Operations manual panel and changes to the shared
Pelican probe transport. Scheduled plans use the same payload plumbing, but
their runner continues to own explicit judge/isolation/restore actions.

### 2. Signatures

- `POST /admin/accounts/:id/pelican-test` accepts `model_id`, `prompt`, and
  `reasoning_effort` and returns SSE `content`, `test_complete`, and `error`.
- `AccountTestService.TestManualPelicanAccountConnection` validates and limits
  manual requests to OpenAI OAuth/API-key text transports.
- `TestPelicanAccountConnection` remains the scheduled probe entry.
- `pelicanTestOptionsFromContext` must be consumed at the actual upstream
  payload builders, not only in standalone helper tests.

### 3. Contracts

- Model is nonempty, at most 100 Unicode characters, without CR/LF/NUL;
  prompt is nonempty and at most 32,000 Unicode characters. The HTTP request
  body is capped at 256 KiB. Effort is empty or `minimal|low|medium|high|xhigh`.
- Responses carries the requested text in `input` and effort in
  `reasoning.effort`; Chat Completions uses the user message and
  `reasoning_effort`. Preserve account mapping, proxy/TLS, credentials and
  explicit identity/header overrides.
- Pelican probes do not write probe extras, permanent authentication errors,
  or rate-limit reconciliation. Ordinary `/test` probes retain their default
  question and state handling. Authentication credential/task lifecycle and
  the scheduled runner's explicit quality actions are separate mechanisms.
- Manual requests do not create schedules or change groups/scheduling.
- Gateway-authored plaintext metadata in final GPT HTTP/WS requests must be
  brand-free. User-authored text and internal deterministic hash salts are
  preserved. A successful transport is not a quality verdict.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Invalid ID/model/prompt/effort or oversized request | Reject before upstream dispatch |
| Unsupported provider or media model in manual mode | Explicit error, no fallback connectivity question |
| OpenAI OAuth / API-key Responses / Chat Completions | Requested prompt and effort reach the actual transport |
| Probe success, 401, 429, transport failure or cancellation | No probe-induced health or scheduling mutation |
| Missing terminal SSE success or empty output | Client reports incomplete/empty result |
| Explicit scheduled judge failure | Existing runner lease/action/audit contract still applies |

### 5. Good / Base / Bad Cases

- Good: compare multiple accounts with one frozen bicycle prompt; each
  response is observable without changing account runtime state.
- Base: a scheduled Pelican probe now sends its configured question rather
  than `hi`, and the quality runner still owns any subsequent account action.
- Bad: helper tests pass while the service still calls the default builder,
  or manual 401 responses silently disable an account.

### 6. Tests Required

Capture outgoing requests through the service/handler for OAuth Responses,
API-key Responses and Chat Completions. Assert actual prompt/effort, ordinary
probe compatibility, input rejection, cancellation and success/401/429
non-mutation. Run `go test -p 2 -tags=unit ./internal/service
./internal/handler/admin -run 'ManualPelican|PelicanTransport|ScheduledPelicanActual|PelicanHandler|PelicanPayloads' -count=1`.

`openai_outbound_branding_test.go` additionally checks actual HTTP forwarding
and native WS writes while preserving caller-authored project-name text.

### 7. Wrong vs Correct

Wrong: store a prompt in context but always call
`createOpenAITestPayload(model, isOAuth)` before sending the request.

Correct: consume Pelican context in that sending path and call
`createPelicanOpenAIPayload(model, isOAuth, options.prompt,
options.reasoningEffort)`, while leaving ordinary probes unchanged.
