# Implementation plan

1. Record existing behavior and a clean baseline for the affected Go packages.
2. Complete the read-only header/body inventory and local CLI request observation.
3. Add failing opt-out header and socket-reuse regressions; repair the final
   HTTP/WS forwarding boundaries while preserving the shared legacy enforcer for
   synthetic callers. Cover Messages post-build restoration and both alpha-search
   builders as well as Responses, compact, and passthrough.
4. Add failing distinct-call-ID, output-pair, boundary, and cross-turn reference
   regressions; apply the existing preservation option to native Responses.
   Update the existing WS native-item test so item validation and opaque call-ID
   preservation are asserted independently.
5. Review all hardcoded request metadata findings, preserving hash namespaces and
   account isolation. Document observed facts and unresolved platform differences.
6. Run focused regression tests, package tests/typecheck, lint/static checks, and
   inspect the integrated diff. Fix only findings within these boundaries.
7. Report changed behavior, the captured CLI mode/platform, tests, and remaining
   upstream/transport limitations. Do not claim to prevent account bans.

Rollback consists of reverting this task's isolated source/test/documentation diff;
no persisted account identity or deployment setting is migrated.

## Completion evidence — 2026-09-12

All seven steps are complete. The independent final review found no blocking source
issues. The full backend unit suite passed (56 package checks), the full integration
suite passed (50 package checks, including PostgreSQL/Redis), golangci-lint v2.13.0
reported zero issues, and service `go vet`, formatting, and diff checks passed.
Validation logs are under `/tmp/sub2api-codex-validation/`; durable commands and
results are recorded in `docs/audits/codex-request-compatibility.md`.

The sanitized official CLI 0.153.4 exec sample is saved beside the audit. Exact-tag
source research confirmed `os_info 3.14.0` serializes Ubuntu `22.04` as `22.4.0`;
no Ubuntu runtime or production/TLS equivalence claim is made. Code remains on
`dev` as uncommitted reviewable changes. Unrelated editor/Trellis setup files and
`.gitattributes` are retained.
