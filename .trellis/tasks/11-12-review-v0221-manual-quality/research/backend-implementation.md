# Backend implementation and verification

## Change boundary

Repair F4 at the actual Pelican OAuth credential dispatch and provider lifecycle, and F5 at the mapped model classifier. Expected product files: `account_test_service.go` (use injected gateway for Pelican), `openai_token_provider.go` (observational failure/missing-refresh/expiry handling), `pelican_test_service.go` (provider-qualified known non-text classification). Regression files: `pelican_oauth_lifecycle_test.go`, `pelican_manual_transport_test.go`. Ordinary connectivity, normal provider refresh fallback/permanent-error policy, Agent Identity signing, fingerprint/branding behavior, published schema and scheduled isolation/actions remain unchanged.

Regression tests are added before product edits. Actual transport tests use in-memory fake upstream responses and instrument credentials, health writes, cache deletion and runtime scheduling calls; no real GPT calls.

## Validation status

Regression-before-edit baseline command: `GOMEMLIMIT=1GiB GOGC=50 GOMAXPROCS=2 go test -p 1 -tags=unit ./internal/service -run 'ManualPelicanOAuthLifecycle|OpenAITokenProviderPelican|ManualPelicanRejectsKnownNonText|ManualPelicanAcceptsCustomText' -count=1`. Interrupted only this task's known Go test/compiler processes after more than 20 minutes compiling, no test execution or diagnostics (~9 minutes compiler CPU). This is incomplete baseline validation, not an assertion failure or pass. Host pressure initially reduced disk headroom to 7.6 GiB, then recovered to 14 GiB and 30% free memory. Parent agreed to use 2 GiB soft limit for final verification.

All subsequent expensive Go checks run sequentially from `backend` with `GOMEMLIMIT=2GiB GOGC=50 GOMAXPROCS=2` and `-p 1` where supported. Raw logs are task-specific `/tmp/sub2api-backend-repair-*.log`; final commands/results will be recorded here.

## Final targeted regression command

```bash
GOMEMLIMIT=2GiB GOGC=50 GOMAXPROCS=2 go test -p 1 -tags=unit ./internal/service ./internal/handler/admin -run 'ManualPelican|PelicanTransport|ScheduledPelicanActual|PelicanOAuth|OpenAITokenProvider|RefreshIfNeeded|PelicanHandler|PelicanPayloads|OpenAISuccessPersistsSnapshot|OpenAI.*401|OpenAI.*429|OpenAIOutboundBranding|Fingerprint|AgentIdentity' -count=1
```

First pre-follow-up run: PASS; raw log `/tmp/sub2api-backend-repair-green.log`.

The first targeted run passed before independent follow-up: service 16.094s, admin 1.339s (exit 0). Independent check then found that adopting a higher-version database credential after the initial expiry guard could bypass expiry validation. Added probe-only expiry validation on the adopted account with PAT exemption, plus an actual transport test using distinct snapshot/database account objects (expired negative with zero requests, fresh positive, PAT compatibility). Final rerun uses the identical command above and `/tmp/sub2api-backend-repair-final.log`; the earlier pass is not represented as final-source evidence.

Final targeted rerun: **PASS**, exit 0; service 11.231s, admin 16.731s. Independent checker reviewed the follow-up and found no additional confirmed backend issue. Logs: `/tmp/sub2api-backend-repair-final.log`.

Full vet command (sequential after targeted tests): `GOMEMLIMIT=2GiB GOGC=50 GOMAXPROCS=2 go vet -p 1 ./...`; log `/tmp/sub2api-backend-repair-vet.log`; **PASS**, exit 0, no diagnostics.


Full configured lint command (sequential after vet): `GOMEMLIMIT=2GiB GOGC=50 GOMAXPROCS=2 golangci-lint run --concurrency 1 --timeout 20m --max-same-issues 0 --max-issues-per-linter 0 ./...`; log `/tmp/sub2api-backend-repair-lint.log`; **PASS**, exit 0, `0 issues.` The resource-bound run completed before its 20-minute timeout (about 19 minutes including compiler/analyzer work).

## Test scope and compatibility

- Actual manual transport uses fresh cached bearer tokens and persisted renewed credentials/token version; failed/cancelled renewal and expired/no-refresh, lock-held and missing refresh wiring send zero upstream requests.
- Provider-level missing-refresh tests assert no permanent error, runtime block or cache deletion. Existing `NoRefreshTokenExpired_DisablesAccount` and refresh-failure forwarding fallback regressions pass unchanged for ordinary requests.
- Direct/mapped/provider-qualified known non-text targets are rejected without upstream requests; custom text/analysis aliases stay accepted and preserve their payload model names.
- Ordinary connectivity continues to use the original snapshot path; Agent Identity bypasses bearer resolution and sends an assertion without exposing its private key.
- The selected existing refresh API, actual manual/scheduled probe, handler, 401/429, fingerprint/identity and outbound branding regressions passed. No repository/schema/migration changes or real account calls occurred; database integration is therefore not rerun for this backend slice.


## Final ownership and remaining handoff

Changed five backend files: `account_test_service.go`, `openai_token_provider.go`, `pelican_test_service.go`, `pelican_manual_transport_test.go`, and new `pelican_oauth_lifecycle_test.go`. This agent also wrote this research record only. Final targeted tests, full vet, full configured lint and source whitespace check pass. No pending backend source edit, known backend finding, commit, live API call or process remains from this lane. Main session owns spec synchronization, integrated frontend/browser checks, embedded server build/version, commit/archive/journal. Full backend unit suite was not repeated; the changed provider, refresh API, manual transport, handler and identity/branding behavior are covered by the named final targeted selection.
