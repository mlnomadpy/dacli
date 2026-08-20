---
id: d-delegate-operating-profile-execution-to-existing-command-strategies
kind: note
note_kind: decision
created: 2026-08-19T15:21:12Z
created_by: a-maintainer-3necr2
about: "[[t-01M0CX03Q4A1BM8JD9YQBCNGV0]]"
github:
  issue: 754
  repo: mlnomadpy/dacli
---
# Delegate operating-profile execution to existing command strategies
## Chose
Delegate operating-profile execution to existing command strategies
## Rejected
Duplicate task, wave, and loop orchestration inside start
## Because
The existing loop path already owns critical-path selection, cheapest-capable routing, claims, verification, landing, budgets, and durable recovery; start resolves policy and invokes that public strategy so direct commands remain compatible
