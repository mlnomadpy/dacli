ALTER TABLE controlplane_tenant_audit_events
    ADD COLUMN action_digest BYTEA,
    ADD COLUMN result TEXT NOT NULL DEFAULT 'succeeded',
    ADD COLUMN reason TEXT NOT NULL DEFAULT 'legacy_success';

-- Existing rows predate exact-action identities. Their immutable after-state
-- digest is the strongest durable legacy binding and is never recomputed.
ALTER TABLE controlplane_tenant_audit_events
    DISABLE TRIGGER controlplane_tenant_audit_append_only;
UPDATE controlplane_tenant_audit_events
SET action_digest = after_digest
WHERE action_digest IS NULL;
ALTER TABLE controlplane_tenant_audit_events
    ENABLE TRIGGER controlplane_tenant_audit_append_only;

ALTER TABLE controlplane_tenant_audit_events
    ALTER COLUMN action_digest SET NOT NULL,
    ADD CONSTRAINT controlplane_tenant_audit_action_digest_size
        CHECK (octet_length(action_digest) = 32),
    ADD CONSTRAINT controlplane_tenant_audit_result_check
        CHECK (result IN ('succeeded', 'refused', 'conflict', 'failed')),
    ADD CONSTRAINT controlplane_tenant_audit_reason_size
        CHECK (char_length(reason) BETWEEN 1 AND 64);

ALTER TABLE controlplane_tenant_audit_events
    ALTER COLUMN result DROP DEFAULT,
    ALTER COLUMN reason DROP DEFAULT;
