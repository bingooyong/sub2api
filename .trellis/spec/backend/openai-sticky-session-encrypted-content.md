# Sticky Sessions and Encrypted Reasoning Content

> Why a Codex session can end up on a different account — and why that makes
> previously issued encrypted reasoning content undecryptable.

---

## Scenario: A sticky session is dropped, so the session lands on another account

### 1. Scope / Trigger

Read this contract when you change:

- OpenAI account selection / scheduling (`openai_account_scheduler.go`,
  `gateway_scheduling.go`, `openai_gateway_scheduling.go`);
- sticky-session binding lifetime, waiting, or exclusion logic;
- failover that excludes an account and retries;
- the encrypted-content recovery path (see
  [OpenAI Encrypted Reasoning Recovery](./openai-encrypted-reasoning-recovery.md)).

Triggered because the two mechanisms are coupled in a way that is invisible
from either side alone: **encrypted reasoning content is bound to the account
that issued it, but the account is chosen per request.** Whenever the binding
between a session and its account is dropped, the next request carries
ciphertext that the new account cannot decrypt.

### 2. Signatures

```go
// backend/internal/service/openai_account_scheduler.go

// Sticky binding lookup / removal. Key is (groupID, sessionHash).
func (s *GatewayService) getStickySessionAccountID(ctx context.Context, groupID int64, sessionHash string) (int64, error)
func (s *GatewayService) deleteStickySessionAccountID(ctx context.Context, groupID int64, sessionHash string) error

// Sticky-aware selection. Returns (nil, false, nil) when no binding applies,
// in which case the caller falls back to load balancing.
func (s *defaultOpenAIAccountScheduler) selectBySessionHash(
    ctx context.Context, req OpenAIAccountScheduleRequest) (*AccountSelectionResult, bool, error)

// Request fields that steer sticky selection.
type OpenAIAccountScheduleRequest struct {
    StickyAccountID       int64  // explicit binding, 0 = look it up
    StickyWeighted        bool
    PreserveStickyBinding bool  // keep the binding even when this attempt fails
    ExcludedIDs           map[int64]struct{}
    SessionHash           string
    RequestedModel        string
    RequiredTransport     string
}
```

```go
// backend/internal/service/openai_gateway_scheduling.go
func (s *OpenAIGatewayService) GenerateSessionHash(c *gin.Context, body []byte) string
func (s *OpenAIGatewayService) GenerateExplicitSessionHash(c *gin.Context, body []byte) string
func (s *OpenAIGatewayService) GenerateSessionHashWithFallback(c *gin.Context, body []byte, fallbackSeed string) string
```

Config keys:

| Key | Meaning |
| --- | --- |
| `gateway.openai_ws.sticky_session_ttl_seconds` | lifetime of a `session_hash -> account_id` binding (default 3600) |
| `gateway.openai_ws.sticky_session_max_waiting` | max requests allowed to wait for the bound account |
| `gateway.openai_ws.sticky_session_wait_timeout` | how long a request may wait for the bound account |

### 3. Contracts

1. **Encrypted content is bound to the issuing account.** The upstream seals
   reasoning/message content with a key tied to the account that produced it.
   The same ciphertext replayed to a *different* account cannot be
   unsealed — the upstream answers `invalid_encrypted_content` with
   `encrypted content hydration failed`.
2. **A sticky binding is `(groupID, sessionHash) -> accountID`.** The session
   hash is derived from the request (`GenerateSessionHash`), so a client that
   keeps its session identity keeps its account — as long as the binding
   survives.
3. **The binding is deliberately dropped in these situations** (each one
   `clearBinding()`s and falls back to load balancing):
   - the bound account is in `ExcludedIDs` (e.g. it just failed and is being
     failed over);
   - the account is no longer schedulable (`getSchedulableAccount` /
     `IsSchedulable()`) — this is the case for a rate-limited or
     over-quota account;
   - `shouldClearStickySession(account, requestedModel)` — the bound account
     cannot serve the requested model;
   - platform / OpenAI-compatibility / transport incompatibility;
   - the re-read from the database no longer matches (group, model,
     capability) — e.g. a concurrent state change;
   - the free-tier soft gate (an over-quota free OAuth account must not be
     pinned);
   - Grok team+model cooldown or model quota blocking.
4. **Consequence: dropping the binding guarantees the next request carries
   undecryptable ciphertext** if the client replays prior encrypted content.
   The upstream error then names an item id (`amsg_…` for assistant messages)
   rather than the ciphertext prefix.
5. **Therefore, a failover that changes the account cannot fix an
   `invalid_encrypted_content` error.** The ciphertext is still bound to the
   old account. Recovery must strip the ciphertext (or drop the continuation
   anchor) — see
   [OpenAI Encrypted Reasoning Recovery](./openai-encrypted-reasoning-recovery.md).
   Switching accounts only adds a second failure on top of the first.
6. **A sticky binding must not be silently destroyed by a transient condition.**
   `PreserveStickyBinding` exists for exactly this reason (used when
   retrying on a guardian/parent account): the binding survives the failed
   attempt so the next turn returns to the original account once it recovers.

### 4. Validation & Error Matrix

| Condition | Required result |
| --- | --- |
| Bound account is healthy and matches the request | Reuse it; the session keeps its account and its ciphertext stays decryptable |
| Bound account rate-limited / over quota | Binding dropped, request goes elsewhere → **the next replayed encrypted content will fail to decrypt**; the request path must handle `invalid_encrypted_content` |
| Bound account excluded by failover | Binding dropped unless `PreserveStickyBinding` is set |
| Bound account cannot serve the requested model | Binding dropped; model mismatch is not a ciphertext problem |
| Upstream answers `invalid_encrypted_content` | Do **not** retry on a different account expecting success; strip the ciphertext / drop the anchor |
| Binding TTL expired | Same as "binding dropped" — the session may land on another account |

### 5. Good/Base/Bad Cases

- **Good**: the bound account is still healthy → the session keeps it → the
  replayed encrypted content still decrypts → no error at all. This is the
  steady state the whole mechanism exists for.
- **Base**: the bound account hit its 5-hour quota → binding dropped → the
  request lands on another account → the replayed ciphertext fails to decrypt
  → the request path strips it and retries → the turn succeeds, with the
  model losing that turn's encrypted reasoning context.
- **Bad**: treating `invalid_encrypted_content` as a normal upstream failure and
  failing over to the next account. The new account cannot decrypt the old
  ciphertext either, so the failover adds latency and burns another account's
  quota without any chance of success.

### 6. Tests Required

- Sticky selection: bind a session, make the bound account unschedulable,
  assert the binding is dropped and the request falls back to load balancing.
- `PreserveStickyBinding`: force a failure on the bound account, assert the
  binding survives so the next turn returns to it.
- Encrypted-content recovery: the recovered request must go to the **same**
  account as the failed attempt, and must not carry the offending ciphertext.
- Assert the account id used by the successful attempt equals the account id
  of the failed attempt (no silent account switch inside one request).

### 7. Wrong vs Correct

Wrong: fail over on `invalid_encrypted_content`

```go
// The new account cannot decrypt the old account's ciphertext.
if isInvalidEncryptedContent(err) {
    return s.forwardWithNextAccount(ctx, c, body) // will fail the same way
}
```

Correct: recover the request in place, on the same account

```go
// Strip the undecryptable ciphertext (or the continuation anchor) and retry
// on the SAME account. The ciphertext is bound to this account's key.
```

Wrong: clearing the binding on a transient condition without a way back

```go
clearBinding() // the session now hops accounts, and its ciphertext is dead
```

Correct: preserve the binding when the account is expected to recover

```go
req.PreserveStickyBinding = true
```

### 8. Production evidence (2026-10-23) and what is still unverified

Observed on 160: encrypted-content failures arrived on OAuth account 6854 while
its 5-hour Codex quota was exhausted (`codex_5h_used_percent = 100`) and on
API-key account 6642 with the ciphertext-prefix form. The failures cluster in
the hours around quota exhaustion, which is consistent with contract 3
("quota-exhausted account is no longer schedulable → binding dropped → session
moves → ciphertext dies").

**Not yet verified**: a direct observation that the failing request's session
hash was bound to a *different* account immediately before the failure. The
chain above is the mechanism, not a measured proof. The remaining check is to
correlate `session_hash` bindings in the scheduler's store with the account ids
seen in `ops_error_logs` for the same `client_request_id`.
