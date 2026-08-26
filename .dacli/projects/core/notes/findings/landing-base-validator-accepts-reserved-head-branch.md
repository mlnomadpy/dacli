---
id: f-landing-base-validator-accepts-reserved-head-branch
kind: note
note_kind: finding
created: 2026-08-26T15:15:04Z
created_by: a-adversarial-reviewer-pdq7xr
about: "[[t-01M0F8DMCN93FCDE59FSEDTJB3]]"
severity: major
against: a-root
---
# Landing base validator accepts reserved HEAD branch
internal/model/landing.go:47-64 (authored by root role a-root) manually approximates Git ref validation but never rejects the reserved branch name HEAD. Trigger: dacli project show p --landing-base HEAD; ValidateLandingPolicy returns nil and SaveProject persists it, while git check-ref-format --branch HEAD exits 128 with 'HEAD is not a valid branch name'. Wrong outcome: project show reports success and later ship/integrate inherit a base Git refuses, violating the unsafe-base and later-policy acceptance criteria. Add HEAD to public-command byte-identity rejection coverage or use a Git-equivalent validator.
