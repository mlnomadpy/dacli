# Private pilot GitHub App event bridge

The App bridge is an optional server-side adapter. It does not replace the
local markdown record and it does not use a developer's `gh` session. The
reference package is `internal/githubapp`; deployments supply HTTPS and a
GitHub API client around it.

## Pilot App registration

Create a **private** GitHub App owned by the pilot organization, install it only
on selected repositories, and use the manifest in
[`examples/github-app-manifest.json`](examples/github-app-manifest.json). Its
complete repository permission set is:

| Permission | Access | Why |
|---|---|---|
| Metadata | read | GitHub-mandatory repository identity used in installation/repository reconciliation |
| Issues | read | receive issue and issue-comment proposal events |
| Pull requests | read | receive pull-request proposal events |
| Checks | read and write | upsert the allowlisted dacli status check |

Do not grant Contents, Administration, Actions, Members, Secrets, or any
organization permission. The implemented remote write is only
`POST /repos/{owner}/{repo}/check-runs`; `UpsertCheck` uses the outbox
idempotency key as a remote marker. Reconciliation reads the installation and
its selected repositories. Expanding the adapter to edit issues or pull
requests requires a separate permission and disclosure review.

The server stores the App private key and mints short-lived installation tokens
inside its GitHub client. Neither is an adapter configuration field, proposal,
outbox value, log payload, nor local-client response. Local clients exchange
only signed control-plane envelopes from `contracts/controlplane/v1`.

## Delivery and recovery

The HTTPS handler passes the untouched request to `Bridge.Ingest`. It reads the
raw body once, verifies `X-Hub-Signature-256` with HMAC-SHA256, and only then
parses or persists it. `X-GitHub-Delivery` is the inbox idempotency identity.
A duplicate with identical content is acknowledged without duplicating its
effect; reuse with different content is rejected. The inbox commit precedes
the proposal write, so a retry repairs a crash between the two.

Inbound issue, comment, and pull-request data is reduced to an attributed,
closed proposal record. The package imports no workspace, store, execution, or
feature package and exposes no command callback. Applying a proposal remains a
local owner action under dacli's normal grant rules.

Outbound `StatusUpdate` is a closed metadata structure. It has no fields for
source, prompts, transcripts, environment, secrets, credentials, or raw command
output. An immutable outbox record is acknowledged separately after an
idempotent check upsert. Dispatch re-checks the installation, tenant,
repository, and current `checks:write` grant immediately before the remote
call.

Run reconciliation periodically and after every `installation`,
`installation_repositories`, and `repository` webhook. GitHub's current view is
authoritative for suspension/removal, selected-repository membership, rename,
and permissions; local tenant/project assignment is never expanded from remote
input. This makes retries, redelivery, permission changes, rename/removal, and
out-of-order webhook arrival converge. A suspended or removed installation is
refused both at ingress and immediately before dispatch.
