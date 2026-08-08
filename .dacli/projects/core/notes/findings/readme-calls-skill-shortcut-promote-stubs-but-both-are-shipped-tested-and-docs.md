---
id: f-readme-calls-skill-shortcut-promote-stubs-but-both-are-shipped-tested-and-docs
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-go-auditor-ag9yhz
about: "[[t-01KZ6SBRBME7WQMTSBRHS55QCS]]"
source_event: 01KZ6SKMZPT0VEBV9MVPPM9J4T
---
# README calls skill/shortcut promote stubs, but both are shipped, tested, and docs say so
README.md:85 (Status) says 'Two commands are still honest stubs that refuse with an explanation: skill promote and shortcut promote' and README.md:278 (Command reference) says 'Still stubbed (each refuses with an explanation): dacli skill promote, dacli shortcut promote'. Both claims are false against today's code. skill promote is fully implemented at internal/features/skillforge/skillforge.go:86-138 (root-gate, lesson lookup, writes a versioned skill.md, prints 'promoted lesson ... → skill ...') and covered by skillforge_test.go:138-147. shortcut promote is fully implemented at internal/features/shortcuts/shortcuts.go:cmdPromote (counts repeats, writes a real shortcut, prints 'promoted ad-hoc command (N runs) → shortcut') and covered by shortcuts_test.go:148. clikit.Planned() (clikit.go:93) — the honest-stub helper — has ZERO callers anywhere in the tree. The test comment cli/supervise_test.go:187 states outright: 'shortcut promote is now real (no more planned stubs left)'. The README even CONTRADICTS its own sibling docs on the same tree: docs/README.md:30 'no planned() stubs remain in product code', docs/SKILLS.md:3 'skill promote ... v1 implemented', docs/SHORTCUTS.md:3 'shortcut promote ... implemented'. Failure scenario: an agent trusting the README's command reference refuses to invoke skill promote / shortcut promote believing they refuse-as-unimplemented, or an auditor files a task to 'implement the remaining stubs', churning already-shipped tested code — exactly the 'misled by a claim that was true three months ago' failure this task exists to prevent.
