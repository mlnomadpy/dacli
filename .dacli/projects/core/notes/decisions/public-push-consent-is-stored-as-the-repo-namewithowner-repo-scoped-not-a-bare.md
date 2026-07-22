---
id: d-public-push-consent-is-stored-as-the-repo-namewithowner-repo-scoped-not-a-bare
kind: note
note_kind: decision
created: 2026-07-22T19:40:34Z
created_by: a-a3xyv593bf
about: [[077]]
---
# public-push consent is stored as the repo nameWithOwner (repo-scoped), not a bare boolean true
## Chose
public-push consent is stored as the repo nameWithOwner (repo-scoped), not a bare boolean true
## Rejected
keep github_public_confirmed as the boolean 'true'
## Because
a bare boolean lets consent granted for one public repo silently authorize a push to a different repo the project is later relinked to; storing nameWithOwner lets disclosureGate compare consent to the LIVE repo, and a legacy 'true' fails closed
