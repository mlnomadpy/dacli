---
id: f-skill-lesson-body-content-is-lost-after-a-disk-round-trip
kind: note
note_kind: finding
created: 2026-08-04T00:39:12Z
created_by: a-fixer-zhxkc6
about: "[[206]]"
severity: moderate
---
# skill/lesson --body content is lost after a disk round-trip
internal/mdstore: when a Doc has Sections=[{Level:1,Title:X,Content:""},{Level:0,Content:body}], WriteFile then ReadFile merges the Level:0 section's content into the preceding Level:1 section's Content field, and skills.load() (internal/skills/skills.go load(), ~line 120) explicitly skips sec.Level==1 when building Skill.Body ('if sec.Level == 1 { continue }'). Net effect: skillforge.cmdAdd's --body (internal/features/skillforge/skillforge.go cmdAdd) and cmdPromote's lesson.Body are written to disk correctly (verified by reading the raw file) but vanish on every subsequent read — 'dacli skill show' never displays the authored body, and 'dacli skill compile' never includes it in inline/context delivery, meaning the actual skill instructions an agent authored are silently dropped from every compiled/inline surface. Verified via a probe test against internal/mdstore.WriteFile/ReadFile: a Level:0 section written right after an empty Level:1 header re-parses merged into the H1's Content, and skills.load() discards Level==1 content. Out of scope for task 206 (coverage-only); filing so it gets its own fix task.
