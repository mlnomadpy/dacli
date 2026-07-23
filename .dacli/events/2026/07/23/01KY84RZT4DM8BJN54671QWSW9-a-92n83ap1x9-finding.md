---
id: 01KY84RZT4DM8BJN54671QWSW9
kind: event
event_kind: finding
created: 2026-07-23T18:46:00Z
created_by: a-92n83ap1x9
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
mdstore has quote-aware GetList read but no SetList write; gates.go:313 is an unguarded duplicate of the antipattern task 111 fixed in runtimefiles

mdstore/mdstore.go:128 GetList decodes inline lists quote-aware via splitTop (mdstore.go:176) + clean (mdstore.go:151), but there is NO symmetric write encoder. Both inline-list writers hand-roll '['+strings.Join(v, ', ')+']' with zero element quoting: internal/store/runtimefiles.go:81 (setInline) and internal/gates/gates.go:313 (writePhase, phase_allows). Task 111 (branch dacli/111, commit e3307d2, done but NOT yet merged to main) fixed ONLY the runtimefiles copy by inline-quoting; gates.go:313 still hand-rolls the unquoted join. Live impact at gates.go:313 is currently nil because Stage.Allow elements are produced by splitting an 'allow:' line on commas (gates.go:168), so no element can contain a comma -- it is a latent duplicate, not an exploitable bug. The value is closing the class: a shared mdstore SetList (inverse of GetList) + a round-trip test (none exists -- no runtimefiles_test.go, confirmed) means the next inline-list writer cannot reintroduce the --allowedTools corruption that 111 just fixed. Filed as task 119.
