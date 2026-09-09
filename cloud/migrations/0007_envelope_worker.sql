CREATE TABLE controlplane_envelope_streams (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    producer_key_id TEXT NOT NULL,
    last_sequence BIGINT NOT NULL DEFAULT 0 CHECK (last_sequence >= 0),
    replay_floor BIGINT NOT NULL DEFAULT 1 CHECK (replay_floor > 0),
    minimum_schema_version INTEGER NOT NULL DEFAULT 1 CHECK (minimum_schema_version > 0),
    maximum_schema_version INTEGER NOT NULL DEFAULT 1 CHECK (maximum_schema_version >= minimum_schema_version),
    PRIMARY KEY (tenant_id, project_id, producer_key_id),
    FOREIGN KEY (tenant_id, project_id)
        REFERENCES controlplane_projects (tenant_id, project_id)
);

-- Identity and sequence rows outlive payload retention. They are the durable
-- replay/idempotency boundary and must never be deleted by payload cleanup.
CREATE TABLE controlplane_envelope_inbox_identities (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    producer_key_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    producer_sequence BIGINT NOT NULL CHECK (producer_sequence > 0),
    outcome TEXT NOT NULL CHECK (outcome IN ('accept-delayed', 'accept-reordered')),
    received_unix_milli BIGINT NOT NULL CHECK (received_unix_milli > 0),
    PRIMARY KEY (tenant_id, project_id, event_id),
    UNIQUE (tenant_id, project_id, idempotency_key),
    UNIQUE (tenant_id, project_id, producer_key_id, producer_sequence),
    FOREIGN KEY (tenant_id, project_id, producer_key_id)
        REFERENCES controlplane_envelope_streams (tenant_id, project_id, producer_key_id)
);

CREATE TABLE controlplane_envelope_inbox_payloads (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    envelope JSONB NOT NULL CHECK (jsonb_typeof(envelope) = 'object'),
    expires_unix_milli BIGINT NOT NULL CHECK (expires_unix_milli > 0),
    PRIMARY KEY (tenant_id, project_id, event_id),
    FOREIGN KEY (tenant_id, project_id, event_id)
        REFERENCES controlplane_envelope_inbox_identities (tenant_id, project_id, event_id)
);

CREATE TABLE controlplane_envelope_outbox (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    outbox_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    envelope JSONB NOT NULL CHECK (jsonb_typeof(envelope) = 'object'),
    state TEXT NOT NULL CHECK (state IN ('pending', 'delivering', 'delivered', 'dead-letter')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_unix_milli BIGINT NOT NULL CHECK (next_attempt_unix_milli > 0),
    lease_until_unix_milli BIGINT NOT NULL DEFAULT 0 CHECK (lease_until_unix_milli >= 0),
    last_error_code TEXT NOT NULL DEFAULT '',
    created_unix_milli BIGINT NOT NULL CHECK (created_unix_milli > 0),
    delivered_unix_milli BIGINT NOT NULL DEFAULT 0 CHECK (delivered_unix_milli >= 0),
    PRIMARY KEY (tenant_id, project_id, outbox_id),
    UNIQUE (tenant_id, project_id, idempotency_key),
    FOREIGN KEY (tenant_id, project_id)
        REFERENCES controlplane_projects (tenant_id, project_id)
);

CREATE TABLE controlplane_envelope_audit_events (
    tenant_id TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_device_id TEXT,
    project_id TEXT NOT NULL,
    event_identity TEXT NOT NULL CHECK (char_length(event_identity) = 64),
    decision TEXT NOT NULL CHECK (decision IN ('accept-delayed', 'accept-reordered', 'ignore-duplicate', 'refuse-incompatible', 'refuse-replay', 'refuse-tampered')),
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 64),
    occurred_unix_milli BIGINT NOT NULL CHECK (occurred_unix_milli > 0),
    PRIMARY KEY (tenant_id, correlation_id),
    FOREIGN KEY (tenant_id)
        REFERENCES controlplane_organizations (tenant_id)
);

CREATE FUNCTION controlplane_refuse_envelope_identity_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'envelope replay identities are append-only';
END;
$$;

CREATE TRIGGER controlplane_envelope_identity_append_only
BEFORE UPDATE OR DELETE ON controlplane_envelope_inbox_identities
FOR EACH ROW EXECUTE FUNCTION controlplane_refuse_envelope_identity_mutation();

CREATE FUNCTION controlplane_refuse_envelope_stream_delete() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'envelope replay streams cannot be deleted';
END;
$$;

CREATE TRIGGER controlplane_envelope_stream_no_delete
BEFORE DELETE ON controlplane_envelope_streams
FOR EACH ROW EXECUTE FUNCTION controlplane_refuse_envelope_stream_delete();

CREATE FUNCTION controlplane_refuse_envelope_audit_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'envelope audit events are append-only';
END;
$$;

CREATE TRIGGER controlplane_envelope_audit_append_only
BEFORE UPDATE OR DELETE ON controlplane_envelope_audit_events
FOR EACH ROW EXECUTE FUNCTION controlplane_refuse_envelope_audit_mutation();

CREATE INDEX controlplane_envelope_inbox_sequence
    ON controlplane_envelope_inbox_identities (tenant_id, project_id, producer_key_id, producer_sequence);
CREATE INDEX controlplane_envelope_payload_expiry
    ON controlplane_envelope_inbox_payloads (tenant_id, expires_unix_milli);
CREATE INDEX controlplane_envelope_outbox_due
    ON controlplane_envelope_outbox (tenant_id, state, next_attempt_unix_milli, lease_until_unix_milli);

ALTER TABLE controlplane_envelope_streams ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_streams FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_inbox_identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_inbox_identities FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_inbox_payloads ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_inbox_payloads FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_outbox FORCE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE controlplane_envelope_audit_events FORCE ROW LEVEL SECURITY;

CREATE POLICY controlplane_envelope_streams_tenant ON controlplane_envelope_streams
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_envelope_inbox_identities_tenant ON controlplane_envelope_inbox_identities
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_envelope_inbox_payloads_tenant ON controlplane_envelope_inbox_payloads
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_envelope_outbox_tenant ON controlplane_envelope_outbox
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
CREATE POLICY controlplane_envelope_audit_events_tenant ON controlplane_envelope_audit_events
    USING (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('dacli.tenant_id', true), ''));
