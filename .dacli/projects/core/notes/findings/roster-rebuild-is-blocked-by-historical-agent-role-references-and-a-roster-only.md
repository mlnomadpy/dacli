---
id: f-roster-rebuild-is-blocked-by-historical-agent-role-references-and-a-roster-only
kind: note
note_kind: finding
created: 2026-08-18T12:50:30Z
created_by: a-codex-maintainer-as8sk8
about: "[[t-01M0AEG5694R7SDMSREJ8KPF4K]]"
severity: major
---
# Roster rebuild is blocked by historical-agent role references and a roster-only claim
Reproduced with '/tmp/dacli-audit-bin role rm codex-maintainer': exit 3 names 16 non-retired agent files, including many completed historical runs, although acceptance requires refusal only while a live agent holds the role. internal/store/remove.go:50-65 checks every non-retired agent rather than procmon liveness. The task claim is only .dacli/roles/**,.dacli/skills/**,docs/ROSTER.md, so fixing that gate or retiring/repointing .dacli/agents is outside claim. Existing roles also cannot be converted in place: internal/features/teamops/teamops.go:375-438 only creates new roles and store.CreateRole refuses an existing name; role rm + add cannot pass the historical-reference gate. The obsolete unreferenced codex-process-architect role was successfully removed before this blocker surfaced.
