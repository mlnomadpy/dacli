---
id: d-keep-the-shipped-gh-adapter-and-proposed-github-app-as-separate-documentation
kind: note
note_kind: decision
created: 2026-08-13T13:32:41Z
created_by: a-fixer-devvfk
about: "[[416]]"
github:
  issue: 578
  repo: mlnomadpy/dacli
---
# Keep the shipped gh adapter and proposed GitHub App as separate documentation contracts
## Chose
Keep the shipped gh adapter and proposed GitHub App as separate documentation contracts
## Rejected
Describe the App as an authentication upgrade to the current command path
## Because
No App receiver or installation-token path ships; a separate adapter boundary preserves explicit sync and local ownership while allowing future event-driven operation
