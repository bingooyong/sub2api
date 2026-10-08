# Technical Design

## Integration Strategy

Create `v0.2.14-hotfix1` from the current `v0.2.10-hotfix1` head and merge the pinned upstream commit with `--no-ff`. A single merge preserves upstream topology and avoids replaying four patch releases independently.

The upstream tag is an unsigned annotated tag. Record both the tag object (`1400a7b482974d98db5b284a8b2afbe3eaf9aaef`) and peeled commit (`0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d`); merge the peeled commit and verify it is the merge's second parent.

## Semantic Merge Review

Git's merge-tree preview has no text conflicts, but 17 paths were modified on both sides. Treat them as semantic integration points:

- Configuration: preserve local Codex identity enforcement controls and incorporate upstream API-key and inflight-reservation settings.
- Codex data plane: preserve local fingerprint, identity, session, tool-call ID, and WebSocket behavior while incorporating upstream GPT-6.1 and catalog behavior.
- Account/admin layers: preserve local plan-tier, Pelican, and quality operations while incorporating upstream TypeSafe and account-priority features.
- Dependency injection, routes, generated code, API contracts, frontend types, settings, account views, tests, and i18n must remain mutually consistent.

Record the decision for each shared path in `research/resolution-matrix.md` during implementation.

## Version Contract

Upstream `v0.2.14` still contains `0.2.13` in `backend/cmd/server/VERSION`. Set it deliberately to `0.2.14-hotfix1`. Validate both source fallback resolution and the final binary/container version output.

## Migration Contract

Do not renumber or edit published migrations. The runner keys migrations by complete filename, so upstream `241_add_payment_order_bonus_amount.sql` and `241_add_typesafe_platform.sql` can be added after local higher-numbered migrations were already recorded.

Validate three database states: empty database, a representative current database with local `243/247/248/249` already recorded, and a second application start proving idempotency.

## Configuration Compatibility

Record explicit decisions for inflight reservation, API-key count and rate limits, Redis failure behavior, concurrent balance accounting, deleted-key settlement, and password-reset link invalidation.

## Rollback Boundary

This task builds and verifies a candidate branch; it does not deploy. A production rollback must not silently restore vulnerable EasyPay behavior. Database restore requires a stop-write/PITR decision because post-upgrade payment and billing records cannot be discarded casually.
