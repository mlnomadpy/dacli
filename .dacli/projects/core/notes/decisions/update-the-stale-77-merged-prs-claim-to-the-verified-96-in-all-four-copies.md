---
id: d-update-the-stale-77-merged-prs-claim-to-the-verified-96-in-all-four-copies
kind: note
note_kind: decision
created: 2026-08-04T11:17:40Z
created_by: a-maintainer-vgqd2d
about: "[[253]]"
---
# update the stale '77 merged PRs' claim to the verified 96 in all four copies (index.md, README.md, overrides/home.html), not just SELFHOSTING
## Chose
update the stale '77 merged PRs' claim to the verified 96 in all four copies (index.md, README.md, overrides/home.html), not just SELFHOSTING
## Rejected
fixing only docs/SELFHOSTING.md and leaving the landing page/README at 77
## Because
the count is one marketing sentence duplicated across docs/index.md:19, README.md:11, overrides/home.html:24+54; correcting SELFHOSTING to 96 while three copies still read 77 leaves the record self-contradictory. 96 is verifiable from git log main (distinct 'Merge pull request #NNN' + squash '(#NNN)' subjects, #39–#293). README/home.html sit outside docs/ but carry the identical false claim; the maintainer axiom is never let the record lie.
