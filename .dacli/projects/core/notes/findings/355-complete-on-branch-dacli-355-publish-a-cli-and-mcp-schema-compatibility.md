---
id: f-355-complete-on-branch-dacli-355-publish-a-cli-and-mcp-schema-compatibility
kind: note
note_kind: finding
created: 2026-08-11T10:06:55Z
created_by: a-fixer-mebe8r
about: "[[355]]"
severity: major
---
# 355 complete on branch dacli/355-publish-a-cli-and-mcp-schema-compatibility-policy-with-migration-notes
Commit be0f4a3 (a-fixer-mebe8r). All 3 acceptance criteria met.

(1) New docs/COMPATIBILITY.md states what's stable: exit codes 0/1/2/3/4/5
(table sourced from ARCHITECTURE.md §4), command paths (single source of
truth: the aggregated command table in internal/cli/cli.go), and the two
--json document shapes that actually exist today — `context --json`
(task_id/sections/omitted) and `task list --json` (array of
id/seq/slug/project/status/priority/title/acceptance_done/acceptance_total),
scoped to exactly the commands enumerated in
internal/cli/json_invariant_test.go's jsonHonoringCommands. MCP tool surface
stability is derived from the same rules per MCP.md §2 (Tier-1 tools =
command-path rule, Tier-2 `cli` escape hatch = JSON-shape rule
transitively). §2 states what may change without notice: human text output,
usage/error wording, undocumented flag-parsing edge cases, internal Go APIs,
and any command/shape not enumerated in §1.

(2) internal/cli/compat_json_shape_test.go: TestDocumentedJSONShapesStillParse
decodes both `context --json` and `task list --json` against the exact
field names/types docs/COMPATIBILITY.md commits to, via a small
requireField helper that fails on a missing field or a type mismatch but
NOT on an extra field (matching the additive-only rule both this doc and
FORMAT.md state). This is the enforcement half of the acceptance criterion:
a future rename/drop of e.g. taskJSON.ID's `json:"id"` tag fails a test, not
just a doc sentence going stale.

(3) docs/COMPATIBILITY.md §4 records four migration notes for surface
changes that landed this week (all 2026-08-10/11, verified against git log
--since so scope wasn't guessed): unknown/typo'd flags now exit 2 instead of
running silently with defaults (a13795d, 51 handlers via one dispatcher
gate); `task list --status <bad value>` now exits 2 naming the bad value +
allowed set instead of silently returning an empty list at exit 0
(d24d5fe); entity-layer policy refusals (e.g. `note add decision` without
--rejected) now exit 3 instead of 1 — a real exit-code contract change, not
cosmetic, since 1 means "retry may help" and 3 means "do not retry"
(7bc0147); and the flag parser's bare `--` handling was fixed to strict
POSIX semantics after FuzzParseFlags found it could swallow a following
flag (b1398a9). Each entry states before/after behavior and the concrete
caller action.

Red-green verified by hand on the new test: mutated taskJSON.ID's json tag
from `id` to `identifier` in internal/features/planning/planning.go, reran
TestDocumentedJSONShapesStillParse -> failed exactly as predicted
(`documented field "id" is missing from the response`). Reverted, green
again; git diff confirmed clean before committing.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l . clean (no
output). go test ./internal/cli/... -run TestDocumentedJSONShapesStillParse
-v: both subtests pass. Full go test ./...: only pre-existing failures are
in internal/features/briefing (TestCatchup*), orchestration/noremote_test.go
(TestIntoRefusesAnUnknownBranchUpFront), and
internal/features/teamops/teamops_test.go
(TestAgentSpawnFailsClosedWhenTheWIPCountCannotBeRead) — all three failing
with "agent token not recognized in this workspace", the same ambient
DACLI_AGENT dogfood-session artifact multiple prior fixers (337, 341, 338)
have already documented as unrelated; this task touched none of those three
packages. Every package this diff touches (cli, and read-only checks of
planning during the mutation test) is green.

golangci-lint could NOT be run: the binary requires interactive approval in
this sandbox, unavailable to a headless agent (`printenv`/`env` themselves
were also blocked) — flagging this gap honestly rather than claiming a
check I did not run, same limitation prior fixers in this project have hit
and documented.

Files changed: docs/COMPATIBILITY.md (new), internal/cli/compat_json_shape_test.go
(new), docs/README.md (added COMPATIBILITY.md row), mkdocs.yml (added nav
entry). No product behavior changed — pure documentation + test addition,
so no CHANGELOG entry (the changes it documents already happened and are
partially reflected there under Unreleased/Fixed).

Owner: dacli accept 355 (task check is gated to a-root; I could not check
the boxes myself). PR-first is off — branch
dacli/355-publish-a-cli-and-mcp-schema-compatibility-policy-with-migration-notes
is ready for accept + integrate/merge --task 355.
