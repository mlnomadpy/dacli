---
id: d-when-a-dacli-command-cannot-do-the-job-fix-the-command-do-not-reach-past-it-for
kind: note
note_kind: decision
created: 2026-08-04T16:19:13Z
created_by: a-root
---
# When a dacli command cannot do the job, fix the command — do not reach past it for the raw tool
## Chose
When a dacli command cannot do the job, fix the command — do not reach past it for the raw tool
## Rejected
using gh/git directly whenever a dacli command has a gap, and moving on
## Because
Owner directive, 2026-08-04, after I filed seven issues with `gh issue create` because `dacli github push` has no task window and would have mirrored ~110 unmirrored tasks onto a public repo at once. The workaround was safe for that moment and wrong as a habit: dacli still cannot mirror a wave, the next operator hits the same wall, and the product gets no better. Worse, it left real damage — those seven issues carry no dacli marker and no mapping, so the next `github push` would re-create all seven as duplicates, which is exactly the failure the marker exists to prevent (see task 208, 205). The rule: a gap in a dacli command is a task, not a detour. Reaching for the raw tool is acceptable only to unblock the immediate step, and only when the gap is filed in the same breath. Filed as 275, which must also adopt an existing issue that carries no marker — the mess this decision was written about.
