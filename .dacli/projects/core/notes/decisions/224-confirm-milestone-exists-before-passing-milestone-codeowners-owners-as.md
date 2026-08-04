---
id: d-224-confirm-milestone-exists-before-passing-milestone-codeowners-owners-as
kind: note
note_kind: decision
created: 2026-08-04T12:22:43Z
created_by: a-maintainer-rmz323
about: "[[224]]"
---
# 224: confirm milestone exists before passing --milestone; CODEOWNERS owners as @owner/role team handles
## Chose
224: confirm milestone exists before passing --milestone; CODEOWNERS owners as @owner/role team handles
## Rejected
Pass --milestone unconditionally after a best-effort create; emit bare role names as CODEOWNERS owners
## Because
gh issue create --milestone HARD-FAILS on an unknown milestone and would abort the whole push, so ensureMilestone must POSITIVELY confirm (POST then re-list) before the flag is passed — an unconfirmed milestone is skipped like the best-effort labels. CODEOWNERS owners must be @-prefixed GitHub identities; @owner/role team handles are the honest mapping of a dacli role to a review team (org from the linked repo or --owner), where a bare role name is not a valid owner.
