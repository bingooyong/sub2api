# Account quality operations for degraded upstream accounts

## Goal

Give administrators an opt-in, scheduled way to detect accounts whose upstream
responses have degraded, quarantine only accounts with an explicit quality
failure, and restore prior scheduling/group state only when a complete later
round proves the account healthy. This addresses observed account-level quality
degradation while preserving the repository's existing account routing,
Codex identity/fingerprint, WebSocket, and scheduled connectivity behavior.

## Background and confirmed facts

- The current scheduled-test path checks connectivity and has an existing
  `auto_recover` behavior for runtime error/rate-limit state. Quality judging is
  a separate concern and must not reuse that recovery state machine.
- The upstream reference branch `ranxi/feat-account-quality-ops` contains
  commits `18b3e94c`, `b034d423`, and `240a7f678`. Its useful behavior is a
  quality probe, an independent judge account/model, strict JSON verdicts,
  bounded retries, account isolation, snapshot-based restore, leases, and
  cursor-paginated history.
- That branch is approximately 930 files and 75k lines different from this
  branch, so wholesale merging would overwrite unrelated local behavior.
- The reference implementation does not establish a verified private `292/312`
  state-injection protocol or prove that the provider routes an account to a
  particular model family. This feature therefore observes quality outcomes
  and quarantines degraded accounts; it does not claim to control private
  provider routing.
- Existing local Codex fingerprint/identity, WebSocket, account-isolation, and
  account plan-tier changes are compatibility constraints and must remain intact.

## Requirements

### R1. Quality plan configuration

Extend the existing scheduled-test administration surface with an opt-in
quality plan. A plan can configure:

- target account scope;
- probe question and expected answer/criteria;
- target model;
- independent judge account group and judge model;
- cron schedule and bounded parallel probe count;
- bounded judge retry count;
- isolation action: remove selected group memberships or disable scheduling;
- automatic restore flag, defaulting to off.

Existing connectivity-only plans continue to work unchanged.

### R2. Probe and judge flow

For each due quality plan, acquire a per-account lease so concurrent workers do
not test the same account. Probe the target account using the existing account
forwarding path and preserve current request metadata/fingerprint behavior.

Select judge accounts that are schedulable, compatible with the configured
judge model, and different from the tested account. Run at most the configured
retry limit. Accept only strict JSON verdicts with one of `correct`, `incorrect`,
or `unknown`, plus bounded diagnostic fields. Treat malformed output as
`unknown` after retries.

### R3. Safe state mutation

- Only an explicit `incorrect` verdict may trigger isolation.
- Connectivity failures, timeouts, authentication errors, unavailable judge
  accounts, malformed/ambiguous verdicts, and `unknown` must record a result
  without mutating account scheduling or group membership.
- Before isolation, save a snapshot of the memberships and scheduling fields
  changed by this operation. A later complete successful round may restore the
  snapshot only when the live state still matches the operation's own changes;
  unrelated administrator edits must win.
- Isolation and restoration must be auditable in operation history.

### R4. Administration API and UI

Add admin endpoints and UI controls for creating, editing, enabling, disabling,
and deleting quality plans, manually triggering a round, viewing the latest
result, and reading cursor-paginated history. Surface verdict, reason,
isolation/restoration action, judge account, retry count, and timestamps.

The UI must make the opt-in nature, destructive isolation action, restore
behavior, and `unknown`/non-mutating outcomes clear before saving a plan.

### R5. Persistence and compatibility

Add forward-only migrations for quality-plan configuration, per-run lease and
snapshot data, and operation history. Migrations must be safe on existing
deployments and preserve all existing scheduled-test rows. API payloads must
use stable enums and nullable fields rather than leaking database-specific
representations.

### R6. Tests and observability

Add focused backend tests for verdict parsing, judge selection/exclusion,
retry bounds, lease contention, isolation/restore conflict handling, and
non-mutating failure paths. Add handler/repository tests for pagination and
frontend tests for plan validation and result rendering. Emit structured logs
with plan, account, run, verdict, and mutation outcome identifiers while
excluding credentials and prompt secrets.

## Acceptance criteria

- [ ] An administrator can enable a quality plan without changing behavior of
      existing connectivity-only plans.
- [ ] A scheduled or manual quality round probes due accounts, excludes the
      tested account from judging itself, and never exceeds configured parallel
      or retry limits.
- [ ] Strict JSON verdicts are classified as `correct`, `incorrect`, or
      `unknown`; every non-explicit-`incorrect` failure leaves account state
      unchanged.
- [ ] An `incorrect` verdict applies the configured isolation action and writes
      an auditable snapshot and operation-history record.
- [ ] Automatic restore is opt-in, happens only after a complete successful
      round, and does not overwrite unrelated administrator edits.
- [ ] Admin API and UI support plan lifecycle, manual execution, latest result,
      and cursor-paginated history with stable error handling.
- [ ] Existing Codex identity/fingerprint, WebSocket, account-isolation,
      account plan-tier, and scheduled connectivity tests remain passing.
- [ ] Database migrations, backend tests, frontend tests, production frontend
      build, and relevant Go checks pass on `v0.2.10-hotfix1`.

## Out of scope

- Implementing or reverse-engineering private `292/312` state injection.
- Claiming or enforcing provider-side routing to Luna, Astra, or another model
  family.
- Cookie acquisition, IP/node rotation, BPS, Mihomo, Codex harvesting, or a
  wholesale scheduler rewrite.
- Changing normal account routing policy outside the explicit quality-plan
  isolation action.
- Mutating account state on ambiguous, transport, timeout, authentication, or
  unavailable-judge failures.

## Open questions

None blocking planning. The implementation should follow existing scheduled
test conventions for authentication, cron parsing, API authorization, and UI
localization unless a focused test demonstrates a missing contract.
