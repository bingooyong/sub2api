# Review repair design

## Scope and baseline

Repair confirmed findings from review baseline `808c8c733` on the current `v0.2.21-hotfix1` branch. Keep the pinned upstream merge, published migrations, dependency versions, fingerprint derivation and client identity behavior. No production deployment, remote push or real GPT calls.

## Credential lifecycle without quality mutations

For Pelican OAuth requests, reuse the injected gateway/token provider rather than reading the access-token snapshot directly. Preserve Agent Identity's per-request signing and explicitly retain dependency-free constructor/test compatibility. The existing Pelican context identifies observational probes; use it to prevent the token provider's missing-refresh branch from permanent SetError/runtime blocking/cache deletion, and to return refresh failures instead of dispatching stale fallback credentials. Normal forwarding keeps its current failure policy.

Validate the effective credential after adopting a higher token version from the repository as well as before fallback; a newer database object may itself carry an expired token. Preserve PAT expiry semantics. Test distinct snapshot/repository objects and zero stale dispatch.

Successful credential refresh may update credentials/token version/cache and the related scheduler snapshot/outbox. It must not apply health, 401/429, extra/quota, group, schedulability or isolation mutations. Existing outer probe guards remain in place. Test stale snapshot/fresh cache, successful refresh and persistence, expired token without refresh, refresh failure/cancellation, ordinary connectivity and normal forwarding behavior.

## Supported text-model boundary

Inspect the mapped target using existing provider-prefix normalization semantics, then reject known image/audio/realtime/transcription/TTS/embedding/moderation/video families before upstream dispatch. Preserve arbitrary custom API-key text names and text analysis aliases. Do not replace the existing guard with a GPT-only allowlist. Verify rejected targets make zero upstream requests.

## Passive generated preview

Reuse the already installed DOMPurify and existing SVG sanitation patterns. Normalize model HTML/SVG to a passive document: remove executable content, meta refresh, active navigation links/targets, base/link/resource hints, embedded browsing contexts, forms and equivalent active constructs. Retain supported SVG artwork, inline styling and CSS animation, plus safe internal SVG references/data images where appropriate. Add the trusted CSP before untrusted markup with script execution disabled and a deny-network subresource policy. All three shared consumers use an empty iframe sandbox with no script/same-origin permissions.

Browser follow-up reproduced a CSS background request during iframe insertion despite final meta CSP. Normalize external and obfuscated CSS resources before srcdoc as well: conservatively discard an offending style element/value containing external url(), @import, image/image-set, backslashes or comments; retain plain styles/keyframes and strict fragment-only paint references. Filter all attributes including the root html element, SVG presentation/animation values, and remove srcset/background/poster/xml:base. This may remove externally dependent or obfuscated styling; original text remains available. Exact browser preload timing is not assumed.

Raw model output remains unchanged and inspectable. Generated JavaScript is not executed; script-dependent drawing may only be inspectable in raw output. Make the preview capability explicit in the operator-facing help and specs. The existing production parent nonce CSP already blocks inline model scripts, so the old srcdoc implementation did not reliably support those animations.

Do not claim that connect-src/default-src or unsupported navigate-to directives prevent document navigation. Real Chromium tests must observe no requests during normalization/render/click and no frame navigation for meta/script/anchor payloads under the actual parent CSP; also verify normal SVG/styling/CSS animation remains visible.

## Administrator streaming authentication

Reuse `refreshAuthTokens` for a single HTTP401 refresh/retry before opening the stream. Preserve shared same-document/cross-tab refresh coordination; avoid a second token-refresh implementation. Original AbortSignal remains authoritative while waiting for a shared refresh and before retry; cancellation must not cancel a refresh shared by other callers. Do not retry account-upstream SSE failures or replay after logout/session replacement. Preserve batch prompt/body and admin UI headers across the retry. Add 401 concurrency, cancellation, failed refresh and retry-limit coverage.

The same pre-stream response boundary must handle HTTP423 with `ADMIN_COMPLIANCE_ACK_REQUIRED`: dispatch the existing `admin-compliance-required` event with `metadata || {}`, preserve status/code/message/metadata on the thrown failure, and do not automatically retry. This must open the existing App/store confirmation flow even when backend compliance state changes after initial page loading. Add metadata/error propagation and no-retry regressions. HTTP401 login refresh and HTTP423 compliance confirmation are separate from account-upstream SSE failures.

## Keyboard disclosure

Give the opener aria-expanded/aria-controls with a stable panel ID, restore opener focus on close, and focus the opened panel or first control after rendering. Keep the mounted-result/session and close-cancels-batch contracts. Prefer changes in AccountQualityView without adding a general dialog abstraction.

## Execution ownership

Use two bounded Trellis implement lanes: backend service/provider/model files and their tests; frontend preview/client/view/components/locales and their tests. The leader owns task/spec updates and sequential resource-bounded verification. An independent Trellis check reviews all repairs, including the inherited no-health and identity contracts. No conflicting writers or concurrent expensive Go checks.
