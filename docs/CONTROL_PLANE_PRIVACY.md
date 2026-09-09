# Control-plane metadata and privacy

**Status: normative for the Phase 1 protocol; the SaaS service is not shipped.**

The machine-readable source is
[`contracts/controlplane/v1/privacy-fields.json`](../contracts/controlplane/v1/privacy-fields.json).
Tests require it to classify every property in every v1 payload schema exactly
once, reject unknown events/fields/retention classes, and remain deny-by-default.

Each classification records five facts: purpose, direction, retention class,
tenant visibility, and whether collection is enabled by default. “Enabled”
means eligible after a user explicitly connects a project; dacli never contacts
a control plane before that connection. Governance downloads and task/approval
proposals are default-off until that product surface is configured.

## Never valid v1 metadata

Source code or diffs, prompt or instruction contents, transcripts, command
stdout/stderr, environment names or values, credentials/tokens/secrets, local
filesystem paths, and arbitrary nested provider payloads have no v1 schema
field. The local client rejects them before signing, and the server must reject
them before persistence. Redaction after upload is not an acceptable control.

Opaque identifiers may correlate records but must not embed local paths,
credentials, email content, or source text. Repository owner/name, default
branch, role/runtime/model, commit IDs, timestamps, and work status are still
sensitive engineering metadata even though they are allowlisted. Tenant access,
retention, export, and deletion rules apply to them.

The pre-API tenant workflow follows the same boundary. Invitations persist an
opaque invitation ID, account/team IDs, closed roles, lifecycle/version,
expiry, and a one-way token digest; the raw token is never serialized or sent
to persistence. Projects, environments, and assignments persist only opaque
relationships, bounded display names, closed kinds, lifecycle, and optimistic
versions. Their Go records and SQL statements are explicit allowlists—there is
no metadata map or catch-all payload column.

HTTP identity tokens, page-cursor signatures, and worker credentials are
transient authorization material, not metadata. They are excluded from JSON
records and structured errors. Cursors expose only already-authorized opaque
tenant/parent identifiers and pagination ordinals inside an authenticated
base64url envelope; they never contain object names, source, or logs.

## Retention and control

The manifest defines pilot defaults, not permission to retain indefinitely.
Organizations may shorten operational-summary retention or disable selected
optional event types before collection. Transport replay floors survive payload
expiry so an old signed event cannot become new again. Legal billing documents
are not control-plane events and need a separate retention policy.

Changing a field from default-off to default-on, broadening visibility,
lengthening retention, or adding a purpose/direction is a disclosure change. It
requires a new reviewed manifest version and migration notice even when the JSON
payload schema itself remains compatible.

## Analytics boundary

Analytics may aggregate allowlisted outcome, timing, gate, runtime/model, and
budget fields. It must show sample counts and uncertainty, enforce minimum
cohorts, and must not rank individual developers. Raw local evidence stays on
the device. Exported aggregates retain links to permitted record identities and
freshness without reconstructing prompts, logs, or code.
