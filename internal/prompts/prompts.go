// Package prompts is the registry for every multi-sentence piece of agent-
// facing prose dacli emits: spawn preambles, supervision corrections, brief
// headers, MCP tool descriptions.
//
// The doctrine is the same one adapters and shortcuts already follow:
// PROMPTS ARE DATA, NOT CODE. A prompt buried in an Fprintf chain cannot be
// audited, diffed in a PR, or improved without recompiling — and for this
// tool the prompts are load-bearing artifacts. So the defaults live as
// template files embedded at build time (still files in the repo: reviewable,
// blame-able), and a workspace may override any of them by placing a file of
// the same name in .dacli/prompts/.
//
// The boundary, stated so it doesn't erode: one-line refusal/usage messages
// STAY in code. They are the exit-code contract's surface, tested by string,
// and versioned with the behavior they describe. Prose blocks move here;
// contract lines do not.
package prompts

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/mlnomadpy/dacli/internal/mdstore"
)

//go:embed tpl
var embedded embed.FS

// Render resolves a prompt by name (workspace override first, embedded
// default second) and executes it as a text/template against data.
func Render(overrideDir, name string, data any) (string, error) {
	content, _, err := Resolve(overrideDir, name)
	if err != nil {
		return "", err
	}
	t, err := template.New(name).Parse(content)
	if err != nil {
		return "", fmt.Errorf("prompt %s: %w", name, err)
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("prompt %s: %w", name, err)
	}
	return b.String(), nil
}

// Resolve returns a prompt's raw template and whether a workspace override
// supplied it.
func Resolve(overrideDir, name string) (content string, overridden bool, err error) {
	if overrideDir != "" {
		if raw, err := os.ReadFile(filepath.Join(overrideDir, name+".md")); err == nil {
			return string(raw), true, nil
		}
	}
	raw, err := embedded.ReadFile("tpl/" + name + ".md")
	if err != nil {
		return "", false, fmt.Errorf("no such prompt %q", name)
	}
	return string(raw), false, nil
}

// Names lists the embedded registry, sorted.
func Names() []string {
	entries, _ := embedded.ReadDir("tpl")
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	sort.Strings(out)
	return out
}

// MCPDesc returns one MCP tool's description from the sectioned registry
// file mcp_tools.md — one file so the entire agent-facing tool documentation
// is auditable in a single review. Missing sections panic at init, guarded
// by a test that walks every registered tool.
func MCPDesc(tool string) string {
	raw, err := embedded.ReadFile("tpl/mcp_tools.md")
	if err != nil {
		panic("prompts: mcp_tools.md missing from embed")
	}
	doc, err := mdstore.Parse(string(raw))
	if err != nil {
		panic("prompts: mcp_tools.md unparseable: " + err.Error())
	}
	s, ok := doc.Section(tool)
	if !ok || strings.TrimSpace(s.Content) == "" {
		panic("prompts: no description for MCP tool " + tool)
	}
	return strings.TrimSpace(s.Content)
}
