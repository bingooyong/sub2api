# Codex Outbound Identity Customization

> Contract for the fork's `GATEWAY_DISABLE_CODEX_IDENTITY_ENFORCEMENT=true` behavior
> and for keeping it compatible with upstream identity changes.

---

## Scenario: Preserve caller-reported Codex identity while upstream normalizes it to CLI

### 1. Scope / Trigger

Read this contract when you change any of:

- outbound Codex identity headers (`User-Agent`, `originator`, `version`) for OAuth /
  ChatGPT-internal requests;
- `gateway.disable_codex_identity_enforcement` or `gateway.force_codex_cli`;
- `resolveCodexOutboundIdentity`, `enforceCodexIdentityHeadersWithUA`,
  `pairCodexIdentityHeaders`, `preserveCodexClientIdentityHeaders`;
- the fork's default originator / canonical User-Agent.

Triggered because upstream and this fork deliberately disagree about the default
Codex identity, and the disagreement is invisible until a request is actually
sent. Upstream changed the default from `codex-tui` to `codex_cli_rs`
(upstream issue #5198: the `codex-tui` identity is risk-tiered by the upstream
and receives `server_is_overloaded`). This fork keeps the caller's real identity
when identity enforcement is disabled.

### 2. Signatures

```go
// backend/internal/service/openai_codex_identity.go

// Publishes the process-wide enforcement switch. Config maps it as
// SetCodexIdentityEnforcementEnabled(!cfg.Gateway.DisableCodexIdentityEnforcement).
func SetCodexIdentityEnforcementEnabled(enabled bool)

// Derives a self-consistent outbound identity triplet.
func resolveCodexOutboundIdentity(candidateUA string) codexOutboundIdentity

// Final identity gate for all outbound paths.
func enforceCodexIdentityHeadersWithUA(h http.Header, overrideUA string)

// Fallback收口 used when enforcement is disabled: keeps the caller's real
// identity, only guarantees originator/UA pairing and version floor.
func pairCodexIdentityHeaders(h http.Header)

// Restores caller metadata on forwarded Codex-protocol requests when
// enforcement is disabled.
func (s *OpenAIGatewayService) preserveCodexClientIdentityHeaders(
    c *gin.Context, account *Account, headers http.Header)
```

```go
// backend/internal/pkg/openai/request.go

const CodexCLIOriginator = "codex_cli_rs"
const CodexTUIOriginator = "codex-tui"

// CodexDefaultOriginator is the gateway's default outbound originator.
const CodexDefaultOriginator = CodexCLIOriginator

// Rewrites a paired TUI identity to CLI identity (originator + UA first segment).
func NormalizeCodexClientIdentityToCLI(originator, userAgent string) (string, string, bool)
```

Config keys:

| Key | Meaning |
| --- | --- |
| `gateway.disable_codex_identity_enforcement` | `true` = keep caller-reported identity (this fork's production setting) |
| `gateway.force_codex_cli` | `true` = always use the CLI identity regardless of caller headers |

### 3. Contracts

1. **Enforcement off (this fork's production setting).** The caller's real
   `User-Agent` / `originator` / `version` are preserved verbatim.
   `enforceCodexIdentityHeadersWithUA` short-circuits to
   `pairCodexIdentityHeaders`, and `preserveCodexClientIdentityHeaders` restores
   the original caller headers afterwards.
2. **TUI → CLI normalization is gated by `codexIdentityEnforcement`.**
   `NormalizeCodexClientIdentityToCLI` must only run when enforcement is ON.
   Upstream calls it unconditionally inside `resolveCodexOutboundIdentity`; in this
   fork it must be wrapped:

   ```go
   if codexIdentityEnforcement.Load() {
       originator, pairedUA, _ = openai.NormalizeCodexClientIdentityToCLI(originator, pairedUA)
   }
   ```

   Rationale: upstream's normalization is a *risk mitigation* for the enforced
   path. With enforcement off, the whole point is to keep the caller's identity,
   so normalizing there would silently defeat the customization.
3. **`ForceCodexCLI` still wins over the caller.** `force_codex_cli=true` means
   "always CLI", regardless of the caller's headers. The fallback default
   originator in that path follows upstream (`codex_cli_rs`). This is an explicit
   administrator opt-in, so it is *not* covered by contract 1.
4. **Account-level `user_agent` overrides still win** over the caller's identity
   (unchanged upstream behavior, and required by this fork too).
5. **The `version` header has exactly one source**: the canonical identity
   (panel setting → auto-synced value → compile-time constant). It is never
   taken from the caller's UA version segment.
6. **`codex-tui` remains in the official-originator set.** It is still a valid
   inbound identity; only the *outbound default* changed.

### 4. Validation & Error Matrix

| Condition | Required result |
| --- | --- |
| Enforcement off, caller sends `codex-tui/...` | Outbound keeps the caller's UA and `codex-tui` originator |
| Enforcement off, caller sends a third-party UA (e.g. `workbench/1.0`) | `preserveCodexClientIdentityHeaders` restores it verbatim; no CLI normalization |
| Enforcement off, `force_codex_cli=true` | UA = canonical CLI UA, originator = `codex_cli_rs` |
| Enforcement off, account has an explicit `user_agent` | Account UA wins; originator paired from it |
| Enforcement **on** (upstream default) | TUI identities are normalized to CLI (upstream behavior preserved) |
| Caller's `version` is below the upstream minimum | Version corrected to the canonical value; identity otherwise untouched |
| Compat-bridge path (originator intentionally removed) | Originator stays absent; identity headers are not re-added |

### 5. Good/Base/Bad Cases

- **Good**: upstream adds a new identity normalization. The fork gates it behind
  `codexIdentityEnforcement`, so the disabled path keeps the caller's identity
  and the enabled path follows upstream.
- **Base**: the fork's production setting is enforcement off. Callers keep their
  own identity; upstream's risk mitigation does not apply to them.
- **Bad**: applying `NormalizeCodexClientIdentityToCLI` unconditionally. Every
  caller-reported `codex-tui` request silently becomes `codex_cli_rs`, which
  contradicts the documented customization and breaks
  `TestOpenAIGatewayService_IdentityOptOutKeepsExplicitUserAgentOverrides`.

### 6. Tests Required

- `go test ./internal/service/ -run 'IdentityOptOut|CodexIdentity' -count=1`
  - enforcement off + account UA override → UA and originator as expected
  - enforcement off + `force_codex_cli=true` → canonical CLI UA + `codex_cli_rs`
  - enforcement off + compat bridge → originator omitted
- `go test ./internal/service/ -count=1` must stay green; the upstream default
  (enforcement on) must still normalize TUI → CLI.
- When changing the fallback default originator, the fork's identity test
  expectations must be updated **in the same commit** — the default is a shared
  contract between the fork and upstream.

### 7. Wrong vs Correct

Wrong: normalize unconditionally (as upstream does)

```go
originator, pairedUA, _ = openai.NormalizeCodexClientIdentityToCLI(originator, pairedUA)
// With enforcement disabled this rewrites the caller's real identity.
```

Correct: gate the normalization on the enforcement switch

```go
if codexIdentityEnforcement.Load() {
    originator, pairedUA, _ = openai.NormalizeCodexClientIdentityToCLI(originator, pairedUA)
}
```

Wrong: assume the gateway default originator is stable

```go
// openai.CodexDefaultOriginator changed from "codex-tui" to "codex_cli_rs" upstream.
wantOriginator = "codex-tui" // breaks on the next upstream merge
```

Correct: read the constant and keep fork expectations in the same commit

```go
wantOriginator = openai.CodexDefaultOriginator
```

### 8. Open question

Upstream issue #5198 records that the `codex-tui` identity is risk-tiered by the
upstream and receives `server_is_overloaded`. This fork deliberately keeps
caller identity when enforcement is off, so that risk mitigation does not
apply. Re-evaluate if requests start failing with `server_is_overloaded` or
capacity-shed errors.
