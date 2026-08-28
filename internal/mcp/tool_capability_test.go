package mcp

import "testing"

func TestToolCapabilitiesAreDerivedFromCompleteRegistry(t *testing.T) {
	got := ToolCapabilities()
	if len(got) != len(tools) {
		t.Fatalf("capability manifest has %d MCP tools, registry has %d", len(got), len(tools))
	}
	for i, tool := range tools {
		if got[i].Name != tool.name || got[i].SchemaVersion != toolSchemaVersion {
			t.Fatalf("tool %d capability = %#v, want %q schema %d", i, got[i], tool.name, toolSchemaVersion)
		}
	}
}
