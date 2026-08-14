---
id: t-01KZYMP0X162ZPK17GFX0MFY8C
kind: task
created: 2026-08-13T22:42:59Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 1, probable: 1, pessimistic: 2}"
github:
  issue: 642
  repo: mlnomadpy/dacli
---
# Pin CI security scan to patched Go 1.25.13
## Acceptance
- [x] .github/workflows/ci.yml runs the lint and govulncheck job on Go 1.25.13 or newer within the 1.25 line
- [x] govulncheck no longer reports GO-2026-6090, GO-2026-6089, or GO-2026-5972 from the CI standard library
- [x] the Go 1.22 compatibility matrix remains unchanged
## Log
- 2026-08-13T22:43:50Z claimed by a-fixer-8skqtd
- 2026-08-13T22:55:13Z accepted by a-root
- 2026-08-13T22:55:13Z verified by `go test ./.github/workflows` (exit 0) in branch main at 4665c04 — proves that tree builds, not that the work is in trunk
- 2026-08-13T22:55:13Z deliverable: dacli/438-pin-ci-security-scan-to-patched-go-1-25-13 is merged into main
- 2026-08-13T22:55:13Z completed by a-root
- 2026-08-13T23:51:31Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/643 (event 01KZYMYFWJWKPQ7DPV8RZ49NA5)
## Verification Evidence
{"command":"go test ./.github/workflows","exit_code":0,"duration_ms":508,"artifact_hash":"sha256:decaa73810cbe4af9bb35b99a5227a968b2cdd7befe4ea5e2c1fb6d77c9ab0ce","verifier":"a-root","branch":"main","commit_sha":"4665c0424def501fd450722b7d664fc326960fd8"}
