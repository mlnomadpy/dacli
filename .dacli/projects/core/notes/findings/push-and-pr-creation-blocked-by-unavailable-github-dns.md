---
id: f-push-and-pr-creation-blocked-by-unavailable-github-dns
kind: note
note_kind: finding
created: 2026-08-27T22:07:51Z
created_by: a-fixer-dqsb6g
about: "[[t-01M12K8SH454ZH3Z1MB1Q3D4TG]]"
severity: moderate
---
# Push and PR creation blocked by unavailable GitHub DNS
After commit 776d6272, /tmp/dacli-main push --task t-01M12K8SH454ZH3Z1MB1Q3D4TG failed: fatal: unable to access https://github.com/mlnomadpy/dacli.git/: Could not resolve host: github.com. No PR was opened and no landing is claimed.
