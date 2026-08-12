---
id: d-make-status-commands-purely-observational-and-retain-finalization-in-wait
kind: note
note_kind: decision
created: 2026-08-12T18:25:31Z
created_by: a-codex-maintainer-j8jbvt
about: "[[382]]"
---
# make status commands purely observational and retain finalization in wait
## Chose
make status commands purely observational and retain finalization in wait
## Rejected
teach status sweeps to distinguish every process-probe failure mode
## Because
a status caller cannot authenticate exit when process visibility is restricted; removing lifecycle writes from reads satisfies the invariant while wait/watchdog/kill remain explicit lifecycle authorities
