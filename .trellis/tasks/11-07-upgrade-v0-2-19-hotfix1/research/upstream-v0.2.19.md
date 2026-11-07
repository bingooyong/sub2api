# Upstream v0.2.19 Evidence

- Local base: `c2c75842d6cdebbe8422286dd19afea035baf45a`
- Upstream annotated tag object: `e60cefeaf78f2fbfa95aa076029e28b40d0a5527`
- Peeled release commit: `dece26153621959ccd7635814f4b2d08304e681a`
- Delta: 71 commits from the local base; 724 files changed by the tag comparison.
- Merge-tree preview: 2 textual conflicts and 23 shared paths.

## Stable-release changes

- Scoped admin API keys with hashed secrets, scope policy, expiry, IP allowlist, revoke/audit.
- Per-image billing and Gemini fileData image accounting.
- DeepSeek cache-hit billing/pricing correction and image-tier token retention.
- Pagination overflow rejection and multipart image upload hard limit.
- Admin guard 400 response, dependency `x/net` security update, and web-search dimension preservation.
- GPT-6.1 Terra, OpenCode DeepSeek developer-role compatibility, malformed tool-call rejection, Gemini retry/model-list fixes, and Grok catalog consistency.

## Compatibility notes

- Gemini defaults remove deprecated 2.x/2.5 models and 3-pro-preview; deployments needing them must use explicit mappings/whitelists.
- Existing Gemini/Antigravity groups are enabled for image generation by migration 246; composite groups require explicit review.
- Upstream migrations 247/248 must coexist with local migration history by full filename.
