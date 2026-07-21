// Package mcp serves the workspace over the Model Context Protocol on stdio,
// per docs/MCP.md: a tiered tool surface (core tools with full schemas plus
// one `cli` escape hatch), refusals returned as results rather than errors,
// and identity bound at process launch — the token never appears in a tool
// parameter, because parameters land in transcripts.
//
// The protocol layer is hand-rolled JSON-RPC 2.0 over newline-delimited
// stdio. It is ~150 lines, and the zero-dependency property of the module is
// worth more than an SDK.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mlnomadpy/dacli/internal/prompts"
)

// Executor runs one dacli command and reports (stdout, stderr+error, exit
// code). The CLI provides it; this package never imports the CLI, which is
// what keeps the two front ends over one core without a cycle.
type Executor func(argv []string, jsonMode bool) (out, msg string, code int)

const protocolVersion = "2025-06-18"

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Serve reads newline-delimited JSON-RPC requests until EOF. It verifies the
// bound identity up front so a bad DACLI_AGENT fails at launch, not on the
// tenth tool call.
func Serve(r io.Reader, w io.Writer, exec Executor) error {
	if _, msg, code := exec([]string{"whoami"}, false); code != 0 {
		return fmt.Errorf("cannot serve: %s", strings.TrimSpace(msg))
	}

	enc := json.NewEncoder(w)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // briefs are big

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(response{JSONRPC: "2.0", Error: &rpcError{-32700, "parse error"}})
			continue
		}
		resp, notify := handle(&req, exec)
		if notify {
			continue // notifications get no response
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return sc.Err()
}

func handle(req *request, exec Executor) (response, bool) {
	resp := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		v := p.ProtocolVersion
		if v == "" {
			v = protocolVersion
		}
		resp.Result = map[string]any{
			"protocolVersion": v,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "dacli", "version": "0.3"},
		}
	case "notifications/initialized", "notifications/cancelled":
		return resp, true
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		defs := make([]map[string]any, 0, len(tools))
		for _, t := range tools {
			defs = append(defs, map[string]any{
				"name":        t.name,
				"description": t.desc,
				"inputSchema": t.schema,
			})
		}
		resp.Result = map[string]any{"tools": defs}
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp.Error = &rpcError{-32602, "invalid params"}
			break
		}
		t, ok := toolByName(p.Name)
		if !ok {
			resp.Error = &rpcError{-32602, "unknown tool: " + p.Name}
			break
		}
		resp.Result = call(t, p.Arguments, exec)
	default:
		if req.ID == nil {
			return resp, true // unknown notification: ignore
		}
		resp.Error = &rpcError{-32601, "method not found: " + req.Method}
	}
	return resp, false
}

// call maps the exit-code contract onto MCP. The load-bearing row: exit 3
// (refused by policy) is a RESULT, not an error — clients retry errors, and
// retrying a refusal is the loop the contract exists to prevent.
func call(t tool, args map[string]any, exec Executor) callResult {
	argv, jsonMode, err := t.build(args)
	if err != nil {
		return callResult{IsError: true, Content: []content{{Type: "text", Text: err.Error()}}}
	}
	out, msg, code := exec(argv, jsonMode)

	switch code {
	case 0:
		text := out
		if strings.TrimSpace(msg) != "" {
			text += "\n[notes]\n" + msg
		}
		if strings.TrimSpace(text) == "" {
			text = "ok"
		}
		return callResult{Content: []content{{Type: "text", Text: text}}}
	case 3:
		next, _ := prompts.Render("", "refusal_next", nil)
		refusal, _ := json.Marshal(map[string]any{"refused": map[string]string{
			"reason": strings.TrimSpace(msg),
			"next":   strings.TrimSpace(next),
		}})
		return callResult{Content: []content{{Type: "text", Text: string(refusal)}}}
	default:
		label := map[int]string{2: "usage error", 4: "not found", 5: "conflict"}[code]
		if label == "" {
			label = "failed"
		}
		return callResult{IsError: true, Content: []content{{Type: "text", Text: label + ": " + strings.TrimSpace(out+"\n"+msg)}}}
	}
}
