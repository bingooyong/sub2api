# Sync upstream v0.2.8 and build v0.2.8-hotfix1

## Goal

Bring local `dev` up to the latest official v0.2.8 line and produce a locally
validated Linux amd64 archive named `v0.2.8-hotfix1`, while the existing Codex
identity and unbranded fingerprint behavior stays in force.

## Background

- Local `dev` is `d9b39597a`. The last verified upstream parent in this history
  is `1c0a69c0ceddb2fd21581c17ab09f6c500b89ba1`. `backend/cmd/server/VERSION`
  is `0.2.7`.
- Official tag `v0.2.8` peels to `fd80b08c90b55edcad5b00171b53f08721d30da1`.
  Official `main` is one commit newer:
  `a3eb7ef302961cba716dc78b39b93b60c467db0e`
  (`chore: sync VERSION to 0.2.8`). That commit is the latest v0.2.8 code.
- Relative to `1c0a69c0c`, upstream is 207 commits ahead and changes about 300
  files. Local history still contains 15 commits that upstream does not have.
- Those local commits keep Codex client identity, unbranded compatibility
  markers, opaque tool-call IDs, and per-frame WebSocket fingerprint
  convergence. The contract is `.trellis/spec/backend/codex-forwarding.md`.
- Existing untracked `.agent/`, `.agents/`, `.cursor/`, and `.trellis/`
  workspace files are user-owned and are not part of the release.

## Requirements

- Fetch and record the official upstream commit before merging. The merge input
  is `a3eb7ef302961cba716dc78b39b93b60c467db0e`, which includes the `v0.2.8`
  tag plus the VERSION sync.
- Create local recovery material before the merge: a backup branch, a Git
  bundle, a source diff, and a SHA-256 inventory of fingerprint-critical files.
- Merge that upstream commit into `dev` with an explicit merge commit. Inspect
  the result, especially any hunk that touches Codex identity or markers.
- Preserve the contracts in `.trellis/spec/backend/codex-forwarding.md`:
  account and API-key isolation, stable internal `sub2api:` hash namespaces,
  caller identity opt-out, opaque tool-call IDs, unbranded markers
  (`<codex-image-generation-bridge>`, `<codex-spark-image-unsupported>`,
  `<claude-code-todo-guard>`, `python__codex`), and per-frame WebSocket
  convergence. Do not restore legacy `<sub2api-...>` markers. Keep the
  identity-restore calls on the HTTP builders, the opt-out config key and
  comment, and the local docs handoff/audit gitignore exceptions.
- Accept upstream changes that do not rewrite that contract, including GPT-6
  Sol/Luna model map entries, `marshalCodexTurnMetadata`, and Claude CLI
  effective-version fingerprint flooring.
- Run focused Codex identity, call-ID, and WebSocket fingerprint tests before
  and after the merge.
- Build from a clean frozen source snapshot with pnpm 9 and a frozen lockfile,
  embedded frontend assets, and Go `linux/amd64`, `CGO_ENABLED=0`, `-tags embed`,
  and `-trimpath`.
- Inject binary version `0.2.8-hotfix1`, the source commit, a UTC build date,
  and `BuildType=release` through Go linker flags.
- Package the binary, `backend/resources/`, `deploy/docker-entrypoint.sh`, and
  the current `Dockerfile.goreleaser` as
  `release/sub2api-v0.2.8-hotfix1-linux-amd64.tar.gz` without overwriting an
  existing artifact.
- Do not push, tag, publish, deploy, clean user-owned untracked files, or add
  those files to Git.

## Acceptance Criteria

- [ ] The recorded upstream commit is `a3eb7ef302961cba716dc78b39b93b60c467db0e`,
      and the merge commit has that commit and the pre-merge `dev` tip as parents.
- [ ] Fingerprint-critical files keep their pre-merge local behavior except where
      an upstream hunk was reviewed and does not restore branded markers or drop
      identity, call-ID, or per-frame WebSocket convergence.
- [ ] Focused Codex identity, call-ID, and WebSocket fingerprint tests pass after
      the merge.
- [ ] Frontend installation uses pnpm major 9 and `--frozen-lockfile`. The build
      output is embedded in the backend snapshot.
- [ ] The archive exists at `release/sub2api-v0.2.8-hotfix1-linux-amd64.tar.gz`
      and its SHA-256 is recorded.
- [ ] The archive contains only the expected runtime payload, and the binary is
      an ELF x86-64 executable that reports `0.2.8-hotfix1`.
- [ ] The source working tree has no new tracked modifications after the build.
      Pre-existing untracked workspace directories remain intact.

## Out of Scope

- Production deployment, remote host changes, Git push, release publication,
  image publication, and tag creation or movement.
- Redesigning the Codex fingerprint mechanism.
- Building darwin, arm64, or Windows archives.

## Risks and Rollback

- Upstream may move after this inspection. Re-fetch immediately before the merge
  and stop if `main` is no longer `a3eb7ef302961cba716dc78b39b93b60c467db0e`.
- A conflict in a fingerprint file stops the build until the resolution is
  checked against the forwarding contract. The backup branch and Git bundle
  keep the exact pre-merge state.
