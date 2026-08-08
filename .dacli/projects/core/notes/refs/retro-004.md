---
id: r-retro-004
kind: note
note_kind: ref
created: 2026-07-21T17:56:56Z
created_by: a-root
about: [[t-01KY2J1BRVAJWS8CEERPPNPH0X]]
scope: workspace
---
# Retro: 004
## Went well
- diff -r against the real library is the only lossless test that counts

## Didn't go well
- the fixture passed while the real skill was unreadable — description-pipe literal blocks broke the parser silently

## Improve next time
- every format reader gets one real-world file in its test corpus, not only synthetic fixtures

