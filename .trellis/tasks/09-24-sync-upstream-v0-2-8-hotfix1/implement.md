# Implementation plan

1. Re-fetch official upstream and confirm `main` is still
   `a3eb7ef302961cba716dc78b39b93b60c467db0e`. If it moved, stop and revise the
   plan. Record the tag peel `fd80b08c90b55edcad5b00171b53f08721d30da1`.
2. Create a uniquely named local backup branch and a private recovery directory
   containing a Git bundle, local commit patch, critical-file checksum inventory,
   and metadata. Do not commit or delete untracked workspace directories.
3. Run focused pre-merge Codex identity, call-ID, and WebSocket fingerprint tests.
4. After the objects are local, list every path changed on both sides with
   `git diff --name-only 1c0a69c0c a3eb7ef` intersected with
   `git diff --name-only 1c0a69c0c HEAD`. The planning intersection was capped
   by the GitHub 300-file compare payload. Any extra overlap is resolved with
   the same rule: keep the local forwarding contract and the upstream hunk.
5. Merge the recorded upstream commit into `dev` with `--no-ff`. Inspect parents,
   conflicts, and the first-parent diff.
6. Confirm `openai_codex_fingerprint.go` and `openai_codex_transform.go` still
   have the local WS helper, unbranded markers, and call-ID preservation, plus
   upstream `marshalCodexTurnMetadata` and GPT-6 Sol/Luna entries. Take the
   Claude effective-version identity changes as upstream wrote them. Confirm
   gateway builders still call `preserveCodexClientIdentityHeaders` after
   enforcement, `PreserveToolCallIDs` remains, and
   `applyOpenCodeUpstreamUserAgent` still no-ops outside OpenCode and official
   Command Code. Keep the local identity opt-out comment and example key, the
   handoff/audit gitignore exceptions, and upstream simple-mode config.
7. Compare critical-file checksums. Re-run the focused regressions, `git diff
   --check`, and targeted vet.
8. Export the merged commit into a private build staging directory. Confirm the
   export excludes untracked workspace state.
9. Use Corepack/pnpm 9 with `pnpm install --frozen-lockfile`, then build the
   frontend into the backend embed directory.
10. Cross-build `backend/cmd/server` for Linux amd64 with embedded assets and
    linker metadata for `0.2.8-hotfix1`.
11. Create `release/sub2api-v0.2.8-hotfix1-linux-amd64.tar.gz` without
    overwriting an existing file.
12. Verify archive contents, SHA-256, ELF architecture, version output, embedded
    UI, source revision, and final repository status. Record any unavailable
    runtime smoke check precisely.

## Validation commands

- `git fetch upstream --prune --tags`
- `git rev-parse a3eb7ef302961cba716dc78b39b93b60c467db0e^{commit}`
- `go test -tags=unit -count=1 ./internal/service -run '<focused patterns>'`
- `go test -count=1 ./internal/pkg/openai -run '<identity patterns>'`
- `go vet -tags=unit ./internal/service`
- `pnpm install --frozen-lockfile && pnpm run build`
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embed -trimpath ...`
- `tar -tzf <archive>`, `shasum -a 256 <archive>`, `file <binary>`
- Linux or container execution of `sub2api -version` when a compatible runtime
  is available.

## Rollback points

- Before merge: backup branch and Git bundle at `d9b39597a`.
- After merge: stop on an unresolved fingerprint conflict or a failed
  preservation check. Do not build.
- During build: discard only the private staging directory. Leave source and
  user-owned untracked workspace state untouched.

## Context for implement and check

- `.trellis/spec/backend/codex-forwarding.md`
- `.trellis/spec/backend/index.md`
- `.trellis/tasks/09-24-sync-upstream-v0-2-8-hotfix1/research/upstream-v028-delta.md`
- `.trellis/tasks/09-24-sync-upstream-v0-2-8-hotfix1/research/upstream-fingerprint-overlap.patch`

## Completion evidence (2026-09-24)

Working-tree note only; not committed.

### Upstream fetch and merge

- Fetch used Clash HTTP proxy (`http://127.0.0.1:7890`) after plain HTTPS DNS
  failure (`Could not resolve host: github.com`).
- `upstream/main` remained `a3eb7ef302961cba716dc78b39b93b60c467db0e`.
- Annotated tag `v0.2.8` object `d7a82d78ca51d42be41cb4daa3510ea401defe9f`
  peels to `fd80b08c90b55edcad5b00171b53f08721d30da1`.
- Backup branch:
  `backup/dev-before-v0.2.8-hotfix1-d9b39597a1-20260924T030529Z`
- Recovery dir (0700):
  `/Users/bingooyong/recovery/sub2api-v028-hotfix1-20260924T030529Z`
- Full overlap after fetch (9 paths): `.gitignore`,
  `backend/internal/config/config.go`,
  `backend/internal/service/openai_codex_fingerprint.go`,
  `backend/internal/service/openai_codex_transform.go`,
  `backend/internal/service/openai_gateway_chat_completions.go`,
  `backend/internal/service/openai_gateway_forward.go`,
  `backend/internal/service/openai_gateway_messages.go`,
  `backend/internal/service/openai_gateway_passthrough.go`,
  `deploy/config.example.yaml`
- Merge commit: `9aa87a187f4129e3eccc3374b38c9c58dc30fcd5`
  - parent1: `d9b39597a12065daecf97d33a11d97a72ec103b9`
  - parent2: `a3eb7ef302961cba716dc78b39b93b60c467db0e`
- Only conflict: `.gitignore`. Kept `docs/handoffs/` exceptions and audits
  exceptions; accepted upstream removal of `!docs/ANTIGRAVITY_ATTRIBUTION_429.md`.

### Tests

- Pre-merge and post-merge focused Codex identity / call-ID / WS fingerprint
  service tests: pass
- `./internal/pkg/openai` identity patterns: pass
- `go vet -tags=unit ./internal/service`: pass
- `git diff --check`: clean

### Build and archive

- Staging export:
  `/Users/bingooyong/recovery/sub2api-build-stage-20260924T031028Z`
  (git archive of merge commit; untracked `.agent`/`.agents`/`.cursor`/task
  dirs excluded)
- pnpm `9.15.9`, `pnpm install --frozen-lockfile`, frontend embedded into
  `backend/internal/web/dist`
- Archive:
  `release/sub2api-v0.2.8-hotfix1-linux-amd64.tar.gz`
- SHA-256:
  `0b8162faa3ed9ea7dc1a307ff835dabc9f51ec7f8e4679bd1771b417a5f456c5`
- Binary: ELF x86-64 statically linked; embedded asset
  `AccountsView-BOJ8MTG1.js` present
- Docker (`alpine:3.20`, `--platform linux/amd64`) `-version`:
  `Sub2API 0.2.8-hotfix1 (commit: 9aa87a187f4129e3eccc3374b38c9c58dc30fcd5, built: 2026-09-24T03:20:52Z)`
- Source tree: no new tracked modifications after build; untracked workspace
  dirs left intact; nothing pushed (`dev` ahead of `origin/dev` by 208).
