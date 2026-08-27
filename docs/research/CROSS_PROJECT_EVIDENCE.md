# Cross-project swarm evidence

Status: **evidence register and case-study template.** This document measures
whether orchestrator agents using dacli deliver accepted product changes across
repositories. It does not measure whether a human memorizes the CLI.

## Evidence levels

| Level | Question | Minimum evidence |
|---|---|---|
| Technical | Can the mechanism work? | Reproducible tests or a landed self-hosted change |
| Cross-project workflow | Does it work on a different product/stack? | Run record joined to task, PR, CI, and landing outcome |
| Independent operation | Can another authority operate it repeatedly? | Repeated outcomes from an independent operator or orchestrator configuration |
| Commercial | Will an organization pay for or depend on it? | Purchase, renewal, funded deployment, or equivalent commitment |

Do not promote one level into another. Self-hosting is strong technical evidence.
A private-project report is a useful cross-project lead, but remains unverified
until its redacted run and landing artifacts are recorded.

## Current cross-project report

**Trust: unverified operator report, recorded 2026-08-27.** The project operator
reports using dacli with Codex to build applications in repositories other than
dacli and receiving product feedback from those applications. Repository names,
run IDs, task counts, PR outcomes, costs, interventions, and release evidence
have not been disclosed or imported. This establishes an evidence slot, not a
quantitative success claim.

## Case-study template

Copy this section for each publishable run. Use ranges or stable anonymized IDs
when exact values would expose proprietary information. Never include secrets,
private prompts, transcripts, customer data, or proprietary source.

```markdown
### Run <stable anonymized ID>

- Evidence date and trust grade:
- Repository archetype and stack:
- Product goal and success criterion:
- Orchestrator environment:
- Coding-agent runtimes and models:
- Operating profile, WIP, and budget policy:
- Planned tasks / completed tasks:
- PRs opened / reviewed / merged:
- CI and independent-review outcome:
- Elapsed time and reported token/cost evidence:
- Human interventions and authority exceptions:
- Policy refusals, cooldowns, recovery, or resumed journals:
- Release or user-feedback outcome:
- Redacted evidence references:
- What changed in the next run:
```

## Outcome metrics

Prefer end-to-end ratios over activity counts:

- Accepted tasks and merged PRs per planned task
- First-pass CI and independent-review success
- Human interventions and exceptional overrides per landed PR
- Tokens and cost per accepted change, labeled estimated or observed
- Recovery success after interruption, provider refusal, or landing uncertainty
- Lead time from product direction to verified trunk and, separately, release
- Defects caught before landing and defects reported after release
- Whether feedback changed the next recorded backlog or model-routing decision

Raw agent turns, spawned-agent count, and command count are diagnostics, not
product outcomes.
