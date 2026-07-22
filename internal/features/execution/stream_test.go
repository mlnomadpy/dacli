package execution

import (
	"errors"
	"os"
	"path/filepath"
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

// The terminating result event carries the ONLY usage numbers and arrives LAST.
// A single very large earlier line (a big assistant/tool message) must NOT abort
// the stream before the result event — the old 16MB Scanner cap silently lost
// usage on an over-long line. A line far larger than the read buffer proves the
// bufio.Reader path grows without truncating.
func TestTeeStreamJSONLongLineDoesNotLoseUsage(t *testing.T) {
	big := strings.Repeat("x", 256*1024) // well beyond the 64KB read buffer
	stream := `{"type":"assistant","message":{"content":[{"type":"text","text":"` + big + `"}]}}` + "\n" +
		`{"type":"result","subtype":"success","usage":{"input_tokens":5,"output_tokens":7},"num_turns":1,"total_cost_usd":0.01}` + "\n"
	var out strings.Builder
	u := teeStreamJSON(strings.NewReader(stream), &out)
	if !u.found || u.OutputTokens != 7 || u.InputTokens != 5 {
		t.Fatalf("usage lost across a long line: %+v", u)
	}
	if u.scanErr != nil {
		t.Errorf("unexpected scanErr on a well-formed stream: %v", u.scanErr)
	}
}

// errAfterReader yields data once, then a non-EOF read error — a stream that
// dies mid-flight before the result event.
type errAfterReader struct {
	data []byte
	done bool
}

func (e *errAfterReader) Read(p []byte) (int, error) {
	if e.done {
		return 0, errors.New("stream read fault")
	}
	e.done = true
	return copy(p, e.data), nil
}

// A read fault that ends the stream before the result event must be reported via
// scanErr, not swallowed as a clean EOF — otherwise a failed capture is
// indistinguishable from a text runtime and usage is silently dropped.
func TestTeeStreamJSONSurfacesReadError(t *testing.T) {
	r := &errAfterReader{data: []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"working"}]}}` + "\n")}
	var out strings.Builder
	u := teeStreamJSON(r, &out)
	if u.found {
		t.Error("no result event was sent; found must be false")
	}
	if u.scanErr == nil {
		t.Error("a mid-stream read error must surface as scanErr, not be swallowed")
	}
}

// Detached stream-json runs write RAW JSON to the transcript; the read-side
// renderer must turn it back into readable text so `dacli logs` / `agents
// --tail` show activity, not JSON.
func TestRenderTranscriptToRendersRawStreamJSON(t *testing.T) {
	var out strings.Builder
	renderTranscriptTo(&out, []byte(streamJSONFixture))
	got := out.String()
	for _, want := range []string{"Looking at the file.", "[tool: Read]", "Done."} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered transcript missing %q; got:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"type":`) {
		t.Errorf("rendered transcript leaked raw JSON:\n%s", got)
	}
}

func TestLastTranscriptLineRendersRawStreamJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "transcript.log")
	if err := os.WriteFile(p, []byte(streamJSONFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	// The final content is the "Done." assistant event; the trailing result
	// event carries no human-facing line and must be skipped.
	if got := lastTranscriptLine(p); got != "Done." {
		t.Errorf("lastTranscriptLine = %q, want %q", got, "Done.")
	}
}
