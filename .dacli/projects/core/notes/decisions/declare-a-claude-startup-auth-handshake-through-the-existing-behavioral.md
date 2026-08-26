---
id: d-declare-a-claude-startup-auth-handshake-through-the-existing-behavioral
kind: note
note_kind: decision
created: 2026-08-22T21:58:15Z
created_by: a-fixer-76xc6t
about: "[[t-01M0CZANEM3TFEMGTW3NTNXGXM]]"
github:
  issue: 779
  repo: mlnomadpy/dacli
---
# Declare a Claude startup-auth handshake through the existing behavioral_preflight adapter capability
## Chose
Declare a Claude startup-auth handshake through the existing behavioral_preflight adapter capability
## Rejected
Add Claude-specific authentication logic to spawn or scheduler
## Because
RuntimeLaunchPreflight already exposes provider-neutral state/layer evidence to doctor, preflight, and resolveLaunch before records are created.
