---
id: f-adopt-provision-roles-seeds-a-role-architect-provisioning-task
kind: note
note_kind: finding
created: 2026-07-22T18:33:05Z
created_by: a-xkktk9s4kk
about: [[067]]
severity: minor
github:
  issue: 17
  repo: mlnomadpy/dacli
---
# adopt --provision-roles seeds a role-architect provisioning task
internal/features/onboard/onboard.go cmdAdopt: new --provision-roles flag seeds ONE 'Provision the team for <project>' task (priority should) whose Context carries the codebase map + languages (reused mapBody) and a directive: analyze stack, decide MINIMAL roster justifying each, per-role 'dacli skill fetch <owner/repo>' + 'dacli role add <name> --kind <k> --grant <g> --model <m> --skills <...>', finish with 'dacli note add decision'. Prints 'next: dacli spawn --task <seq> --role role-architect'. No cross-slice import — SEEDS + PRINTS only. Body uses bold labels not ATX headings so it stays inside the Context section (renderMap trap). go build+vet+gofmt clean; go test ./internal/cli green (TestAdoptExistingRepo: default path unchanged). Branch dacli/067-provision-adopt-provision-roles-seeds-a-team-provisioning-task-for-a-role.
