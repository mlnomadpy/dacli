ALTER TABLE controlplane_tenant_audit_events
    DROP CONSTRAINT controlplane_tenant_audit_events_target_kind_check;
ALTER TABLE controlplane_tenant_audit_events
    ADD CONSTRAINT controlplane_tenant_audit_events_target_kind_check
    CHECK (target_kind BETWEEN 1 AND 9);

CREATE TABLE controlplane_invitations (
    tenant_id TEXT NOT NULL,
    invitation_id TEXT NOT NULL,
    account_id TEXT NOT NULL REFERENCES controlplane_accounts (account_id),
    team_id TEXT,
    roles JSONB NOT NULL CHECK (jsonb_typeof(roles) = 'array'),
    token_digest BYTEA NOT NULL CHECK (octet_length(token_digest) = 32),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    expires_unix BIGINT NOT NULL CHECK (expires_unix > 0),
    PRIMARY KEY (tenant_id, invitation_id),
    UNIQUE (tenant_id, token_digest),
    FOREIGN KEY (tenant_id)
        REFERENCES controlplane_organizations (tenant_id),
    FOREIGN KEY (tenant_id, team_id)
        REFERENCES controlplane_teams (tenant_id, team_id)
);

CREATE TABLE controlplane_project_assignments (
    tenant_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    PRIMARY KEY (tenant_id, assignment_id),
    UNIQUE (tenant_id, project_id, account_id),
    FOREIGN KEY (tenant_id, project_id)
        REFERENCES controlplane_projects (tenant_id, project_id),
    FOREIGN KEY (tenant_id, account_id)
        REFERENCES controlplane_memberships (tenant_id, account_id)
);

ALTER TABLE controlplane_environments
    ADD CONSTRAINT controlplane_environments_tenant_project_environment_key
    UNIQUE (tenant_id, project_id, environment_id);

CREATE TABLE controlplane_environment_assignments (
    tenant_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 4),
    version BIGINT NOT NULL CHECK (version > 0),
    PRIMARY KEY (tenant_id, assignment_id),
    UNIQUE (tenant_id, environment_id, account_id),
    FOREIGN KEY (tenant_id, project_id, environment_id)
        REFERENCES controlplane_environments (tenant_id, project_id, environment_id),
    FOREIGN KEY (tenant_id, account_id)
        REFERENCES controlplane_memberships (tenant_id, account_id)
);

ALTER TABLE controlplane_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_invitations FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_project_assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_project_assignments FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_environment_assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_environment_assignments FORCE ROW LEVEL SECURITY;

CREATE POLICY controlplane_invitations_tenant ON controlplane_invitations
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_project_assignments_tenant ON controlplane_project_assignments
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_environment_assignments_tenant ON controlplane_environment_assignments
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
