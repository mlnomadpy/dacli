---
id: f-burn-rate-population-filter-is-defeated-by-any-ro-grant-role-missing-role-kind
kind: note
note_kind: finding
created: 2026-08-04T00:37:40Z
created_by: a-1hwz5pcjva
about: "[[t-01KY60QM1Y7DK05WXB954YNDHJ]]"
source_event: 01KYFP68109BKE3PCP2ZFE79R9
---
# burn Rate population filter is defeated by any ro-grant role missing role_kind (reviewer.md)
Verified against source. burn.go:261 reviewerRoles builds its exclusion set ONLY from roles where team.Role.Kind=='reviewer' (role_kind frontmatter, store/roles.go:91). .dacli/roles/reviewer.md (the primary review seat: grant: ro, model opus, 'never implements') has NO role_kind field, so store.LoadRoles yields Kind=='' and its runs are NOT excluded from burnSeries' per-run Rate (burn.go:142-152). This re-introduces the exact population mismatch task 149 (merged 828f164) was built to remove: reviewer runs are not in the Ceiling's calibration population (an ro role cannot commit/complete a task -> never a store calibration sample, cf. TestPRRefusesReadOnlyGrant), yet they now count in the Rate, so the Ratio=Rate/Ceiling that yells at 1.5x compares mismatched populations. Root cause is a class, not one file: ANY ro role (auditor/researcher/reviewer) authored without role_kind silently contaminates the Rate. The durable fix is to align the Rate population with the Ceiling population in code -- exclude runs whose role has grant==model.GrantRO (burn.go reviewerRoles), since an ro role never produces a ceiling sample -- rather than only backfilling role_kind into one workspace file. team.Role already carries Grant (team.go:42) so LoadRoles exposes it. Frontend-reviewer.md is unaffected (it declares role_kind: reviewer).
