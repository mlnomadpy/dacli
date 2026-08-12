---
id: d-preserve-explicit-implementer-provenance-in-loopcfg-and-route-only-defaults
kind: note
note_kind: decision
created: 2026-08-12T19:24:06Z
created_by: a-codex-maintainer-hyzqzv
about: "[[373]]"
github:
  issue: 528
  repo: mlnomadpy/dacli
---
# Preserve explicit implementer provenance in loopCfg and route only defaults automatically
## Chose
Preserve explicit implementer provenance in loopCfg and route only defaults automatically
## Rejected
Infer explicitness later from the selected role name or disable phase routing
## Because
The selected string cannot reveal whether it came from --impl-role or the project stack; a boolean preserves operator intent while buildRole can still enforce phase-kind gates.
