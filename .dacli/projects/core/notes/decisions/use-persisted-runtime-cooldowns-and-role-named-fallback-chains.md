---
id: d-use-persisted-runtime-cooldowns-and-role-named-fallback-chains
kind: note
note_kind: decision
created: 2026-08-13T10:20:24Z
created_by: a-codex-maintainer-76ksyq
about: "[[406]]"
---
# Use persisted runtime cooldowns and role-named fallback chains
## Chose
Use persisted runtime cooldowns and role-named fallback chains
## Rejected
Infer a replacement provider or encode vendor-specific retry branches in spawn
## Because
Role fallback_to is ordered and opt-in, while a provider-neutral classifier and breaker preserve grant/capability floors and survive loop process restarts.
