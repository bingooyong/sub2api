# Preview navigation browser reproduction

## Method

Executed the current `extractPelicanHtml` implementation (transpiled with the installed TypeScript compiler) in isolated Chromium. The fixture parent received the exact `DefaultCSPPolicy` from backend config, replacing only its nonce placeholder with a fixed fixture nonce. Used the production iframe attributes `sandbox="allow-scripts"` and `referrerpolicy="no-referrer"`.

All browser requests were intercepted and fulfilled from in-memory dummy HTML. No application account, real external endpoint or GPT API was called. The test host was `https://preview-review.test`.

## Observed production-policy evidence

| Payload | Result |
| --- | --- |
| meta refresh to same-site dummy path | Request observed; frame left srcdoc |
| meta refresh to CSP-allowlisted captcha host dummy path | Request observed; frame left srcdoc |
| SVG anchor clicked toward same-site dummy path | Request observed; frame left srcdoc |
| Inline script navigating the frame | Blocked by inherited parent nonce policy |
| Inline script marking a drawing as animated | `inlineRan=false` under production nonce policy |

No page errors occurred. Parent DOM/storage isolation remains intact; this is a preview navigation boundary failure rather than parent-page XSS. The preview's own `default-src`/`connect-src` restrictions do not enforce its documented no-network guarantee for document navigation. Removing only scripts is insufficient because meta refresh and SVG links reproduce without scripts.

The parent-without-header comparison allowed the inline drawing script, but some concurrent navigation fixtures became browser error documents. No stronger navigation conclusion is inferred from that comparison; the production-policy requests above independently reproduce the issue.

## Artifacts and lifecycle

- `output/playwright/review-v0221-csp.js`: executable reproduction and exact normalized fixture outputs.
- `output/playwright/review-v0221-csp.log`: browser result and run-code transcript.
- The isolated `review-v0221` browser was closed after observation.

## Repair direction

Reuse the already-installed DOMPurify for a passive HTML/SVG preview, disable iframe scripts and script CSP, and remove navigation-capable constructs consistently in all consumers. Keep CSS/SMIL animation and raw model output. The production parent nonce policy already prevents inline model scripts; a script-driven canvas/DOM animation cannot be promised by the current srcdoc implementation. Make this rendering contract explicit in the reviewed fix plan.
