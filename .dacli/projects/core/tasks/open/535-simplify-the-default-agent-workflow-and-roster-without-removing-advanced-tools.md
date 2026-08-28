---
id: t-01M146B9RBYKTG8XCHNMZ72T9K
kind: task
created: 2026-08-28T12:43:36Z
created_by: a-root
owner: a-root
github:
  issue: 863
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: "[t-01M12QX9HEPKAAS1033W6HS45D, t-01M11KPNTMDJZAHR1YZHH6RF3A]"
---
# Simplify the default agent workflow and roster without removing advanced tools
## Context
Adopted from GitHub issue #863.

## Parent

Part of #855. Coordinate documentation changes with #825.

## Observed symptom

The full CLI exposes more than one hundred command entries and a large default role roster. Agent operators must understand overlapping lifecycle and portfolio commands before discovering the primary product-building journey.

## Objective

Make the default agent-facing workflow obvious while preserving advanced capabilities:

```text
inspect -> plan -> claim -> implement -> verify -> review -> PR -> CI -> merge
```

Provide a small default roster/preset and keep expert leaf commands available as explicit recovery or advanced tools.



## Non-goals

- Removing advanced functionality solely to reduce command count.
- Hiding safety gates.
- Silently selecting another coding harness.

## Manual workaround today

Agents depend on the skill and operator playbook to select a workable subset of the CLI.

## Acceptance
- [ ] Top-level and parent help present one primary bounded workflow, clearly distinguish ordinary orchestration from advanced/recovery/portfolio commands, and support parent help such as `dacli task --help`.
- [ ] A default roster contains planner, implementer, security implementer, reviewer, security reviewer, and integration owner capabilities without silently changing harness family.
- [ ] Small projects can run inspect/task modes without mandatory estimation or portfolio configuration; wave/loop scheduling still explains when estimates are needed.
- [ ] Overlapping `task done`, `accept`, `integrate`, and `ship` help states one owner and invariant for each lifecycle boundary, consistent with the resolution of #841.
- [ ] An executable walkthrough starts from a clean fixture, uses the default journey, and reaches a merged/verified task without relying on undocumented commands.
- [ ] Existing advanced commands remain addressable and backward-compatible or receive an explicit migration/deprecation contract.
- [ ] Snapshot/help tests fail when the primary journey or harness-preservation guarantee disappears.
## Log
- 2026-08-28T12:44:47Z dependency edit by a-root (event 01M146DFA1VY59BP40623WFEV5)
