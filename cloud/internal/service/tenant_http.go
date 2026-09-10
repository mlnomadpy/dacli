package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/cloud/internal/ratelimit"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantapi"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantrepo"
)

const (
	defaultPageSize     = 25
	projectPageKind     = "project"
	environmentPageKind = "environment"
)

type tenantHTTP struct {
	api      *API
	verifier IdentityVerifier
	backend  tenantapi.Backend
	cursors  *cursorCodec
}

// EnableTenantAPI installs the tenant routes only when all security-boundary
// dependencies are explicit. Health and readiness remain usable without it.
func (a *API) EnableTenantAPI(verifier IdentityVerifier, backend tenantapi.Backend, cursorSecret string) error {
	if a == nil || verifier == nil || backend == nil {
		return errors.New("tenant API requires verifier and backend")
	}
	cursors, err := newCursorCodec(cursorSecret)
	if err != nil {
		return err
	}
	a.tenant = &tenantHTTP{api: a, verifier: verifier, backend: backend, cursors: cursors}
	return nil
}

func (h *tenantHTTP) projects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.api.writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "endpoint requires GET or POST", false)
		return
	}
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		h.createProject(w, r, identity)
		return
	}
	if !h.api.allow(w, r, h.api.tenantReadLimit, ratelimit.IdentityKey(h.api.limitSecret, identity, projectPageKind)) {
		return
	}
	request, ok := h.pageRequest(w, r, identity.Scope, projectPageKind, "")
	if !ok {
		return
	}
	page, err := h.backend.ListProjects(r.Context(), identity, request)
	if err != nil {
		h.backendError(w, r, err)
		return
	}
	next, err := h.cursors.encode(identity.Scope, projectPageKind, "", page.Snapshot, page.Next)
	if err != nil {
		h.api.writeError(w, r, http.StatusInternalServerError, "internal_error", "request could not be completed", true)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schema": "controlplane-page/v1", "items": page.Items, "next_cursor": next})
}

func (h *tenantHTTP) project(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identity(w, r)
	if !ok {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1/projects/")
	parts := strings.Split(path, "/")
	if len(parts) == 3 && parts[1] == "environments" {
		h.environment(w, r, identity, parts[0], parts[2])
		return
	}
	if len(parts) == 2 && parts[1] == "environments" {
		h.environments(w, r, identity, parts[0])
		return
	}
	if len(parts) != 1 || parts[0] == "" {
		h.api.writeError(w, r, http.StatusNotFound, "not_found", "endpoint does not exist", false)
		return
	}
	id, err := tenant.NewProjectID(parts[0])
	if err != nil {
		h.api.writeError(w, r, http.StatusNotFound, "resource_unavailable", "resource is unavailable", false)
		return
	}
	switch r.Method {
	case http.MethodGet:
		version, ok := h.version(w, r, r.URL.Query().Get("version"))
		if !ok {
			return
		}
		value, err := h.backend.Project(r.Context(), identity, id, version)
		if err != nil {
			h.backendError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	case http.MethodPatch:
		h.updateProject(w, r, identity, id)
	default:
		h.api.writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "endpoint requires GET or PATCH", false)
	}
}

func (h *tenantHTTP) environments(w http.ResponseWriter, r *http.Request, identity tenant.VerifiedIdentity, rawProject string) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.api.writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "endpoint requires GET or POST", false)
		return
	}
	project, err := tenant.NewProjectID(rawProject)
	if err != nil {
		h.api.writeError(w, r, http.StatusNotFound, "resource_unavailable", "resource is unavailable", false)
		return
	}
	if r.Method == http.MethodPost {
		h.createEnvironment(w, r, identity, project)
		return
	}
	if !h.api.allow(w, r, h.api.tenantReadLimit, ratelimit.IdentityKey(h.api.limitSecret, identity, environmentPageKind)) {
		return
	}
	request, ok := h.pageRequest(w, r, identity.Scope, environmentPageKind, string(project))
	if !ok {
		return
	}
	page, err := h.backend.ListEnvironments(r.Context(), identity, project, request)
	if err != nil {
		h.backendError(w, r, err)
		return
	}
	next, err := h.cursors.encode(identity.Scope, environmentPageKind, string(project), page.Snapshot, page.Next)
	if err != nil {
		h.api.writeError(w, r, http.StatusInternalServerError, "internal_error", "request could not be completed", true)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schema": "controlplane-page/v1", "items": page.Items, "next_cursor": next})
}

type projectUpdate struct {
	Name            string           `json:"name"`
	State           tenant.Lifecycle `json:"state"`
	ExpectedVersion tenant.Version   `json:"expected_version"`
}

type projectCreate struct {
	ID   tenant.ProjectID `json:"project_id"`
	Name string           `json:"name"`
}

func (h *tenantHTTP) createProject(w http.ResponseWriter, r *http.Request, identity tenant.VerifiedIdentity) {
	var input projectCreate
	if err := decodeClosed(r.Body, &input); err != nil {
		h.decodeError(w, r, err)
		return
	}
	value := tenant.Project{Tenant: identity.Scope.Organization, ID: input.ID, Name: input.Name, State: tenant.LifecycleActive, Version: 1}
	if err := h.backend.CreateProject(r.Context(), identity, h.mutation(r, identity), value); err != nil {
		h.backendError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (h *tenantHTTP) updateProject(w http.ResponseWriter, r *http.Request, identity tenant.VerifiedIdentity, id tenant.ProjectID) {
	var input projectUpdate
	if err := decodeClosed(r.Body, &input); err != nil {
		h.decodeError(w, r, err)
		return
	}
	value := tenant.Project{Tenant: identity.Scope.Organization, ID: id, Name: input.Name, State: input.State, Version: input.ExpectedVersion + 1}
	if err := h.backend.UpdateProject(r.Context(), identity, h.mutation(r, identity), input.ExpectedVersion, value); err != nil {
		h.backendError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

type environmentCreate struct {
	ID   tenant.EnvironmentID   `json:"environment_id"`
	Name string                 `json:"name"`
	Kind tenant.EnvironmentKind `json:"kind"`
}

type environmentUpdate struct {
	Name            string                 `json:"name"`
	Kind            tenant.EnvironmentKind `json:"kind"`
	State           tenant.Lifecycle       `json:"state"`
	ExpectedVersion tenant.Version         `json:"expected_version"`
}

func (h *tenantHTTP) createEnvironment(w http.ResponseWriter, r *http.Request, identity tenant.VerifiedIdentity, project tenant.ProjectID) {
	var input environmentCreate
	if err := decodeClosed(r.Body, &input); err != nil {
		h.decodeError(w, r, err)
		return
	}
	value := tenant.Environment{Tenant: identity.Scope.Organization, Project: project, ID: input.ID, Name: input.Name, Kind: input.Kind, State: tenant.LifecycleActive, Version: 1}
	if err := h.backend.CreateEnvironment(r.Context(), identity, h.mutation(r, identity), value); err != nil {
		h.backendError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (h *tenantHTTP) environment(w http.ResponseWriter, r *http.Request, identity tenant.VerifiedIdentity, rawProject, rawEnvironment string) {
	if r.Method != http.MethodPatch {
		h.api.writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "endpoint requires PATCH", false)
		return
	}
	project, projectErr := tenant.NewProjectID(rawProject)
	environment, environmentErr := tenant.NewEnvironmentID(rawEnvironment)
	if projectErr != nil || environmentErr != nil {
		h.api.writeError(w, r, http.StatusNotFound, "resource_unavailable", "resource is unavailable", false)
		return
	}
	var input environmentUpdate
	if err := decodeClosed(r.Body, &input); err != nil {
		h.decodeError(w, r, err)
		return
	}
	value := tenant.Environment{Tenant: identity.Scope.Organization, Project: project, ID: environment, Name: input.Name, Kind: input.Kind, State: input.State, Version: input.ExpectedVersion + 1}
	if err := h.backend.UpdateEnvironment(r.Context(), identity, h.mutation(r, identity), input.ExpectedVersion, value); err != nil {
		h.backendError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *tenantHTTP) mutation(r *http.Request, identity tenant.VerifiedIdentity) tenant.Mutation {
	return tenant.Mutation{Actor: identity.Account, Device: identity.Device, CorrelationID: r.Header.Get("X-Request-ID")}
}

func (h *tenantHTTP) decodeError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		h.api.writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large", "request exceeds the configured byte limit", false)
		return
	}
	h.api.writeError(w, r, http.StatusBadRequest, "invalid_request", "request body is invalid", false)
}

func (h *tenantHTTP) identity(w http.ResponseWriter, r *http.Request) (tenant.VerifiedIdentity, bool) {
	if !h.api.allow(w, r, h.api.identityLimit, ratelimit.NetworkKey(h.api.limitSecret, r.RemoteAddr)) {
		return tenant.VerifiedIdentity{}, false
	}
	identity, err := h.verifier.Verify(r.Context(), r.Header.Get("Authorization"))
	if err != nil || tenant.ValidateVerifiedIdentity(identity) != nil {
		h.api.writeError(w, r, http.StatusUnauthorized, "unauthorized", "verified identity is required", false)
		return tenant.VerifiedIdentity{}, false
	}
	return identity, true
}

func (h *tenantHTTP) pageRequest(w http.ResponseWriter, r *http.Request, scope tenant.Scope, kind, parent string) (tenantrepo.PageRequest, bool) {
	limit := defaultPageSize
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > tenantrepo.MaxPageSize {
			h.api.writeError(w, r, http.StatusBadRequest, "invalid_page", "page request is invalid", false)
			return tenantrepo.PageRequest{}, false
		}
		limit = parsed
	}
	rawCursor := r.URL.Query().Get("cursor")
	if rawCursor == "" {
		return tenantrepo.PageRequest{Limit: limit}, true
	}
	request, err := h.cursors.decode(rawCursor, scope, kind, parent, limit)
	if err != nil {
		h.api.writeError(w, r, http.StatusBadRequest, "invalid_page", "page request is invalid", false)
		return tenantrepo.PageRequest{}, false
	}
	return request, true
}

func (h *tenantHTTP) version(w http.ResponseWriter, r *http.Request, raw string) (tenant.Version, bool) {
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 {
		h.api.writeError(w, r, http.StatusBadRequest, "invalid_version", "an exact positive version is required", false)
		return 0, false
	}
	return tenant.Version(parsed), true
}

func (h *tenantHTTP) backendError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, tenantapi.ErrUnavailable):
		h.api.writeError(w, r, http.StatusNotFound, "resource_unavailable", "resource is unavailable", false)
	case errors.Is(err, tenantapi.ErrConflict):
		h.api.writeError(w, r, http.StatusConflict, "version_conflict", "resource version changed", false)
	default:
		h.api.writeError(w, r, http.StatusServiceUnavailable, "dependency_unavailable", "service dependency is unavailable", true)
	}
}

func decodeClosed(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}
