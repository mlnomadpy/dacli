package worker

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

type workVerifier struct {
	identity tenant.VerifiedIdentity
	err      error
}

func (v workVerifier) VerifyWork(context.Context, string) (tenant.VerifiedIdentity, error) {
	return v.identity, v.err
}

type recordingProcessor struct {
	scope   tenant.Scope
	calls   int
	payload TenantPayload
}

func (p *recordingProcessor) ProcessTenant(_ context.Context, scope tenant.Scope, _ tenant.VerifiedIdentity, payload TenantPayload) error {
	p.scope, p.calls, p.payload = scope, p.calls+1, payload
	return nil
}

func TestTenantDispatchUsesVerifiedScopeNotClaimedTenant(t *testing.T) {
	identity := workerIdentity()
	processor := &recordingProcessor{}
	dispatcher, err := NewTenantDispatcher(workVerifier{identity: identity}, processor)
	if err != nil {
		t.Fatal(err)
	}
	if err := dispatcher.Dispatch(context.Background(), TenantWork{Credential: "opaque", ClaimedTenant: "tenant-b", Kind: "project", Identity: "same-id"}); err != nil {
		t.Fatal(err)
	}
	if processor.calls != 1 || processor.scope != identity.Scope || processor.payload.Kind != "project" || processor.payload.Identity != "same-id" {
		t.Fatalf("processor calls=%d scope=%+v", processor.calls, processor.scope)
	}
}

func TestTenantDispatchFailsClosedBeforeProcessor(t *testing.T) {
	processor := &recordingProcessor{}
	for _, verifier := range []WorkIdentityVerifier{workVerifier{err: errors.New("bad signature")}, workVerifier{identity: tenant.VerifiedIdentity{}}} {
		dispatcher, _ := NewTenantDispatcher(verifier, processor)
		if err := dispatcher.Dispatch(context.Background(), TenantWork{}); !errors.Is(err, ErrTenantWorkDenied) {
			t.Fatalf("dispatch = %v", err)
		}
	}
	if processor.calls != 0 {
		t.Fatal("denied work reached processor")
	}
	if _, err := NewTenantDispatcher(nil, processor); err == nil {
		t.Fatal("nil verifier accepted")
	}
}

func TestTenantWorkNeverSerializesCredential(t *testing.T) {
	raw, err := json.Marshal(TenantWork{Credential: "raw-secret", ClaimedTenant: "tenant-a", Kind: "project", Identity: "project-a"})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || strings.Contains(string(raw), "raw-secret") || strings.Contains(string(raw), "Credential") {
		t.Fatalf("serialized work leaked credential: %s", raw)
	}
}

func workerIdentity() tenant.VerifiedIdentity {
	scope, _ := tenant.NewScope("tenant-a")
	return tenant.VerifiedIdentity{Scope: scope, Account: "account-a", Device: "device-a", MembershipVersion: 2, PolicyRevision: 3}
}
