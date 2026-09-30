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
  and non-mutating failure paths.
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
