# Implementation Plan

## 1. Baseline and Branch

- Record clean tracked status, current head, remotes, tag object, and peeled commit.
- Run the current targeted smoke tests needed to establish a known baseline.
- Create a backup reference for `36b75566c` and branch `v0.2.14-hotfix1`.
- Set Trellis branch metadata.

## 2. Upstream Merge and Version

- Merge `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d` with `--no-ff`.
- Verify the merge's second parent matches the pinned commit.
- Set `backend/cmd/server/VERSION` to `0.2.14-hotfix1`.
- Populate `research/resolution-matrix.md` for all shared paths plus `VERSION`.

## 3. Semantic Integration

- Review configuration, Codex gateway/transform, account services, handlers, routes, Wire, API contracts, frontend types, settings/accounts views, tests, and i18n.
- Preserve upstream security, billing, TypeSafe, remote catalog, and GPT-6.1 behavior.
- Preserve local fingerprint, identity, WebSocket, session isolation, account isolation, Pelican, and account quality behavior.
- Record explicit production-default decisions for API-key limits and inflight reservation.

## 4. Migration Verification

- Keep both upstream `241_*` files and local `243/247/248/249` files unchanged.
- Test empty database migration, representative current database upgrade, and second-start idempotency.

Release gate: representative upgrade-database verification is mandatory.

## 5. Targeted Regression

- Verify EasyPay callback security, password reset, verification rate limits, API-key limits, inflight reservation, Redis failure, concurrent balance, and deleted-key settlement.
- Verify GPT-6.1, remote model catalog, fingerprint, identity, session isolation, tool IDs, WebSocket, account isolation, Pelican, quality operations, priority, TypeSafe, settings, filters, types, and i18n.

## 6. Full Verification and Build

- Run backend generation, tests, lint/static checks, and build.
- Run frontend frozen install, lint check, typecheck, tests, and build.
- Run local health/config/migration/API smoke checks when supported.
- Verify the artifact reports `0.2.14-hotfix1` and excludes `.omc/` and unrelated work.

## 7. Review and Finish

- Dispatch Trellis quality check and fix verified findings.
- Update specs only for durable new project knowledge.
- Commit the upgrade and run the Trellis finish workflow.
- Push after all release gates pass, as already authorized by the user.
