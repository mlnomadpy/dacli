---
id: f-158-ci-yml-frontend-node-setup-deduped-cross-compile-reuses-test-job-s-build
kind: note
note_kind: finding
created: 2026-07-26T21:32:38Z
created_by: a-a6t1k0z52j
about: [[158]]
severity: moderate
---
# 158: ci.yml frontend node setup deduped, cross-compile reuses test job's build via artifact
Commit a75c611. .github/workflows/ci.yml: cross-compile job no longer has its own actions/setup-node + npm ci + npm run build (previously repeated per matrix leg, 6x for 3 goos x 2 goarch). It now declares needs: test and downloads the dashboard-ui-dist artifact the test job uploads right after its own npm ci/build+test:unit+lint+go build (ci.yml:45-49), placing it at internal/features/dashboard/ui/dist before go build ./cmd/dacli runs (ci.yml:65-68), same path go:embed all:ui/dist reads. Net: exactly one node setup + one npm ci/build across ci.yml, test:unit and lint remain as steps in the test job (already wired by 154, ci.yml:29-34). Verified: go build ./..., go vet ./..., gofmt -l . all clean locally (no Go source touched, only ci.yml). Did NOT verify the actual GitHub Actions run (headless sandbox has no way to trigger/observe a live workflow run before push) -- correctness of the artifact upload/download wiring rests on the standard actions/upload-artifact@v4 + actions/download-artifact@v4 name-matching contract; owner should confirm via the PR's own CI run. release.yml was left untouched -- it's a separate workflow triggered only on version tags, out of scope for task 158 which named only ci.yml.
