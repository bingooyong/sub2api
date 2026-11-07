# Implementation Plan

## 1. Baseline

- Record current head `c2c75842d`, branch, upstream tag/commit, and targeted green tests.
- Create `backup/v0.2.18-hotfix1-before-v0.2.19` and branch `v0.2.19-hotfix1`.
- Set Trellis branch/base/scope metadata.

## 2. Merge and Resolve

- Merge pinned upstream `dece26153621959ccd7635814f4b2d08304e681a` with `--no-ff`.
- Resolve `.gitignore` and `VERSION`, setting `VERSION=0.2.19-hotfix1`.
- Review 23 shared paths and create `research/resolution-matrix.md`.

## 3. Selected Post-tag Backports and Regression

- Backport isolated compression, Bedrock, authentication and frontend dependency security fixes; document exact SHAs in latest-main triage.
- Keep resource management/MCP, request-body timeout and optional Responses/WS changes deferred as documented.
- Run scoped-admin-key policy, hashing, expiration/IP, revoke/audit, middleware, and route tests.
- Run image/per-request/Gemini/DeepSeek billing and pricing tests.
- Run pagination, upload-limit, admin-guard, malformed-tool-call, web-search, model catalog, Gemini retry, GPT-6.1 Terra, and OpenCode role tests.
- Run local encrypted-content recovery, sticky session/session-hash, Codex identity/fingerprint/WebSocket, account filters, Pelican, and quality tests.

## 4. Schema and Build

- Keep migrations unchanged; test empty DB, representative `v0.2.18-hotfix1` upgrade, and second-start idempotency.
- Run `make generate` and inspect generated diffs.
- Run `go test ./...`, backend build, and binary version smoke.
- Run frontend frozen install, lint, typecheck, full tests, and production build when pnpm is available.

## 5. Finish

- Dispatch independent Trellis check and fix verified findings.
- Update durable specs only for new contracts.
- Commit, archive this task, journal, and push `v0.2.19-hotfix1`; do not deploy or create a release tag.
