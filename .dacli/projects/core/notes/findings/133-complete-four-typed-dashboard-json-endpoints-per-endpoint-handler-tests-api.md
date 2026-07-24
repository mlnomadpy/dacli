---
id: f-133-complete-four-typed-dashboard-json-endpoints-per-endpoint-handler-tests-api
kind: note
note_kind: finding
created: 2026-07-24T00:54:53Z
created_by: a-wey686j4cx
about: [[133]]
severity: moderate
---
# 133 complete: four typed dashboard JSON endpoints + per-endpoint handler tests, /api/state preserved
Commit c544c50 on branch dacli/133 (frontend-engineer). internal/features/dashboard/dashboard.go adds GET /api/overview (overviewResponse: generated/project_count/task_count/counts/pending_events/live_agents), /api/projects (projectsResponse.projects = []projectView, same shape as /api/state.projects), /api/tasks (tasksResponse.tasks = []taskView with id/project/seq/slug/title/status/priority/owner/points/estimated; optional ?project=<slug> filter), /api/agents (agentsResponse.agents = []agentView, same shape as /api/state.agents). Each is a fresh no-cache workspace read reusing buildProjectView/buildAgentView/liveAgents + eventlog.List, so they reflect the live store+event log and never drift from the combined snapshot. Legacy / and /api/state kept verbatim (acceptance: existing behavior preserved). Shapes documented via Go doc comments on each response struct. One handler test per endpoint in dashboard_test.go: TestAPIOverview/TestAPIProjects/TestAPITasks (incl. ?project= filter)/TestAPIAgents, all driving newHandler via httptest. go build ./... clean; gofmt -l clean; go test ./internal/... green (DACLI_AGENT stripped per known test-isolation gap). Owner: verify + close via task check/done + merge --task 133.
