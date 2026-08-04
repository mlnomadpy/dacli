---
id: d-require-2-shared-significant-words-word-set-not-substring-for-a-lesson-to
kind: note
note_kind: decision
created: 2026-08-04T10:17:19Z
created_by: a-maintainer-07d86s
about: "[[248]]"
---
# Require >=2 shared significant words (word-set, not substring) for a lesson to attach to a task
## Chose
Require >=2 shared significant words (word-set, not substring) for a lesson to attach to a task
## Rejected
Keep single-word substring match; or weight/rank by proportion of overlap
## Because
insight.go lessonMatchesTask used strings.Contains(hay, w) on the lesson's raw text and returned on the FIRST shared task word, so a task word 'port' hit 'report' and one common word painted every paragraph-length lesson onto every task. Exact word-set intersection with a minLessonOverlap=2 bar is the minimal honest fix: measured on the real workspace it drops the lesson-x-task match rate from 29.9% to 3.8% while keeping the hint low-cost (a spurious hint is one ignorable line). A proportional ranking is P5 (PROPOSALS) and unwarranted until evidence the 2-word bar misranks.
