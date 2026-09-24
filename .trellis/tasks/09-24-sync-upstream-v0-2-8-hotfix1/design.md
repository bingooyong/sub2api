# Design: upstream v0.2.8 sync and local hotfix build

## Source-control boundary

Fetch the official `upstream` remote and merge
`a3eb7ef302961cba716dc78b39b93b60c467db0e` into `dev` with `--no-ff`. That
commit is official `main` at planning time: annotated tag `v0.2.8`
(`fd80b08c90b55edcad5b00171b53f08721d30da1`) plus
`chore: sync VERSION to 0.2.8`.

Before the merge, create a local backup branch at the current `dev` revision
and a Git bundle outside the repository. Do not reset, clean, rebase, tag,
push, or publish.

The expected merge base is
`1c0a69c0ceddb2fd21581c17ab09f6c500b89ba1`. Accept the merge only after the
first-parent diff is inspected. If `main` has moved, stop and update this
design before merging a different commit.

HTTPS fetch failed once with an empty server reply. Retry the same remote.
If it fails again, fetch the recorded commit through another authenticated
GitHub path and verify the SHA before merging. Do not merge an unverified tip.

## Fingerprint preservation boundary

The authoritative behavior contract is
`.trellis/spec/backend/codex-forwarding.md`. Capture checksums and a source
diff for the Codex identity and fingerprint files, and run focused regressions
before the merge. Afterward, compare the same files and repeat the regressions.

Upstream hunks that must be kept when they merge cleanly:

- `identity_service.go`: Claude CLI floor and default fingerprint follow
  `EffectiveCLIVersion`. This file has no local delta since the merge base.
- `openai_codex_account_identity.go` and the two metadata rebuilds in
  `openai_codex_fingerprint.go`: use `marshalCodexTurnMetadata`.
- `openai_codex_transform.go`: add `gpt-6-sol` and `gpt-6-luna` map entries.

Local hunks that must remain:

- Unbranded markers and `python__codex`. Do not restore `<sub2api-...>` markers.
- `PreserveToolCallIDs` / `PreserveCallIDs` and the cross-turn reference rule.
- `applyOpenAIWSFingerprintClientMetadata` and its ingress / v2 passthrough
  call sites.
- Caller identity opt-out, account and API-key isolation, and stable internal
  `sub2api:` hash domains.

`openai_codex_fingerprint.go` and `openai_codex_transform.go` change on both
sides in different regions, so a textual auto-merge is expected. Read the merged
functions anyway. A conflict is resolved by keeping both the upstream helper or
model entry and the local contract, never by taking one side wholesale.

The GitHub file list is capped at 300, so these additional overlaps are a lower
bound. Recompute the full name intersection after fetch. Already confirmed:

- `openai_gateway_forward.go`, `openai_gateway_passthrough.go`, and
  `openai_gateway_messages.go`: keep `preserveCodexClientIdentityHeaders` after
  `enforceCodexIdentityHeaders*`. Keep `PreserveToolCallIDs` on the native
  Responses transform. Upstream adds `applyOpenCodeUpstreamUserAgent` and
  `applyMappedGPT55LiteCompatibility` in the same builders. That UA helper
  returns immediately unless the target is an official Command Code host or the
  account/target is OpenCode. Leave that guard intact. Do not move preservation
  after it, and do not delete it to protect Codex opt-out: Command Code's
  canonical UA is a separate Cloudflare 1010 guard, not a sub2api marker.
- `openai_gateway_chat_completions.go`: keep `PreserveToolCallIDs` for the
  Responses-shaped path, and keep upstream's model assignment plus
  `normalizeGPT6ResponsesSampling`.
- `config.go`: keep the local comment on `DisableCodexIdentityEnforcement`.
  Take upstream `SimpleMode` fields; they sit in a different struct region.
- `deploy/config.example.yaml`: keep
  `disable_codex_identity_enforcement: false` and its opt-out comment. Fold in
  upstream simple-mode keys from other sections. The compare response omitted
  this patch, so inspect the merged file directly.
- `.gitignore`: same docs-exception hunk on both sides. Keep local
  `docs/handoffs/` and `docs/audits/` exceptions. Accept upstream removal of
  `!docs/ANTIGRAVITY_ATTRIBUTION_429.md`.

`identity_service_user_agent_validation_test.go` is upstream-only in this range.
After `defaultFingerprint` becomes a function, call sites in that test must use
`defaultFingerprint()`. Local did not edit the test.

## Build boundary

Export the merged commit to a private `0700` staging directory so user-owned
untracked workspace files cannot enter the build. Install frontend dependencies
with pnpm 9 and the frozen lockfile, build into `backend/internal/web/dist`,
and cross-compile from `backend/` with embedded assets.

Use `0.2.8-hotfix1` as the linker-injected binary version. A single hyphen
matches the requested label `v0.2.8-hotfix1`. The previous archive used a
double hyphen (`0.2.7--hotfix1`); this build does not repeat that spelling.
The Git source stays traceable through the merged commit. Do not create a tag.

## Package and verification

Assemble an archive rooted at `sub2api-v0.2.8-hotfix1-linux-amd64/`. Include
the binary, resources, entrypoint, and `Dockerfile.goreleaser`. Verify the tar
manifest, SHA-256, ELF architecture, embedded UI, and binary `-version` output
through a Linux runtime when available.

The artifact is local only. A build failure leaves the source merge and
recovery material available. It does not trigger deployment or publication.
