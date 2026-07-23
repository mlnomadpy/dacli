---
id: f-mkdocs-build-could-not-be-smoke-tested-locally-sandbox-denies-python3-pip3-gh
kind: note
note_kind: finding
created: 2026-07-23T19:09:35Z
created_by: a-7k5fg7wzv7
about: [[121]]
severity: minor
scope: task
---
# mkdocs build could not be smoke-tested locally: sandbox denies python3/pip3/gh execution
This headless session's permission mode requires approval for python3, pip3, and gh (go/git/dacli run fine) with no human to approve it, so mkdocs-material could not be installed/run here to verify 'mkdocs build' end to end. Verified by hand instead: grep-audited every relative markdown link under docs/*.md (only two pointed outside docs_dir -- docs/README.md:7 and docs/GITHUB.md:201, both '../DESIGN.md' -- rewritten to absolute https://github.com/mlnomadpy/dacli/blob/main/DESIGN.md links since DESIGN.md lives outside docs_dir and isn't in the site nav); confirmed docs/ has no non-markdown assets; confirmed mkdocs.yml's nav lists all 19 docs/*.md files (18 existing + new docs/index.md) so none are orphaned. Owner should run 'pip install mkdocs-material && mkdocs build --strict' once locally/in CI to confirm before fully trusting the Pages deploy. Also note: GitHub Pages must be switched to Source=GitHub Actions once under repo Settings -> Pages -- a one-time manual/admin step this agent has no access to perform.
