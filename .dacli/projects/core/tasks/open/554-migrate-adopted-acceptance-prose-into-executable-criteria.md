---
id: t-01M1493JHEAX82WX246TPCDR43
kind: task
created: 2026-08-28T13:31:49Z
created_by: a-root
owner: a-root
github:
  issue: 875
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Migrate adopted acceptance prose into executable criteria
## Context
Adopted from GitHub issue #875.

## Parent

Extracted from #871 and observed while landing #813: adopted GitHub issues can have detailed prose acceptance but zero executable dacli criteria.

## Observed symptom

`github pull` preserves an issue body as task context but does not reliably materialize prose or checklist acceptance as structured task checkboxes. Implementation can finish with strong evidence while `task done` correctly refuses because the task has no acceptance contract.

## Objective

Make acceptance extraction explicit, previewable, and auditable during adoption and for existing tasks.

Suggested recovery surface:

```bash
dacli task acceptance migrate <ref> --from-section "Acceptance criteria" --dry-run
```



## Non-goals

- Marking acceptance criteria complete during import.
- Inventing requirements absent from the issue.
- Weakening `task done`'s empty-acceptance refusal.

## Manual workaround today

The owner edits the adopted task contract or GitHub issue manually, then records evidence outside the original adoption transaction.

## Acceptance
- [ ] GitHub adoption recognizes checkbox lists and clearly headed acceptance sections without treating arbitrary bullets, examples, or non-goals as criteria.
- [ ] Dry-run shows the exact normalized criteria, source issue/body digest, ambiguities, and skipped lines; it writes nothing.
- [ ] Apply consumes the same immutable plan, is idempotent, preserves the original body, and records actor/source/time.
- [ ] Ambiguous prose fails closed for owner editing; no model-generated criterion is silently asserted as human intent.
- [ ] Existing structured criteria are preserved and deduplicated; checked remote boxes never become locally verified solely because they were checked on GitHub.
- [ ] Tasks with zero criteria receive an actionable migration/edit path before implementation or landing begins.
- [ ] GitHub re-pull does not erase locally refined criteria or duplicate migrated ones.
- [ ] Fixtures cover Markdown checkboxes, prose acceptance bullets, nested lists, code blocks, non-goals, duplicates, and zero usable criteria.
## Log
