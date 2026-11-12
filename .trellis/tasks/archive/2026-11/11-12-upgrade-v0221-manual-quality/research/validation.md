# Validation record

## Release

- Merge commit `81231fa121b39e0c36ece7ab18eb613cddba796a` has parents local baseline `8bb5def3d82f2162b546899c3f6b0ce7a7f928b7` and upstream v0.2.21 `10ba8f9b4dc2e6023d4a9879ab77b21841babd10`.
- Version is `0.2.21-hotfix1`, branch `v0.2.21-hotfix1`; backup `backup/v0.2.19-hotfix1-before-v0.2.21`.
- Core identity/fingerprint/account-isolation/native-transform files are byte-identical to baseline. Final identity restoration call sites remain in HTTP/passthrough/Messages; WS compatibility/staging reviewed independently.
- All three conflicts resolved; seven pnpm override entries match package/workspace configuration; frozen install passed with pnpm 11.16.0.

## Frontend

- `pnpm lint:check` and `pnpm typecheck`: passed.
- `pnpm test:run`: 390 files / 3,143 tests passed.
- Independent review fixed a reproduced pre-head-script CSP ordering issue; panel/client/helper/old modal regression pass: 4 files / 25 tests; focused ESLint passed.
- `pnpm build`: passed, including 3 locale completeness tests; Vite reports existing mixed-import/chunk-size warnings only.

## Browser (mock accounts and SSE, no real GPT calls)

- Playwright CLI isolated browser at localhost:3107, fake test admin and API fixtures.
- Three-account batch produced independent results: two previews and a plain-text answer with a missing-preview hint.
- Second batch stopped at `success,success,cancelled`; close/reopen retained all three cards.
- 390-pixel viewport settled to a 382-pixel main region and no horizontal page overflow.
- Clean-session fixture initialization skips sandbox frames; no app console errors observed after fixture corrections. Earlier errors were test-fixture announcement shape/setup routing and fixture script access to opaque iframe localStorage, corrected before final smoke.
- Final production-preview review scrolled the frames into view after Stop, close/reopen and mobile resize. Bicycle SVG rendered at 270×202 desktop and 288×216 mobile, with original background/styles; no preview regression reproduced. Evidence: `preview-browser-review.md` and ignored `output/playwright/pelican-prod-{desktop-iframe,mobile-iframe,mobile-viewport}.png`. Sample artwork is a UI fixture, not model-generated validation. Browser sessions and this task's preview server were closed afterwards.

## Backend and infrastructure

- Pre-merge baseline could not finish: compiler ran out of local disk space before testing. Removed this task's incomplete Go temp directory; later removed rebuildable Yarn download cache to keep compilation possible. No project dependencies or user documents removed.
- Initial targeted Go compile overlapped addition of a test import; sources subsequently frozen and fresh run required.
- Frozen-source targeted `GOMEMLIMIT=1GiB GOGC=50 GOMAXPROCS=2 go test -p 1 -tags=unit ./internal/service ./internal/handler/admin -run 'ManualPelican|PelicanTransport|ScheduledPelicanActual|PelicanHandler|PelicanPayloads|OpenAISuccessPersistsSnapshot|OpenAI.*401|OpenAI.*429|OpenAIOutboundBranding' -count=1`: passed (service 3.600s, admin 1.643s).
- `go generate ./ent` and `go generate ./cmd/server`: passed with no generated/module drift.
- Full `go test -p 1 -tags=unit ./... -count=1`: passed (609.8s overall; service package 284.295s).
- Initial repository integration selection passed (64.4s including compilation); full `go vet -p 1 ./...` passed (60.3s).
- Initial configured golangci-lint found five mechanical issues: ignored channel receives (three tests), unchecked `strings.Builder.WriteRune`, and equivalent outcome if/else branches. The initial explicit channel-discard attempt conflicted with staticcheck S1005 and was superseded by bound shutdown results used in test diagnostics. WriteRune retains explicit discards and the outcome branch uses a tagged switch; no business behavior changed by these lint edits. Existing memory-budget/WS/plan-tier/quality tests lock those behaviors. Fresh affected unit tests passed (589.4s including recompilation; httputil 0.448s, service 3.926s). Exact selection: `RequestBodyMemoryBudgetReusesTransientCapacity|OpenAIOutboundBranding|ReadOpenAIWSClientMessage|NormalizePlanType|PlanTier|ManualPelican|PelicanTransport`. Subsequent complete-report lint expanded the shutdown sites to fourteen locations across six test files; all original protocol assertions remain. See `lint-loop.md`. Final checks follow below.
- Docker server responds with version 28.4.0 on the existing colima context. Initial image inventory timed out under host pressure; retry succeeded and required `postgres:18.1-alpine3.23` / `redis:8.4-alpine` images are already present.

## Trellis review and scope

- Independent review confirmed the final fourteen shutdown diagnostic edits preserve prior behavior/assertions, including fingerprint metadata and cases that require actual server-error assertions. Scheduled real-schema restoration and the new quality guidelines were also reviewed with no remaining confirmed findings.
- Independent Trellis check completed; the confirmed CSP ordering defect was fixed and its regression checks passed. Final production-browser follow-up found no remaining confirmed issue.
- Backend/frontend package quality-check indexes and dependency/build contracts reviewed; task context manifests validate (10 implement + 10 check entries).
- No production deployment, remote push or live GPT account calls were performed. No automatic intelligence verdict or invented quota/token metrics were added.

## Expanded integration finding

With `CI=true` and verbose output, the v0.2.19→v0.2.21 upgrade, migration idempotency and Pelican persistence tests passed on PostgreSQL/Redis containers. Adding `QualityActionsRestoreOwnershipAndStaleRuns` exposed a pre-existing schema mismatch before exercising restoration: `account_groups.allowed_models` does not exist in any published migration or the Ent schema. The production restoration INSERT had the same mismatch. A bounded fix restores the actual four membership columns and extends the existing regression to compare priority and created_at. Fresh integration/review/check evidence follows below.

The legacy quality fixture also assumed host and database clocks matched when claiming a plan just triggered with SQL `NOW()`. It now reads `clock_timestamp()` from the same disposable database for claim/lease times; the original successful-claim assertion remains strict, with no sleeps or production scheduling changes. Independent review confirmed real-schema parity and compatibility with existing JSON snapshots.

## Final correction recheck

- `CI=true go test -p 1 -v -tags=integration ./internal/repository -run 'V0219UpgradeToV0221|MigrationsRunner_IsIdempotent|AccountQuality|Pelican|QualityActionsRestoreOwnershipAndStaleRuns' -count=1 -timeout=300s`: passed (67.3s including recompilation/startup; repository tests 18.682s). Logs explicitly show PostgreSQL/Redis startup and all selected tests passing; this was not an infrastructure skip.
- The passing selection includes both scheduled isolation/restore actions, ownership/stale/conflict/outbox/audit behavior, upgrade history preservation, full-filename checksums, actual new schema objects, idempotency, and scheduled Pelican retention/leases.
- Fresh affected-package `go vet -p 1 ./internal/service ./internal/repository ./internal/pkg/httputil`: passed (15.3s).

- Final complete-report `golangci-lint run --concurrency 1 --timeout 20m --max-same-issues 0 --max-issues-per-linter 0 ./...`: passed, **0 issues** (381.7s).
- Final changed-test regression `go test -p 1 ./internal/pkg/httputil ./internal/service -run 'RequestBodyMemoryBudgetReusesTransientCapacity|OpenAIOutboundBranding|ReadOpenAIWSClientMessage|PassthroughIngress|PassthroughFingerprint|WSFingerprint|PassthroughLifecycle' -count=1`: passed (45.0s; httputil 0.492s, service 27.409s). This run uses no extra build tag and exercises the final shutdown diagnostics, branding captures, fingerprint frame parity, cancellation and lifecycle assertions.

- Embedded server build `go build -p 1 -tags embed -trimpath -ldflags '-s -w -X main.Version=0.2.21-hotfix1' -o bin/server-v0.2.21-hotfix1 ./cmd/server`: passed (184.5s), using the final frontend production assets.
- Binary `-version`: passed (2.6s), reports `0.2.21-hotfix1`. Artifact: ignored `backend/bin/server-v0.2.21-hotfix1`.
- Final source whitespace check passed. All required checks are complete; there are no remaining confirmed review findings. Pre-merge baseline assertions and real-account/live-deployment behavior were not claimed as verified.

## Work commits

- `81231fa12`: pinned upstream merge preserving fork customizations.
- `61dbfc2b4`: real-schema group restoration and fresh-database regression.
- `6d96a1ee8`: compatible shutdown diagnostics and backend lint contracts.
- `f5109cf14`: manual comparison feature, transport/branding/migration regressions and Trellis records.
