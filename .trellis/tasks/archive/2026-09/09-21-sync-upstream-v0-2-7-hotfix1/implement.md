# Implementation plan

1. Record the clean tracked state and retry official upstream fetch/reference
   verification.
2. Create a uniquely named local backup branch and a private recovery directory
   containing a Git bundle, local commit patch, critical-file checksum inventory,
   and metadata.
3. Run focused pre-merge Codex identity, call-ID, and WebSocket fingerprint tests.
4. Merge the recorded `upstream/main` commit into `dev` with an explicit merge
   commit. Inspect parents, changed paths, conflicts, and the full first-parent
   diff.
5. Compare critical-file checksums and run the same focused regressions plus
   `git diff --check` and targeted vet/static checks.
6. Export the merged commit into a private build staging directory. Confirm the
   export excludes untracked workspace state and inspect the frozen build inputs.
7. Use Corepack/pnpm 9 with `pnpm install --frozen-lockfile`, run relevant frontend
   validation, and build the frontend into the backend embed directory.
8. Cross-build `backend/cmd/server` for Linux amd64 with embedded assets and linker
   metadata for `0.2.7--hotfix1`.
9. Create the versioned runtime archive without overwriting an existing file.
10. Verify archive contents, SHA-256, ELF architecture, version output, embedded UI,
    source revision, and final repository status. Record any unavailable runtime
    smoke check precisely.

## Completion evidence - 2026-09-22

All implementation and acceptance steps are complete. The final official
`upstream/main` is `1c0a69c0ceddb2fd21581c17ab09f6c500b89ba1`. The final local
source is merge commit `d329d452b3af056894886ea48d548c07e51bf20b`, with parents
`00e5b2bdeabecd20e0754a4aaf8f3c69968f1488` and the recorded upstream commit.
The four local fingerprint commits remain ancestors of the final source, and the
35-file critical checksum inventory passed after the final merge.

Focused Codex identity, call-ID, WebSocket convergence, and OpenAI package tests
passed. Migration tests, service vet, reasoning-effort billing tests, frontend
i18n tests, Vue typecheck, and the production frontend build passed. The complete
service suite has two DNS-limited channel-monitor cases; both were reproduced on
the pre-merge source and return `CHANNEL_MONITOR_ENDPOINT_UNREACHABLE`, so no
production change was made for them.

The final artifact is
`release/sub2api-v0.2.7--hotfix1-linux-amd64.tar.gz`, SHA-256
`f35cbe2578efc65840853994687f541a61318a5ea6d78df93b05d43aeb7c3ad6`.
Its manifest contains exactly five regular files under one versioned root, five
`0755` directories, no links or special entries, and the expected file modes.
The extracted binary is byte-identical to the build and smoke inputs (SHA-256
`e23ff3f7a22e9af21a6137eb645db7ca8ca3ae2d935be08108b49d3c8b2eff76`).
A Linux amd64 scratch-container run reports version `0.2.7--hotfix1`, commit
`d329d452b3af056894886ea48d548c07e51bf20b`, and build time
`2026-09-22T01:46:28Z`; the embedded frontend asset was also found.

Spec review found no new forwarding contract: the existing
`.trellis/spec/backend/codex-forwarding.md` remains current. The archive directory
mode normalization is recorded here as task-specific release evidence rather than
as a backend behavior rule. User-owned untracked workspace directories remain
untouched, and no push, tag, publication, or deployment was performed.

## Validation commands

- `git fetch upstream --prune --tags`
- `go test -tags=unit -count=1 ./internal/service -run '<focused patterns>'`
- `go test -count=1 ./internal/pkg/openai -run '<identity patterns>'`
- `go vet -tags=unit ./internal/service`
- `pnpm install --frozen-lockfile && pnpm run build`
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embed -trimpath ...`
- `tar -tzf <archive>`, `shasum -a 256 <archive>`, `file <binary>`
- Linux/container execution of `sub2api -version` when a compatible runtime is
  available.

## Rollback points

- Before merge: backup branch and Git bundle at the original `dev` revision.
- After merge: abort on conflicts or failed preservation checks; do not build.
- During build: discard only the private staging directory; leave source and
  user-owned untracked workspace state untouched.
