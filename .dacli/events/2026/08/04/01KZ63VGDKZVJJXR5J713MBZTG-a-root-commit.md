---
id: 01KZ63VGDKZVJJXR5J713MBZTG
kind: event
event_kind: commit
created: 2026-08-04T10:07:07Z
created_by: a-root
about: "[[t-01KZ63T6SPC4TP4RC9F8WGWAPG]]"
origin: agent
applied: false
---
a99c245 251: assert on discriminator-length windows, not 4-char coincidences

TestIDDoesNotLeakTheToken rejected any 4-character window of the token
appearing anywhere in the id. The id's discriminator is 6 characters, so
that assertion is not testing derivation — it is testing whether a
45-window scan happens to collide with a 6-character string, which it
does on roughly one run in twenty across 25 spawns. It went red on PR
#283 on code that leaks nothing.

The window now matches the discriminator length. Any scheme that derived
the discriminator from the token — prefix, suffix, hash slice — still
produces a full-length match and trips on the first spawn, so the test
keeps the property it was written for and loses the coincidence.

Verified with -count=200 (5000 spawns), green.
role: root
