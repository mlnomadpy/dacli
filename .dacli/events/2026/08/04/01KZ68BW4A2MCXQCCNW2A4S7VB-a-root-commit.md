---
id: 01KZ68BW4A2MCXQCCNW2A4S7VB
kind: event
event_kind: commit
created: 2026-08-04T11:25:57Z
created_by: a-root
origin: agent
applied: false
---
87b267a 253 review: document the UNENFORCED half of the grant/runtime coupling

The docs pass covered the ro direction — a ro grant on a runtime with
no read-only sandbox is refused at exit 3 — which is real and enforced.
It is also the half that has never cost anyone a run.

sandboxFor returns early for any grant that is not ro, so an rw role
spawns on ANY runtime, including one whose adapter grants no write tool.
junior shipped exactly that: grant rw, runtime cc, and cc's allowlist is
Read/Grep/Glob/LS plus the dacli binary. Nothing refuses it. The agent
starts, reads its brief, tries to edit a file, cannot, and the run is
spent — which is what happened on task 183 today.

A reader told only about the ro refusal concludes the coupling is
handled and routes implementation work to a role that cannot write. Both
directions are now stated, in TEAM.md and in the generated ROSTER
preamble, with the check being: read the role's runtime, then that
runtime's allowlist in .dacli/runtimes/, and confirm Edit and Write are
actually there. A grant says what the workspace permits; the runtime's
allowlist says what the process can do, and only the second is real.

The ROSTER preamble lives in catalog.go, not the generated page — the
page edit alone would have been overwritten by the next `dacli catalog`.
Its test now asserts the rw half is present too.
role: root
