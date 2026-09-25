# Admin account list filters

## 1. Scope / Trigger

Read this when changing `GET /admin/accounts` filters, account table filter UI,
or shared table page-size defaults.

## 2. Signatures

- Query: `platform`, `type`, `status`, `search`, `group`, `privacy_mode`,
  `plan_tier`, `sort_by`, `sort_order`, `page`, `page_size`
- `type` = account auth kind (`oauth` / `apikey` / …)
- `plan_tier` = subscription plan filter ID (platform-scoped aliases)

Shared page size:

- Code default: `DEFAULT_TABLE_PAGE_SIZE`
- Setting key: `table_default_page_size`
- Override order: `__APP_CONFIG__.table_default_page_size` →
  `localStorage['table-page-size']` → code/setting default

## 3. Contracts

- Never reuse `type` for Free / Plus / Business Premium. Those are `plan_tier`.
- `plan_tier` without `platform` is ignored. The UI disables the control when
  platform is empty.
- Alias expansion is platform-scoped. OpenAI aliases must not apply under
  `platform=grok` or `antigravity`.
- OpenAI / Antigravity match `credentials.plan_type` (normalize like
  `normalizePlanType`). Empty child plan with `parent_account_id` may fall back
  to the parent’s `plan_type`.
- Grok matches `credentials.subscription_tier` OR `credentials.plan_type`.
  Usage-snapshot Heavy inference is not part of SQL filtering.
- List / count / export / bulk filtered snapshots / ETag inputs that clone list
  filters must forward `plan_tier`.

## 4. Validation & Error Matrix

| Condition | Result |
| --- | --- |
| `plan_tier` set, `platform` empty | Ignore `plan_tier` |
| `platform=openai`, `plan_tier=business_premium` | Match normalized aliases including `self_serve_business_prolite` |
| `platform=grok`, OpenAI-only filter ID | No OpenAI alias expansion |
| Unknown `plan_tier` for platform | Empty result set (no match), not 500 |

## 5. Good / Base / Bad Cases

- Good: OpenAI + Free returns rows whose stored plan normalizes to `free`.
- Base: no `plan_tier` keeps previous list behavior.
- Bad: filtering Business Premium via `type=free` or applying ChatGPT labels to
  Antigravity Pro.

## 6. Tests Required

- Alias tables for OpenAI / Antigravity / Grok
- Empty platform ignores `plan_tier`
- Parent fallback for empty child OpenAI `plan_type`
- Default page size 50 with config/localStorage override precedence
- Accounts filter UI disables plan tier without platform

## 7. Wrong vs Correct

Wrong: overload `type` for plan badges.

```text
GET /admin/accounts?type=plus
```

Correct: keep auth type and plan tier separate.

```text
GET /admin/accounts?platform=openai&type=oauth&plan_tier=plus
```
