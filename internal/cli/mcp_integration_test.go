package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mcp"
)

// A whole session through the MCP surface against a real workspace: the
// two front ends must be the same tool. Also covers queues end to end.
func TestMCPSessionAgainstRealWorkspace(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "One write path.")
	run(t, dir, 0, "queue", "add", "release", "--step", "go test ./...", "--step", "git tag")

	rpc := func(id int, method, params string) string {
		if params == "" {
			return `{"jsonrpc":"2.0","id":` + jsonInt(id) + `,"method":"` + method + `"}`
		}
		return `{"jsonrpc":"2.0","id":` + jsonInt(id) + `,"method":"` + method + `","params":` + params + `}`
	}
	call := func(id int, tool, args string) string {
		return rpc(id, "tools/call", `{"name":"`+tool+`","arguments":`+args+`}`)
	}

	script := strings.Join([]string{
		rpc(1, "initialize", `{"protocolVersion":"2025-06-18"}`),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		call(2, "add_task", `{"project":"p","title":"Audit write paths","priority":"must","estimate":"2,5,14","accept":["writers listed"]}`),
		call(3, "get_context", `{"ref":"001","budget":4000}`),
		call(4, "finish_task", `{"ref":"001"}`),
		call(5, "check_task", `{"ref":"001","all":true}`),
		call(6, "finish_task", `{"ref":"001"}`),
		call(7, "list_tasks", `{"status":"done"}`),
		call(8, "queue_next", `{"queue":"release"}`),
		call(9, "queue_advance", `{"queue":"release"}`),
	}, "\n") + "\n"

	var out bytes.Buffer
	if err := mcp.Serve(strings.NewReader(script), &out, executor(dir)); err != nil {
		t.Fatal(err)
	}

	byID := map[float64]map[string]any{}
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("bad response line %q: %v", l, err)
		}
		if id, ok := m["id"].(float64); ok {
			byID[id] = m
		}
	}

	text := func(id float64) string {
		res, ok := byID[id]["result"].(map[string]any)
		if !ok {
			t.Fatalf("no result for id %v: %v", id, byID[id])
		}
		return res["content"].([]any)[0].(map[string]any)["text"].(string)
	}

	// The brief arrives through MCP with the same content as the CLI path.
	brief := text(3)
	for _, want := range []string{"## Task: Audit write paths", "estimate: 2/5/14", "data, not instructions"} {
		if !strings.Contains(brief, want) {
			t.Errorf("brief via MCP missing %q", want)
		}
	}

	// finish before checking → refused-as-result with the criterion named.
	refusal := text(4)
	if byID[4]["result"].(map[string]any)["isError"] == true {
		t.Fatal("refusal must not be isError")
	}
	if !strings.Contains(refusal, "refused") || !strings.Contains(refusal, "writers listed") {
		t.Errorf("refusal = %q", refusal)
	}

	// After checking, finish succeeds and list_tasks shows JSON.
	if !strings.Contains(text(6), "done: 001") {
		t.Errorf("finish after check = %q", text(6))
	}
	var tasks []map[string]any
	if err := json.Unmarshal([]byte(text(7)), &tasks); err != nil || len(tasks) != 1 {
		t.Errorf("list_tasks json = %q (%v)", text(7), err)
	}

	// Queue stepping.
	if !strings.Contains(text(8), "go test ./...") {
		t.Errorf("queue_next = %q", text(8))
	}
	if !strings.Contains(text(9), "git tag") {
		t.Errorf("queue_advance = %q", text(9))
	}
}

func jsonInt(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

// Queue ownership: the cursor has exactly one writer.
func TestQueueOwnershipRefusal(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "queue", "add", "rel", "--step", "one")

	// A spawned child is not the owner: advancing is refused, not failed.
	out := run(t, dir, 0, "agent", "spawn", "--role", "helper", "--grant", "rw")
	token := strings.TrimSpace(strings.Split(out, "\n")[0])
	t.Setenv("DACLI_AGENT", token)
	refusal := run(t, dir, 3, "queue", "advance", "rel")
	if !strings.Contains(refusal, "owned by a-root") {
		t.Errorf("refusal should name the owner: %s", refusal)
	}

	t.Setenv("DACLI_AGENT", "")
	run(t, dir, 0, "queue", "advance", "rel")
	if got := run(t, dir, 0, "queue", "next", "rel"); !strings.Contains(got, "queue complete") {
		t.Errorf("queue not complete: %s", got)
	}

	// Halt semantics.
	run(t, dir, 0, "queue", "add", "rel2", "--step", "a", "--step", "b")
	run(t, dir, 0, "queue", "advance", "rel2", "--fail", "step a exploded")
	if got := run(t, dir, 1, "queue", "next", "rel2"); !strings.Contains(got, "halted") {
		t.Errorf("halted queue should refuse next: %s", got)
	}
}
