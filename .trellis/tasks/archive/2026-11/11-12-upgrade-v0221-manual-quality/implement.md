# Execution plan

## Planning gate

- [x] Inspect current branch, identity/quality specs, reference checkout, upstream tag and merge preview.
- [x] Persist PRD, design, execution plan and curated agent contexts.
- [x] Latest planning summary approved by user (`start`); task activated for implementation.

## 1. Baseline and upgrade (leader)

- [x] Record pinned upstream and local baseline, create backup ref and release branch.
- [x] Attempt targeted pre-merge regressions; compilation was blocked by exhausted disk before assertions. Record this limitation and run fresh post-merge verification.
- [x] Merge v0.2.21 with `--no-ff`; resolve ignore/VERSION/admin-key scope conflicts.
- [x] Review shared/automatically merged local fix paths, published migration identities and pnpm workspace override/lock parity; write resolution matrix.

## 2. Backend manual probe lane (Trellis implement)

- [x] Add transport-level regressions for the currently ignored custom prompt/effort, default probe compatibility and no health mutation on success/401/429/errors.
- [x] Wire options into actual OAuth/API-key Responses and Chat Completions payloads; validate bounded inputs and supported modes.
- [x] Preserve credential/proxy/identity/fingerprint contracts, and scheduled runner's explicit judge actions.
- [x] Add final serialized outbound assertions for generated branding and cancellation.

## 3. Frontend manual comparison lane (Trellis implement)

Depends on the agreed existing-endpoint/SSE contract; may run alongside backend edits after upstream integration, with disjoint ownership.

- [x] Add focused manual component and typed SSE client using canonical question/HTML helpers.
- [x] Expose it in Quality Operations with account search, multi-select, shared settings, bounded request queue, result cards and raw output.
- [x] Implement stop/close/unmount cancellation, stable batch settings and stale-result prevention.
- [x] Add English/Chinese labels and meaningful component/client tests for success/failure, real prompt payload, queue bounds, cancellation, selection pagination and preview isolation.

## 4. Verification and review

- [x] Backend focused manual/branding/state regressions passed; full `go test -p 1 -tags=unit ./... -count=1` also passed, covering identity/fingerprint/tool-correlation/quality suites. See `research/validation.md` for exact executed commands.
- [x] Backend full test/check/build: project generation if upstream requires it, `go test ./...`, `go vet ./...`, configured golangci-lint when available, and server build/version smoke with matching frontend snapshot.
- [x] Frontend `pnpm install --frozen-lockfile`, `pnpm lint:check`, `pnpm typecheck`, focused tests and `pnpm test:run`, `pnpm build` (includes i18n gate).
- [x] Review migration compatibility and run available empty/upgrade/idempotency checks if local integration services are available; report infrastructure limits precisely.
- [x] Browser smoke with local app if available; use mocked transport for repeatable manual-card/cancellation behavior. Live GPT account probes only if a safe configured local environment supports them; do not invent evidence.
- [x] Dispatch independent Trellis check; fix verified findings and rerun affected checks.

## 5. Finish (leader)

- [x] Update quality/identity specs with implemented contracts and final validation record.
- [x] Commit merge and manual feature in reviewable commits, including task-owned artifacts only; no unrelated task or runtime files.
- [x] Archive completed task and record journal following Trellis finish-work. Report branch/version, checks and remaining validation gaps; no remote push or deployment.
