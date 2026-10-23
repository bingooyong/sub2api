# Implementation Plan

## 1. Baseline and Branch

- Record current head and targeted green baseline.
- Create `backup/v0.2.14-hotfix1-before-v0.2.17` and `v0.2.17-hotfix1`.
- Set Trellis branch/base/scope metadata.

## 2. Merge and Resolve

- Merge `060d0160417782cd962a0c7f460f6cd4565d4916` with `--no-ff`.
- Resolve all seven conflicts while retaining both contracts.
- Set `backend/cmd/server/VERSION` to `0.2.17-hotfix1`.
- Verify parent 2 and populate `research/resolution-matrix.md`.

## 3. Semantic Integration

- Review all 30 shared paths against both parents.
- Reconcile platform catalog with filters, Pelican, quality operations, quota settings, types, and locales.
- Regenerate Wire/generated sources and inspect deltas.
- Verify the exact encrypted-content failure reported on server 160.

## 4. Database

- Preserve all upstream and local migration filenames.
- Test empty database, representative hotfix upgrade, and second-start idempotency.
- Assert upstream platform/Gemini schema and local Pelican/quality schema coexist.

## 5. Verification

- Run targeted encrypted-content, Codex, account filter, Pelican/quality, platform catalog, payment/security, Gemini, and retirement tests.
- Run backend generation, full tests, compatible lint, build, and version smoke.
- Run frontend frozen install, lint, typecheck, full tests, and production build.
- Run PostgreSQL/Redis migration integration tests.

## 6. Finish

- Dispatch independent Trellis check and fix findings.
- Update durable specs when warranted.
- Commit, archive this task, journal, and push `v0.2.17-hotfix1` after gates pass.
- Do not deploy or create a release tag.
