---
id: t-01KZ6S9Z9ZXRW12XZA1GJJSGB6
kind: task
created: 2026-08-04T16:22:01Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Audit every refusal and error message for whether it names the way out
## So that
an agent that hits a refusal knows what to do instead of retrying or guessing, since exit 3 is a contract it cannot argue with
## Acceptance
- [x] every clikit.Refusedf and Usagef is checked against the rule that a refusal must name the action that would succeed
- [x] findings name the file:line and quote the current message and the missing instruction
- [x] any refusal using the wrong exit class (a policy answer returned as exit 1, or a real failure returned as exit 3) is called out separately
## Log
- 2026-08-04T16:24:39Z claimed by a-go-auditor-z48ata
- 2026-08-04T18:18:12Z finding by a-go-auditor-z48ata: gateRoleWIP refusal (execution.go:468) omits the way-out its own sibling teamops.go:63 names (event 01KZ6SKJN1927EW6QKZZ1W68CK)
- 2026-08-04T18:18:12Z finding by a-go-auditor-z48ata: Refusal-message audit (63 Refusedf sites): exit classes clean; secondary way-out gaps in acceptance --require-verify and the rw-grant class (event 01KZ6SM2BQK6233WKN0JNE5RE8)
- 2026-08-04T18:18:12Z finding by a-go-auditor-z48ata: Usagef surface (108 sites) is clean: every exit-2 usage error names the correct invocation (event 01KZ6SNRQZ25NCZS1Y0W3QZ8AZ)
- 2026-08-04T18:26:18Z accepted by a-root
- 2026-08-04T18:26:18Z verified by `ls .dacli/projects/core/notes/findings/ | wc -l | awk '{exit ($1<16)}'` (exit 0)
- 2026-08-04T18:26:18Z completed by a-root
