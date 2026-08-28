---
id: t-01M1493JN9F1F22S30ZW9HCVNC
kind: task
created: 2026-08-28T13:31:49Z
created_by: a-root
owner: a-root
github:
  issue: 873
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[t-01M12QX9HEPKAAS1033W6HS45D, t-01M1493JQ62NY4E39PQHNEZ6TP]"
---
# Make public GitHub projection safe by default and issue closure explicit
## Context
Adopted from GitHub issue #873.

## Parent

Extracted from #871. Coordinate with disclosure consent, GitHub projection, task slices, and lifecycle transaction work in #841.

## Observed symptom

The PR path currently assembles acceptance, findings, verdicts, agent attribution, and a closing issue reference for publication. In a public repository, an intermediate/partial PR can close the parent issue prematurely or expose internal operational evidence that was never approved for publication.

## Objective

Make public-repository projection safe by default and make issue-closing semantics an explicit terminal-delivery decision.



## Non-goals

- Redacting source-code diffs selected for publication.
- Treating public-safe projection as authorization to merge.
- Uploading workspace transcripts.

## Manual workaround today

Operators inspect and edit every PR body/comment manually and replace closing keywords with `Refs` for partial deliveries.

## Acceptance
- [ ] A linked public repository defaults to a documented public-safe allowlist; private findings, transcripts, journals, recovery details, local paths, tokens/costs, and internal agent identities are withheld unless separately disclosed.
- [ ] Nonterminal and slice PRs use a non-closing reference to the issue; a closing keyword is emitted only when the command proves this PR is the terminal accepted delivery or the owner explicitly selects a closing mode.
- [ ] Dry-run shows the exact public title/body/comments/reviews and every withheld field with its policy reason, without mutation.
- [ ] Explicit include/findings/verdict options require recorded disclosure authority and cannot broaden a prior narrow approval silently.
- [ ] Private-repository behavior remains explicit and tested; repository visibility unknown fails closed to public-safe output.
- [ ] GitHub issue projection never publishes internal decision/finding issues by default merely because `--allow-public` linked the repository.
- [ ] Text, JSON, CLI, and MCP use one typed redaction/projection policy.
- [ ] Mutation tests fail if a partial PR gains `Fixes`/`Closes` or a private evidence field reaches a public fixture.
## Log
- 2026-08-28T13:32:49Z dependency edit by a-root (event 01M1495D4VRGS2B61QEP9B4ZNM)
