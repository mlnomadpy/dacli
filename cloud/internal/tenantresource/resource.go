// Package tenantresource coordinates invitation, project, environment, and
// assignment workflows without coupling authorization to SQL or HTTP.
package tenantresource

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

const minimumInvitationTokenBytes = 32

var ErrDenied = errors.New("tenant resource operation denied")

type Mutation struct {
	Actor                     tenant.AccountID
	Device                    tenant.DeviceID
	ExpectedMembershipVersion tenant.Version
	CorrelationID             string
	OccurredAt                time.Time
}

type Store interface {
	IssueInvitation(context.Context, tenant.Scope, tenant.Mutation, tenant.Invitation) error
	AcceptInvitation(context.Context, tenant.Scope, tenant.Mutation, [32]byte, tenant.AccountID, tenant.Version) (tenant.Membership, error)
	TransitionInvitation(context.Context, tenant.Scope, tenant.Mutation, tenant.InvitationID, tenant.Version, tenant.InvitationState) error
	Project(context.Context, tenant.Scope, tenant.ProjectID) (tenant.Project, error)
	CreateProject(context.Context, tenant.Scope, tenant.Mutation, tenant.Project) error
	UpdateProject(context.Context, tenant.Scope, tenant.Mutation, tenant.Version, tenant.Project) error
	Environment(context.Context, tenant.Scope, tenant.EnvironmentID) (tenant.Environment, error)
	CreateEnvironment(context.Context, tenant.Scope, tenant.Mutation, tenant.Environment) error
	UpdateEnvironment(context.Context, tenant.Scope, tenant.Mutation, tenant.Version, tenant.Environment) error
	CreateProjectAssignment(context.Context, tenant.Scope, tenant.Mutation, tenant.ProjectAssignment) error
	UpdateProjectAssignment(context.Context, tenant.Scope, tenant.Mutation, tenant.Version, tenant.ProjectAssignment) error
	CreateEnvironmentAssignment(context.Context, tenant.Scope, tenant.Mutation, tenant.EnvironmentAssignment) error
	UpdateEnvironmentAssignment(context.Context, tenant.Scope, tenant.Mutation, tenant.Version, tenant.EnvironmentAssignment) error
}

type Manager struct {
	store      Store
	authorizer *tenant.Authorizer
	now        func() time.Time
}

// AuthorizeVerified reloads membership state for an already authenticated
// identity. Callers may use it before read-only repository or cache access.
func (m *Manager) AuthorizeVerified(ctx context.Context, identity tenant.VerifiedIdentity, permission tenant.Permission) error {
	if err := tenant.ValidateVerifiedIdentity(identity); err != nil {
		return ErrDenied
	}
	return m.authorize(ctx, identity.Scope, Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion}, permission)
}

func New(store Store, memberships tenant.MembershipSource, now func() time.Time) (*Manager, error) {
	if store == nil {
		return nil, errors.New("tenant resource manager requires a store")
	}
	authorizer, err := tenant.NewAuthorizer(memberships, now)
	if err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	return &Manager{store: store, authorizer: authorizer, now: now}, nil
}

func (m *Manager) IssueInvitation(ctx context.Context, scope tenant.Scope, mutation Mutation, invitation tenant.Invitation, token []byte) error {
	if len(token) < minimumInvitationTokenBytes || invitation.State != tenant.InvitationPending || invitation.Version != 1 || invitation.ExpiresUnix <= m.now().Unix() {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionMembershipManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	invitation.TokenDigest = sha256.Sum256(token)
	if err := tenant.ValidateInvitation(scope, invitation); err != nil {
		return ErrDenied
	}
	return m.store.IssueInvitation(ctx, scope, auditMutation(mutation), invitation)
}

// AcceptInvitation is authorized by possession of the one-time credential and
// its bound account identity. The invited account does not have a membership
// yet, so applying the ordinary membership authorizer here would be circular.
func (m *Manager) AcceptInvitation(ctx context.Context, scope tenant.Scope, mutation Mutation, account tenant.AccountID, token []byte, expected tenant.Version) (tenant.Membership, error) {
	if len(token) < minimumInvitationTokenBytes || expected == 0 || mutation.Actor != account {
		return tenant.Membership{}, ErrDenied
	}
	mutation.OccurredAt = m.now()
	return m.store.AcceptInvitation(ctx, scope, auditMutation(mutation), sha256.Sum256(token), account, expected)
}

func (m *Manager) CancelInvitation(ctx context.Context, scope tenant.Scope, mutation Mutation, id tenant.InvitationID, expected tenant.Version) error {
	return m.transitionInvitation(ctx, scope, mutation, id, expected, tenant.InvitationCancelled)
}

func (m *Manager) ExpireInvitation(ctx context.Context, scope tenant.Scope, mutation Mutation, id tenant.InvitationID, expected tenant.Version) error {
	return m.transitionInvitation(ctx, scope, mutation, id, expected, tenant.InvitationExpired)
}

func (m *Manager) transitionInvitation(ctx context.Context, scope tenant.Scope, mutation Mutation, id tenant.InvitationID, expected tenant.Version, state tenant.InvitationState) error {
	if expected == 0 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionMembershipManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.TransitionInvitation(ctx, scope, auditMutation(mutation), id, expected, state)
}

func (m *Manager) CreateProject(ctx context.Context, scope tenant.Scope, mutation Mutation, value tenant.Project) error {
	if err := tenant.ValidateProject(scope, value); err != nil || value.State != tenant.LifecycleActive || value.Version != 1 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionProjectManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.CreateProject(ctx, scope, auditMutation(mutation), value)
}

func (m *Manager) UpdateProject(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.Project) error {
	if err := tenant.ValidateProject(scope, value); err != nil || expected == 0 || value.Version != expected+1 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionProjectManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.UpdateProject(ctx, scope, auditMutation(mutation), expected, value)
}

func (m *Manager) ArchiveProject(ctx context.Context, scope tenant.Scope, mutation Mutation, id tenant.ProjectID, expected tenant.Version) error {
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionProjectManage); err != nil {
		return err
	}
	value, err := m.store.Project(ctx, scope, id)
	if err != nil || value.Version != expected || value.State != tenant.LifecycleActive {
		return ErrDenied
	}
	value.State, value.Version = tenant.LifecycleArchived, expected+1
	mutation.OccurredAt = m.now()
	return m.store.UpdateProject(ctx, scope, auditMutation(mutation), expected, value)
}

func (m *Manager) CreateEnvironment(ctx context.Context, scope tenant.Scope, mutation Mutation, value tenant.Environment) error {
	if err := tenant.ValidateEnvironment(scope, value); err != nil || value.State != tenant.LifecycleActive || value.Version != 1 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionEnvironmentManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.CreateEnvironment(ctx, scope, auditMutation(mutation), value)
}

func (m *Manager) UpdateEnvironment(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.Environment) error {
	if err := tenant.ValidateEnvironment(scope, value); err != nil || expected == 0 || value.Version != expected+1 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionEnvironmentManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.UpdateEnvironment(ctx, scope, auditMutation(mutation), expected, value)
}

func (m *Manager) ArchiveEnvironment(ctx context.Context, scope tenant.Scope, mutation Mutation, id tenant.EnvironmentID, expected tenant.Version) error {
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionEnvironmentManage); err != nil {
		return err
	}
	value, err := m.store.Environment(ctx, scope, id)
	if err != nil || value.Version != expected || value.State != tenant.LifecycleActive {
		return ErrDenied
	}
	value.State, value.Version = tenant.LifecycleArchived, expected+1
	mutation.OccurredAt = m.now()
	return m.store.UpdateEnvironment(ctx, scope, auditMutation(mutation), expected, value)
}

func (m *Manager) AssignProject(ctx context.Context, scope tenant.Scope, mutation Mutation, value tenant.ProjectAssignment) error {
	if err := tenant.ValidateProjectAssignment(scope, value); err != nil || value.State != tenant.LifecycleActive || value.Version != 1 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionProjectManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.CreateProjectAssignment(ctx, scope, auditMutation(mutation), value)
}

func (m *Manager) UpdateProjectAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.ProjectAssignment) error {
	if err := tenant.ValidateProjectAssignment(scope, value); err != nil || expected == 0 || value.Version != expected+1 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionProjectManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.UpdateProjectAssignment(ctx, scope, auditMutation(mutation), expected, value)
}

func (m *Manager) AssignEnvironment(ctx context.Context, scope tenant.Scope, mutation Mutation, value tenant.EnvironmentAssignment) error {
	if err := tenant.ValidateEnvironmentAssignment(scope, value); err != nil || value.State != tenant.LifecycleActive || value.Version != 1 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionEnvironmentManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.CreateEnvironmentAssignment(ctx, scope, auditMutation(mutation), value)
}

func (m *Manager) UpdateEnvironmentAssignment(ctx context.Context, scope tenant.Scope, mutation Mutation, expected tenant.Version, value tenant.EnvironmentAssignment) error {
	if err := tenant.ValidateEnvironmentAssignment(scope, value); err != nil || expected == 0 || value.Version != expected+1 {
		return ErrDenied
	}
	if err := m.authorize(ctx, scope, mutation, tenant.PermissionEnvironmentManage); err != nil {
		return err
	}
	mutation.OccurredAt = m.now()
	return m.store.UpdateEnvironmentAssignment(ctx, scope, auditMutation(mutation), expected, value)
}

func (m *Manager) authorize(ctx context.Context, scope tenant.Scope, mutation Mutation, permission tenant.Permission) error {
	_, err := m.authorizer.Authorize(ctx, tenant.Authorization{Scope: scope, Account: mutation.Actor, Permission: permission, ExpectedMembershipVersion: mutation.ExpectedMembershipVersion})
	if err != nil {
		return ErrDenied
	}
	return nil
}

func auditMutation(value Mutation) tenant.Mutation {
	return tenant.Mutation{Actor: value.Actor, Device: value.Device, CorrelationID: value.CorrelationID, OccurredAt: value.OccurredAt}
}
