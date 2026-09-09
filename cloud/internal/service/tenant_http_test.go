package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantapi"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantrepo"
)

const testSigningSecret = "0123456789abcdef0123456789abcdef"

type backendCall struct {
	identity    tenant.VerifiedIdentity
	page        tenantrepo.PageRequest
	project     tenant.ProjectID
	mutation    tenant.Mutation
	expected    tenant.Version
	value       tenant.Project
	environment tenant.Environment
}

type fakeTenantBackend struct {
	calls           []backendCall
	projectPage     tenantrepo.Page[tenant.Project]
	environmentPage tenantrepo.Page[tenant.Environment]
	projectValue    tenant.Project
	err             error
}

func (b *fakeTenantBackend) ListProjects(_ context.Context, identity tenant.VerifiedIdentity, page tenantrepo.PageRequest) (tenantrepo.Page[tenant.Project], error) {
	b.calls = append(b.calls, backendCall{identity: identity, page: page})
	return b.projectPage, b.err
}
func (b *fakeTenantBackend) ListEnvironments(_ context.Context, identity tenant.VerifiedIdentity, project tenant.ProjectID, page tenantrepo.PageRequest) (tenantrepo.Page[tenant.Environment], error) {
	b.calls = append(b.calls, backendCall{identity: identity, project: project, page: page})
	return b.environmentPage, b.err
}
func (b *fakeTenantBackend) Project(_ context.Context, identity tenant.VerifiedIdentity, project tenant.ProjectID, version tenant.Version) (tenant.Project, error) {
	b.calls = append(b.calls, backendCall{identity: identity, project: project, expected: version})
	return b.projectValue, b.err
}
func (b *fakeTenantBackend) CreateProject(_ context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, value tenant.Project) error {
	b.calls = append(b.calls, backendCall{identity: identity, mutation: mutation, value: value})
	return b.err
}
func (b *fakeTenantBackend) UpdateProject(_ context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, expected tenant.Version, value tenant.Project) error {
	b.calls = append(b.calls, backendCall{identity: identity, mutation: mutation, expected: expected, value: value})
	return b.err
}
func (b *fakeTenantBackend) CreateEnvironment(_ context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, value tenant.Environment) error {
	b.calls = append(b.calls, backendCall{identity: identity, mutation: mutation, project: value.Project, environment: value})
	return b.err
}
func (b *fakeTenantBackend) UpdateEnvironment(_ context.Context, identity tenant.VerifiedIdentity, mutation tenant.Mutation, expected tenant.Version, value tenant.Environment) error {
	b.calls = append(b.calls, backendCall{identity: identity, mutation: mutation, project: value.Project, expected: expected, environment: value})
	return b.err
}

func httpIdentity(t *testing.T, tenantID string) (tenant.VerifiedIdentity, string) {
	t.Helper()
	scope, _ := tenant.NewScope(tenant.OrganizationID(tenantID))
	identity := tenant.VerifiedIdentity{Scope: scope, Account: "account-a", Device: "device-a", MembershipVersion: 7, PolicyRevision: 3}
	verifier, err := NewHMACIdentityVerifier(testSigningSecret)
	if err != nil {
		t.Fatal(err)
	}
	token, err := verifier.Sign(identity)
	if err != nil {
		t.Fatal(err)
	}
	return identity, token
}

func tenantHandler(t *testing.T, backend tenantapi.Backend) (*API, string) {
	t.Helper()
	_, token := httpIdentity(t, "tenant-a")
	verifier, _ := NewHMACIdentityVerifier(testSigningSecret)
	api := NewAPI(testConfig(), nil, nil)
	if err := api.EnableTenantAPI(verifier, backend, testSigningSecret); err != nil {
		t.Fatal(err)
	}
	return api, token
}

func serveTenant(api *API, token, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("X-Request-ID", "request-1")
	recorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, request)
	return recorder
}

func TestSignedIdentityRejectsTamperUnknownFieldsAndInvalidShape(t *testing.T) {
	identity, token := httpIdentity(t, "tenant-a")
	verifier, _ := NewHMACIdentityVerifier(testSigningSecret)
	verified, err := verifier.Verify(context.Background(), "Bearer "+token)
	if err != nil || verified != identity {
		t.Fatalf("verified=%+v error=%v", verified, err)
	}
	for _, authorization := range []string{"", "Bearer " + token + "x", "Basic " + token, "Bearer malformed"} {
		if _, err := verifier.Verify(context.Background(), authorization); !errors.Is(err, ErrIdentityDenied) {
			t.Fatalf("%q = %v", authorization, err)
		}
	}
	if _, err := NewHMACIdentityVerifier("short"); err == nil {
		t.Fatal("short identity key accepted")
	}
	unknown := []byte(`{"tenant_id":"tenant-a","account_id":"account-a","membership_version":1,"policy_revision":1,"prompt":"forbidden"}`)
	unknownToken := base64.RawURLEncoding.EncodeToString(unknown) + "." + base64.RawURLEncoding.EncodeToString(sign([]byte(testSigningSecret), "identity", unknown))
	if _, err := verifier.Verify(context.Background(), "Bearer "+unknownToken); !errors.Is(err, ErrIdentityDenied) {
		t.Fatalf("unknown identity field = %v", err)
	}
}

func TestSignedIdentityExpires(t *testing.T) {
	now := time.Unix(1_000, 0)
	verifier, err := newHMACIdentityVerifier(testSigningSecret, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	scope, _ := tenant.NewScope("tenant-a")
	identity := tenant.VerifiedIdentity{Scope: scope, Account: "account-a", MembershipVersion: 1, PolicyRevision: 1}
	token, err := verifier.Sign(identity)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(5 * time.Minute)
	if _, err := verifier.Verify(context.Background(), "Bearer "+token); !errors.Is(err, ErrIdentityDenied) {
		t.Fatalf("expired identity = %v", err)
	}
}

func TestProjectPaginationCursorIsOpaqueSnapshotBoundAndTenantBound(t *testing.T) {
	backend := &fakeTenantBackend{projectPage: tenantrepo.Page[tenant.Project]{Items: []tenant.Project{{Tenant: "tenant-a", ID: "project-1", Name: "One", State: tenant.LifecycleActive, Version: 1}}, Snapshot: 9, Next: 5}}
	api, token := tenantHandler(t, backend)
	first := serveTenant(api, token, http.MethodGet, "/v1/projects?limit=1", "")
	if first.Code != http.StatusOK {
		t.Fatalf("first = %d %s", first.Code, first.Body.String())
	}
	var response struct {
		Next string `json:"next_cursor"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &response); err != nil || response.Next == "" || strings.Contains(response.Next, "tenant-a") {
		t.Fatalf("cursor response = %+v, %v", response, err)
	}
	backend.projectPage = tenantrepo.Page[tenant.Project]{Snapshot: 9}
	second := serveTenant(api, token, http.MethodGet, "/v1/projects?limit=1&cursor="+response.Next, "")
	if second.Code != http.StatusOK || len(backend.calls) != 2 || backend.calls[1].page.Snapshot != 9 || backend.calls[1].page.After != 5 {
		t.Fatalf("second=%d calls=%+v", second.Code, backend.calls)
	}
	_, otherToken := httpIdentity(t, "tenant-b")
	wrongTenant := serveTenant(api, otherToken, http.MethodGet, "/v1/projects?limit=1&cursor="+response.Next, "")
	if wrongTenant.Code != http.StatusBadRequest || len(backend.calls) != 2 {
		t.Fatalf("cross-tenant cursor=%d calls=%d", wrongTenant.Code, len(backend.calls))
	}
	tampered := serveTenant(api, token, http.MethodGet, "/v1/projects?limit=1&cursor="+response.Next+"x", "")
	if tampered.Code != http.StatusBadRequest || len(backend.calls) != 2 {
		t.Fatalf("tampered cursor=%d calls=%d", tampered.Code, len(backend.calls))
	}
}

func TestEnvironmentCursorIsBoundToParentProject(t *testing.T) {
	backend := &fakeTenantBackend{environmentPage: tenantrepo.Page[tenant.Environment]{Snapshot: 8, Next: 4}}
	api, token := tenantHandler(t, backend)
	first := serveTenant(api, token, http.MethodGet, "/v1/projects/project-a/environments?limit=2", "")
	var response struct {
		Next string `json:"next_cursor"`
	}
	_ = json.Unmarshal(first.Body.Bytes(), &response)
	wrongParent := serveTenant(api, token, http.MethodGet, "/v1/projects/project-b/environments?limit=2&cursor="+response.Next, "")
	if first.Code != http.StatusOK || response.Next == "" || wrongParent.Code != http.StatusBadRequest || len(backend.calls) != 1 {
		t.Fatalf("first=%d wrong=%d calls=%d", first.Code, wrongParent.Code, len(backend.calls))
	}
}

func TestProjectUpdateDerivesTenantAndActorOnlyFromVerifiedIdentity(t *testing.T) {
	backend := &fakeTenantBackend{}
	api, token := tenantHandler(t, backend)
	response := serveTenant(api, token, http.MethodPatch, "/v1/projects/project-a", `{"name":"Renamed","state":1,"expected_version":3}`)
	if response.Code != http.StatusOK || len(backend.calls) != 1 {
		t.Fatalf("response=%d %s calls=%d", response.Code, response.Body.String(), len(backend.calls))
	}
	call := backend.calls[0]
	if call.value.Tenant != "tenant-a" || call.value.ID != "project-a" || call.value.Version != 4 || call.mutation.Actor != "account-a" || call.mutation.Device != "device-a" {
		t.Fatalf("derived mutation = %+v", call)
	}
	override := serveTenant(api, token, http.MethodPatch, "/v1/projects/project-a", `{"tenant_id":"tenant-b","name":"X","state":1,"expected_version":4}`)
	if override.Code != http.StatusBadRequest || len(backend.calls) != 1 {
		t.Fatalf("body override=%d calls=%d", override.Code, len(backend.calls))
	}
}

func TestProjectAndEnvironmentCreatesDeriveTenantAndClosedLifecycle(t *testing.T) {
	backend := &fakeTenantBackend{}
	api, token := tenantHandler(t, backend)
	project := serveTenant(api, token, http.MethodPost, "/v1/projects", `{"project_id":"project-a","name":"Project"}`)
	environment := serveTenant(api, token, http.MethodPost, "/v1/projects/project-a/environments", `{"environment_id":"environment-a","name":"Production","kind":3}`)
	updated := serveTenant(api, token, http.MethodPatch, "/v1/projects/project-a/environments/environment-a", `{"name":"Archived","kind":3,"state":3,"expected_version":4}`)
	if project.Code != http.StatusCreated || environment.Code != http.StatusCreated || updated.Code != http.StatusOK || len(backend.calls) != 3 {
		t.Fatalf("project=%d environment=%d updated=%d calls=%d", project.Code, environment.Code, updated.Code, len(backend.calls))
	}
	if backend.calls[0].value.Tenant != "tenant-a" || backend.calls[0].value.State != tenant.LifecycleActive || backend.calls[0].value.Version != 1 {
		t.Fatalf("project create = %+v", backend.calls[0])
	}
	if backend.calls[1].environment.Tenant != "tenant-a" || backend.calls[1].environment.Project != "project-a" || backend.calls[1].environment.State != tenant.LifecycleActive || backend.calls[1].environment.Version != 1 {
		t.Fatalf("environment create = %+v", backend.calls[1])
	}
	if backend.calls[2].environment.Tenant != "tenant-a" || backend.calls[2].environment.Project != "project-a" || backend.calls[2].environment.ID != "environment-a" || backend.calls[2].environment.Version != 5 {
		t.Fatalf("environment update = %+v", backend.calls[2])
	}
}

func TestAPIRefusalsAreBoundedAndDoNotDiscloseExistence(t *testing.T) {
	backend := &fakeTenantBackend{err: tenantapi.ErrUnavailable}
	api, token := tenantHandler(t, backend)
	missing := serveTenant(api, token, http.MethodGet, "/v1/projects/missing?version=1", "")
	wrong := serveTenant(api, token, http.MethodGet, "/v1/projects/collision?version=1", "")
	if missing.Code != http.StatusNotFound || wrong.Code != http.StatusNotFound || missing.Body.String() != wrong.Body.String() || !strings.Contains(missing.Body.String(), `"code":"resource_unavailable"`) {
		t.Fatalf("missing=%d %s wrong=%d %s", missing.Code, missing.Body.String(), wrong.Code, wrong.Body.String())
	}
	unauthorized := serveTenant(api, "", http.MethodGet, "/v1/projects", "")
	if unauthorized.Code != http.StatusUnauthorized || len(backend.calls) != 2 {
		t.Fatalf("unauthorized=%d calls=%d", unauthorized.Code, len(backend.calls))
	}
	invalidPage := serveTenant(api, token, http.MethodGet, "/v1/projects?limit=101", "")
	if invalidPage.Code != http.StatusBadRequest || len(backend.calls) != 2 {
		t.Fatalf("invalid page=%d calls=%d", invalidPage.Code, len(backend.calls))
	}
}

func TestMutationHTTPDistinguishesConflictFromNonDisclosingRefusalAndAuditFailure(t *testing.T) {
	backend := &fakeTenantBackend{err: tenantapi.ErrUnavailable}
	api, token := tenantHandler(t, backend)
	body := `{"project_id":"project-a","name":"Project"}`
	refused := serveTenant(api, token, http.MethodPost, "/v1/projects", body)
	if refused.Code != http.StatusNotFound || !strings.Contains(refused.Body.String(), `"code":"resource_unavailable"`) {
		t.Fatalf("refused = %d %s", refused.Code, refused.Body.String())
	}
	backend.err = tenantapi.ErrConflict
	conflict := serveTenant(api, token, http.MethodPost, "/v1/projects", body)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"code":"version_conflict"`) {
		t.Fatalf("conflict = %d %s", conflict.Code, conflict.Body.String())
	}
	backend.err = errors.New("durable audit unavailable")
	failed := serveTenant(api, token, http.MethodPost, "/v1/projects", body)
	if failed.Code != http.StatusServiceUnavailable || !strings.Contains(failed.Body.String(), `"code":"dependency_unavailable"`) || !strings.Contains(failed.Body.String(), `"retryable":true`) {
		t.Fatalf("failed = %d %s", failed.Code, failed.Body.String())
	}
	unauthenticated := serveTenant(api, "", http.MethodPost, "/v1/projects", body)
	if unauthenticated.Code != http.StatusUnauthorized || len(backend.calls) != 3 {
		t.Fatalf("unauthenticated = %d calls=%d", unauthenticated.Code, len(backend.calls))
	}
}

func TestChunkedOversizeTenantBodyReturnsStructured413(t *testing.T) {
	backend := &fakeTenantBackend{}
	api, token := tenantHandler(t, backend)
	body := `{"name":"` + strings.Repeat("x", 2000) + `","state":1,"expected_version":1}`
	request := httptest.NewRequest(http.MethodPatch, "/v1/projects/project-a", io.NopCloser(strings.NewReader(body)))
	request.ContentLength = -1
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge || !strings.Contains(recorder.Body.String(), `"code":"request_too_large"`) || len(backend.calls) != 0 {
		t.Fatalf("response=%d %s calls=%d", recorder.Code, recorder.Body.String(), len(backend.calls))
	}
}
