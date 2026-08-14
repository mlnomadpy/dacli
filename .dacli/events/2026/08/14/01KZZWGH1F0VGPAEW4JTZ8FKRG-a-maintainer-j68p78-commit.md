---
id: 01KZZWGH1F0VGPAEW4JTZ8FKRG
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-14T10:19:02Z
created_by: a-maintainer-j68p78
about: "[[t-01KZZVFWZWP3M2KX52E1FF6CMA]]"
origin: agent
applied: true
checksum: sha256:77af7f2cbdd71601fea4e6608f116743f8f8d87c13130018cf98c240f19cb4c9
---
fab4345 t-01KZZVFWZWP3M2KX52E1FF6CMA: preserve transcript-active detached claims

Issue #672 keeps non-empty transcript evidence live through the configured runtime timeout unless the guardian records runtime exit. Named wait now skips already-terminal records so finalization is exactly once.

Mutation: removing the transcript lease made TestTranscriptActiveUnobservableRunSurvivesStatusWaitAndClaimLookup fail at claim_release_test.go:141: status probe lost transcript-active run.
role: maintainer
