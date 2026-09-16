# Document Codex fingerprint findings with source examples

## Goal

Produce a Chinese reference document explaining the gateway's request fingerprint
findings, with concrete filenames, current source locations, and example code.

## Requirements

- Reuse the completed compatibility audit and sanitized official CLI observation.
- Identify exact files, symbols, line numbers, activation conditions, and observable
  fields for each finding; show small source excerpts or clearly labeled examples.
- Distinguish confirmed defects, deliberate metadata/body changes, internal hash
  inputs, and transport/provider behavior that has not been verified.
- Distinguish pre-fix `4726bdd08` examples from fixes committed locally in
  `06bb4df4c` on dev; do not imply upstream merge or deployment.
- Include the Ubuntu version-formatting correction and other providers' fixed UAs
  without claiming their traffic should match Codex.
- Change documentation and its narrow gitignore allowlist only; preserve runtime
  code, existing workspace edits, isolation mechanisms, and credentials.

## Acceptance Criteria

- [x] The Chinese document includes a finding index and file-linked code examples.
- [x] Every source excerpt and local link is checked against its stated revision.
- [x] Previously fixed issues and remaining configured behaviors are distinct.
- [x] Source evidence and capture limitations support the document's conclusions.
- [x] Documentation/whitespace checks pass; no runtime code changes are introduced.

## Notes

- This is a lightweight documentation follow-up to
  `.trellis/tasks/archive/2026-09/09-11-codex-request-compatibility-audit`.
- Deliverable: `docs/audits/sub2api-fingerprint-findings.zh-CN.md`.
- Chinese is used for the requested reader-facing reference; the existing English
  audit remains the detailed validation record.

## Validation — 2026-09-12

- 19 core findings and 7 other-provider/utility UA examples documented.
- All 38 source blocks matched their declared ranges in `4726bdd08` or
  `06bb4df4c`; all 7 inline UA examples matched source.
- All 80 local links resolved, including 78 valid source line anchors.
- CLI headers and body-shape statements matched the saved sanitized observation.
- Independent semantic review completed; revision status, passthrough model gate,
  Messages template scope, and configured device ID precedence were corrected.
- `git diff --check`, document whitespace/fence checks, and the narrow gitignore
  exception passed. Staged and unstaged diffs for `backend/` and `deploy/` are empty.
- No backend tests were rerun for this documentation-only task. The earlier code
  validation remains recorded in the English audit.
- Spec review: existing `codex-forwarding.md` already records the relevant
  forwarding contracts and Ubuntu formatting caveat; no new runtime contract or
  convention was introduced, so no spec change was needed.
- Preserve the pre-existing local commit `06bb4df4c` and unrelated workspace files;
  this documentation follow-up is delivered without creating or pushing a commit.
