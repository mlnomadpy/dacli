package tenantrepo

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

func (r *Repository) IssueInvitation(ctx context.Context, scope tenant.Scope, mutation Mutation, value tenant.Invitation) error {
	if err := tenant.ValidateInvitation(scope, value); err != nil {
		return err
	}
	if value.Version != 1 || value.State != tenant.InvitationPending {
		return errors.New("new invitation must be pending at version 1")
	}
	roles, err := encodeRoles(value.Roles)
	if err != nil {
		return err
	}
	return r.mutate(ctx, scope, mutation, func(tx *sql.Tx) (tenant.AuditEvent, error) {
		_, err := tx.ExecContext(ctx, `INSERT INTO controlplane_invitations
(tenant_id, invitation_id, account_id, team_id, roles, token_digest, state, version, expires_unix)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9)`, value.Tenant, value.ID, value.Account, value.Team,
			roles, value.TokenDigest[:], value.State, value.Version, value.ExpiresUnix)
		if err != nil {
			return tenant.AuditEvent{}, fmt.Errorf("insert invitation: %w", classifyWrite(err))
		}
		return newAudit(scope, mutation, tenant.AuditActionCreate, tenant.TargetInvitation, string(value.ID), 0, value.Version, [32]byte{}, digest(value))
	})
}

func (r *Repository) AcceptInvitation(ctx context.Context, scope tenant.Scope, mutation Mutation, token [32]byte, account tenant.AccountID, expected tenant.Version) (tenant.Membership, error) {
	var membership tenant.Membership
	err := r.withTenant(ctx, scope, false, func(tx *sql.Tx) error {
		if err := validateMutation(mutation); err != nil {
			return err
		}
		var invitation tenant.Invitation
		var tokenBytes, rolesJSON []byte
		err := tx.QueryRowContext(ctx, `SELECT tenant_id, invitation_id, account_id, COALESCE(team_id, ''), roles, token_digest, state, version, expires_unix
FROM controlplane_invitations WHERE tenant_id = $1 AND token_digest = $2 FOR UPDATE`, scope.Organization, token[:]).Scan(
			&invitation.Tenant, &invitation.ID, &invitation.Account, &invitation.Team, &rolesJSON, &tokenBytes,
			&invitation.State, &invitation.Version, &invitation.ExpiresUnix)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return fmt.Errorf("read invitation: %w", err)
		}
		if len(tokenBytes) != len(invitation.TokenDigest) {
			return errors.New("stored invitation has invalid token digest")
		}
		copy(invitation.TokenDigest[:], tokenBytes)
		roles, err := decodeRoles(rolesJSON)
		if err != nil {
			return err
		}
		invitation.Roles = roles
		if err := tenant.ValidateInvitation(scope, invitation); err != nil ||
			invitation.Account != account || invitation.State != tenant.InvitationPending || invitation.Version != expected ||
			invitation.ExpiresUnix <= mutation.OccurredAt.Unix() || subtle.ConstantTimeCompare(token[:], invitation.TokenDigest[:]) != 1 {
			return ErrConflict
		}
		accepted := invitation
		accepted.State, accepted.Version = tenant.InvitationAccepted, expected+1
		result, err := tx.ExecContext(ctx, `UPDATE controlplane_invitations SET state = $3, version = $4
WHERE tenant_id = $1 AND invitation_id = $2 AND state = $5 AND version = $6`, scope.Organization, invitation.ID,
			accepted.State, accepted.Version, tenant.InvitationPending, expected)
		if err != nil {
			return fmt.Errorf("accept invitation: %w", err)
		}
		if err := requireOne(result); err != nil {
			return err
		}
		membership = tenant.Membership{Tenant: scope.Organization, Account: account, Team: invitation.Team, Roles: invitation.Roles, State: tenant.LifecycleActive, Version: 1}
		membershipRoles, err := encodeRoles(membership.Roles)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO controlplane_memberships
(tenant_id, account_id, team_id, roles, state, version, expires_unix)
VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7)`, membership.Tenant, membership.Account, membership.Team,
			membershipRoles, membership.State, membership.Version, membership.ExpiresUnix); err != nil {
			return fmt.Errorf("create invited membership: %w", classifyWrite(err))
		}
		acceptedAudit, err := newAudit(scope, mutation, tenant.AuditActionAssign, tenant.TargetInvitation, string(invitation.ID), invitation.Version, accepted.Version, digest(invitation), digest(accepted))
		if err != nil {
			return err
		}
		if err := appendAudit(ctx, tx, mutation.CorrelationID+".invitation", acceptedAudit); err != nil {
			return err
		}
		membershipAudit, err := newAudit(scope, mutation, tenant.AuditActionCreate, tenant.TargetMembership, string(account), 0, membership.Version, [32]byte{}, digest(membership))
		if err != nil {
			return err
		}
		return appendAudit(ctx, tx, mutation.CorrelationID+".membership", membershipAudit)
	})
	return membership, err
}

func (r *Repository) TransitionInvitation(ctx context.Context, scope tenant.Scope, mutation Mutation, id tenant.InvitationID, expected tenant.Version, state tenant.InvitationState) error {
	if state != tenant.InvitationCancelled && state != tenant.InvitationExpired {
		return errors.New("invitation transition must cancel or expire")
	}
	return r.mutate(ctx, scope, mutation, func(tx *sql.Tx) (tenant.AuditEvent, error) {
		before, err := readInvitation(ctx, tx, scope, id)
		if errors.Is(err, ErrNotFound) {
			return tenant.AuditEvent{}, ErrConflict
		}
		if err != nil {
			return tenant.AuditEvent{}, err
		}
		if before.Version != expected || before.State != tenant.InvitationPending {
			return tenant.AuditEvent{}, ErrConflict
		}
		if state == tenant.InvitationExpired && before.ExpiresUnix > mutation.OccurredAt.Unix() {
			return tenant.AuditEvent{}, ErrConflict
		}
		after := before
		after.State, after.Version = state, expected+1
		result, err := tx.ExecContext(ctx, `UPDATE controlplane_invitations SET state = $3, version = $4
WHERE tenant_id = $1 AND invitation_id = $2 AND state = $5 AND version = $6`, scope.Organization, id, state, after.Version, tenant.InvitationPending, expected)
		if err != nil {
			return tenant.AuditEvent{}, fmt.Errorf("transition invitation: %w", err)
		}
		if err := requireOne(result); err != nil {
			return tenant.AuditEvent{}, err
		}
		action := tenant.AuditActionRevoke
		if state == tenant.InvitationExpired {
			action = tenant.AuditActionArchive
		}
		return newAudit(scope, mutation, action, tenant.TargetInvitation, string(id), before.Version, after.Version, digest(before), digest(after))
	})
}

func (r *Repository) Environment(ctx context.Context, scope tenant.Scope, id tenant.EnvironmentID) (tenant.Environment, error) {
	var value tenant.Environment
	err := r.withTenant(ctx, scope, true, func(tx *sql.Tx) error {
		return scanEnvironment(tx.QueryRowContext(ctx, `SELECT tenant_id, project_id, environment_id, name, kind, state, version
FROM controlplane_environments WHERE tenant_id = $1 AND environment_id = $2`, scope.Organization, id), &value)
	})
	return value, err
}

func (r *Repository) CreateEnvironment(ctx context.Context, scope tenant.Scope, mutation Mutation, value tenant.Environment) error {
	if err := tenant.ValidateEnvironment(scope, value); err != nil {
		return err
	}
	if value.Version != 1 || value.State != tenant.LifecycleActive {
		return errors.New("new environment must be active at version 1")
	}
	return r.mutate(ctx, scope, mutation, func(tx *sql.Tx) (tenant.AuditEvent, error) {
		_, err := tx.ExecContext(ctx, `INSERT INTO controlplane_environments
(tenant_id, project_id, environment_id, name, kind, state, version) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			value.Tenant, value.Project, value.ID, value.Name, value.Kind, value.State, value.Version)
		if err != nil {
			return tenant.AuditEvent{}, fmt.Errorf("insert environment: %w", classifyWrite(err))
		}
		return newAudit(scope, mutation, tenant.AuditActionCreate, tenant.TargetEnvironment, string(value.ID), 0, value.Version, [32]byte{}, digest(value))
	})
}

func (r *Repository) UpdateEnvironment(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.Environment) error {
	if err := tenant.ValidateEnvironment(scope, value); err != nil {
		return err
	}
	if expected == 0 || value.Version != expected+1 {
		return errors.New("environment update must advance the expected version exactly once")
	}
	return r.mutate(ctx, scope, mutation, func(tx *sql.Tx) (tenant.AuditEvent, error) {
		before, err := readEnvironment(ctx, tx, scope, value.ID)
		if err != nil {
			return tenant.AuditEvent{}, err
		}
		if before.Version != expected || before.Project != value.Project {
			return tenant.AuditEvent{}, ErrConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE controlplane_environments SET name = $3, kind = $4, state = $5, version = $6
WHERE tenant_id = $1 AND environment_id = $2 AND version = $7`, scope.Organization, value.ID, value.Name, value.Kind, value.State, value.Version, expected)
		if err != nil {
			return tenant.AuditEvent{}, fmt.Errorf("update environment: %w", err)
		}
		if err := requireOne(result); err != nil {
			return tenant.AuditEvent{}, err
		}
		return newAudit(scope, mutation, lifecycleAuditAction(before.State, value.State), tenant.TargetEnvironment, string(value.ID), before.Version, value.Version, digest(before), digest(value))
	})
}

func (r *Repository) CreateProjectAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, value tenant.ProjectAssignment) error {
	if err := tenant.ValidateProjectAssignment(scope, value); err != nil {
		return err
	}
	if value.Version != 1 || value.State != tenant.LifecycleActive {
		return errors.New("new project assignment must be active at version 1")
	}
	return r.createAssignment(ctx, scope, mutation, `INSERT INTO controlplane_project_assignments
(tenant_id, assignment_id, project_id, account_id, state, version) VALUES ($1, $2, $3, $4, $5, $6)`,
		[]any{value.Tenant, value.ID, value.Project, value.Account, value.State, value.Version}, tenant.TargetProjectAssignment, string(value.ID), digest(value))
}

func (r *Repository) UpdateProjectAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.ProjectAssignment) error {
	if err := tenant.ValidateProjectAssignment(scope, value); err != nil {
		return err
	}
	if expected == 0 || value.Version != expected+1 {
		return errors.New("project assignment update must advance the expected version exactly once")
	}
	return r.updateProjectAssignment(ctx, scope, mutation, expected, value)
}

func (r *Repository) CreateEnvironmentAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, value tenant.EnvironmentAssignment) error {
	if err := tenant.ValidateEnvironmentAssignment(scope, value); err != nil {
		return err
	}
	if value.Version != 1 || value.State != tenant.LifecycleActive {
		return errors.New("new environment assignment must be active at version 1")
	}
	return r.createAssignment(ctx, scope, mutation, `INSERT INTO controlplane_environment_assignments
(tenant_id, assignment_id, project_id, environment_id, account_id, state, version) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		[]any{value.Tenant, value.ID, value.Project, value.Environment, value.Account, value.State, value.Version}, tenant.TargetEnvironmentAssignment, string(value.ID), digest(value))
}

func (r *Repository) UpdateEnvironmentAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.EnvironmentAssignment) error {
	if err := tenant.ValidateEnvironmentAssignment(scope, value); err != nil {
		return err
	}
	if expected == 0 || value.Version != expected+1 {
		return errors.New("environment assignment update must advance the expected version exactly once")
	}
	return r.updateEnvironmentAssignment(ctx, scope, mutation, expected, value)
}

func (r *Repository) createAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, query string, args []any, kind tenant.TargetKind, id string, after [32]byte) error {
	return r.mutate(ctx, scope, mutation, func(tx *sql.Tx) (tenant.AuditEvent, error) {
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return tenant.AuditEvent{}, fmt.Errorf("insert assignment: %w", classifyWrite(err))
		}
		return newAudit(scope, mutation, tenant.AuditActionAssign, kind, id, 0, 1, [32]byte{}, after)
	})
}

func (r *Repository) updateProjectAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.ProjectAssignment) error {
	return r.mutate(ctx, scope, mutation, func(tx *sql.Tx) (tenant.AuditEvent, error) {
		var before tenant.ProjectAssignment
		err := tx.QueryRowContext(ctx, `SELECT tenant_id, assignment_id, project_id, account_id, state, version
FROM controlplane_project_assignments WHERE tenant_id = $1 AND assignment_id = $2`, scope.Organization, value.ID).
			Scan(&before.Tenant, &before.ID, &before.Project, &before.Account, &before.State, &before.Version)
		if errors.Is(err, sql.ErrNoRows) {
			return tenant.AuditEvent{}, ErrConflict
		}
		if err != nil {
			return tenant.AuditEvent{}, err
		}
		if before.Version != expected || before.Project != value.Project || before.Account != value.Account {
			return tenant.AuditEvent{}, ErrConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE controlplane_project_assignments SET state = $3, version = $4
WHERE tenant_id = $1 AND assignment_id = $2 AND version = $5`, scope.Organization, value.ID, value.State, value.Version, expected)
		if err != nil {
			return tenant.AuditEvent{}, err
		}
		if err := requireOne(result); err != nil {
			return tenant.AuditEvent{}, err
		}
		return newAudit(scope, mutation, lifecycleAuditAction(before.State, value.State), tenant.TargetProjectAssignment, string(value.ID), before.Version, value.Version, digest(before), digest(value))
	})
}

func (r *Repository) updateEnvironmentAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.EnvironmentAssignment) error {
	return r.mutate(ctx, scope, mutation, func(tx *sql.Tx) (tenant.AuditEvent, error) {
		var before tenant.EnvironmentAssignment
		err := tx.QueryRowContext(ctx, `SELECT tenant_id, assignment_id, project_id, environment_id, account_id, state, version
FROM controlplane_environment_assignments WHERE tenant_id = $1 AND assignment_id = $2`, scope.Organization, value.ID).
			Scan(&before.Tenant, &before.ID, &before.Project, &before.Environment, &before.Account, &before.State, &before.Version)
		if errors.Is(err, sql.ErrNoRows) {
			return tenant.AuditEvent{}, ErrConflict
		}
		if err != nil {
			return tenant.AuditEvent{}, err
		}
		if before.Version != expected || before.Project != value.Project || before.Environment != value.Environment || before.Account != value.Account {
			return tenant.AuditEvent{}, ErrConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE controlplane_environment_assignments SET state = $3, version = $4
WHERE tenant_id = $1 AND assignment_id = $2 AND version = $5`, scope.Organization, value.ID, value.State, value.Version, expected)
		if err != nil {
			return tenant.AuditEvent{}, err
		}
		if err := requireOne(result); err != nil {
			return tenant.AuditEvent{}, err
		}
		return newAudit(scope, mutation, lifecycleAuditAction(before.State, value.State), tenant.TargetEnvironmentAssignment, string(value.ID), before.Version, value.Version, digest(before), digest(value))
	})
}

func readInvitation(ctx context.Context, tx *sql.Tx, scope tenant.Scope, id tenant.InvitationID) (tenant.Invitation, error) {
	var value tenant.Invitation
	var rolesJSON, tokenBytes []byte
	err := tx.QueryRowContext(ctx, `SELECT tenant_id, invitation_id, account_id, COALESCE(team_id, ''), roles, token_digest, state, version, expires_unix
FROM controlplane_invitations WHERE tenant_id = $1 AND invitation_id = $2`, scope.Organization, id).Scan(
		&value.Tenant, &value.ID, &value.Account, &value.Team, &rolesJSON, &tokenBytes, &value.State, &value.Version, &value.ExpiresUnix)
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	if err != nil {
		return value, err
	}
	if len(tokenBytes) != len(value.TokenDigest) {
		return value, errors.New("stored invitation has invalid token digest")
	}
	copy(value.TokenDigest[:], tokenBytes)
	value.Roles, err = decodeRoles(rolesJSON)
	return value, err
}

func readEnvironment(ctx context.Context, tx *sql.Tx, scope tenant.Scope, id tenant.EnvironmentID) (tenant.Environment, error) {
	var value tenant.Environment
	err := scanEnvironment(tx.QueryRowContext(ctx, `SELECT tenant_id, project_id, environment_id, name, kind, state, version
FROM controlplane_environments WHERE tenant_id = $1 AND environment_id = $2`, scope.Organization, id), &value)
	return value, err
}

func scanEnvironment(row rowScanner, value *tenant.Environment) error {
	if err := row.Scan(&value.Tenant, &value.Project, &value.ID, &value.Name, &value.Kind, &value.State, &value.Version); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("read environment: %w", err)
	}
	return nil
}

func encodeRoles(set tenant.RoleSet) ([]byte, error) {
	names := make([]string, 0, len(set.Roles()))
	for _, role := range set.Roles() {
		names = append(names, role.String())
	}
	return json.Marshal(names)
}

func decodeRoles(raw []byte) (tenant.RoleSet, error) {
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return 0, fmt.Errorf("decode roles: %w", err)
	}
	roles := make([]tenant.Role, 0, len(names))
	for _, name := range names {
		role, err := tenant.ParseRole(name)
		if err != nil {
			return 0, fmt.Errorf("decode roles: %w", err)
		}
		roles = append(roles, role)
	}
	return tenant.Roles(roles...)
}

func requireOne(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return ErrConflict
	}
	return nil
}

func lifecycleAuditAction(before, after tenant.Lifecycle) tenant.AuditAction {
	if before != tenant.LifecycleArchived && after == tenant.LifecycleArchived {
		return tenant.AuditActionArchive
	}
	if before == tenant.LifecycleArchived && after == tenant.LifecycleActive {
		return tenant.AuditActionRestore
	}
	return tenant.AuditActionUpdate
}
