---
id: f-283-complete-on-branch-dacli-283-readme-no-longer-calls-skill-shortcut-promote
kind: note
note_kind: finding
created: 2026-08-04T20:14:08Z
created_by: a-maintainer-d26jvm
about: "[[283]]"
severity: major
---
# 283 complete on branch dacli/283-...: README no longer calls skill/shortcut promote stubs
Commit e2d756a on branch dacli/283-readme-calls-skill-shortcut-promote-unimplemented-stubs-but-both-are-shipped. Fixes (all 4 acceptance criteria): (1) README.md:85 rewrote the 'honest stubs that refuse' line to 'The whole command surface is implemented and tested — including skill promote ... and shortcut promote ...'. (2) README.md:278 'Still stubbed ...' line removed; added proper reference rows: 'dacli shortcut promote' under Shortcuts & queues and 'dacli skill promote' under Skills, templates & stage gates, so the command reference matches shipped behavior. (3) Verified against code: skillforge.go:86-138 cmdPromote (registered skillforge.go:28) and shortcuts.go:75 cmdPromote (registered shortcuts.go:24) both implement the commands; grep confirms clikit.Planned has ZERO callers (only its definition at clikit.go:93). (4) Swept all *.md: no remaining doc calls an implemented command a stub/planned; docs/README.md:30, docs/SKILLS.md:3, docs/SHORTCUTS.md:3 already correct; docs/GITHUB.md:73 and docs/REVIEW.md:69 hits are unrelated (github pull / a dated historical review record). Added regression test internal/cli/readme_status_test.go (TestReadmeDoesNotCallPromoteStubbed): FAILS on the pre-change README (4 assertions incl. the exact 'honest stubs that refuse' and 'Still stubbed' phrasings), PASSES after — verified via git stash. Proof: go build ./... clean, go test ./... all green, go vet clean, gofmt -l internal/ empty. Owner: dacli accept 283 then integrate/merge --task 283.
