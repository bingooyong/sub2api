# Design: upstream sync and local hotfix build

## Source-control boundary

Fetch the official `upstream` remote, record its immutable commit, and merge that
commit into `dev` with `--no-ff`. Before the merge, create a local backup branch at
the current `dev` revision and a Git bundle outside the repository. Never use
reset, clean, rebase, tag, push, or publication commands.

The merge is accepted only after its first-parent diff is inspected. The verified
upstream delta contains 99 commits and 200 changed files, so run a merge-tree
simulation and enumerate overlap with the local fingerprint inventory before
modifying `dev`. Review every conflict against both the upstream intent and the
local Codex forwarding contract.

## Fingerprint preservation boundary

Capture checksums and a source diff for the Codex identity/fingerprint files and
run focused regressions before the merge. Afterward, compare the same files and
repeat the regressions. This proves that the upstream sync retained local behavior
instead of relying on the absence of textual conflicts.

The authoritative behavior contract is
`.trellis/spec/backend/codex-forwarding.md`. Internal `sub2api:` hash domains remain
unchanged because they are stable derivation inputs, while upstream-visible
compatibility markers remain unbranded.

## Build boundary

Export the merged commit to a private `0700` staging directory so user-owned
untracked workspace files cannot enter the build. Install frontend dependencies
with pnpm 9 and the frozen lockfile, build into
`backend/internal/web/dist`, and cross-compile from `backend/` with the project Go
toolchain requirements and embedded assets.

Use `0.2.7--hotfix1` as the linker-injected binary version. The Git source remains
traceable through the merged commit rather than through a newly created tag.

## Package and verification

Assemble a deterministic-shape archive rooted at
`sub2api-v0.2.7--hotfix1-linux-amd64/`. Include the binary, resources,
entrypoint, and `Dockerfile.goreleaser`. Verify the tar manifest, SHA-256, ELF
architecture, embedded UI, and binary `-version` output through a Linux runtime
when available.

The artifact is local only. A build failure leaves the source merge and recovery
material available for diagnosis; it does not trigger deployment or publication.
