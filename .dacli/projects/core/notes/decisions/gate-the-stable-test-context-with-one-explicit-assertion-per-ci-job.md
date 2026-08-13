---
id: d-gate-the-stable-test-context-with-one-explicit-assertion-per-ci-job
kind: note
note_kind: decision
created: 2026-08-13T19:21:26Z
created_by: a-codex-maintainer-xytv4d
about: "[[428]]"
github:
  issue: 602
  repo: mlnomadpy/dacli
---
# Gate the stable test context with one explicit assertion per CI job
## Chose
Gate the stable test context with one explicit assertion per CI job
## Rejected
Rely on needs alone or add a second aggregate context
## Because
if: always() lets the stable check run after failures and cancellations, while explicit result equality makes it fail closed and preserves the branch-protected test context without repository-setting churn.
