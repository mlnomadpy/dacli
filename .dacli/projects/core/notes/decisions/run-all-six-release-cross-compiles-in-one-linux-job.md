---
id: d-run-all-six-release-cross-compiles-in-one-linux-job
kind: note
note_kind: decision
created: 2026-08-28T10:31:52Z
created_by: a-fixer-7cpqs2
about: "[[t-01M13YH646HH1BWH0NHCM3DQP6]]"
---
# Run all six release cross-compiles in one Linux job
## Chose
Run all six release cross-compiles in one Linux job
## Rejected
Keep the six-leg GitHub Actions cross-compile matrix
## Because
A sequential target loop preserves each GOOS/GOARCH build while sharing the checkout, Go setup, and dashboard artifact instead of incurring six separately rounded runner jobs.
