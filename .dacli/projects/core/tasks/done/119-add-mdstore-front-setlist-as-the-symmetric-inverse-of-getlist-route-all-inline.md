---
id: t-01KY84R5NYX4QWTCYVGM5G8MD3
kind: task
created: 2026-07-23T18:45:33Z
created_by: a-92n83ap1x9
owner: a-root
priority: should
---
# Add mdstore.Front.SetList as the symmetric inverse of GetList; route all inline-list writers through it
## Acceptance
- [x] mdstore exposes a Front.SetList(key, []string) (or equivalent) that encodes an inline list as the exact inverse of GetList: any element containing a comma, bracket, brace, quote, or leading/trailing whitespace is quoted so it round-trips losslessly through splitTop/clean
- [x] internal/store/runtimefiles.go setInline (runtimefiles.go:79-83) and internal/gates/gates.go writePhase (gates.go:313) both call the shared encoder instead of hand-rolling '['+strings.Join(v, ', ')+']', so the two known duplicate call sites of the unquoted-join antipattern are eliminated
- [x] a new mdstore round-trip test proves SetList->GetList is identity for values containing commas, spaces, brackets, and quotes (the class the a-xrcxmhwz96 finding said had no coverage: no runtimefiles_test.go exists today)
- [x] go build ./... and go test ./internal/... are green
## Log
- 2026-07-23T18:47:24Z claimed by a-bk3fqg83sm
- 2026-07-23T18:50:34Z adopted by a-root (owner a-92n83ap1x9 orphaned)
- 2026-07-23T18:50:34Z accepted by a-root
- 2026-07-23T18:50:34Z completed by a-root
