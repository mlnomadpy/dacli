---
id: t-01KZPWSJ4CGN3F923P4C3ZDRG2
kind: task
created: 2026-08-10T22:30:48Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Produce reproducible cross-platform release artifacts with checksums and an SBOM, verified by snapshot
## Acceptance
- [ ] goreleaser emits binaries for every supported platform, a checksums file, and an SBOM per artifact
- [ ] the whole path is verified with a SNAPSHOT build in CI, so it is proven without publishing anything
- [ ] the README install commands are exercised against a built artifact, which is the acceptance criterion issue #437 names
- [ ] no v* tag is pushed: publication stays the maintainer's explicit act, and release.yml still never creates a tag
## Log
