# Upgrade to v0.2.17-hotfix1

## Goal

Merge upstream `v0.2.17` into the customized `v0.2.14-hotfix1` fork and produce a verified `v0.2.17-hotfix1` branch and build while preserving local Codex identity/fingerprint, account filters, Pelican scheduled tests, account isolation, and quality operations.

## Requirements

- Merge pinned upstream commit `060d0160417782cd962a0c7f460f6cd4565d4916` with ancestry preserved.
- Set the server version and built artifact to `0.2.17-hotfix1`.
- Include upstream encrypted-reasoning recovery commit `a4ea8059b`, matching the production error on server 160.
- Preserve local Codex identity/fingerprint/session/WebSocket handling, account type and plan-tier filtering, Pelican scheduled tests, account isolation, and quality operations.
- Retain upstream platform catalog, Command Code/Cline, Gemini cache/Argon, payment/security fixes, model retirement rules, and Responses compatibility changes.
- Resolve all 7 textual conflicts and review all 30 paths changed on both sides.
- Preserve published migration filenames; duplicate numeric prefixes are valid because migration identity is the full filename.
- Exclude `.omc/` and unrelated Trellis tasks.
- Do not deploy to production in this task.

## Acceptance Criteria

- [x] The merge's second parent is `060d0160417782cd962a0c7f460f6cd4565d4916`.
- [x] The built binary reports `0.2.17-hotfix1`.
- [x] The exact `thinking_signature_invalid` encrypted-content error retries after stripping stale encrypted reasoning and records lineage.
- [x] Local Codex identity/fingerprint/session/tool-call/WebSocket regressions pass.
- [x] Local account filters, page size, plan-tier, Pelican, isolation, and quality operations remain present and pass targeted tests.
- [x] Upstream platform catalog, payment fixes, Gemini features, model retirement, and protocol routing pass relevant backend tests.
- [x] Migration package and empty-database paths pass; representative live upgrade remains environment-dependent.
- [x] Backend tests, generation, and build pass.
- [ ] Frontend frozen install, lint, typecheck, full tests, and production build pass.
- [x] A resolution matrix documents all shared paths, conflicts, version, migrations, and configuration decisions.
- [x] Final commits exclude `.omc/` and unrelated work.

## Notes

- `v0.2.17` retires GPT-5.5 for ChatGPT OAuth/SetupToken subscription accounts; API-key accounts remain supported.
- Production rollout is separate from this merge/build task.
- Frontend frozen install/lint/typecheck/full tests/build and live PostgreSQL upgrade remain pending because Corepack could not download the pinned pnpm binary in this environment.
