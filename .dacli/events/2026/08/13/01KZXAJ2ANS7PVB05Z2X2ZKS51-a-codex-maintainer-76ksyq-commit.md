---
id: 01KZXAJ2ANS7PVB05Z2X2ZKS51
kind: event
event_kind: commit
created: 2026-08-13T10:26:49Z
created_by: a-codex-maintainer-76ksyq
about: "[[t-01KZX7PXQBEVM1M0N2BKWYD4RK]]"
origin: agent
applied: true
---
108ad3c 406: add explicit provider limit policies

Add typed provider outcomes, reset-aware bounded retry policy, persistent
runtime circuit breakers, and ordered role fallback eligibility that preserves
grant and capability floors.

Red-test mutation: adding PermanentInput to Fallbackable failed
TestFallbackCannotWeakenPolicy with: permanent_input triggered fallback.
role: codex-maintainer
