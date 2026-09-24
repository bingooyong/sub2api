# Upstream v0.2.8 delta vs local dev

Recorded 2026-09-24 before merge. HTTPS `git fetch upstream` failed with
`Empty reply from server`. Commit identities below come from the GitHub API.

## Immutable refs

- Local `dev` HEAD: `d9b39597a` (`docs: fix hotfix1 release handoff details`).
- Previous verified upstream parent inside local history:
  `1c0a69c0ceddb2fd21581c17ab09f6c500b89ba1`.
- Official annotated tag `v0.2.8` object `d7a82d78ca51d42be41cb4daa3510ea401defe9f`
  peels to commit `fd80b08c90b55edcad5b00171b53f08721d30da1`.
- Official `main` at inspection time: `a3eb7ef302961cba716dc78b39b93b60c467db0e`,
  one commit after the tag: `chore: sync VERSION to 0.2.8 [skip ci]`
  (2026-09-23T09:18:46Z). Release published 2026-09-23T09:18:31Z.
- Compare `1c0a69c0c...v0.2.8`: 206 commits, about 300 files.
- Compare local fork `d9b39597a` with `Wei-Shaw/sub2api@a3eb7ef`: diverged,
  upstream 207 commits ahead, local 15 commits not in upstream.

## Merge base expectation

`1c0a69c0c` is an ancestor of local `dev` and of `a3eb7ef`, so the three-way
merge base is that commit. Local work since that base is the docs handoff plus
the fingerprint tree already preserved by `d329d452b`.

## Local behavior that must survive

`git diff --stat 1c0a69c0c HEAD` is 69 files, +2803/−53. The Codex contract
lives in `.trellis/spec/backend/codex-forwarding.md`. Behavior-bearing files
still different from previous upstream include:

- `backend/internal/service/openai_codex_identity.go`
- `backend/internal/service/openai_codex_fingerprint.go` (+15, WS per-frame helper)
- `backend/internal/service/openai_codex_transform.go` (unbranded markers, call-ID preservation)
- `backend/internal/service/openai_messages_todo_guard.go`
- `backend/internal/service/openai_ws_forwarder_ingress.go`
- `backend/internal/service/openai_ws_v2_passthrough_adapter.go`
- `backend/internal/service/openai_ws_pool.go` and related tests
- `deploy/config.example.yaml`, `backend/internal/config/config.go`

Local source commits that own this behavior:

- `06bb4df4c` preserve Codex client identity
- `175875895` remove gateway brand markers
- `25fc7d27e` align websocket fingerprint frames
- `f2b15cdfc` drop legacy branded markers

## Review correction

The GitHub compare payload lists exactly 300 files, which is the API cap, so
that list is a lower bound. `git fetch` of `a3eb7ef` failed twice (empty reply,
then port 443 timeout). The intersection below is only against those 300 names.
After a successful fetch, recompute `git diff --name-only 1c0a69c0c a3eb7ef`
before treating any other local file as untouched.

## Upstream overlap on those files

Confirmed intersection of the local-only tree with the capped upstream file list:

- `.gitignore`
- `backend/internal/config/config.go`
- `backend/internal/service/openai_codex_fingerprint.go`
- `backend/internal/service/openai_codex_transform.go`
- `backend/internal/service/openai_gateway_chat_completions.go`
- `backend/internal/service/openai_gateway_forward.go`
- `backend/internal/service/openai_gateway_messages.go`
- `backend/internal/service/openai_gateway_passthrough.go`
- `deploy/config.example.yaml` (compare marked it modified but omitted the patch)

From `1c0a69c0c...a3eb7ef` (see `upstream-fingerprint-overlap.patch`):

| File | Upstream change | Local delta since base | Expected merge |
| --- | --- | --- | --- |
| `identity_service.go` | Claude CLI floor uses `EffectiveCLIVersion`; `defaultFingerprint()` | none | take upstream |
| `openai_codex_account_identity.go` | `json.Marshal` → `marshalCodexTurnMetadata` | none | take upstream |
| `openai_codex_fingerprint.go` | same marshal helper at two metadata rebuilds | new WS helper near line 57 | different hunks; auto-merge expected, then re-read both |
| `openai_codex_transform.go` | add `gpt-6-sol` / `gpt-6-luna` map entries | unbranded markers and call-ID preservation | different hunks; keep local markers |

Upstream did not restore `<sub2api-...>` markers in the inspected patch.
`openai_codex_identity.go`, WS ingress, v2 passthrough, and todo-guard were
not in the upstream fingerprint-file set for this range.

## Version and packaging precedent

Previous local artifact: `release/sub2api-v0.2.7--hotfix1-linux-amd64.tar.gz`.
`backend/cmd/server/VERSION` is currently `0.2.7`. Upstream main sets it to
`0.2.8` in `a3eb7ef`. Binary version for this hotfix is injected with ldflags
and must read `0.2.8-hotfix1`, matching the requested label `v0.2.8-hotfix1`.
