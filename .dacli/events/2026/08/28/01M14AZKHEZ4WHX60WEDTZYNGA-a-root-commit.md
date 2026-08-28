---
id: 01M14AZKHEZ4WHX60WEDTZYNGA
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-28T14:04:36Z
created_by: a-root
about: "[[t-01M12QX9HEPKAAS1033W6HS45D]]"
origin: agent
applied: true
checksum: sha256:9d464e92553f8809fb341ee6bb69fd8a6566ff21826f6bdaddf47f8c0b9a37ef
---
3afd1fd7 529: make PR shipping land before acceptance

Make explicit PR shipping checks-gated and PR-first, verify the reviewed head on fresh configured trunk, accept only afterward, then run scoped GitHub projection. Keep nonterminal PR bodies from closing issues early and make direct refusals point to the non-cyclic transaction.

Owner review added pre-merge guards for nonempty/fully checked acceptance, mandatory post-landing verification, no-merge conflicts, and done/mixed windows.

Mutation proofs: disabling landThenAccept makes TestShipPRExplicitOpenTaskLandsThenAcceptsAndClosesIssue fail with 'task became terminal before the PR merge'; suppressing the unchecked guard makes TestShipPRTransactionRefusesUncheckedTaskBeforeIntegration succeed unexpectedly and fail its assertion.
role: root
