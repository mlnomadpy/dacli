---
id: d-kept-native-title-tooltips-two-segment-burndown-bar-instead-of-forcing-shadcn
kind: note
note_kind: decision
created: 2026-07-26T17:40:13Z
created_by: a-tja4fdtr3z
about: [[152]]
---
# Kept native title tooltips + two-segment burndown bar instead of forcing shadcn Tooltip/Progress into every section
## Chose
Kept native title tooltips + two-segment burndown bar instead of forcing shadcn Tooltip/Progress into every section
## Rejected
Swap every hover-title to a reka-ui Tooltip and rebuild both burndown bars on the Progress primitive to literally use each named component
## Because
The acceptance's 'Card/Table/Badge/Progress/Tooltip/etc.' is an illustrative list, not a mandate that all six appear. AgentRow's freshness dot + state badge and the DAG use native title attributes that the tests assert on and that a screen reader/hover reads without JS — a Tooltip would break those tests and add focus-trap complexity to a read-only console for no gain. The ProjectCard burndown is a done(green)/remaining(blue) SPLIT; Progress renders a single-color fill, so forcing it would drop the two-hue meaning that mirrors status done=green. Sections are genuinely rebuilt on Card/Table/Badge/Button + Tailwind theme tokens; Progress/Tooltip stay exercised by the ui foundation smoke test (ui.test.ts).
