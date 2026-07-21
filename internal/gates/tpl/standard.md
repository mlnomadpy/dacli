---
name: standard
summary: a team, a real backlog, work others depend on
cost: "three gates; skip it if nobody else depends on the work"
---
# standard

## stage: define
cone: definition
- project_sections: Goal
- glossary: min_terms 2
- decisions: min 1

## stage: build
cone: approach
- tasks: all_have_acceptance
- tasks: all_have_estimate
- risks: rank1_have_action

## stage: ship
cone: design
- tasks: musts_done
- retro: required
