package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// fake is an Executor scripted by command name.
func fake(responses map[string][3]any) Executor {
	return func(argv []string, jsonMode bool) (string, string, int) {
		key := strings.Join(argv, " ")
		for prefix, r := range responses {
			if strings.HasPrefix(key, prefix) {
				return r[0].(string), r[1].(string), r[2].(int)
			}
		}
		return "ok:" + key, "", 0
	}
}

func serve(t *testing.T, exec Executor, lines ...string) []map[string]any {
	t.Helper()
	in := strings.Join(lines, "\n") + "\n"
	var out bytes.Buffer
	if err := Serve(strings.NewReader(in), &out, exec); err != nil {
		t.Fatal(err)
	}
	var msgs []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("unparseable response %q: %v", l, err)
		}
		msgs = append(msgs, m)
	}
	return msgs
}

const initReq = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`
const initedNote = `{"jsonrpc":"2.0","method":"notifications/initialized"}`

func TestHandshakeAndToolsList(t *testing.T) {
	msgs := serve(t, fake(nil),
		initReq,
		initedNote, // must produce no response
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	if len(msgs) != 2 {
		t.Fatalf("got %d responses, want 2 (notification must be silent)", len(msgs))
	}
	initRes := msgs[0]["result"].(map[string]any)
	if initRes["protocolVersion"] != "2025-06-18" {
		t.Errorf("protocolVersion = %v", initRes["protocolVersion"])
	}
	toolsRes := msgs[1]["result"].(map[string]any)["tools"].([]any)
	names := map[string]bool{}
	for _, tl := range toolsRes {
		names[tl.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{"get_context", "add_note", "finish_task", "run_shortcut", "cli"} {
		if !names[want] {
			t.Errorf("tool %q missing from tools/list", want)
		}
	}
	if len(toolsRes) > 20 {
		t.Errorf("%d tools — the tiered surface exists to avoid a 50-schema catalog", len(toolsRes))
	}
}

func TestToolCallSuccess(t *testing.T) {
	msgs := serve(t, fake(map[string][3]any{"whoami": {"a-root (grant: rw)\n", "", 0}}),
		initReq, initedNote,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"whoami","arguments":{}}}`,
	)
	res := msgs[len(msgs)-1]["result"].(map[string]any)
	if res["isError"] == true {
		t.Fatalf("whoami errored: %v", res)
	}
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "a-root") {
		t.Errorf("text = %q", text)
	}
}

// The load-bearing row of the exit-code mapping: a refusal is a RESULT the
// model reads, never an error a client retries.
func TestRefusalIsResultNotError(t *testing.T) {
	msgs := serve(t, fake(map[string][3]any{"task done": {"", "acceptance unmet:\n  - suite green", 3}}),
		initReq, initedNote,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"finish_task","arguments":{"ref":"001"}}}`,
	)
	res := msgs[len(msgs)-1]["result"].(map[string]any)
	if res["isError"] == true {
		t.Fatal("refusal surfaced as isError — clients retry errors, and retrying a refusal is the forbidden loop")
	}
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	var payload struct {
		Refused struct{ Reason, Next string }
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("refusal payload not JSON: %q", text)
	}
	if !strings.Contains(payload.Refused.Reason, "acceptance unmet") || !strings.Contains(payload.Refused.Next, "do not retry") {
		t.Errorf("refusal payload = %+v", payload)
	}
}

func TestOperationalErrorIsError(t *testing.T) {
	msgs := serve(t, fake(map[string][3]any{"task claim": {"", "not found: ghost", 4}}),
		initReq, initedNote,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"claim_task","arguments":{"ref":"ghost"}}}`,
	)
	res := msgs[len(msgs)-1]["result"].(map[string]any)
	if res["isError"] != true {
		t.Fatal("not-found must be isError")
	}
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "not found") {
		t.Errorf("text = %q", text)
	}
}

func TestUnknownToolAndMethod(t *testing.T) {
	msgs := serve(t, fake(nil),
		initReq, initedNote,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"frobnicate","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"no/such"}`,
	)
	if msgs[1]["error"] == nil {
		t.Error("unknown tool should be a JSON-RPC error")
	}
	if msgs[2]["error"] == nil {
		t.Error("unknown method with id should be a JSON-RPC error")
	}
}

func TestMissingRequiredArgumentIsError(t *testing.T) {
	msgs := serve(t, fake(nil),
		initReq, initedNote,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"get_context","arguments":{}}}`,
	)
	res := msgs[len(msgs)-1]["result"].(map[string]any)
	if res["isError"] != true {
		t.Error("missing required arg should be isError")
	}
}

func TestServeRefusesBadIdentity(t *testing.T) {
	exec := func(argv []string, jsonMode bool) (string, string, int) {
		return "", "agent token not recognized", 1
	}
	err := Serve(strings.NewReader(""), &bytes.Buffer{}, exec)
	if err == nil || !strings.Contains(err.Error(), "not recognized") {
		t.Errorf("bad identity should fail at launch, got %v", err)
	}
}
