---
id: d-acceptall-gained-a-force-bool-param-mirroring-the-single-ref-override-ship
kind: note
note_kind: decision
created: 2026-07-23T10:34:43Z
created_by: a-c4n7ak99hj
about: [[092]]
---
# acceptAll gained a force bool param mirroring the single-ref override; ship always forwards --force to accept --all
## Chose
acceptAll gained a force bool param mirroring the single-ref override; ship always forwards --force to accept --all
## Rejected
a separate 'reconcile' subcommand or ship-only --force flag that toggles whether to forward it
## Because
accept already has the exact root+--force override semantics for the single-ref path (acceptance.go cmdAccept); reusing it for --all keeps one override policy instead of two, and forwarding --force unconditionally from ship is safe because accept only honors it when the acting identity is root — so ship run by a non-root identity is unaffected
