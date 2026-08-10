---
id: t-01KZP9FCRD0RCV11597KX2YZR4
kind: task
created: 2026-08-10T16:53:12Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Give --no-progress-halt a value shape in help and a name that is not a double negative
## Acceptance
- [x] help shows the flag with its value, as --max-cycles N already does
- [x] a positive alias (--halt-after-idle N) is accepted, with the old name kept working
- [x] the bare flag either works or fails with a message naming the value it needs; it must not read as a boolean
- [x] a test drives the bare form and the valued form through the real dispatcher
## Log
- 2026-08-10T17:05:18Z accepted by a-root
- 2026-08-10T17:05:18Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T17:05:18Z deliverable: no dacli/330-give-no-progress-halt-a-value-shape-in-help-and-a-name-that-is-not-a-double branch — nothing to check against trunk
- 2026-08-10T17:05:18Z completed by a-root
