---
id: d-158-share-one-frontend-build-via-upload-download-artifact-between-test-and
kind: note
note_kind: decision
created: 2026-07-26T21:32:21Z
created_by: a-a6t1k0z52j
about: [[158]]
---
# 158: share one frontend build via upload/download-artifact between test and cross-compile jobs
## Chose
158: share one frontend build via upload/download-artifact between test and cross-compile jobs
## Rejected
keeping cross-compile's own setup-node+npm ci+npm run build (6x duplicate, once per matrix leg), or inlining frontend build into a single mega-job with the goreleaser matrix
## Because
test job (154) already builds+tests+lints the SPA once; cross-compile only needs the resulting internal/features/dashboard/ui/dist artifact for go:embed, not a fresh npm install per matrix leg -- actions/upload-artifact+download-artifact lets cross-compile depend on (needs: test) and reuse that single build, cutting 6 redundant npm ci/build invocations down to 1 without touching release.yml (separate workflow, out of scope for 158)
