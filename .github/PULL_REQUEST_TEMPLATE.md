<!--
Delete any section that genuinely does not apply, and say why in one line.
An empty heading is worse than a removed one.
-->

## What was wrong

<!--
The DEFECT, not the change. What did the tool do, and what should it have done?
If you can, give the observable symptom a user would report — most of this
repo's worst bugs presented as "it said it worked".
-->

## Why it happened

<!--
The cause, not the fix. If two individually-correct things composed badly, say
both halves — that pattern accounts for most of the expensive bugs here.
-->

## How it was verified

<!--
Required. "Tests pass" is not verification: this codebase has repeatedly shipped
tests that passed while measuring nothing.

State the MUTATION: break the code this PR fixes, and confirm the new test goes
red. Paste the failure line.

    $ # revert `if requireVerify {` to `if false {`
    $ go test ./internal/features/acceptance/ -run Unlanded
    --- FAIL: TestAcceptOneRefusesUnlandedUnderRequireVerify

If the change is not testable, say so and say why.
-->

## Checklist

- [ ] `go test ./...`, `gofmt -l .`, `go vet ./...` and `golangci-lint run` are clean
- [ ] The new test fails against the pre-fix code (mutation stated above)
- [ ] Exit codes follow the contract: `0` ok, `2` usage, `3` policy refusal, `4` not found, `1` other
- [ ] A new command declares `JSON` / `Mutates` / `Usage` on its `clikit.Command`
- [ ] No feature slice imports another feature slice (`internal/cli/arch_test.go` enforces this)
- [ ] Comments explain *why*, and name the defect that shaped the code

## Anything you are unsure about

<!--
Say it here rather than leaving it for review to discover. A stated uncertainty
is cheap; an unstated one costs a round trip.
-->
