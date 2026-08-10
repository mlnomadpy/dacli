---
id: t-01KZPWRY9ARRDAAWKNPS6NFP43
kind: task
created: 2026-08-10T22:30:28Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Add fuzz targets and failure injection for orchestration, git operations, and event-log recovery
## Acceptance
- [ ] fuzz targets exist for the parsers that read untrusted or malformed input: task frontmatter, event records, and flag parsing
- [ ] a failure-injection test proves the loop recovers when a git command fails mid-cycle
- [ ] a failure-injection test proves the event log is readable after an interrupted write
- [ ] CI runs the fuzz corpus as a bounded seed run, so a crasher fails the build rather than waiting for someone to run it
## Log
