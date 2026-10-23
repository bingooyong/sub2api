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

### 7.1 Observability contract

The diagnostic log line for a 400 upstream error carries the request's
`session_hash`:

```text
[OpenAI] upstream 400 in forward loop (account: 6854, session_hash: openai:0376..., upstream_code: "invalid_encrypted_content", invalid_encrypted: true, outbound: keys=..., input=[...], msg: ...)
```

The `session_hash` is the Redis key suffix of the sticky binding
(`sticky_session:{groupID}:{sessionHash}`), so the log line plus a Redis read
answers the question "which account was this session bound to?".

Rules:

- The handler computes the session hash and stores it in the gin context
  (`SetOpenAISessionHashToContext`); the service layer reads it with
  `OpenAISessionHashFromContext`. **The service layer must not recompute it**:
  the outbound body has already been rewritten by the codex transform, so the
  content-derived fallback seed would differ from the original request.
- The value is diagnostic only. It never influences scheduling.
- `ops_error_logs` does **not** store the session hash, so a failure that is
  only visible in the database cannot be correlated with the binding. When
  adding a column for it, remember the table is maintained outside the ent
  schema (no `ent/schema/ops_error_log.go`).

### 8. Production evidence (2026-10-23)

**Sticky sessions do work on 160.** Measured directly against the live Redis:

- Keys are `sticky_session:{groupID}:{sessionHash}`; the session hash is
  `openai:` + xxhash64 (16 hex chars, current format) **or** `openai:` +
  sha256 (64 hex chars, legacy format — both formats are written, see
  `SessionHashDualWriteOld`).
- Observed bindings on 160 mapped to real account ids (e.g. group 8 →
  6854, group 4 → 6847), with TTLs counting down from the 3600 s default.
- A probe request carrying a `prompt_cache_key` **created a new binding**
  (`sticky_session:4:openai:<xxhash>` → the account that served it), proving
  the write path works for that session signal.

**Observability trap (cost a wrong conclusion once):** `redis-cli --scan`
returned **empty** on this deployment even though `dbsize` reported 6456 keys
and the keys were really there. Only `redis-cli KEYS "sticky_session:*"` listed
them. Always use `KEYS` (or `SCAN` with a verified cursor) when auditing
sticky bindings; never conclude "no bindings exist" from an empty `--scan`.

**Remaining open question.** The encrypted-content failures on 160 (OAuth
account 6854, `for item amsg_... hydration failed`) all landed on 6854
itself, and the user's session was bound to 6854 at observation time. So the
mechanism in contract 3-5 explains *how* a session can end up on an account
that cannot decrypt its ciphertext, but this particular incident was **not**
observed to involve an account switch — the ciphertext was simply not
decryptable by 6854. The two candidate root causes remain open:

1. the ciphertext was issued by a different account earlier in the session and
   the session moved accounts (sticky binding dropped) before the failure; or
2. the issuing account's key was rotated / the upstream re-issued the
   ciphertext under a new key, so even the same account can no longer decrypt
   it.

Distinguishing them needs the session hash of the failing request, which is
not stored in `ops_error_logs`. The self-healing path (strip the ciphertext /
drop the anchor) addresses both.
