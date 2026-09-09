package worker

import (
	"context"
	"errors"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

var ErrTenantWorkDenied = errors.New("tenant work item denied")

// TenantWork keeps an untrusted claimed tenant only so adapters can prove it
// is never treated as authority. Credential verification supplies the scope.
type TenantWork struct {
	Credential    string                `json:"-"`
	ClaimedTenant tenant.OrganizationID `json:"claimed_tenant"`
	Kind          string                `json:"kind"`
	Identity      string                `json:"identity"`
}

type TenantPayload struct {
	Kind     string `json:"kind"`
	Identity string `json:"identity"`
}

type WorkIdentityVerifier interface {
	VerifyWork(context.Context, string) (tenant.VerifiedIdentity, error)
}

type TenantProcessor interface {
	ProcessTenant(context.Context, tenant.Scope, tenant.VerifiedIdentity, TenantPayload) error
}

type TenantDispatcher struct {
	verifier  WorkIdentityVerifier
	processor TenantProcessor
}

func NewTenantDispatcher(verifier WorkIdentityVerifier, processor TenantProcessor) (*TenantDispatcher, error) {
	if verifier == nil || processor == nil {
		return nil, errors.New("tenant worker requires verifier and processor")
	}
	return &TenantDispatcher{verifier: verifier, processor: processor}, nil
}

func (d *TenantDispatcher) Dispatch(ctx context.Context, work TenantWork) error {
	if d == nil || d.verifier == nil || d.processor == nil {
		return ErrTenantWorkDenied
	}
	identity, err := d.verifier.VerifyWork(ctx, work.Credential)
	if err != nil || tenant.ValidateVerifiedIdentity(identity) != nil {
		return ErrTenantWorkDenied
	}
	return d.processor.ProcessTenant(ctx, identity.Scope, identity, TenantPayload{Kind: work.Kind, Identity: work.Identity})
}
