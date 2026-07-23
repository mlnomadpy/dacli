---
id: d-add-identity-mutaterefusal-distinguishing-ro-grant-from-non-ownership-accept-s
kind: note
note_kind: decision
created: 2026-07-23T10:52:05Z
created_by: a-n9n6r0nn4w
about: [[094]]
---
# add Identity.MutateRefusal() distinguishing ro-grant from non-ownership; accept's non-owner case also names --force root override
## Chose
add Identity.MutateRefusal() distinguishing ro-grant from non-ownership; accept's non-owner case also names --force root override
## Rejected
keep the single hardcoded '(read-only grant)' string and just special-case accept
## Because
claim/done/block and accept all call CanMutate for the same two distinct reasons (Grant != RW vs RW-but-not-owner); a shared Identity method keeps the distinction in one place instead of four ad-hoc checks, and only accept exposes a --force root override so only its message mentions it
