package execution

import (
	"strings"
	"testing"
)

// A minimal but realistic `claude --output-format stream-json` transcript: an
// init line, an assistant turn with text + a tool_use, and the terminating
// result event carrying usage/turns/cost.
const streamJSONFixture = `{"type":"system","subtype":"init","tools":["Read"]}
{"type":"assistant","message":{"content":[{"type":"text","text":"Looking at the file."},{"type":"tool_use","name":"Read","input":{"path":"x.go"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Done."}]}}
{"type":"result","subtype":"success","usage":{"input_tokens":1200,"output_tokens":345},"num_turns":2,"total_cost_usd":0.0421}
`

func TestTeeStreamJSONParsesUsageAndReadableText(t *testing.T) {
	var out strings.Builder
	u := teeStreamJSON(strings.NewReader(streamJSONFixture), &out)

	if !u.found {
		t.Fatal("expected a result event to be found")
	}
	if u.OutputTokens != 345 || u.InputTokens != 1200 {
		t.Errorf("tokens: got in=%d out=%d, want in=1200 out=345", u.InputTokens, u.OutputTokens)
	}
	if u.NumTurns != 2 {
		t.Errorf("num_turns: got %d, want 2", u.NumTurns)
	}
	if u.CostUSD != 0.0421 {
		t.Errorf("cost_usd: got %v, want 0.0421", u.CostUSD)
	}

	text := out.String()
	// Readable transcript keeps assistant text and marks tool use — so logs -f
	// and agents --tail still show activity, not raw JSON.
	for _, want := range []string{"Looking at the file.", "[tool: Read]", "Done."} {
		if !strings.Contains(text, want) {
			t.Errorf("transcript missing %q; got:\n%s", want, text)
		}
	}
	if strings.Contains(text, `"type":"assistant"`) {
		t.Errorf("transcript leaked raw JSON:\n%s", text)
	}
}

// A plain-text transcript (a text runtime's output) carries no result event, so
// the parser reports nothing found and passes the lines through verbatim — the
// property that keeps text runtimes byte-for-byte unaffected.
func TestTeeStreamJSONPlainTextHasNoUsage(t *testing.T) {
	var out strings.Builder
	u := teeStreamJSON(strings.NewReader("hello world\nnot json at all\n"), &out)
	if u.found {
		t.Errorf("plain text should carry no usage, got %+v", u)
	}
	if got := out.String(); !strings.Contains(got, "hello world") || !strings.Contains(got, "not json at all") {
		t.Errorf("plain lines should pass through verbatim; got:\n%s", got)
	}
}
