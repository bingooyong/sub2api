# OpenAI Upstream Error Handling

> How this gateway classifies, retries, recovers and reports errors returned by
> OpenAI-compatible upstreams.

---

## Scenario: Classifying and recovering from a 400 whose semantics live only in the message

### 1. Scope / Trigger

Read this contract when you change:

- upstream error extraction (`extractUpstreamErrorCode`,
  `extractUpstreamErrorMessage`, `sanitizeUpstreamErrorMessage`);
- the retry / recovery branches in `openai_gateway_forward.go`;
- failover decisions (`shouldFailoverUpstreamError`,
  `shouldFailoverOpenAIUpstreamResponse`);
- the ops error events written to `ops_error_logs`.

Triggered because upstream error bodies are **not** a stable schema. The same
logical failure arrives with different shapes, and a classifier that keys on
one field silently stops matching — which turns a recoverable upstream
condition into a hard client-visible error.

### 2. Signatures

```go
// backend/internal/service/gateway_upstream_response.go

// "invalid_encrypted_content" | "thinking_signature_invalid" | ...
// Returns "" when error.code is null / absent.
func extractUpstreamErrorCode(body []byte) string

// The human-readable message from the upstream error body.
func extractUpstreamErrorMessage(body []byte) string
```

```go
// backend/internal/service/gemini_messages_compat_service.go

// Strips sensitive query parameters from an upstream message before it is
// logged or returned.
func sanitizeUpstreamErrorMessage(msg string) string
```

```go
// backend/internal/service/openai_gateway_forward.go

// 400 handling order:
//   1. agent-identity task recovery   (401 only)
//   2. encrypted-content recovery     (400 only, retried once)
//   3. rejected-field normalization    (400)
//   4. compact model fallback          (400/404)
//   5. failover decision
//   6. handleErrorResponse            (terminal)
```

### 3. Contracts

1. **The upstream error body is multi-signal.** A single logical failure may
   present as `error.code`, as `error.type`, as `error.param`, or only as
   free-form text inside `error.message`. Observed production shapes for the
   same failure:
   - `{"error":{"message":"The encrypted content ... could not be verified...","type":"invalid_request_error","param":"","code":null}}`
   - `{"error":{"code":"invalid_encrypted_content","message":"The encrypted content for item amsg_... could not be verified...","param":"input","type":"invalid_request_error"}}`

   A classifier that reads only `error.code` misses the first shape entirely.
2. **Classification must combine the error code with the message text.**
   Verdict functions take `(code, msg)` and match on both, requiring the
   message to contain the specific semantic markers (not just a generic
   "invalid request").
3. **A 400 is deterministic for the same request body.** Retrying the same
   body against a different account does not help; it burns another
   account's quota and adds latency. Recovery of a 400 must therefore change
   the request (strip the offending field / drop the anchor) or change nothing
   at all.
4. **Recovery is budgeted: one retry.** Each recovery branch is guarded by a
   `...RetryTried` flag. A second failure of the same kind is terminal and is
   returned to the client.
5. **The upstream error body is what the client sees** (after sanitization)
   when no recovery applies. `handleErrorResponse` writes the upstream body
   through, so the client's error message is the upstream's wording — not a
   generic "upstream error".
6. **Sanitization happens before logging and before returning.**
   `sanitizeUpstreamErrorMessage` strips sensitive query parameters; the
   upstream body may also contain account-identifying material, so
   `redactAgentIdentitySensitiveBody` is applied to agent-identity accounts.
7. **Ops error events are additive, one per upstream attempt.** A request that
   made two upstream attempts produces two `ops_error_logs` rows. That count
   is the evidence for whether a recovery branch fired: **one row = no retry
   happened**.
8. **Do not read the error code from `ops_error_logs.provider_error_code`.**
   Observed 2026-10-23: a production row whose `upstream_error_detail`
   clearly contained `"code":"invalid_encrypted_content"` had an **empty**
   `provider_error_code`. `ops_error_logs` is maintained outside the ent
   schema (no `ent/schema/ops_error_log.go`; no `ProviderErrorCode` symbol in
   the Go code), so that column is not the value produced by
   `extractUpstreamErrorCode` and must not be used to conclude that the
   upstream body carried no code. Read the body itself from
   `upstream_error_detail` / `error_body` instead.

### 4. Validation & Error Matrix

| Condition | Required result |
| --- | --- |
| Upstream 400 whose meaning is only in the message | Verdict must match on the message; recovery fires; one extra upstream attempt is logged |
| Upstream 400 with an explicit `error.code` | Verdict matches on the code; the code form must not regress |
| Upstream 400 unrelated to the classified family | No recovery, no retry; the body is returned to the client |
| Recovery attempt fails again | Terminal; the client gets the error; no second recovery attempt |
| Upstream 429 / 5xx | Handled by the rate-limit / failover paths, **not** by the 400 recovery branches |
| Upstream message contains a sensitive query parameter | Stripped before logging and before returning |
| Agent-identity account returns an error body | Redacted before logging |

### 5. Good/Base/Bad Cases

- **Good**: upstream returns `code: null` plus a message that names the
  failure. The verdict matches on the message, the offending field is
  stripped, the request is retried once, and the client gets a successful
  response. `ops_error_logs` shows two upstream attempts.
- **Base**: upstream returns a 400 that is genuinely the client's fault
  (malformed request). No recovery fires; the client sees the upstream's
  own error message, which is the most useful thing to show them.
- **Bad**: classifying on `error.code` only. The `code: null` shape is
  missed, the 400 is returned verbatim, and the client retries the same
  history forever — the request appears "stuck" with no server-side
  explanation.

### 6. Tests Required

- Verdict functions: table-driven, covering **each observed production body
  shape verbatim**, plus near-miss bodies that must not match
  (e.g. `Invalid signature`).
- Recovery: assert the number of upstream attempts (the mock upstream's
  recorded bodies) — one attempt for a non-matching error, two for a
  matching one, never more.
- Recovery: assert the retried body no longer contains the offending field
  and still contains everything else (messages, tool outputs, tools).
- Sanitization: assert a message with a sensitive query parameter is
  stripped in both the log line and the client-facing body.
- Ops events: assert one `ops_error_logs` row per upstream attempt.

### 7. Wrong vs Correct

Wrong: classify on the error code alone

```go
if upstreamCode == "invalid_encrypted_content" { /* recover */ }
// production sends code=null for the same failure → never matches
```

Correct: classify on code **and** message

```go
if isOpenAIInvalidEncryptedContentUpstreamError(upstreamCode, upstreamMsg) { /* recover */ }
```

Wrong: retry a 400 on a different account

```go
// The request body is unchanged and the ciphertext is bound to the old
// account, so this cannot succeed.
```

Correct: change the request, retry once, on the same account

```go
// strip the offending field / drop the anchor, then retry the same account once
```

Wrong: conclude "there was no error code" from an empty ops column

```sql
select provider_error_code from ops_error_logs where ...  -- empty, even though the body had a code
```

Correct: read the code from the stored body, not from that column

```sql
select upstream_error_detail from ops_error_logs where ...  -- contains error.code
```
