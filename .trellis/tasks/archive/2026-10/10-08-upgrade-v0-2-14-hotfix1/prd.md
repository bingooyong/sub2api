# Upgrade to v0.2.14-hotfix1

## Goal

Integrate upstream `v0.2.14` into the current `v0.2.10-hotfix1` fork and produce a verified `v0.2.14-hotfix1` branch without losing local Codex identity, fingerprint, WebSocket, account isolation, Pelican, or account quality operations behavior.

## Requirements

- Merge the pinned upstream `v0.2.14` commit `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d` with preserved ancestry.
- Set the server version to `0.2.14-hotfix1` and verify the built artifact reports that version.
- Preserve upstream payment, authentication, billing, TypeSafe, remote model catalog, and GPT-6.1 fixes.
- Preserve all intentional local hotfix behavior, especially Codex fingerprint and identity propagation, WebSocket forwarding, account isolation, Pelican scheduled tests, and account quality operations.
- Review every path changed by both sides since the common ancestor, even when Git resolves it automatically.
- Keep published migration filenames unchanged. Allow both upstream `241_*` migrations and local `243/247/248/249` migrations to coexist.
- Exclude `.omc/` and unrelated Trellis tasks from commits.
- Do not deploy to production as part of this task.

## Acceptance Criteria

- [x] Branch `v0.2.14-hotfix1` contains a merge whose second parent is exactly `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d`.
- [x] `backend/cmd/server/VERSION` and the built binary report `0.2.14-hotfix1`.
- [x] The 17 automatically merged shared paths plus `VERSION` have a documented resolution review.
- [x] Upstream security fixes remain effective, including EasyPay callback/replay protection and password-reset token hardening.
- [x] Upstream API-key limits, inflight reservation defaults, remote model catalog, GPT-6.1, TypeSafe, and deleted-key settlement behavior are covered by tests or explicit verification.
- [x] Local Codex fingerprint, identity, session isolation, tool-call correlation, WebSocket, account isolation, Pelican, and quality-operations regressions pass.
- [x] Empty-database migrations, upgrade from a representative `v0.2.10-hotfix1` database, and second-start idempotency pass.
- [x] Backend tests and build pass; generated files are clean after regeneration.
- [x] Frontend frozen install, lint check, typecheck, tests, and build pass.
- [x] The final diff contains no `.omc/` files or unrelated changes.

## Notes

- This upgrade is security-driven. Future upstream releases should be screened weekly, with immediate integration for security/payment/auth/billing/data-correctness/model-availability changes and batched integration for ordinary updates.
- Production canary, PITR, and deployment are separate operational work.
