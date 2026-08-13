---
id: 01KZXKRJ2QKJEEYFVZHQZRB428
kind: event
event_kind: commit
created: 2026-08-13T13:07:39Z
created_by: a-fixer-rsb99q
about: "[[t-01KZXKHGDD0JJ1GK11MF5F26TG]]"
origin: agent
applied: true
---
af601d2 413: gate second title verb on explicit modifier

Treat the first title word as authoritative unless it is a supported modifier,
so implementation verbs cannot be overridden by later reviewer keywords while
Full audit/review/verify still route correctly.

Red: TestLeadingImplementationIntentBlocksLaterReviewerVerb failed all 12
Test/Check/Improve/Cover x verify/audit/review cases via the second word.
role: fixer
