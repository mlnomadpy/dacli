ALTER TABLE controlplane_tenant_audit_events
    DROP CONSTRAINT controlplane_tenant_audit_events_target_kind_check;
ALTER TABLE controlplane_tenant_audit_events
    ADD CONSTRAINT controlplane_tenant_audit_events_target_kind_check
    CHECK (target_kind BETWEEN 1 AND 7);

CREATE TABLE controlplane_device_sessions (
    tenant_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    credential_digest BYTEA NOT NULL CHECK (octet_length(credential_digest) = 32),
    state SMALLINT NOT NULL CHECK (state BETWEEN 1 AND 2),
    version BIGINT NOT NULL CHECK (version > 0),
    expires_unix BIGINT NOT NULL CHECK (expires_unix > 0),
    PRIMARY KEY (tenant_id, session_id),
    UNIQUE (tenant_id, credential_digest),
    FOREIGN KEY (tenant_id, account_id)
        REFERENCES controlplane_memberships (tenant_id, account_id),
    FOREIGN KEY (tenant_id, device_id)
        REFERENCES controlplane_devices (tenant_id, device_id)
);

ALTER TABLE controlplane_device_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_device_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY controlplane_device_sessions_tenant ON controlplane_device_sessions
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
