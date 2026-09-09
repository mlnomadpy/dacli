// Package tenantapi composes verified identities, authorization, scoped
// caching, and the tenant repository for HTTP and worker entry points.
package tenantapi

import (
	"context"
	"errors"

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

func (o *Operations) CreateProject(ctx context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, value tenant.Project) error {
	if identity.PolicyRevision != o.policyRevision || mutation.Actor != identity.Account || mutation.Device != identity.Device {
		return ErrUnavailable
	}
	change := tenantresource.Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion, CorrelationID: mutation.CorrelationID}
	if err := o.resources.CreateProject(ctx, identity.Scope, change, value); err != nil {
		return normalize(err)
	}
	o.projects.InvalidateTenant(identity.Scope.Organization)
	return nil
}

type Operations struct {
	repository     *tenantrepo.Repository
	resources      *tenantresource.Manager
	projects       *tenantcache.Cache[tenant.Project]
	policyRevision uint64
}

func New(repository *tenantrepo.Repository, resources *tenantresource.Manager, cache *tenantcache.Cache[tenant.Project], policyRevision uint64) (*Operations, error) {
	if repository == nil || resources == nil || policyRevision == 0 {
		return nil, errors.New("tenant API requires repository and resource manager")
	}
	if cache == nil {
		cache = tenantcache.New[tenant.Project](1_000)
	}
	return &Operations{repository: repository, resources: resources, projects: cache, policyRevision: policyRevision}, nil
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
	if identity.PolicyRevision != o.policyRevision || mutation.Actor != identity.Account || mutation.Device != identity.Device {
		return ErrUnavailable
	}
	change := tenantresource.Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion, CorrelationID: mutation.CorrelationID}
	if err := o.resources.UpdateProject(ctx, identity.Scope, change, expected, value); err != nil {
		return normalize(err)
	}
	o.projects.InvalidateTenant(identity.Scope.Organization)
	return nil
}

func (o *Operations) CreateEnvironment(ctx context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, value tenant.Environment) error {
	if identity.PolicyRevision != o.policyRevision || mutation.Actor != identity.Account || mutation.Device != identity.Device {
		return ErrUnavailable
	}
	change := tenantresource.Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion, CorrelationID: mutation.CorrelationID}
	return normalize(o.resources.CreateEnvironment(ctx, identity.Scope, change, value))
}

func (o *Operations) UpdateEnvironment(ctx context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, expected tenant.Version, value tenant.Environment) error {
	if identity.PolicyRevision != o.policyRevision || mutation.Actor != identity.Account || mutation.Device != identity.Device {
		return ErrUnavailable
	}
	change := tenantresource.Mutation{Actor: identity.Account, Device: identity.Device, ExpectedMembershipVersion: identity.MembershipVersion, CorrelationID: mutation.CorrelationID}
	return normalize(o.resources.UpdateEnvironment(ctx, identity.Scope, change, expected, value))
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
