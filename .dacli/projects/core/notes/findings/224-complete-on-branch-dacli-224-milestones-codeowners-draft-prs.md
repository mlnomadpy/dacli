---
id: f-224-complete-on-branch-dacli-224-milestones-codeowners-draft-prs
kind: note
note_kind: finding
created: 2026-08-04T12:23:08Z
created_by: a-maintainer-rmz323
about: "[[224]]"
severity: moderate
---
# 224 complete on branch dacli/224: milestones + CODEOWNERS + draft PRs
Commit c339cff by a-maintainer-rmz323 on branch dacli/224-add-milestones-draft-prs-and-codeowners-to-the-github-surface. Acceptance (projects map to milestones AND roles emit CODEOWNERS) both satisfied. (1) MILESTONES: ghmirror cmdPush ensures one milestone per project (milestoneTitle = project title or slug) via ensureMilestone in internal/features/ghmirror/ghmirror.go — it POSTs to the REST milestones endpoint then re-lists to CONFIRM before passing --milestone, because gh issue create --milestone hard-fails on an unknown milestone and would abort the push; task issues (new, adopted, mapped) are assigned via applyMilestone. (2) CODEOWNERS: new internal/features/ghmirror/codeowners.go plus command github codeowners (project arg or --owner flag) writes .github/CODEOWNERS from role scope globs to at-owner-slash-role team handles, patterns most-general-first for last-match-wins, and refuses to write a hollow file when no role has scope. (3) DRAFT PRs (title): dacli pr --draft threads a flag into openPR and gh pr create in internal/features/vcs/lifecycle.go; integration never drafts. Tests: milestone_test.go, codeowners_test.go, prdraft_test.go. Verified: go build, go vet, gofmt -l internal all clean; go test ./... green with DACLI_AGENT stripped (the lone catalog failure is the pre-existing agent-token env leak in this session per f-016/f-021, not this change). NOTE: I first edited via absolute paths into the MAIN checkout (sibling branch dacli/261) by mistake, then restored those files with git restore plus git clean and re-applied everything in the worktree; the main checkout is untouched.
