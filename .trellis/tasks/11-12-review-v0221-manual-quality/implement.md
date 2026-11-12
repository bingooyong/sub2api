# Review and repair execution plan

## Planning and evidence

- [x] Create the approved review task and curate real context manifests.
- [x] Independently review backend, frontend and upgrade preservation boundaries.
- [x] Reproduce iframe navigation with real Chromium and intercepted-only requests.
- [x] Finish the six-finding inventory (F1–F6), acceptance mapping and PRD/design convergence.
- [x] Present final scope/compatibility/test plan; user explicitly approved implementation: “同意，继续”.

## Backend repair lane

- [x] Add meaningful regressions for known non-text mapped/provider-qualified models and custom text aliases.
- [x] Fix text-only dispatch classification with zero upstream requests on rejection.
- [x] Add cache/refresh/no-refresh/failure/cancellation and adopted-version-expiry tests with health/runtime write spies.
- [x] Use gateway/provider credential lifecycle with observational safeguards; preserve normal forwarding and Agent Identity/default connectivity behavior.

## Frontend repair lane

- [x] Add passive-preview sanitation, navigation/external CSS and root-attribute regressions using existing DOMPurify.
- [x] Apply no-script sandbox consistently to ManualQualityPanel, IQTestModal and PelicanRecordsDashboard; preserve raw output and ordinary SVG/style rendering.
- [x] Add one shared-coordinated auth refresh retry to manual streaming, abort/session safeguards and tests.
- [x] Add HTTP423 compliance event/metadata and error propagation without retry; verify the existing confirmation flow.
- [x] Repair disclosure state/focus, and add keyboard interaction coverage.
- [x] Update clear operator-facing preview capability copy.

## Verification and independent check

- [x] Run focused frontend helper/client/component/view and token-refresh regressions, lint, typecheck, locale/build checks; full frontend suite when warranted by shared helper changes.
- [x] Sequential bounded backend targeted manual/token-provider/model/ordinary-test/identity-branding regressions; configured vet/lint with complete reports.
- [x] Browser test normal artwork plus script/meta/anchor requests under production-like parent CSP, cancellation/reopen and keyboard focus. Use fake accounts only.
- [x] Run independent Trellis check, repair confirmed findings and rerun affected checks.
- [x] Build final frontend assets and embedded server/version after source changes settle. Preserve migration identity; rerun database integration only if repository/schema code changes.

## Finish

- [x] Sync domain/frontend/quality specs to the final contracts and record exact evidence/limitations.
- [ ] Commit scoped repairs and task/spec records, no push.
- [ ] Archive only this task, record journal, confirm clean working tree and report the repaired findings.

## Validation command shape

Run from `frontend`: `pnpm test:run` (focused files first, full suite after shared helper changes), `pnpm lint:check`, `pnpm typecheck`, `pnpm build` (includes locale completeness). Browser tests use intercepted fixtures and the backend production CSP; observe actual requests/navigation, not only sanitizer string snapshots.

Run expensive backend commands sequentially from `backend`, with `GOMEMLIMIT=2GiB GOGC=50 GOMAXPROCS=2` (1 GiB baseline compilation was interrupted under host pressure; final checks use 2 GiB):

- Targeted tests: `go test -p 1 -tags=unit ./internal/service ./internal/handler/admin -run '<actual repair tests plus existing manual/token-provider/identity/branding selections>' -count=1`; record the final concrete selection and output in `research/validation.md`.
- `go vet -p 1 ./...`.
- `golangci-lint run --concurrency 1 --timeout 20m --max-same-issues 0 --max-issues-per-linter 0 ./...`.
- After the final frontend build, `go build -p 1 -tags embed -trimpath -ldflags '-s -w -X main.Version=0.2.21-hotfix1' -o bin/server-v0.2.21-hotfix1 ./cmd/server`, then binary `-version`.

Broaden the backend test selection if token-provider behavior or independent review requires it. Do not run expensive Go checks concurrently. Do not treat the previous task's passing tests or reviewer-reported checks without saved logs as fresh repair evidence. Changes to repository/schema code require the affected upgrade/idempotency/quality-restoration integration selection again.

## Risk and rollback points

The token-provider observation boundary and shared preview helper have the largest compatibility surface. Keep their changes independently reviewable, preserve normal non-probe policy, and add regressions before changing behavior. Use existing utilities and dependencies. Backend and frontend implementers own disjoint files; neither reverts another agent's changes or recursively delegates. The leader owns task/spec records and final verification; the independent checker reviews the integrated result. Revert only task-owned changes if a repair fails, preserving inherited fingerprint and upgrade commits.
