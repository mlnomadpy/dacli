---
id: f-071-complete-on-branch-dacli-071-commit-45e21af-all-3-acceptance-criteria-met
kind: note
note_kind: finding
created: 2026-07-22T20:15:45Z
created_by: a-rncv68m0fd
about: [[071]]
severity: moderate
---
# 071 complete on branch dacli/071 — commit 45e21af, all 3 acceptance criteria met, build+test green
Implemented dacli github project <project> in internal/features/ghmirror/project.go (+ ghmirror.go registration, project_test.go). Staged ONLY the 3 scoped files (git add + dacli commit --no-add). ACCEPTANCE: (1) cmdProject creates/links a Project v2 (ensureProject: stored github_project block number+id → gh project list adopt-by-title → gh project create) and adds every mirrored issue via ensureItem (gh project item-add by issue URL); operator-triggered and behind the SAME disclosureGate as push (project.go:~250). (2) Fields map dacli→Project: Status from task folder, Severity from finding severity, Area from area: label (boardFields/taskItemFields/findingItemFields); idempotent — ensureProject/ensureFields(reuse-by-name)/ensureItem(itemIndexByNumber snapshot keyed on content issue-number) mean a re-run duplicates no board, field, or item (project.go itemIndexByNumber, ensureItem). (3) Uses gh project create/list/field-list/field-create/item-list/item-add/item-edit; NO live gh in tests — project_test.go unit-tests the mapping (boardFields, severityValue, areaValue, item assignments) and the pure JSON parsers (parseProjectList/parseFieldList/parseItemList, optionID, itemIndexByNumber, stored-project round-trip). go build ./... clean; go test ./internal/... all green incl. internal/features/ghmirror and the TestFeatureSlicesAreIsolated arch test. Field-set is best-effort: a single-select value that is not an option on the resolved field (e.g. a built-in Status with Todo/In-Progress/Done) is skipped, not fatal — see decision notes. Box-checking is owner-only (a-root); owner: verify and close via dacli accept 071 (or task check/done) + dacli merge --task 071.
