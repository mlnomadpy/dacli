package tenantrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mlnomadpy/dacli/cloud/internal/devicesession"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

func (r *Repository) Issue(ctx context.Context, scope tenant.Scope, mutation devicesession.Mutation, record devicesession.Record) error {
	return r.mutate(ctx, scope, sessionMutation(mutation), func(tx *sql.Tx) (tenant.AuditEvent, error) {
		_, err := tx.ExecContext(ctx, `INSERT INTO controlplane_device_sessions
(tenant_id, session_id, account_id, device_id, credential_digest, state, version, expires_unix)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, record.Tenant, record.ID, record.Account, record.Device,
			record.CredentialDigest[:], record.State, record.Version, record.ExpiresUnix)
		if err != nil {
			return tenant.AuditEvent{}, fmt.Errorf("insert device session: %w", classifyWrite(err))
		}
		return newAudit(scope, sessionMutation(mutation), tenant.AuditActionCreate, tenant.TargetSession, string(record.ID), 0, record.Version, [32]byte{}, digest(record))
	})
}

func (r *Repository) Lookup(ctx context.Context, scope tenant.Scope, credential [32]byte) (devicesession.Snapshot, error) {
	var value devicesession.Snapshot
	err := r.withTenant(ctx, scope, true, func(tx *sql.Tx) error {
		var roles, credentialBytes, publicKeyBytes []byte
		err := tx.QueryRowContext(ctx, `SELECT s.tenant_id, s.session_id, s.account_id, s.device_id, s.credential_digest, s.state, s.version, s.expires_unix,
d.name, d.platform, d.public_key, d.state, d.version,
COALESCE(m.team_id, ''), m.roles, m.state, m.version, m.expires_unix
FROM controlplane_device_sessions s
JOIN controlplane_devices d ON d.tenant_id = s.tenant_id AND d.device_id = s.device_id
JOIN controlplane_memberships m ON m.tenant_id = s.tenant_id AND m.account_id = s.account_id
WHERE s.tenant_id = $1 AND s.credential_digest = $2`, scope.Organization, credential[:]).Scan(
			&value.Session.Tenant, &value.Session.ID, &value.Session.Account, &value.Session.Device, &credentialBytes,
			&value.Session.State, &value.Session.Version, &value.Session.ExpiresUnix,
			&value.Device.Name, &value.Device.Platform, &publicKeyBytes, &value.Device.State, &value.Device.Version,
			&value.Membership.Team, &roles, &value.Membership.State, &value.Membership.Version, &value.Membership.ExpiresUnix)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read device session: %w", err)
		}
		if len(credentialBytes) != len(value.Session.CredentialDigest) || len(publicKeyBytes) != len(value.Device.PublicKey) {
			return errors.New("stored device session has invalid key material")
		}
		copy(value.Session.CredentialDigest[:], credentialBytes)
		copy(value.Device.PublicKey[:], publicKeyBytes)
		value.Device.Tenant, value.Device.ID, value.Device.Account = scope.Organization, value.Session.Device, value.Session.Account
		value.Membership.Tenant, value.Membership.Account = scope.Organization, value.Session.Account
		var names []string
		if err := json.Unmarshal(roles, &names); err != nil {
			return fmt.Errorf("decode membership roles: %w", err)
		}
		parsed := make([]tenant.Role, 0, len(names))
		for _, name := range names {
			role, err := tenant.ParseRole(name)
			if err != nil {
				return err
			}
			parsed = append(parsed, role)
		}
		value.Membership.Roles, err = tenant.Roles(parsed...)
		return err
	})
	return value, err
}

func (r *Repository) Revoke(ctx context.Context, scope tenant.Scope, mutation devicesession.Mutation, credential [32]byte, expected tenant.Version) error {
	return r.changeSession(ctx, scope, mutation, credential, expected, nil)
}

func (r *Repository) Rotate(ctx context.Context, scope tenant.Scope, mutation devicesession.Mutation, credential [32]byte, expected tenant.Version, next devicesession.Record) error {
	return r.changeSession(ctx, scope, mutation, credential, expected, &next)
}

func (r *Repository) changeSession(ctx context.Context, scope tenant.Scope, mutation devicesession.Mutation, credential [32]byte, expected tenant.Version, next *devicesession.Record) error {
	converted := sessionMutation(mutation)
	if err := validateMutation(converted); err != nil {
		return err
	}
	return r.withTenant(ctx, scope, false, func(tx *sql.Tx) error {
		var current devicesession.Record
		var credentialBytes []byte
		err := tx.QueryRowContext(ctx, `SELECT tenant_id, session_id, account_id, device_id, credential_digest, state, version, expires_unix
FROM controlplane_device_sessions WHERE tenant_id = $1 AND credential_digest = $2`, scope.Organization, credential[:]).Scan(
			&current.Tenant, &current.ID, &current.Account, &current.Device, &credentialBytes, &current.State, &current.Version, &current.ExpiresUnix)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		if len(credentialBytes) != len(current.CredentialDigest) {
			return errors.New("stored device session has invalid credential digest")
		}
		copy(current.CredentialDigest[:], credentialBytes)
		if current.Version != expected || current.State != devicesession.StateActive {
			return ErrConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE controlplane_device_sessions SET state = $3, version = $4
WHERE tenant_id = $1 AND session_id = $2 AND state = $5 AND version = $6`, scope.Organization, current.ID, devicesession.StateRevoked, expected+1, devicesession.StateActive, expected)
		if err != nil {
			return err
		}
		if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			return ErrConflict
		}
		revoked := current
		revoked.State, revoked.Version = devicesession.StateRevoked, expected+1
		revokeAudit, err := newAudit(scope, converted, tenant.AuditActionRevoke, tenant.TargetSession, string(current.ID), expected, expected+1, digest(current), digest(revoked))
		if err != nil {
			return err
		}
		if next == nil {
			return appendAudit(ctx, tx, mutation.CorrelationID, revokeAudit)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO controlplane_device_sessions
(tenant_id, session_id, account_id, device_id, credential_digest, state, version, expires_unix)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, next.Tenant, next.ID, next.Account, next.Device, next.CredentialDigest[:], next.State, next.Version, next.ExpiresUnix); err != nil {
			return classifyWrite(err)
		}
		if err := appendAudit(ctx, tx, mutation.CorrelationID+".revoke", revokeAudit); err != nil {
			return err
		}
		issueAudit, err := newAudit(scope, converted, tenant.AuditActionCreate, tenant.TargetSession, string(next.ID), 0, next.Version, [32]byte{}, digest(*next))
		if err != nil {
			return err
		}
		return appendAudit(ctx, tx, mutation.CorrelationID+".issue", issueAudit)
	})
}

func sessionMutation(value devicesession.Mutation) Mutation {
	return Mutation{Actor: value.Actor, Device: value.Device, CorrelationID: value.CorrelationID, OccurredAt: value.OccurredAt}
}
