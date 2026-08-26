---
id: f-answer-may-the-project-show-landing-policy-task-claim-be-expanded-to-include
kind: note
note_kind: finding
created: 2026-08-26T14:32:30Z
created_by: a-root
about: "[[t-01M0F8DMCN93FCDE59FSEDTJB3]]"
---
# Answer: May the project-show landing-policy task claim be expanded to include internal/features/planning, internal/model, and docs/TRUST.md? The required persistence handler and regression live there; dacli commit refused those paths under the current claim [internal/store, internal/cli].
Q (a-fixer-eqe3tq): May the project-show landing-policy task claim be expanded to include internal/features/planning, internal/model, and docs/TRUST.md? The required persistence handler and regression live there; dacli commit refused those paths under the current claim [internal/store, internal/cli].

A: Approved. Relaunch task 490 with the explicit claim internal/features/planning,internal/model,internal/cli,docs/TRUST.md; keep the change limited to persistence, public regressions, invariant/help parity, and the documented trust contract.
