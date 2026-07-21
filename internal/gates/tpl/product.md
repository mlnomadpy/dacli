---
name: product
summary: a full product lifecycle — discovery through release, phase-gated
cost: "five phases with role gating; use when who-does-what-when actually matters"
---
# product

Each stage is a lifecycle phase that only certain role KINDS may work. You
cannot spawn an implementer during discovery — advance the gate first. This
is how the team knows when to use which sub-agent.

## stage: discovery
cone: definition
phase: discovery
allow: researcher, reviewer
- project_sections: Goal | Out of scope
- glossary: min_terms 3

## stage: research
cone: elicitation
phase: research
allow: researcher, reviewer
- decisions: min 1
- risks: rank1_have_action

## stage: planning
cone: approach
phase: planning
allow: planner, researcher, reviewer
- tasks: all_have_acceptance
- tasks: all_have_estimate

## stage: build
cone: design
phase: implementation
allow: implementer, reviewer
- tasks: musts_done

## stage: release
cone: design
phase: release
allow: implementer, reviewer
- retro: required
