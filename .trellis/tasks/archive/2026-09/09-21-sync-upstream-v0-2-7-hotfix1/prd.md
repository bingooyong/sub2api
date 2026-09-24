# Sync upstream and build v0.2.7--hotfix1

## Goal

Bring the local `dev` branch up to the latest verified official `upstream/main`
while preserving all local Codex identity and fingerprint behavior, then produce a
locally validated Linux amd64 release archive identified exactly as
`v0.2.7--hotfix1`.

## Background

- The pre-merge local revision is `3ed4afcc920e87b5db136bb27d93839eb89a4d87`.
- The final verified official revision is
  `1c0a69c0ceddb2fd21581c17ab09f6c500b89ba1`; the official `v0.2.7` tag peels to
  `aea725f2ea644d5592d0bbb1d63b607efa7e200a`.
- Upstream advanced after the first merge. The final source commit is
  `d329d452b3af056894886ea48d548c07e51bf20b`, whose parents are the first merge
  `00e5b2bdeabecd20e0754a4aaf8f3c69968f1488` and final verified upstream
  `1c0a69c0ceddb2fd21581c17ab09f6c500b89ba1`.
- Relative to the recorded merge base, the original local branch had 12 commits
  and final `upstream/main` has 102 commits. The final upstream delta touches 254
  files, including
  backend, frontend, release tooling, dependencies, and `VERSION` (`0.2.7`).
- Local commits contain deliberate Codex identity, request fingerprint, WebSocket
  per-frame convergence, opaque tool-call ID, and unbranded marker behavior.
- Existing untracked `.agent/`, `.agents/`, `.cursor/`, `.trellis/`, and `.zcode/`
  content is user-owned workspace state.

## Requirements

- Re-fetch and verify the official `upstream` remote before merging.
- Create local recovery material before the merge: a backup branch, Git bundle,
  source diff, and SHA-256 inventory of fingerprint-critical files.
- Merge `upstream/main` into `dev` with an explicit merge commit and inspect the
  resulting diff.
- Preserve the contracts in `.trellis/spec/backend/codex-forwarding.md`, including
  account/API-key isolation, stable internal hash namespaces, caller identity
  opt-out, opaque tool-call IDs, unbranded markers, and per-frame WebSocket
  convergence.
- Run focused Codex/fingerprint regressions before and after the merge.
- Build from a clean frozen source snapshot using pnpm 9 with a frozen lockfile,
  embedded frontend assets, and Go `linux/amd64`, `CGO_ENABLED=0`, `-tags embed`,
  and `-trimpath`.
- Inject binary version `0.2.7--hotfix1`, source commit, UTC build date, and
  `BuildType=release` through Go linker flags.
- Package the binary, `backend/resources/`, `deploy/docker-entrypoint.sh`, and the
  current `Dockerfile.goreleaser` as
  `release/sub2api-v0.2.7--hotfix1-linux-amd64.tar.gz` without overwriting an
  existing artifact.
- Do not push, tag, publish, deploy, clean user-owned untracked files, or add those
  files to Git.

## Acceptance Criteria

- [x] The verified upstream commit is recorded and the merge commit has two
  expected parents.
- [x] Fingerprint-critical files match their pre-merge content unless an upstream
  conflict requires a reviewed integration change.
- [x] Focused Codex identity, call-ID, and WebSocket fingerprint tests pass after
  the merge.
- [x] Frontend installation uses pnpm major 9 and `--frozen-lockfile`; the build
  output is embedded in the backend snapshot.
- [x] The archive exists at the requested versioned path and its SHA-256 is
  recorded.
- [x] The archive contains only the expected runtime payload, and the binary is an
  ELF x86-64 executable reporting `0.2.7--hotfix1`.
- [x] The source working tree has no new tracked modifications after the build;
  pre-existing untracked workspace directories remain intact.

## Out of Scope

- Production deployment or remote host changes.
- Git push, release publication, image publication, or tag creation/movement.
- Re-designing the existing Codex fingerprint mechanism.

## Risks and Rollback

- Upstream may advance between checks; the final re-fetch confirmed
  `1c0a69c0ceddb2fd21581c17ab09f6c500b89ba1` immediately before final artifact
  verification, and that immutable commit is the reported upstream input.
- Any failed merge or regression stops the build until repaired. The backup branch
  and Git bundle preserve the exact pre-merge state.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
