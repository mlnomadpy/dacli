// Package tenantapi composes verified identities, authorization, scoped
// caching, and the tenant repository for HTTP and worker entry points.
package tenantapi

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantcache"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantrepo"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantresource"
)

var (
	ErrUnavailable = errors.New("tenant resource unavailable")
	ErrConflict    = errors.New("tenant resource version conflict")
)

type Backend interface {
	ListProjects(context.Context, tenant.VerifiedIdentity, tenantrepo.PageRequest) (tenantrepo.Page[tenant.Project], error)
	ListEnvironments(context.Context, tenant.VerifiedIdentity, tenant.ProjectID, tenantrepo.PageRequest) (tenantrepo.Page[tenant.Environment], error)
	Project(context.Context, tenant.VerifiedIdentity, tenant.ProjectID, tenant.Version) (tenant.Project, error)
	CreateProject(context.Context, tenant.VerifiedIdentity, tenant.Mutation, tenant.Project) error
	UpdateProject(context.Context, tenant.VerifiedIdentity, tenant.Mutation, tenant.Version, tenant.Project) error
	CreateEnvironment(context.Context, tenant.VerifiedIdentity, tenant.Mutation, tenant.Environment) error
	UpdateEnvironment(context.Context, tenant.VerifiedIdentity, tenant.Mutation, tenant.Version, tenant.Environment) error
}

type AttemptRecorder interface {
	RecordAttempt(context.Context, tenant.Scope, tenant.Mutation, tenant.AuditEvent) error
}

func (o *Operations) CreateProject(ctx context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, value tenant.Project) error {
	attempt := o.attemptMutation(identity, mutation)
	if identity.PolicyRevision != o.policyRevision || mutation.Actor != identity.Account || mutation.Device != identity.Device {
		return o.recordFailure(ctx, identity, attempt, tenant.AuditActionCreate, tenant.TargetProject, string(value.ID), 0, 1, [32]byte{}, stateDigest(value), tenantresource.ErrDenied)
	}
	change := tenantresource.Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion, CorrelationID: mutation.CorrelationID}
	if err := o.resources.CreateProject(ctx, identity.Scope, change, value); err != nil {
		return o.recordFailure(ctx, identity, attempt, tenant.AuditActionCreate, tenant.TargetProject, string(value.ID), 0, 1, [32]byte{}, stateDigest(value), err)
	}
	o.projects.InvalidateTenant(identity.Scope.Organization)
	return nil
}

type Operations struct {
	repository     *tenantrepo.Repository
	resources      *tenantresource.Manager
	projects       *tenantcache.Cache[tenant.Project]
	policyRevision uint64
	attempts       AttemptRecorder
	now            func() time.Time
}

func New(repository *tenantrepo.Repository, resources *tenantresource.Manager, cache *tenantcache.Cache[tenant.Project], policyRevision uint64) (*Operations, error) {
	if repository == nil || resources == nil || policyRevision == 0 {
		return nil, errors.New("tenant API requires repository and resource manager")
	}
	if cache == nil {
		cache = tenantcache.New[tenant.Project](1_000)
	}
	return &Operations{repository: repository, resources: resources, projects: cache, policyRevision: policyRevision, attempts: repository, now: time.Now}, nil
}

func (o *Operations) ListProjects(ctx context.Context, identity tenant.VerifiedIdentity, request tenantrepo.PageRequest) (tenantrepo.Page[tenant.Project], error) {
	if identity.PolicyRevision != o.policyRevision {
		return tenantrepo.Page[tenant.Project]{}, ErrUnavailable
	}
	if err := o.resources.AuthorizeVerified(ctx, identity, tenant.PermissionProjectRead); err != nil {
		return tenantrepo.Page[tenant.Project]{}, ErrUnavailable
	}
	page, err := o.repository.ListProjects(ctx, identity.Scope, request)
	return page, normalize(err)
}

func (o *Operations) ListEnvironments(ctx context.Context, identity tenant.VerifiedIdentity, project tenant.ProjectID, request tenantrepo.PageRequest) (tenantrepo.Page[tenant.Environment], error) {
	if identity.PolicyRevision != o.policyRevision {
		return tenantrepo.Page[tenant.Environment]{}, ErrUnavailable
	}
	if err := o.resources.AuthorizeVerified(ctx, identity, tenant.PermissionProjectRead); err != nil {
		return tenantrepo.Page[tenant.Environment]{}, ErrUnavailable
	}
	page, err := o.repository.ListEnvironments(ctx, identity.Scope, project, request)
	return page, normalize(err)
}

func (o *Operations) Project(ctx context.Context, identity tenant.VerifiedIdentity, id tenant.ProjectID, version tenant.Version) (tenant.Project, error) {
	if identity.PolicyRevision != o.policyRevision {
		return tenant.Project{}, ErrUnavailable
	}
	if err := o.resources.AuthorizeVerified(ctx, identity, tenant.PermissionProjectRead); err != nil {
		return tenant.Project{}, ErrUnavailable
	}
	key := tenantcache.Key{Tenant: identity.Scope.Organization, Kind: tenantcache.ResourceProject, ResourceID: string(id), Subject: identity.Account, ResourceVersion: version, MembershipVersion: identity.MembershipVersion, PolicyRevision: identity.PolicyRevision}
	if value, ok := o.projects.Get(key); ok {
		return value, nil
	}
	value, err := o.repository.Project(ctx, identity.Scope, id)
	if err != nil {
		return tenant.Project{}, normalize(err)
	}
	if value.Version != version {
		return tenant.Project{}, ErrConflict
	}
	o.projects.Put(key, value)
	return value, nil
}

func (o *Operations) UpdateProject(ctx context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, expected tenant.Version, value tenant.Project) error {
	attempt := o.attemptMutation(identity, mutation)
	action := lifecycleAttemptAction(value.State)
	if identity.PolicyRevision != o.policyRevision || mutation.Actor != identity.Account || mutation.Device != identity.Device {
		return o.recordFailure(ctx, identity, attempt, action, tenant.TargetProject, string(value.ID), expected, value.Version, expectedVersionDigest(expected), stateDigest(value), tenantresource.ErrDenied)
	}
	change := tenantresource.Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion, CorrelationID: mutation.CorrelationID}
	if err := o.resources.UpdateProject(ctx, identity.Scope, change, expected, value); err != nil {
		return o.recordFailure(ctx, identity, attempt, action, tenant.TargetProject, string(value.ID), expected, value.Version, expectedVersionDigest(expected), stateDigest(value), err)
	}
	o.projects.InvalidateTenant(identity.Scope.Organization)
	return nil
}

func (o *Operations) CreateEnvironment(ctx context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, value tenant.Environment) error {
	attempt := o.attemptMutation(identity, mutation)
	if identity.PolicyRevision != o.policyRevision || mutation.Actor != identity.Account || mutation.Device != identity.Device {
		return o.recordFailure(ctx, identity, attempt, tenant.AuditActionCreate, tenant.TargetEnvironment, string(value.ID), 0, 1, [32]byte{}, stateDigest(value), tenantresource.ErrDenied)
	}
	change := tenantresource.Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion, CorrelationID: mutation.CorrelationID}
	err := o.resources.CreateEnvironment(ctx, identity.Scope, change, value)
	if err != nil {
		return o.recordFailure(ctx, identity, attempt, tenant.AuditActionCreate, tenant.TargetEnvironment, string(value.ID), 0, 1, [32]byte{}, stateDigest(value), err)
	}
	return nil
}

func (o *Operations) UpdateEnvironment(ctx context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, expected tenant.Version, value tenant.Environment) error {
	attempt := o.attemptMutation(identity, mutation)
	action := lifecycleAttemptAction(value.State)
	if identity.PolicyRevision != o.policyRevision || mutation.Actor != identity.Account || mutation.Device != identity.Device {
		return o.recordFailure(ctx, identity, attempt, action, tenant.TargetEnvironment, string(value.ID), expected, value.Version, expectedVersionDigest(expected), stateDigest(value), tenantresource.ErrDenied)
	}
	change := tenantresource.Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion, CorrelationID: mutation.CorrelationID}
	err := o.resources.UpdateEnvironment(ctx, identity.Scope, change, expected, value)
	if err != nil {
		return o.recordFailure(ctx, identity, attempt, action, tenant.TargetEnvironment, string(value.ID), expected, value.Version, expectedVersionDigest(expected), stateDigest(value), err)
	}
	return nil
}

func (o *Operations) attemptMutation(identity tenant.VerifiedIdentity, requested tenant.Mutation) tenant.Mutation {
	return tenant.Mutation{Actor: identity.Account, Device: identity.Device, CorrelationID: requested.CorrelationID, OccurredAt: o.now()}
}

func (o *Operations) recordFailure(ctx context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, action tenant.AuditAction, kind tenant.TargetKind, target string, before, after tenant.Version, beforeDigest, afterDigest [32]byte, cause error) error {
	if after == 0 || after <= before {
		before, after = 0, 1
	}
	result, reason := failureEvidence(cause)
	event, err := tenant.NewAuditAttempt(identity.Scope, identity.Account, identity.Device, action, kind, target, before, after, beforeDigest, afterDigest, result, reason, mutation.OccurredAt.UnixMilli())
	if err != nil {
		return fmt.Errorf("construct authenticated mutation audit: %w", err)
	}
	if err := o.attempts.RecordAttempt(ctx, identity.Scope, mutation, event); err != nil {
		return fmt.Errorf("record authenticated mutation audit: %w", err)
	}
	return normalize(cause)
}

func failureEvidence(err error) (tenant.AuditResult, tenant.AuditReason) {
	switch {
	case errors.Is(err, tenantresource.ErrInvalidState):
		return tenant.AuditResultRefused, tenant.AuditReasonInvalidState
	case errors.Is(err, tenantresource.ErrDenied):
		return tenant.AuditResultRefused, tenant.AuditReasonAuthorizationDenied
	case errors.Is(err, tenantrepo.ErrNotFound):
		return tenant.AuditResultRefused, tenant.AuditReasonResourceUnavailable
	case errors.Is(err, tenantrepo.ErrConflict):
		return tenant.AuditResultConflict, tenant.AuditReasonVersionConflict
	default:
		return tenant.AuditResultFailed, tenant.AuditReasonPersistenceFailed
	}
}

func lifecycleAttemptAction(state tenant.Lifecycle) tenant.AuditAction {
	if state == tenant.LifecycleArchived {
		return tenant.AuditActionArchive
	}
	return tenant.AuditActionUpdate
}

func expectedVersionDigest(version tenant.Version) [32]byte {
	var value [8]byte
	binary.BigEndian.PutUint64(value[:], uint64(version))
	return sha256.Sum256(append([]byte("dacli-expected-version/v1:"), value[:]...))
}

func stateDigest(value any) [32]byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic("tenant API closed state is not serializable: " + err.Error())
	}
	return sha256.Sum256(raw)
}

func normalize(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, tenantresource.ErrDenied) || errors.Is(err, tenantrepo.ErrNotFound) {
		return ErrUnavailable
	}
	if errors.Is(err, tenantrepo.ErrConflict) {
		return ErrConflict
	}
	return err
}
