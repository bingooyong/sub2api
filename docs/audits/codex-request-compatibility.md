# Codex request metadata and compatibility audit

This audit covers Codex-related request headers, body transformations, identifiers,
settings/UI overrides, and HTTP/WebSocket transport paths in the fork. It does not
treat every application constant, brand name, or dependency string as a defect.

Two compatibility defects were identified: the identity-enforcement opt-out still
changed client metadata, and native Responses tool-call normalization could merge
distinct calls. The changes address those defects without rotating persisted
identifiers or replacing client metadata with a captured identity.

For a Chinese reference with filenames, source excerpts, and per-finding status,
see [Sub2API 请求指纹与硬编码核查清单](sub2api-fingerprint-findings.zh-CN.md).

## Local CLI observation

The installed `codex-cli 0.153.4` was run in `exec` mode on macOS 26.6.2 / arm64,
using an ephemeral scratch directory, `--ignore-user-config`, a read-only tool
sandbox, and a custom Responses provider bound to `127.0.0.1`. The provider used a
dummy key and returned a canned `OK` SSE response. The CLI completed successfully.
Only allowlisted header values and body structure were recorded; authorization,
session/request/installation identifier values, and prompt contents were excluded.

[Sanitized observation](codex-cli-0.153.4-exec-observation.json):

```text
originator: codex_exec
user-agent: codex_exec/0.153.4 (Mac OS 26.6.2; arm64) iTerm.app/3.4.22 (codex_exec; 0.153.4)
accept: text/event-stream
content-type: application/json
version: absent
```

The request used Responses Lite, with developer/user messages and `additional_tools`
in `input`; top-level `instructions` and `tools` were absent. This custom-provider
observation alone does not establish the requirements of the ChatGPT OAuth endpoint.

This is an HTTP application-layer loopback observation, not a production capture,
TLS fingerprint, Ubuntu sample, or interactive TUI sample. It demonstrates why one
fixed Ubuntu/x86_64/terminal tuple is not a universal CLI identity.

The provider setup follows the official
[custom-provider documentation](https://developers.openai.com/codex/config-advanced/#custom-model-providers).
No real provider credentials or upstream API requests were needed.

### Ubuntu version formatting

The exact official tag `rust-v0.153.4` (commit
`3d2ee51ca2d5db578f328aa75e20aa22c0197c9a`) resolves the apparent `22.4.0` versus
`22.04` discrepancy:

- Codex's [`get_codex_user_agent`](https://github.com/openai/codex/blob/rust-v0.153.4/codex-rs/login/src/auth/default_client.rs#L164-L175)
  formats `os_info.os_type()` and `os_info.version()` directly into the UA.
- Its [`Cargo.lock`](https://github.com/openai/codex/blob/rust-v0.153.4/codex-rs/Cargo.lock#L10406-L10420)
  pins `os_info 3.14.0`. That dependency reads Linux release versions through
  `Version::from_string`, for both [`lsb_release`](https://github.com/stanislav-tkach/os_info/blob/v3.14.0/os_info/src/linux/lsb_release.rs#L9-L16)
  and [`VERSION_ID`](https://github.com/stanislav-tkach/os_info/blob/v3.14.0/os_info/src/linux/file_release.rs#L164-L168).
- The dependency's [parser and formatter](https://github.com/stanislav-tkach/os_info/blob/v3.14.0/os_info/src/version.rs#L55-L83)
  parse numeric components, default a missing patch to zero, and print
  `major.minor.patch`: `22.04` becomes `(22, 4, 0)`, then `22.4.0`.

Thus `Ubuntu 22.4.0` can be a legitimate official CLI UA representation of the
Ubuntu 22.04 release. Replacing the literal solely because the release label has
a leading zero would not be a justified compatibility correction. This is an
exact-version source check, not an Ubuntu runtime capture.

## Inventory

| Area | Value or behavior | Source | Assessment |
| --- | --- | --- | --- |
| Default request identity | `codex-tui`, version `0.146.0`, fixed Ubuntu/x86_64/xterm suffix | [gateway constants](../../backend/internal/service/openai_gateway_service.go), [originator constant](../../backend/internal/pkg/openai/request.go) | Legacy fallback metadata. It is not discovered from the caller's machine. |
| Version floor | `0.144.0` | [identity helper](../../backend/internal/service/openai_codex_identity.go) | Repository compatibility policy, not an independently verified current upstream requirement; opt-out must not silently raise the caller's version. |
| Canonical identity selection | Global UA setting; administrator version, synchronized version, then compiled fallback | [runtime settings](../../backend/internal/service/setting_gateway_runtime.go), [identity resolver](../../backend/internal/service/openai_codex_identity.go) | Already configurable. The legacy resolver rebuilds candidate UA versions; a literal UA is not necessarily sent unchanged. |
| Forced identity | Replaces UA/originator/version by default, independently of convergence | [identity helper](../../backend/internal/service/openai_codex_identity.go), [config](../../backend/internal/config/config.go) | Existing opt-out forwarding repaired in this change. |
| Missing instructions | Model-specific bundled instructions are inserted on applicable Codex/OAuth conversion paths | [forwarding](../../backend/internal/service/openai_gateway_forward.go), [transform](../../backend/internal/service/openai_codex_transform.go), [bundled instructions](../../backend/internal/pkg/openai/instructions.txt) | The audited `instructions*.txt` files contain no `sub2api` text. Nonempty client instructions are retained by defaulting logic. |
| Forced instructions template | Explicit template can replace the final instructions | [template renderer](../../backend/internal/service/openai_codex_instructions_template.go) | Already optional. `{{ .ExistingInstructions }}` is required in the template to retain the converted client instructions. |
| Image bridge text | `<codex-image-generation-bridge>` | [transform](../../backend/internal/service/openai_codex_transform.go) | Actual upstream-visible instruction marker. Automatic bridge injection defaults off; account override precedes channel override and global config. The marker also supports idempotence. |
| Spark image limitation | `<codex-spark-image-unsupported>` | [transform](../../backend/internal/service/openai_codex_transform.go) | Actual upstream-visible marker on the Spark-specific path, independent of the general image-bridge switch. |
| Messages compatibility guard | `<claude-code-todo-guard>` | [guard](../../backend/internal/service/openai_messages_todo_guard.go), [Messages conversion](../../backend/internal/service/openai_gateway_messages.go) | Actual developer-message content in this compatibility path. |
| Reserved tool alias | `python__codex` | [tool-name mapping](../../backend/internal/service/openai_codex_tool_names.go) | Actual wire-level alias with reverse response mapping and collision checks. Changing it requires a compatibility migration. |
| Oversized call IDs | SHA-256 domain `sub2api:codex-call-id:v1:`; maximum 64 bytes | [transform](../../backend/internal/service/openai_codex_transform.go) | The domain string is an input to hashing, not a plaintext field sent upstream. Valid native call IDs now retain their identity; oversized pairs remain deterministically bounded. |
| Converged installation/session/thread IDs | Versioned `sub2api:codex-...` hash domains | [fingerprint derivation](../../backend/internal/service/openai_codex_fingerprint.go) | UUID-shaped outputs are sent, not the domain strings. Renaming domains rotates stable IDs. Convergence defaults off. |
| Account/API-key isolation | Credential- and API-key-scoped session/cache/metadata derivation | [account identity](../../backend/internal/service/openai_codex_account_identity.go) | Separate from optional convergence. Preserves isolation across credentials and gateway users; retained. |
| Compact probes | `sub2api:codex-compact-probe:v1:` hash domain | [compact probe](../../backend/internal/service/openai_compact_probe.go) | Internal derivation input for synthetic probes; not emitted as plaintext. |
| Generated request identities | Canonical identity and separately configured overrides for probes, model discovery, and Live requests | [usage probe](../../backend/internal/service/account_usage_service.go), [model discovery](../../backend/internal/service/upstream_models.go), [Live](../../backend/internal/service/openai_live.go) | These flows construct gateway-originated requests rather than forward incoming client headers. Their shared identity-enforcement behavior is retained. |
| TLS profiles | Configurable TLS-profile feature | [account gating](../../backend/internal/service/account.go), [OpenAI transport](../../backend/internal/service/openai_plugin_transport.go) | Account-level TLS-profile support is gated to Anthropic OAuth/setup-token accounts, not a Codex TLS profile. OpenAI HTTP/WebSocket transport is distinct from the CLI transport. |

Hash-derived identifiers can still have observable formatting and deterministic
relationships. The narrower finding is that these particular domain strings are
not plaintext request markers. Replacing `sub2api:` with `codex-client:` does not
fix a protocol error and can invalidate established correlation/cache identities.

### Additional outbound UA constants found by the repository scan

These values belong to other provider integrations or utility requests. They are
recorded to distinguish their scope from the Codex inference forwarding changes.
Their upstream behavior was not exercised by the local Codex observation.

| Flow | Hardcoded value | Source |
| --- | --- | --- |
| Grok OAuth token exchange/refresh | `sub2api-grok-oauth/1.0` | [OAuth client](../../backend/internal/repository/grok_oauth_client.go) |
| Ollama Cloud usage query | `sub2api-ollama-usage/1` | [usage service](../../backend/internal/service/ollama_cloud_usage.go) |
| Gemini CLI integration | `GeminiCLI/0.1.5 (Windows; AMD64)` | [constants](../../backend/internal/pkg/geminicli/constants.go) |
| Claude usage fallback | `claude-code/2.1.7` | [usage client](../../backend/internal/repository/claude_usage_service.go) |
| Grok billing probes | Pinned `0.2.120` pager/shell UA with `(macos; aarch64)` | [billing identity](../../backend/internal/pkg/xai/billing.go) |
| OpenAI image download fallback | Browser-style Windows/x64 UA with Chrome `131.0.0.0` | [image download](../../backend/internal/service/openai_images.go) |
| Administrator proxy quality checks | Browser-style Windows/x64 UA with Chrome `136.0.0.0` | [proxy constants](../../backend/internal/service/admin_service.go), [probe](../../backend/internal/service/admin_proxy.go) |

## Corrected behavior

### Forwarded identity metadata

With `gateway.disable_codex_identity_enforcement: true`, and without an explicit
account UA override or forced CLI mode, forwarded Codex-protocol requests preserve supplied valid
`User-Agent`, `originator`, and `version` values. Unknown client names and separately
declared versions are not replaced with a canonical identity. Missing Codex fields
are not synthesized; the underlying HTTP library may still use its own default UA.
Malformed header values are not forwarded.

The behavior is applied at Responses HTTP, compact, passthrough, Messages, both
alpha-search builders, and WebSocket forwarding boundaries, after legacy identity
enforcement. Ordinary API-key forwarding is unaffected. In opt-out mode all values
of the effective three metadata fields participate in WebSocket connection
compatibility, including the distinction between missing and explicitly empty
fields, so a socket established with one metadata set is not reused for a request
declaring a different set.

Default legacy enforcement, explicit account UA/forced CLI settings, and synthetic
authentication/probe/Live identities remain separate behaviors. This option does not
disable account/API-key isolation or alter configured convergence modes. An upstream
can still reject unsupported clients or versions.

The configuration example now documents the current key and its deprecated alias:

```yaml
gateway:
  disable_codex_identity_enforcement: true
  force_codex_cli: false
```

### Native tool-call correlation

Native Responses calls use the existing `PreserveToolCallIDs` option. For example,
`call_abc` and `fc_abc` remain distinct instead of both becoming `fc_abc`. Call/output
pairs preserve their mapping, values up to 64 bytes remain unchanged, and longer
values still use the established deterministic compression.

Item IDs and call IDs are separate protocol concepts. Existing item-ID validation
and real item-reference precedence remain intact; the cross-turn legacy reference
fallback now observes the preservation option. Legacy transforms with preservation
disabled retain their prior normalization behavior.

## Validation

Validated on 2026-09-12 with Go 1.27.0 on macOS arm64:

| Check | Result |
| --- | --- |
| `go test -tags=unit -count=1 ./...` | Passed; 56 packages reported successful tests. |
| `go test -tags=integration -count=1 ./...` | Passed; 50 packages reported successful tests, including the PostgreSQL/Redis repository suite. |
| `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.0 run --timeout=30m ./...` | Passed; `0 issues.` |
| `go vet -tags=unit ./internal/service` | Passed. |
| `gofmt -l` on the 15 changed Go files; `git diff --check` | Passed. |
| Independent review of the final source diff | No blocking findings. |
| Audit links and sanitized observation | Local links resolve; JSON matches the successful capture and contains only allowlisted header values. |

Go commands run from `backend/`. Integration tests used the current local Colima
Docker context (Docker 28.4.0), supplied through `DOCKER_HOST`, with
`TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock` for the container socket
mount. The test harness selects PostgreSQL `18.1-alpine3.23` and Redis `8.4-alpine`.

Regression cases were observed failing before their corresponding fixes. The final
tests cover identity opt-out across forwarding routes, explicit overrides, malformed
and absent headers, synthetic cached-UA behavior, complete WebSocket handshake
metadata, distinct tool-call pairing, and cross-turn references. The existing WS
test now checks preserved `fc_hotfix` call/output IDs while retaining its assertion
that an invalid native custom-tool item ID is removed.

No frontend files changed. The checks above do not test production provider policy,
interactive CLI behavior, or transport fingerprint equivalence.

## Remaining behavior and limits

Optional convergence may deliberately collapse device/session/thread cardinality.
In particular, session/full convergence can overwrite previously scoped metadata;
that is different from the normal account-isolation path. It is audited here, not
changed by rotating seeds or removing tenant isolation.

The legacy fallback still contains the fixed Ubuntu tuple. This patch provides
accurate opt-out forwarding instead of replacing it with another machine's captured
tuple. It does not remove functional bridge markers, copy an official transport
fingerprint, or claim to prevent upstream account restrictions. Such restrictions
cannot be attributed to a specific string from source inspection alone.
