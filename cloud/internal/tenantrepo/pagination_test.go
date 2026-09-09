package tenantrepo

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

func TestProjectPageUsesTenantSnapshotAndLookahead(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(list_ordinal\), 0\) FROM controlplane_projects WHERE tenant_id = \$1`).WithArgs(scope.Organization).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(9))
	mock.ExpectQuery(`WHERE tenant_id = \$1 AND list_ordinal <= \$2 AND list_ordinal > \$3\s+ORDER BY list_ordinal ASC LIMIT \$4`).WithArgs(scope.Organization, int64(9), int64(0), 3).WillReturnRows(projectPageRows().
		AddRow("tenant-a", "project-1", "One", 1, 1, 2).
		AddRow("tenant-a", "project-2", "Two", 1, 1, 5).
		AddRow("tenant-a", "project-3", "Lookahead", 1, 1, 8))
	mock.ExpectCommit()
	page, err := repo.ListProjects(context.Background(), scope, PageRequest{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Snapshot != 9 || page.Next != 5 || page.Items[1].ID != "project-2" {
		t.Fatalf("page = %+v", page)
	}
	assertExpectations(t, mock)
}

func TestProjectNextPageReusesSnapshotWithoutGapOrDuplicate(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_projects\s+WHERE tenant_id = \$1 AND list_ordinal <= \$2 AND list_ordinal > \$3`).WithArgs(scope.Organization, int64(9), int64(5), 3).WillReturnRows(projectPageRows().
		AddRow("tenant-a", "project-3", "Three", 1, 1, 8))
	mock.ExpectCommit()
	page, err := repo.ListProjects(context.Background(), scope, PageRequest{Snapshot: 9, After: 5, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "project-3" || page.Next != 0 || page.Snapshot != 9 {
		t.Fatalf("page = %+v", page)
	}
	assertExpectations(t, mock)
}

func TestEnvironmentPageBindsTenantProjectAndSnapshot(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-b")
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`MAX\(list_ordinal\).*WHERE tenant_id = \$1 AND project_id = \$2`).WithArgs(scope.Organization, tenant.ProjectID("same-project")).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(7))
	mock.ExpectQuery(`WHERE tenant_id = \$1 AND project_id = \$2 AND list_ordinal <= \$3 AND list_ordinal > \$4`).WithArgs(scope.Organization, tenant.ProjectID("same-project"), int64(7), int64(0), 2).WillReturnRows(environmentPageRows().
		AddRow("tenant-b", "same-project", "environment-b", "B", 2, 1, 1, 7))
	mock.ExpectCommit()
	page, err := repo.ListEnvironments(context.Background(), scope, "same-project", PageRequest{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Tenant != scope.Organization || page.Next != 0 {
		t.Fatalf("page = %+v", page)
	}
	assertExpectations(t, mock)
}

func TestPageBoundsFailBeforeTransaction(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	for _, request := range []PageRequest{{Limit: 0}, {Limit: MaxPageSize + 1}, {Limit: 1, Snapshot: -1}, {Limit: 1, After: 1}, {Limit: 1, Snapshot: 2, After: 3}} {
		if _, err := repo.ListProjects(context.Background(), scope, request); err == nil {
			t.Fatalf("invalid page accepted: %+v", request)
		}
	}
	if _, err := repo.ListEnvironments(context.Background(), scope, "", PageRequest{Limit: 1}); err == nil {
		t.Fatal("empty project accepted")
	}
	assertExpectations(t, mock)
}

func TestProjectPageRejectsCrossTenantRowsWithoutPartialDisclosure(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_projects`).WillReturnRows(projectPageRows().AddRow("tenant-b", "same-id", "Other", 1, 1, 1))
	mock.ExpectRollback()
	page, err := repo.ListProjects(context.Background(), scope, PageRequest{Snapshot: 1, Limit: 1})
	if err == nil || len(page.Items) != 0 {
		t.Fatalf("cross-tenant page=%+v error=%v", page, err)
	}
	assertExpectations(t, mock)
}

func TestPageQueryFailureRollsBackWithoutPartialResult(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope := mustScope(t, "tenant-a")
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_projects`).WillReturnError(errors.New("database unavailable"))
	mock.ExpectRollback()
	page, err := repo.ListProjects(context.Background(), scope, PageRequest{Snapshot: 5, Limit: 1})
	if err == nil || len(page.Items) != 0 {
		t.Fatalf("page=%+v error=%v", page, err)
	}
	assertExpectations(t, mock)
}

func projectPageRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"tenant_id", "project_id", "name", "state", "version", "list_ordinal"})
}

func environmentPageRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"tenant_id", "project_id", "environment_id", "name", "kind", "state", "version", "list_ordinal"})
}
