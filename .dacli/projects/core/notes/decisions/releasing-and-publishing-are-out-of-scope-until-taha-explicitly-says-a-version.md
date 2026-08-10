---
id: d-releasing-and-publishing-are-out-of-scope-until-taha-explicitly-says-a-version
kind: note
note_kind: decision
created: 2026-08-10T18:07:06Z
created_by: a-root
---
# Releasing and publishing are OUT OF SCOPE until Taha explicitly says a version is solid enough to publish (2026-08-10). Task 155 (v0.1.0 + Homebrew tap) is deferred by owner decision, not blocked on a missing HOMEBREW_TAP_GITHUB_TOKEN — the token was never the real constraint. Recorded in the project's Out of scope section so every assembled brief carries it to every spawned agent, rather than living only in a task nobody reads. Verified that no autonomous path can publish: release.yml triggers only on a manually pushed v* tag, ship --release requires an explicit flag, and the loop never passes it.
## Chose
Releasing and publishing are OUT OF SCOPE until Taha explicitly says a version is solid enough to publish (2026-08-10). Task 155 (v0.1.0 + Homebrew tap) is deferred by owner decision, not blocked on a missing HOMEBREW_TAP_GITHUB_TOKEN — the token was never the real constraint. Recorded in the project's Out of scope section so every assembled brief carries it to every spawned agent, rather than living only in a task nobody reads. Verified that no autonomous path can publish: release.yml triggers only on a manually pushed v* tag, ship --release requires an explicit flag, and the loop never passes it.
## Rejected
Leaving task 155 in the backlog as 'blocked on the PAT'. Rejected because it states the wrong cause: it reads as work waiting on a credential, so a review agent could reasonably file follow-ups to unblock it, and the release path would keep resurfacing. Also rejected: deleting task 155 outright, which would lose the record of what a release actually requires.
## Because

