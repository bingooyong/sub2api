# Research: Latest main changes after v0.2.19

- Query: Which changes in `v0.2.19..upstream/main` should be backported into the customized `v0.2.19-hotfix1`, without replacing local Codex identity, encrypted-content recovery, sticky sessions, or account quality operations?
- Scope: mixed; locally fetched upstream Git history and patches, current merge worktree, local Trellis contracts. Read-only Git inspection explicitly authorized by the parent for this research slice.
- Date: 2026-11-07
- Output boundary: research only; no implementation, staging, branch changes, or test execution performed by this researcher.

## Findings

### Provenance and release boundary

The latest fetched main is **not** the stable release. `upstream/main` is `d22e0724d6a28435bcf21280333e89883d069a5e`, dated 2026-11-07 11:27:14 +0800. Compared with `v0.2.19`, it changes **446 files, +29,083/-1,417 lines**. Importing all of main would add substantial features and resource-management refactors beyond a bounded stable hotfix.

Verified references:

- Annotated `v0.2.19` tag object: `e60cefeaf78f2fbfa95aa076029e28b40d0a5527`.
- Peeled `v0.2.19^{}` release commit: `dece26153621959ccd7635814f4b2d08304e681a`.
- Local pre-merge base supplied by parent: `c2c75842d6cdebbe8422286dd19afea035baf45a` (`v0.2.18-hotfix1`).
- The existing PRD incorrectly identifies `dece261...` as both tag object and commit; the two references must be distinguished in release evidence.

The stable merge already introduces scoped administrator keys and image-count billing. Those are capabilities; their existence alone does **not** establish that the previous fork contained an exploitable authorization or billing vulnerability. Claims must name a concrete defect and trigger.

### Must backport now: narrow, demonstrable defects in present code

| Commit / component | Evidence and recommended treatment | Dependencies / verification |
| --- | --- | --- |
| `b6c0ca4256b3b437f1fd025690795cc12dda8dd3` — brotli/compress | Current `backend/go.mod:13` contains brotli `v1.2.0`; `:31` contains compress `v1.18.2`. Patch upgrades to `v1.2.7` / `v1.20.2`. Upstream cites reachable GO-2026-6888 (decompression allocation) and GO-2026-6865 (zstd panic); local `backend/internal/pkg/httputil/body.go:170` uses zstd decoding. Backport the two modules and matching checksums. | Independent of MCP despite upstream go.mod context containing MCP. Resolve module versions rather than copying all of upstream go.mod. Run dependency resolution, backend tests/build, and govulncheck if available. Advisory claims originate in upstream commit text; not independently queried here. |
| `a0b058a9e9d73a4f5d9bed2bdccb503eb7ed8cc4` — Bedrock EventStream | Local `backend/internal/service/bedrock_stream.go:265` reads peer-supplied lengths, `:277` allocates from total length, and `:294` slices headers without an upper-bound check. Patch checks headers <=128 KiB, headers <=total-16, payload <=16 MiB **before allocation**. A forged frame can otherwise request about 4 GiB or panic outside Gin recovery. Backport source and upstream regression test. | Small standalone two-file patch. Does not require broad resource framework `457602c8f`. Test oversized payload, oversized/inconsistent headers, and valid frames. Bedrock exposure depends on configured accounts, but hardening is isolated enough to include. |
| `d851a38477597157e78450d30af2d8504764d085` — frontend dependency remediation | Current `frontend/package.json:35` retains vue-i18n `^9.14.5`. Patch upgrades vue-i18n and `@intlify/message-compiler` to `11.4.16`, overrides decode-uri-component >=0.5.1, removes obsolete Vite JIT define, and updates test expectations/CI Node. Upstream cites GHSA-83rm-6c8r-g728 and GHSA-qr28-fpjm-wq7f. Include as one compatible frontend slice. | Vue-i18n 11 requires Node >=22; upstream moves CI/release to Node 24. Include compiler, lockfile, Vite and relevant test/CI changes together. Frozen install, typecheck, lint, tests, build, and dependency audit are required. This is a major dependency upgrade, not a lockfile-only patch. |
| Package/lockfile hunk of `ff04296510f4df36f60b800cc16db432e7bfb0dc` — follow-redirects | Upstream raises the override from >=1.16.1 to >=1.16.2. Extract this dependency hunk with the frontend remediation instead of importing unrelated administrator permission UI changes. | Preserve existing pnpm overrides and regenerate/verify the lockfile. The patch itself does not name a vulnerability ID; do not invent one. |

These items should be represented explicitly in the release plan if the final build is described as incorporating necessary latest-main fixes. A stable-tag-only merge does not include them.

### Optional now: valuable bounded fixes, selected by actual paths in use

| Commit | Actual patch behavior | Integration judgment |
| --- | --- | --- |
| `40d9c0964b05f192f3b31ab155aa037eebf42593` | In `proxyOpenAIWSHTTPBridgeTurn`, clears staged fingerprint IDs, resolves them for the account/client headers, rewrites client_metadata, and stages the same IDs before request construction. Current bridge `backend/internal/service/openai_ws_http_bridge.go:450` proceeds directly from prepared body to Grok adaptation without staging. | Strongly recommended if Codex uses WS→HTTP bridge with fingerprint convergence enabled. It prevents bridge identity/cache-routing divergence. Existing resolver supports off/session/device modes; preserve those modes, caller headers, account isolation, and local identity enforcement switch. This is not proof that OpenAI changes the selected model. Add bridge tests for all modes, stale-ID clearing, account switches, and local enforcement-off behavior. |
| `f10a45c03088b4658bca63141673ed9026cc6c37` | Rejects non-UTF8 keys before cache/database lookup and maps already-cancelled lookup context to 499 in both Google and regular API-key middleware; logs other lookup failures. Local `backend/internal/service/api_key_service.go:737` lacks UTF8 validation. | Small operational correctness fix; recommended to avoid false internal-error alarms. Keep true repository faults at 500. Include middleware and service tests; no fingerprint overlap. |
| `2d6b64a18e619e57ac7364cc7f42918eb49fd76b` | Uses shared wire encoding for completed message output and preserves output_text fields and refusal content. Current `backend/internal/pkg/apicompat/types.go:432` special-cases only tool_search_call; `responses_stream_event_wire.go:210` reduces content to type/text. | Recommended for Responses compatibility bridges. Verify completed JSON and streamed representation retain annotations/refusal fields while preserving existing tool_search_call encoding. Self-contained source/test slice. |
| `30ec2ecc985f07674f2b9384457efbde6fd1972d` | Adds `responsesReasoningText`, preferring nonempty summary and otherwise collecting content[type=reasoning_text]. Uses it in Responses→Anthropic/Chat Completions buffered conversion while retaining encrypted_content as thinking.signature. Current `responses_to_anthropic.go:30` and `responses_to_chatcompletions.go:59` inspect summary only. | Recommended if buffered compatible upstreams use reasoning_text. It preserves visible reasoning; it neither decrypts ciphertext nor proves model substitution. No need to import the earlier streaming memory refactor to implement this small semantic change, but tests may need adaptation to local signatures. |
| `6b2fdfcac262ec826be24c9ca857f39e40ef7119` → `9b5407449c8d0b8f41abba4756da3bab47a67945` → `4db8402b5081d2f3caa2b870b64aa7c18ff057ed` | Keeps the original client stream flag when OAuth forces upstream streaming; sets JSON Content-Type explicitly after aggregation; handles incomplete/cancelled terminal Responses; detects OAuth SSE bodies without Content-Type. Current `openai_gateway_passthrough.go:181` overwrites reqStream from normalized stream=true, and `:2342` only checks response headers. | Treat as one ordered protocol slice if non-stream Responses clients matter. First two patches touch SSE aggregation in both normal and passthrough paths. Preserve compact behavior, usage, terminal errors, body limits, and local recovery. Tests should cover stream=false, compact, completed/incomplete/cancelled, absent Content-Type and ordinary stream=true. Codex normally streams, so not a universal blocker. |
| `e16e69a7c4e0c6145887b615422e0e632658bdde` | In `openai_ws_v2/passthrough_relay.go`, resolves the turn before measuring its first token even when deltas lack response.id. Fixes missing TTFT after the first turn. | Useful for local quality/observability and scheduler samples. Adapt to local response-model observer added in this same block. Upstream two-turn regression is meaningful. Does not fix encrypted-content failures. |
| `c476edbf40f100f04a08619da665439d3dd8e795` | In `openai_apikey_responses_probe.go`, returns early for forced Responses/Chat Completions routing modes. | Small optional reduction of unnecessary upstream probe calls; tests cover forced versus auto mode. Not an account/model routing fix. |

### Defer: broad resource framework and its required follow-ups

The current server intentionally lacks a whole-request ReadTimeout (`backend/internal/server/http.go:124`); current Gemini compatible paths use unbounded `ReadString`, and Antigravity non-stream uses `io.ReadAll` (`backend/internal/service/antigravity_gateway_upstream.go:127`). These are real robustness surfaces, not a claim that a production exploit occurred.

`457602c8fc07112fb6e38642adea88a61d7c9759` introduces **104 files, +4,958/-299**, including process/request memory reservations, read/write timeouts, SSE line/event limits, session-memory caches, bounded workers, usage context lifetime fixes, and generated Wire changes. It touches customized OpenAI forwarding, retries, WS bridge, billing and handlers. Treat as a separate planned integration rather than a casual hotfix cherry-pick. Important related patches:

- `3425ba36637e07e296008113d825f3cd9e0d330e`: independent stalled request-body idle timeout/classification foundation. Default 60s inactivity, 408/499 classification, HTTP/1 drain and HTTP/2 deadline semantics, 33 files. It predates the broader framework; can be isolated if urgently required, but must include server wrapper, config defaults/validation, shared error diagnostics, middleware and handler changes together. Newer resource framework builds on this state.
- `0853d160365dbc87c1ebb33eda18b59b8ed2bb72`: async image tasks must detach their own body reservation after ingress reservation is released. Depends on body-budget APIs introduced by resource framework. Do not import a budget framework without this lifetime follow-up when async images are enabled.
- `957a83f32e68e2dbe5dcc5e02785929b5ace42c9`: settles patch reservations per attempt, releases superseded outputs across failovers, restores decoded-map fallback, and fingerprints reused body snapshots to detect in-place mutation. Depends on resource framework **and** `6c140b89762d3a985c1ccbe214d653d019589598` JSON-copy/parse optimization. Particularly high overlap with local encrypted-content body rewrites/retries.
- `d446c4a6bbf798b34527ef50b20e14a2d90378f6`: replaces Gemini unbounded ReadString with framework's bounded SSE reader, threads cfg through collectors, and bounds Antigravity non-stream responses. Depends on `sse_transport_limits.go` and ReadUpstreamResponseBody from resource framework; not standalone in stable. If those platforms are exposed to untrusted custom upstreams, prioritize a separate bounded-read backport instead of copying missing framework APIs.
- `22693fb4feed5c71c8b18883fca1f7e5e6e50639`: Grok Voice request-body failure classification; inventory-only review, belongs with the shared body-classification lane.

Deferral is a release-size judgment, not dismissal of the risk. For a publicly exposed gateway lacking an edge upload timeout, stalled-body protection should be elevated after confirming the deployed reverse proxy's actual behavior.

### Defer: new management MCP surface and broad administrator permission expansion

- `f17918ea1746df840ef766ba43e019629bf7708e`: OAuth-authenticated management MCP backend feature.
- `cf330063361b265c932b0b11cab2c59323687549`: MCP OAuth consent/connections frontend.
- `ee71c7f379d56d8b28595beb032883acac8c438d`: MCP hardening follow-up.
- `575dd9bc2b10825f27ddf2826cd4643df75aba20`: stateless MCP registration follow-up.
- `fb22c92126b4d4877ce67caf39eb40cb88501b12`: MCP SDK 1.4.1 and management server/route changes for GO-2026-4770/4773.
- `8a34f1ecf30bbcd60a7243e8bbbf3050c07eba7b`, `b3eb7194c29e82ae3738fec14eaec4164d6806cd`: MCP visibility/client-ID UI fixes.

Current stable merge's `backend/go.mod` has no modelcontextprotocol/go-sdk dependency, and MCP management functionality is absent. Do **not** add a new management API merely to absorb its SDK security update. If MCP is adopted later, take feature and hardening as one reviewed unit. Only the SDK patch's source changes/statistics were examined here; other MCP commits are inventory classifications, not a complete security review.

`194d69e636275d6489fd0341ace603b80bcafcbc` is a 61-file expansion: new full_access policy with explicit confirmation, identity/usage/statistics/OpenAI option permissions, response projections, scope-aware search, resource filters, and optimistic update metadata. Actual inspected policy/service changes add `RestrictedFromContext`, `ConfirmFullAccess`, and prior-access/version state; this is **not** a single authorization bypass fix. It also changes scoped account-list signatures, directly overlapping the local plan_tier query adaptation. Defer unless the user requires full-access compatibility or newly exposed scoped operations. Do not pick policy bypass helpers without their call sites, guards, audit/rate-limit/idempotency identity and tests.

`ff04296510f4df36f60b800cc16db432e7bfb0dc` permissions UI portion plus `b0db521ceb9ceea183559f0d8bec35d6fc670911` ES2020 fix depend on that permissions expansion. Extract only the dependency remediation described earlier for this hotfix.

### Remaining inventory: optional platform features or defer until separately reproduced

These entries were classified from inventory/statistics unless explicitly noted above; they are not approved patch-level backports:

| Class | Commits and rationale |
| --- | --- |
| Defer performance rewrites | `6c140b89762d3a985c1ccbe214d653d019589598`, `4ca1a2097ca05e686ade53c9c7a6aa96b878a4c4`, `f654c23eb7af2da100c3c23c7a4e58afa2ee6400`, `00660a227ae18d223af013244501b33e5c2bb6b3`, `9bb073d0d18db5b9940d138c77d6b73273c3e061`, `76221c9cb04a8b09484626456f33fcc75b4ce697`, `1c83bb825a845c7d820194d5bfbee692864cf465`. No demonstrated local latency issue requiring this churn; bootstrap/compaction/body reuse overlap local recovery and auth projection contracts. |
| Defer image capability/routing feature unit | `7f2876defa360317cc876798bed6d95ce26dc38f`, `e452e41e4010b48028e16e5047540974e1b4b78d`, `6b565e45a6d8705bbfdf8f0e847f4bb7812f074a`, `890120120d7e7446041085e6a72233d5c359bb91`. GPT Image 2.6 and 1K/2K/4K selection add feature requirements; sticky follow-up must accompany new size-based routing if adopted. |
| Optional Codex exports | `1e0f56d76fad149e80745fb621712767806fe565`, `1afe5ffe35ce2ede434267f5166f900ee0eabc2a`, `0d2459d986239d5b3e5ab3fedc9c772a5f4c3900`, `e97a48d84be5f4b23898ab2d2f968482dbd0b3ce`. Review desired exported model and image/search settings against local canonical choices before importing. These do not establish actual server-side model routing. |
| Optional platform-specific correctness | `a8d90a72ea05579fae9061783fd886ef42c779e7` (Anthropic weekly-only quota), `17772947bda617439f20300f9930b4010d64d8a8` (Kimi concurrency 403), `aae31c7438a1452bbdb81f7de72f184f0c4f7045` (Gemini tool media), `c35418032b1c549fada003ffff60bd2cdcb65877` (Antigravity direct Gemini streaming), `2106081510f98cbe71e5525593d7e3c762c27f9c` (Cloudflare beta query), `0b535b5c0700d901208136250b1400189c2646d3` (Gemini response schema), `522f1fb999db5380dc88724a71098c7902b094fc` (Grok delegated message normalization). Elevate only for deployed affected platforms and after patch-level review. |
| Defer Antigravity plan/identity unit | `0ee6b6c5d11f9fc91764db380840620c311f12a3`, `455751402f3a85f7720beb67b4f69189b819b30b`, `2ce1849bdfe0acc6c48d3362af4303658bc10950`, `bf4e16b90e3335d4f0c026b60c08c7eb5150f012`, `835803a7021c3255c6f19019329de923085f5f47`, `01a1a78db2c3ce88565d924c0804a40062472221`. New quota windows/account-tier configuration or system identity rewriting need their own behavior validation. |
| Defer behavior-changing OpenAI controls | `a699c2baedc886a6970997cf6b362e4f46f354b0` (empty instructions and response echo, 400 insertions/96 deletions), `7e3b6c72faded2427585a6d528b977943184e0e5` (cyber transcript prefix/session-block matching, Redis/cache changes), `489cfd5f4f7893ac12d3bb93a38d9f1baea7171f` (disable transport cooldown), `fa6f06510b6de03b80c329e2be4e5e0b112c079b` (body-limit passthrough precedence). Need a reproduced requirement and separate policy review; no evidence here of a local trigger. |
| Defer unrelated administrator feature / optional CSV UI | `721dfb282e3bd94b513ad6d2d65dcb27df246369` adds registration email blacklist and bulk disabling; `e47abd8f9bc958949989958f0fe4cb59ff3339c5` changes usage CSV pagination/rate-limit retry. No requirement for this hotfix. |
| Defer release housekeeping | `a9240d5016c226b341242a8c54a01a8605cbbc72` version sync, `c01b2cb817663e79c3ce1db1d611ffe33f2d589a` CI retrigger, sponsor-only commits `e6c19082f24740a098f6753d0fbfef754098c8b5`, `25608d216bb7be04e4a646830cb065cd09b7d7e9`, `876b96d204855ad1aac486b7dd39a95d377d4760`, `251094436325175c9e0ab06e7b962d05369fd766`, `e353c832261c9358949e2071bfb17e99d36aa8cc`. Keep local VERSION `0.2.19-hotfix1`; no need to recreate upstream merge commits. |

### Local contracts and validation boundaries

- Fingerprint convergence controls device/session IDs; outbound identity controls UA/originator/version. They are related but distinct. Preserve enforcement-off caller identity, account UA override precedence, and force_cli precedence. Upstream issue descriptions about risk tiers do not prove silent Luna routing on this user's account.
- Encrypted-content recovery must still classify code-null message errors on HTTP and WS, handle reasoning and message ciphertext, retry within local bounded/same-account rules, and preserve visible/tool content where possible. Neither visible-reasoning preservation nor fingerprint convergence decrypts content or guarantees a specific model.
- Sticky session key remains `(groupID, sessionHash) -> accountID`. Preserve handler-context hash diagnostics rather than recomputing from transformed bodies. Existing production evidence did not prove account switching caused the observed incident; upstream key rotation remains an alternative explanation.
- Broad JSON-copy/bootstrap/compaction optimizations require re-running encrypted-content, orphan tool-call, account isolation and fingerprint tests, not just performance benchmarks.
- Local response-model observation in `openai_ws_v2/passthrough_relay.go` must survive optional TTFT patch placement.
- Source line references describe the in-progress stable-merge worktree when inspected; the parent is concurrently integrating changes, so subsequent line numbers may shift.

### Files found and related specifications

- `.trellis/workflow.md` — requires persistent research, scoped planning and validation.
- `.trellis/tasks/11-07-upgrade-v0-2-19-hotfix1/prd.md` — stable release target, current priority claims and acceptance criteria.
- `.trellis/tasks/11-07-upgrade-v0-2-19-hotfix1/design.md` — pinned stable merge and local preservation boundaries.
- `.trellis/spec/backend/codex-identity-customization.md` — caller identity preservation and enforcement switch contract.
- `.trellis/spec/backend/openai-encrypted-reasoning-recovery.md` — code-null ciphertext classification and bounded recovery.
- `.trellis/spec/backend/openai-sticky-session-encrypted-content.md` — account/session association and failover boundaries; production causation still requires logs.
- `.trellis/spec/backend/openai-upstream-error-handling.md` — error classification and passthrough/retry semantics.
- `backend/go.mod`, `backend/go.sum` — backend dependency versions/checksums.
- `frontend/package.json`, `frontend/pnpm-lock.yaml`, `frontend/vite.config.ts`, `.github/workflows/{backend-ci,release,security-scan}.yml` — frontend dependency/runtime remediation unit.
- `backend/internal/service/bedrock_stream.go` — unchecked peer-controlled EventStream allocation/slicing in stable.
- `backend/internal/service/openai_ws_http_bridge.go`, `openai_codex_fingerprint.go`, `openai_codex_identity.go` — bridge staging and local identity controls.
- `backend/internal/pkg/apicompat/{types,responses_stream_event_wire,responses_to_anthropic,responses_to_chatcompletions}.go` — completed output serialization and buffered reasoning conversion.
- `backend/internal/service/openai_gateway_{passthrough,response_handling}.go` — non-stream OAuth SSE aggregation and local forwarding overlap.
- `backend/internal/server/middleware/api_key_auth{,_google}.go`, `backend/internal/service/api_key_service.go` — lookup failure mapping and key input validation.
- `backend/internal/service/openai_ws_v2/passthrough_relay.go` — per-turn TTFT and local response-model observation.
- `backend/internal/service/{gemini_messages_compat_service,antigravity_gateway_upstream}.go`, `backend/internal/server/http.go` — bounded-read follow-up surfaces.

### External references

- Upstream repository: `https://github.com/Wei-Shaw/sub2api`; each full commit above can be inspected at `/commit/<SHA>`.
- Brotli/compress upstream commit cites `https://pkg.go.dev/vuln/GO-2026-6888` and `https://pkg.go.dev/vuln/GO-2026-6865`.
- Vue-i18n/decode-uri upstream commit cites `https://github.com/advisories/GHSA-83rm-6c8r-g728` and `https://github.com/advisories/GHSA-qr28-fpjm-wq7f`.
- MCP upstream commit cites `https://pkg.go.dev/vuln/GO-2026-4770` and `https://pkg.go.dev/vuln/GO-2026-4773`; not applicable to a build that never adds that SDK/API surface.
- AWS EventStream protocol length constraints are described in the inspected patch as matching AWS SDK: headers 128 KiB, payload 16 MiB. External specification not separately fetched.

## Caveats / Not Found

- This audit inspected actual patches for the priority dependency fixes, Bedrock, fingerprint bridge, auth failure handling, completed-message serialization, buffered reasoning, non-stream Responses sequence, TTFT, routing probe, body-idle timeout, bounded Gemini reads, patch budgeting, async image reservations, and portions of admin permission expansion. Remaining inventory entries are marked accordingly; a title/stat classification is not approval to backport unreviewed code.
- No tests, builds, vulnerability scanners, registry advisory requests, production log access, or deployment performed by this researcher. Advisory reachability described by upstream must be confirmed on the final dependency graph where a formal security claim is needed.
- Do not claim the final artifact contains all latest-main fixes. Record stable merge plus exact selected backports and explicit deferrals.
- No evidence gathered here proves official silent model substitution, guarantees that a fingerprint forces a particular model, or isolates the user's ciphertext failure to account switching.
