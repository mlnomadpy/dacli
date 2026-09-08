CREATE TABLE controlplane_accounts (
    account_id TEXT PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0)
);

CREATE TABLE controlplane_organizations (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    PRIMARY KEY (tenant_id, organization_id),
    UNIQUE (tenant_id),
    CONSTRAINT organization_is_tenant CHECK (tenant_id = organization_id)
);

CREATE TABLE controlplane_teams (
    tenant_id TEXT NOT NULL,
    team_id TEXT NOT NULL,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    PRIMARY KEY (tenant_id, team_id),
    FOREIGN KEY (tenant_id)
        REFERENCES controlplane_organizations (tenant_id)
);

CREATE TABLE controlplane_memberships (
    tenant_id TEXT NOT NULL,
    account_id TEXT NOT NULL REFERENCES controlplane_accounts (account_id),
    team_id TEXT,
    roles JSONB NOT NULL CHECK (jsonb_typeof(roles) = 'array'),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    expires_unix BIGINT NOT NULL DEFAULT 0 CHECK (expires_unix >= 0),
    PRIMARY KEY (tenant_id, account_id),
    FOREIGN KEY (tenant_id)
        REFERENCES controlplane_organizations (tenant_id),
    FOREIGN KEY (tenant_id, team_id)
        REFERENCES controlplane_teams (tenant_id, team_id)
);

CREATE TABLE controlplane_devices (
    tenant_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
    platform SMALLINT NOT NULL CHECK (platform BETWEEN 1 AND 3),
    public_key BYTEA NOT NULL CHECK (octet_length(public_key) = 32),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    PRIMARY KEY (tenant_id, device_id),
    FOREIGN KEY (tenant_id, account_id)
        REFERENCES controlplane_memberships (tenant_id, account_id)
);

CREATE TABLE controlplane_projects (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    PRIMARY KEY (tenant_id, project_id),
    FOREIGN KEY (tenant_id)
        REFERENCES controlplane_organizations (tenant_id)
);

CREATE TABLE controlplane_environments (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
    kind SMALLINT NOT NULL CHECK (kind BETWEEN 1 AND 3),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    PRIMARY KEY (tenant_id, environment_id),
    FOREIGN KEY (tenant_id, project_id)
        REFERENCES controlplane_projects (tenant_id, project_id)
);

CREATE TABLE controlplane_tenant_audit_events (
    tenant_id TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_device_id TEXT,
    action SMALLINT NOT NULL CHECK (action BETWEEN 1 AND 7),
    target_kind SMALLINT NOT NULL CHECK (target_kind BETWEEN 1 AND 6),
    target_id TEXT NOT NULL,
    version_before BIGINT NOT NULL CHECK (version_before >= 0),
    version_after BIGINT NOT NULL CHECK (version_after > version_before),
    before_digest BYTEA NOT NULL CHECK (octet_length(before_digest) = 32),
    after_digest BYTEA NOT NULL CHECK (octet_length(after_digest) = 32),
    occurred_unix_milli BIGINT NOT NULL CHECK (occurred_unix_milli > 0),
    PRIMARY KEY (tenant_id, correlation_id),
    FOREIGN KEY (tenant_id)
        REFERENCES controlplane_organizations (tenant_id)
);

CREATE FUNCTION controlplane_refuse_audit_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'tenant audit events are append-only';
END;
$$;

CREATE TRIGGER controlplane_tenant_audit_append_only
BEFORE UPDATE OR DELETE ON controlplane_tenant_audit_events
FOR EACH ROW EXECUTE FUNCTION controlplane_refuse_audit_mutation();

ALTER TABLE controlplane_organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_organizations FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_teams ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_teams FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_memberships FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_devices FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_projects FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_environments ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_environments FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_tenant_audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_tenant_audit_events FORCE ROW LEVEL SECURITY;

CREATE POLICY controlplane_organizations_tenant ON controlplane_organizations
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_teams_tenant ON controlplane_teams
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_memberships_tenant ON controlplane_memberships
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_devices_tenant ON controlplane_devices
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_projects_tenant ON controlplane_projects
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_environments_tenant ON controlplane_environments
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_tenant_audit_events_tenant ON controlplane_tenant_audit_events
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
