#!/usr/bin/env bash
# Proves that a supported PostgreSQL snapshot restores without losing the
# migration ledger, tenant isolation, or durable envelope recovery state.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
IMAGE=${DACLI_RECOVERY_POSTGRES_IMAGE:-postgres:17.6-bookworm}
if [[ "$IMAGE" != "postgres:17.6-bookworm" ]]; then
  echo "refusing unreviewed PostgreSQL image: $IMAGE" >&2
  exit 3
fi
for command in docker openssl awk; do
  command -v "$command" >/dev/null || { echo "missing required command: $command" >&2; exit 4; }
done

digest_file() {
  if command -v sha256sum >/dev/null; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    echo "missing required SHA-256 command: sha256sum or shasum" >&2
    exit 4
  fi
}

SCRATCH=$(mktemp -d "${TMPDIR:-/tmp}/dacli-recovery.XXXXXX")
SUFFIX=$(basename "$SCRATCH" | tr -cd 'a-zA-Z0-9')
SOURCE="dacli-recovery-source-$SUFFIX"
RESTORED="dacli-recovery-restored-$SUFFIX"
DATABASE=dacli_recovery
USER=dacli_recovery
APP_ROLE=dacli_recovery_app
PASSWORD=$(openssl rand -hex 24)
BACKUP="$SCRATCH/control-plane.dump"

cleanup() {
  docker rm -f "$SOURCE" "$RESTORED" >/dev/null 2>&1 || true
  rm -rf "$SCRATCH"
}
trap cleanup EXIT INT TERM

start_postgres() {
  local name=$1
  docker run --detach --name "$name" \
    --env "POSTGRES_DB=$DATABASE" \
    --env "POSTGRES_USER=$USER" \
    --env "POSTGRES_PASSWORD=$PASSWORD" \
    "$IMAGE" >/dev/null
  for _ in $(seq 1 60); do
    if docker exec "$name" pg_isready --quiet --username "$USER" --dbname "$DATABASE"; then
      return
    fi
    sleep 1
  done
  echo "PostgreSQL did not become ready: $name" >&2
  exit 1
}

psql_exec() {
  local name=$1
  shift
  docker exec --interactive "$name" psql --no-psqlrc --set ON_ERROR_STOP=1 --username "$USER" --dbname "$DATABASE" "$@"
}

create_application_role() {
  psql_exec "$1" --command "CREATE ROLE $APP_ROLE NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;" >/dev/null
}

start_postgres "$SOURCE"
SOURCE_VERSION=$(psql_exec "$SOURCE" --tuples-only --no-align --command "SHOW server_version;")
[[ "$SOURCE_VERSION" == 17.6* ]] || { echo "unexpected source version: $SOURCE_VERSION" >&2; exit 1; }
create_application_role "$SOURCE"

psql_exec "$SOURCE" --command "CREATE TABLE controlplane_schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP);" >/dev/null
MIGRATION_COUNT=0
for migration in "$ROOT"/cloud/migrations/*.sql; do
  filename=$(basename "$migration")
  version=${filename%%_*}
  name=${filename#*_}
  name=${name%.sql}
  checksum=$(digest_file "$migration")
  psql_exec "$SOURCE" <"$migration" >/dev/null
  psql_exec "$SOURCE" --command "INSERT INTO controlplane_schema_migrations(version, name, checksum) VALUES (${version#0}, '$name', '$checksum');" >/dev/null
  ((MIGRATION_COUNT += 1))
done

psql_exec "$SOURCE" <<'SQL' >/dev/null
INSERT INTO controlplane_accounts(account_id, name, state, version)
VALUES ('account-a', 'Account A', 1, 1), ('account-b', 'Account B', 1, 1);
INSERT INTO controlplane_organizations(tenant_id, organization_id, name, state, version)
VALUES ('tenant-a', 'tenant-a', 'Tenant A', 1, 1), ('tenant-b', 'tenant-b', 'Tenant B', 1, 1);
INSERT INTO controlplane_memberships(tenant_id, account_id, roles, state, version)
VALUES ('tenant-a', 'account-a', '["owner"]', 1, 1), ('tenant-b', 'account-b', '["owner"]', 1, 1);
INSERT INTO controlplane_projects(tenant_id, project_id, name, state, version)
VALUES ('tenant-a', 'project-a', 'Project A', 1, 1), ('tenant-b', 'project-b', 'Project B', 1, 1);
INSERT INTO controlplane_envelope_streams(tenant_id, project_id, producer_key_id, last_sequence, replay_floor)
VALUES ('tenant-a', 'project-a', 'key-a', 7, 3), ('tenant-b', 'project-b', 'key-b', 11, 5);
INSERT INTO controlplane_envelope_inbox_identities(tenant_id, project_id, producer_key_id, event_id, idempotency_key, producer_sequence, outcome, received_unix_milli)
VALUES ('tenant-a', 'project-a', 'key-a', 'event-a', 'idem-a', 7, 'accept-delayed', 1000),
       ('tenant-b', 'project-b', 'key-b', 'event-b', 'idem-b', 11, 'accept-delayed', 1000);
INSERT INTO controlplane_envelope_inbox_payloads(tenant_id, project_id, event_id, envelope, expires_unix_milli)
VALUES ('tenant-a', 'project-a', 'event-a', '{}', 9000000), ('tenant-b', 'project-b', 'event-b', '{}', 9000000);
INSERT INTO controlplane_envelope_outbox(tenant_id, project_id, outbox_id, idempotency_key, envelope, state, attempts, next_attempt_unix_milli, lease_until_unix_milli, last_error_code, created_unix_milli)
VALUES ('tenant-a', 'project-a', 'outbox-a', 'outbound-a', '{}', 'dead-letter', 4, 2000, 0, 'provider_refused', 1000);
INSERT INTO controlplane_tenant_audit_events(tenant_id, correlation_id, actor_id, action, target_kind, target_id, version_before, version_after, before_digest, after_digest, occurred_unix_milli, action_digest, result, reason)
VALUES ('tenant-a', 'audit-a', 'account-a', 1, 1, 'project-a', 0, 1, decode(repeat('00', 32), 'hex'), decode(repeat('01', 32), 'hex'), 1000, decode(repeat('02', 32), 'hex'), 'succeeded', 'committed');
INSERT INTO controlplane_envelope_audit_events(tenant_id, correlation_id, actor_id, project_id, event_identity, decision, reason, occurred_unix_milli)
VALUES ('tenant-a', 'envelope-audit-a', 'account-a', 'project-a', repeat('a', 64), 'accept-delayed', 'accepted', 1000);
GRANT USAGE ON SCHEMA public TO dacli_recovery_app;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO dacli_recovery_app;
SQL

SOURCE_LEDGER=$(psql_exec "$SOURCE" --tuples-only --no-align --command "SELECT version || ':' || name || ':' || checksum FROM controlplane_schema_migrations ORDER BY version;")
RECOVERY_STATE_SQL="
SELECT 'project:' || tenant_id || ':' || project_id || ':' || version FROM controlplane_projects
UNION ALL SELECT 'stream:' || tenant_id || ':' || project_id || ':' || producer_key_id || ':' || last_sequence || ':' || replay_floor || ':' || minimum_schema_version || ':' || maximum_schema_version FROM controlplane_envelope_streams
UNION ALL SELECT 'identity:' || tenant_id || ':' || project_id || ':' || producer_key_id || ':' || event_id || ':' || idempotency_key || ':' || producer_sequence FROM controlplane_envelope_inbox_identities
UNION ALL SELECT 'outbox:' || tenant_id || ':' || project_id || ':' || outbox_id || ':' || idempotency_key || ':' || state || ':' || attempts || ':' || lease_until_unix_milli || ':' || last_error_code FROM controlplane_envelope_outbox
UNION ALL SELECT 'tenant-audit:' || tenant_id || ':' || correlation_id || ':' || encode(action_digest, 'hex') || ':' || result || ':' || reason FROM controlplane_tenant_audit_events
UNION ALL SELECT 'envelope-audit:' || tenant_id || ':' || correlation_id || ':' || event_identity || ':' || decision || ':' || reason FROM controlplane_envelope_audit_events
ORDER BY 1;"
SOURCE_RECOVERY_STATE=$(psql_exec "$SOURCE" --tuples-only --no-align --command "$RECOVERY_STATE_SQL")

# pg_dump takes one transactionally consistent database snapshot. Global roles,
# credentials, and runtime configuration are deliberately outside this backup.
docker exec "$SOURCE" pg_dump --format=custom --no-owner --username "$USER" --dbname "$DATABASE" >"$BACKUP"
[[ -s "$BACKUP" ]] || { echo "pg_dump produced an empty artifact" >&2; exit 1; }

start_postgres "$RESTORED"
RESTORED_VERSION=$(psql_exec "$RESTORED" --tuples-only --no-align --command "SHOW server_version;")
[[ "$RESTORED_VERSION" == 17.6* ]] || { echo "unexpected restore version: $RESTORED_VERSION" >&2; exit 1; }
create_application_role "$RESTORED"
docker exec --interactive "$RESTORED" pg_restore --exit-on-error --no-owner --username "$USER" --dbname "$DATABASE" <"$BACKUP" >/dev/null

RESTORED_LEDGER=$(psql_exec "$RESTORED" --tuples-only --no-align --command "SELECT version || ':' || name || ':' || checksum FROM controlplane_schema_migrations ORDER BY version;")
RESTORED_RECOVERY_STATE=$(psql_exec "$RESTORED" --tuples-only --no-align --command "$RECOVERY_STATE_SQL")
[[ "$RESTORED_LEDGER" == "$SOURCE_LEDGER" ]] || { echo "migration ledger changed during restore" >&2; exit 1; }
[[ "$RESTORED_RECOVERY_STATE" == "$SOURCE_RECOVERY_STATE" ]] || { echo "durable replay state changed during restore" >&2; exit 1; }

rls_count() {
  local tenant=$1
  psql_exec "$RESTORED" --tuples-only --no-align --command "BEGIN; SET LOCAL ROLE $APP_ROLE; SELECT set_config('dacli.tenant_id', '$tenant', true); SELECT count(*) FROM controlplane_projects; COMMIT;" | awk '/^[0-9]+$/{value=$0} END{print value}'
}
[[ "$(rls_count tenant-a)" == "1" ]] || { echo "tenant-a RLS smoke check failed" >&2; exit 1; }
[[ "$(rls_count tenant-b)" == "1" ]] || { echo "tenant-b RLS smoke check failed" >&2; exit 1; }

ABSENT_COUNT=$(psql_exec "$RESTORED" --tuples-only --no-align --command "BEGIN; SET LOCAL ROLE $APP_ROLE; SELECT count(*) FROM controlplane_projects; COMMIT;" | awk '/^[0-9]+$/{value=$0} END{print value}')
[[ "$ABSENT_COUNT" == "0" ]] || { echo "missing tenant scope exposed rows" >&2; exit 1; }

echo "control-plane recovery smoke passed: PostgreSQL $RESTORED_VERSION, $MIGRATION_COUNT migrations, two isolated tenants"
