---
id: f-223-complete-on-branch-dacli-223-add-releases-tags-and-generated-notes-to-the
kind: note
note_kind: finding
created: 2026-08-04T12:13:22Z
created_by: a-maintainer-prf0dg
about: "[[223]]"
severity: moderate
---
# 223 complete on branch dacli/223-add-releases-tags-and-generated-notes-to-the-github-surface
Commit 8161e20 by a-maintainer-prf0dg. Adds the release half of the GitHub surface so ship can cut a tagged release with notes. (1) New 'dacli github release <project> <tag>' in the ghmirror slice (internal/features/ghmirror/release.go): cuts a tagged GitHub release with gh --generate-notes by default on the project's LINKED repo (--repo, dacli 221); --notes overrides (and then --generate-notes is not also passed); --title/--target/--draft/--prerelease pass through. rw-gated (release.go:47); idempotent — an existing release (release view succeeds) is reported not duplicated (release.go:60); usage errors for unlinked project / missing tag. NOT run through disclosureGate by design (generated notes are the repo's own PR/commit history, not workspace findings — see decision note). (2) ship gains --release <tag> (internal/features/ship/ship.go): after integrate+record+push it shells 'dacli github release <project> <tag> --target <into>'. Preconditions validated up front (before accept, never half-ship): requires --push (release tags the REMOTE state), requires --project (resolve the repo), refuses --pr (async merges could tag before the wave lands). dry-run shows step 5. Tests: ghmirror/release_test.go (6 cases), ship_test.go (TestShipCutsReleaseAfterPush real push to a bare remote + 4 precondition/dry-run cases). Proof: go build ./... clean; go test ./... green with DACLI_AGENT stripped (the one catalog failure is the pre-existing env-leak, unrelated); go vet clean; gofmt -l internal/ empty. PR-first off: owner accepts + integrates the branch.
