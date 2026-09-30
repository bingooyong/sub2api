# Reference research: ranxi account quality operations

Source branch: `ranxi/feat-account-quality-ops`

Relevant commits:

- `18b3e94c`: initial scheduled account quality operations.
- `b034d423`: configurable judge model/group and operation rounds.
- `240a7f678`: regression test fix.

The branch introduces scheduled account probes, independent judge selection,
strict JSON verdicts (`correct`, `incorrect`, `unknown`), bounded retries,
per-account leases, isolation actions, snapshot-based restore, and cursor
pagination for operation history. Judge accounts are schedulable,
model-compatible, and exclude the tested account. Only explicit `incorrect`
mutates account state; transport errors and `unknown` do not.

The branch is too divergent to cherry-pick wholesale (roughly 930 files and
75k lines difference from this repository). It also does not implement a
verified `292/312` `current_turn_state` injection or prove provider routing to
Luna. The local implementation should import the operational safeguards and
adapt them to the existing scheduled-test/account isolation boundaries.
