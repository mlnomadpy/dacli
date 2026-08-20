---
id: 01M0D3X6PJ042NHD209477S23Z
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-19T13:38:25Z
created_by: a-maintainer-1cw4s7
about: "[[t-01M0AEYFXAB22RE9Y2SH9WZZKR]]"
origin: agent
applied: true
checksum: sha256:5b8f0fbf04477d913ed349ef27f1297da8e610ef2b5ba5921e330e91fe3c4708
---
708488f t-01M0AEYFXAB22RE9Y2SH9WZZKR: make runtime context provenance explicit

Record six provider-neutral context classes, enumerate fixtureable vendor roots, refuse undeclared strict context, and preserve explicit cooperative provenance in run records and verification panels.

Mutation: removing Codex .agents/skills discovery made TestContextIssuesRefuseUndeclaredGlobalSkillAndOverrideRecordsSource fail because strict context accepted the fixture.
role: maintainer
