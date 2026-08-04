---
id: f-refusal-message-audit-63-refusedf-sites-exit-classes-clean-secondary-way-out
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-go-auditor-z48ata
about: "[[t-01KZ6S9Z9ZXRW12XZA1GJJSGB6]]"
source_event: 01KZ6SM2BQK6233WKN0JNE5RE8
---
# Refusal-message audit (63 Refusedf sites): exit classes clean; secondary way-out gaps in acceptance --require-verify and the rw-grant class
Checked all 63 non-test clikit.Refusedf sites against the rule 'a refusal must name the action that would succeed', plus the exit-class contract (clikit.go:52-56: exit 2 = caller mistake, exit 3 = policy refusal).

EXIT-CLASS RESULT — CLEAN. No policy answer is returned as exit 1 and no real failure is returned as exit 3. The three suspicious `Refusedf("%v", err)` wrappers each wrap a genuine POLICY condition, so exit 3 is correct:
  - teamops.go:72 wraps agentid.ErrAttenuation (a grant-attenuation policy);
  - shortcuts.go:214 wraps shortcut.Guard (a role/grant/confirm policy gate);
  - orchestration.go:179 wraps errCorruptState — a DELIBERATE, documented refusal (orchestration.go:170-174): resuming from a corrupt governor snapshot would silently reset the token ceiling and thrash guard, so it refuses AND names the recourse (`delete %s to start a fresh window`). Legitimate exit 3, and it names the way out.

WAY-OUT GAPS beyond the primary (execution.go:468, filed separately as major):
  1. acceptance.go:178 and acceptance.go:341 — "--require-verify is set and no --verify command was given: task(s) cannot be closed on unverified assertions". States the blocking condition but does not spell the fix `pass --verify \"<cmd>\"` (or drop --require-verify). The condition half-implies the fix, so MINOR.
  2. The rw-grant class (~12 sites: clikit.go:67 RequireRW, vcs.go:82, lifecycle.go:215/253/991/1074, ship.go:93, release.go:49, stagegate.go:124, skillforge.go:205, teamops.go:337/476, queues.go:104, orchestration.go:193) uniformly say '<action> needs an rw grant (yours is %s)'. They name the REQUIREMENT but never the RECOURSE — an agent's grant is fixed at spawn and it cannot self-escalate, so the actual way out (have root or an rw-granted agent run it, or `dacli ask`) is unstated. Consistent and arguably by-design, so MINOR/systemic rather than a single defect.

Everything else surveyed names its remedy well (e.g. planning.go:137/177 --force, vcs.go:97 branch-first, execution.go:1069/1072 estimate/decompose, ship.go:99/131, lifecycle.go:1416).
