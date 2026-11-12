# Backend Quality Guidelines

## 1. Scope / Trigger

Read when changing backend tests or resolving configured lint findings,
especially after Go or golangci-lint upgrades.

## 2. Signatures

Run from `backend`:

```bash
go test -tags=unit ./...
go vet ./...
golangci-lint run --max-same-issues 0 --max-issues-per-linter 0 ./...
```

`.golangci.yml` enables errcheck, staticcheck, govet, gosec, depguard,
ineffassign and unused. Bound Go memory and compilation concurrency when
the host is under pressure; run expensive checks sequentially.

## 3. Contracts

- Read full lint output. Default duplicate limits can hide additional
  locations; disable those limits before planning a fix.
- Validate an idiom against all configured rules. In golangci-lint 2.15,
  bare receives from error channels trigger errcheck; replacing them with
  `case _ = <-errors` triggers staticcheck S1005.
- For a shutdown channel only joined after protocol assertions, bind its
  result and include it in test diagnostics. Preserve close-code, frame,
  timeout and account-state assertions. A deliberate connection close may
  return an error; do not add an unrelated nil-error assertion to satisfy lint.
- In a branch that must always fail, include the received error in `Fatalf`.
  For an operation whose documented error is always nil, such as
  `strings.Builder.WriteRune`, explicit result discards are appropriate.
- Fix the contract rather than disabling a check or weakening assertions.
  Keep production behavior unchanged for lint-only edits.

## 4. Validation & Error Matrix

| Finding | Action |
| --- | --- |
| Only three identical diagnostics appear | Disable duplicate limits before editing |
| errcheck and staticcheck reject alternate receive forms | Bind the error and use meaningful test diagnostics |
| An operation error can invalidate the test | Assert the expected result, rather than only logging it |
| Integration fixture names a nonexistent column | Compare published migrations and Ent schema |
| Compilation fails before tests run | Record infrastructure failure separately from assertions |

## 5. Good / Base / Bad Cases

- Good: protocol assertions stay strict; shutdown results are visible in
  verbose or failed-test output.
- Base: affected regressions pass, followed by the full configured lint set.
- Bad: fixing only the first three diagnostics or adding blanket suppressions.

## 6. Tests Required

Run affected tests after diagnostic or fixture edits. Fresh-database
integration proves SQL/schema compatibility. Full unit, vet, lint and
embedded build checks apply to release integration; record which checks
ran before versus after final changes.

## 7. Wrong vs Correct

Wrong: `case _ = <-serverErr:` moves the finding to staticcheck.

Correct for a join-only shutdown result after protocol assertions:

```go
case shutdownErr := <-serverErr:
    t.Logf("passthrough shutdown result: %v", shutdownErr)
```
