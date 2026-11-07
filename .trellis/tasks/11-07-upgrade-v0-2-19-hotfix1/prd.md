# Upgrade to v0.2.19-hotfix1

## Goal

Audit upstream `v0.2.19`, merge the required fixes into the customized `v0.2.18-hotfix1` branch, and build a verified `v0.2.19-hotfix1` hotfix without losing local Codex/session, account quality, or observability behavior.

## Requirements

- Pin upstream tag object `e60cefeaf78f2fbfa95aa076029e28b40d0a5527` and commit `dece26153621959ccd7635814f4b2d08304e681a`.
- Preserve local encrypted-content recovery, sticky session behavior, session-hash diagnostics, Codex fingerprint/identity/WebSocket, account filters, Pelican, quality operations, and migration history.
- Include upstream security/robustness fixes: pagination overflow rejection, multipart image size rejection, admin guard 400 behavior, dependency security update, and web-search dimension preservation.
- Retain new scoped admin-key and opt-in per-image billing features as part of the complete stable-release merge. Include billing correctness fixes: Gemini fileData image accounting, DeepSeek cache-hit accounting/pricing, and payment/model pricing corrections.
- Include protocol/model compatibility fixes: GPT-6.1 Terra, OpenCode DeepSeek developer-role handling, malformed tool-call rejection, Gemini model-list/cache retry fixes, and Grok catalog consistency.
- Audit all 108 commits after the tag through pinned upstream/main `d22e0724d6a28435bcf21280333e89883d069a5e`; backport bounded security fixes and document deferred protocol changes.
- Treat optional UI-only assets and diagnostics as lower priority, but retain them when the full merge does not conflict with local contracts.
- Resolve the two text conflicts and review all 23 shared paths semantically.
- Preserve every published migration filename; upstream `247/248` must coexist with local migrations by full filename.
- Do not commit `.omc/`, unrelated Trellis tasks, or production deployment changes.

## Acceptance Criteria

- [ ] Merge parent 2 is pinned upstream commit `dece26153621959ccd7635814f4b2d08304e681a` and branch is `v0.2.19-hotfix1`.
- [ ] `backend/cmd/server/VERSION` and built binary report `0.2.19-hotfix1`.
- [ ] Scoped admin-key authorization, hashing, expiration/IP checks, and audit/revocation behavior pass targeted tests.
- [ ] Pagination overflow, multipart upload limits, admin guard errors, and dependency checks pass.
- [ ] Per-image/Gemini/DeepSeek billing and payment pricing regressions pass.
- [ ] Server-160 encrypted-content recovery, local sticky session/session-hash diagnostics, Codex identity/fingerprint/WebSocket, account filters, Pelican, and quality operations pass.
- [ ] Empty and representative upgrade migrations apply upstream `247/248` alongside local migration history and are idempotent.
- [ ] Backend generation, tests, build, and version smoke pass.
- [ ] Frontend frozen install, lint, typecheck, tests, and production build pass where the local pnpm toolchain is available.
- [ ] Resolution matrix documents mandatory/optional decisions, 2 conflicts, and 23 shared paths.
- [ ] No `.omc/` or unrelated task files are committed.

## Notes

- This is a security and billing correctness release; ordinary UI additions are not release blockers.
- Production rollout and database backup/PITR are separate from this local branch/build task.
