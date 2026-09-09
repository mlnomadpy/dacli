package tenantrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

const MaxPageSize = 100

type PageRequest struct {
	Snapshot int64
	After    int64
	Limit    int
}

type Page[T any] struct {
	Items    []T
	Snapshot int64
	Next     int64
}

func (r *Repository) ListProjects(ctx context.Context, scope tenant.Scope, request PageRequest) (Page[tenant.Project], error) {
	var page Page[tenant.Project]
	if err := validatePage(request); err != nil {
		return page, err
	}
	err := r.withTenant(ctx, scope, true, func(tx *sql.Tx) error {
		snapshot, err := pageSnapshot(ctx, tx, "controlplane_projects", scope, request)
		if err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT tenant_id, project_id, name, state, version, list_ordinal
FROM controlplane_projects
WHERE tenant_id = $1 AND list_ordinal <= $2 AND list_ordinal > $3
ORDER BY list_ordinal ASC LIMIT $4`, scope.Organization, snapshot, request.After, request.Limit+1)
		if err != nil {
			return fmt.Errorf("list projects: %w", err)
		}
		defer func() { _ = rows.Close() }()
		page.Snapshot = snapshot
		hasMore := false
		for rows.Next() {
			var value tenant.Project
			var ordinal int64
			if err := rows.Scan(&value.Tenant, &value.ID, &value.Name, &value.State, &value.Version, &ordinal); err != nil {
				return fmt.Errorf("scan project page: %w", err)
			}
			if err := tenant.ValidateProject(scope, value); err != nil {
				return fmt.Errorf("validate stored project page: %w", err)
			}
			if len(page.Items) == request.Limit {
				hasMore = true
				break
			}
			page.Items = append(page.Items, value)
			page.Next = ordinal
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate project page: %w", err)
		}
		if !hasMore {
			page.Next = 0
		}
		return nil
	})
	if err != nil {
		return Page[tenant.Project]{}, err
	}
	return page, nil
}

func (r *Repository) ListEnvironments(ctx context.Context, scope tenant.Scope, project tenant.ProjectID, request PageRequest) (Page[tenant.Environment], error) {
	var page Page[tenant.Environment]
	if project == "" {
		return page, errors.New("environment page requires a project")
	}
	if err := validatePage(request); err != nil {
		return page, err
	}
	err := r.withTenant(ctx, scope, true, func(tx *sql.Tx) error {
		snapshot := request.Snapshot
		if snapshot == 0 {
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(list_ordinal), 0) FROM controlplane_environments
WHERE tenant_id = $1 AND project_id = $2`, scope.Organization, project).Scan(&snapshot); err != nil {
				return fmt.Errorf("read environment page snapshot: %w", err)
			}
		}
		rows, err := tx.QueryContext(ctx, `SELECT tenant_id, project_id, environment_id, name, kind, state, version, list_ordinal
FROM controlplane_environments
WHERE tenant_id = $1 AND project_id = $2 AND list_ordinal <= $3 AND list_ordinal > $4
ORDER BY list_ordinal ASC LIMIT $5`, scope.Organization, project, snapshot, request.After, request.Limit+1)
		if err != nil {
			return fmt.Errorf("list environments: %w", err)
		}
		defer func() { _ = rows.Close() }()
		page.Snapshot = snapshot
		hasMore := false
		for rows.Next() {
			var value tenant.Environment
			var ordinal int64
			if err := rows.Scan(&value.Tenant, &value.Project, &value.ID, &value.Name, &value.Kind, &value.State, &value.Version, &ordinal); err != nil {
				return fmt.Errorf("scan environment page: %w", err)
			}
			if err := tenant.ValidateEnvironment(scope, value); err != nil || value.Project != project {
				return errors.New("stored environment page escaped its tenant or project scope")
			}
			if len(page.Items) == request.Limit {
				hasMore = true
				break
			}
			page.Items = append(page.Items, value)
			page.Next = ordinal
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate environment page: %w", err)
		}
		if !hasMore {
			page.Next = 0
		}
		return nil
	})
	if err != nil {
		return Page[tenant.Environment]{}, err
	}
	return page, nil
}

func validatePage(request PageRequest) error {
	if request.Limit < 1 || request.Limit > MaxPageSize || request.Snapshot < 0 || request.After < 0 || (request.Snapshot == 0 && request.After != 0) || (request.Snapshot > 0 && request.After > request.Snapshot) {
		return errors.New("page request is outside its bounds")
	}
	return nil
}

func pageSnapshot(ctx context.Context, tx *sql.Tx, table string, scope tenant.Scope, request PageRequest) (int64, error) {
	if request.Snapshot != 0 {
		return request.Snapshot, nil
	}
	query := `SELECT COALESCE(MAX(list_ordinal), 0) FROM ` + table + ` WHERE tenant_id = $1`
	var snapshot int64
	if err := tx.QueryRowContext(ctx, query, scope.Organization).Scan(&snapshot); err != nil {
		return 0, fmt.Errorf("read page snapshot: %w", err)
	}
	return snapshot, nil
}
