# Bug Analysis: error-channel diagnostics and incomplete lint reports

## 1. Root Cause Category

- **Category E — Implicit assumption**: explicitly discarding an error-channel
  receive satisfies errcheck but staticcheck rejects it as S1005.
- **Category C — Change propagation failure**: duplicate report limits showed
  only three equivalent diagnostics, hiding eleven later locations.

## 2. Why Fixes Failed

The initial five-item report was addressed mechanically. Three blank receives
then failed staticcheck, while the next capped report exposed three more bare
receives. An unchanged-source cached rerun with `--max-same-issues 0` produced
the full fourteen-item report. This distinguishes report truncation from
nondeterministic findings; no production behavior change is implicated.

## 3. Prevention Mechanisms

| Priority | Mechanism | Action | Status |
| --- | --- | --- | --- |
| P0 | Complete evidence | Disable duplicate/per-linter report limits | Applied |
| P0 | Compatible idiom | Bind join-only errors and include test diagnostics | Applied |
| P0 | Behavior protection | Retain frame/identity/close/timeout assertions | Reviewed |
| P1 | Documentation | Record interaction in backend quality guidelines | Done |

## 4. Systematic Expansion

Fourteen locations across six test files were changed, with no new helper or
suppression. Actual operation errors still need assertions when they define
success. These shutdown channels only join goroutines after independent
protocol assertions.

## 5. Knowledge Capture

- [x] Updated backend quality guidelines and index.
- [x] Included complete-report flags in final verification.
- [x] Recorded scheduled SQL/schema and clock-fixture findings in domain spec.
- [x] Confirmed this consumer repository has no `src/templates/markdown/spec`
  source tree; no template mirror applies here.
- Work commits include these records; final evidence is in `validation.md`.
