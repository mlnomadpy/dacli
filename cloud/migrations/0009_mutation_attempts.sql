ALTER TABLE controlplane_tenant_audit_events
    DROP CONSTRAINT controlplane_tenant_audit_reason_size,
    ADD CONSTRAINT controlplane_tenant_audit_reason_check
        CHECK (reason IN (
            'committed',
            'legacy_success',
            'authorization_denied',
            'invalid_state',
            'resource_unavailable',
            'version_conflict',
            'persistence_failed'
        )),
    ADD CONSTRAINT controlplane_tenant_audit_result_reason_check
        CHECK (
            (result = 'succeeded' AND reason IN ('committed', 'legacy_success')) OR
            (result = 'refused' AND reason IN (
                'authorization_denied', 'invalid_state', 'resource_unavailable'
            )) OR
            (result = 'conflict' AND reason = 'version_conflict') OR
            (result = 'failed' AND reason = 'persistence_failed')
        );
