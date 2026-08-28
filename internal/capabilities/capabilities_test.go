package capabilities

import (
	"reflect"
	"testing"
)

func fixtureManifest() Manifest {
	return Manifest{
		Binary: Binary{Name: "dacli", Version: "v1", Path: "/bin/dacli"},
		Commands: []Command{{
			ID: "cli.command.task.check", SchemaVersion: 1,
			Flags: []Flag{{ID: "cli.command.task.check.flag.verify", SchemaVersion: 1}},
		}},
		MCP: MCP{Tools: []VersionedCapability{{ID: "mcp.tool.status", SchemaVersion: 1}}},
	}
}

func TestDiagnoseClassifiesSupportedOptionalMissingAndIncompatibleSchema(t *testing.T) {
	req := Requirements{SchemaVersion: 1,
		Required: []Requirement{{ID: "cli.command.task.check", SchemaVersion: 1}},
		Optional: []Requirement{
			{ID: "cli.command.task.check.flag.verify", SchemaVersion: 2, Fallback: "run verifier separately"},
			{ID: "mcp.tool.future", SchemaVersion: 1, Fallback: "use cli"},
		},
	}
	d, err := Diagnose(fixtureManifest(), req, "/skill/capabilities.json")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Compatible {
		t.Fatal("optional gaps must not make the skill incompatible")
	}
	want := []Finding{
		{ID: "cli.command.task.check", Kind: "required", Status: "supported", Required: 1, Installed: 1},
		{ID: "cli.command.task.check.flag.verify", Kind: "optional", Status: "incompatible-schema", Required: 2, Installed: 1, Fallback: "run verifier separately"},
		{ID: "mcp.tool.future", Kind: "optional", Status: "optional-missing", Required: 1, Fallback: "use cli"},
	}
	if !reflect.DeepEqual(d.Findings, want) {
		t.Fatalf("findings = %#v, want %#v", d.Findings, want)
	}
}

func TestDiagnoseRequiredMissingFailsClosed(t *testing.T) {
	d, err := Diagnose(fixtureManifest(), Requirements{SchemaVersion: 1, Required: []Requirement{{ID: "cli.command.task.check.flag.verify", SchemaVersion: 1}}}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Compatible || len(d.Findings) != 1 || d.Findings[0].Status != "supported" {
		t.Fatalf("existing flag diagnosis = %#v", d)
	}

	// Regression for issue #807: old binaries whose manifest lacks --verify
	// must not be treated as compatible with guidance that emits that flag.
	m := fixtureManifest()
	m.Commands[0].Flags = nil
	d, err = Diagnose(m, Requirements{SchemaVersion: 1, Required: []Requirement{{ID: "cli.command.task.check.flag.verify", SchemaVersion: 1}}}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if d.Compatible || d.Findings[0].Status != "required-missing" {
		t.Fatalf("guidance-only flag was accepted against an installed manifest that lacks it: %#v", d)
	}
}

func TestDiagnoseRequiredIncompatibleSchemaFailsClosed(t *testing.T) {
	d, err := Diagnose(fixtureManifest(), Requirements{SchemaVersion: 1, Required: []Requirement{{ID: "cli.command.task.check", SchemaVersion: 2}}}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if d.Compatible || d.Findings[0].Status != "incompatible-schema" || d.Findings[0].Installed != 1 {
		t.Fatalf("incompatible required schema was accepted: %#v", d)
	}
}

func TestNormalizeIsDeterministic(t *testing.T) {
	m := Manifest{
		Commands: []Command{{ID: "z", Flags: []Flag{{ID: "z.2"}, {ID: "z.1"}}}, {ID: "a"}},
		State:    []VersionedCapability{{ID: "z"}, {ID: "a"}},
		MCP:      MCP{Tools: []VersionedCapability{{ID: "z"}, {ID: "a"}}},
		Runtime:  RuntimeAdapters{Capabilities: []VersionedCapability{{ID: "z"}, {ID: "a"}}},
	}
	m.Normalize()
	if m.Commands[0].ID != "a" || m.Commands[1].Flags[0].ID != "z.1" || m.State[0].ID != "a" || m.MCP.Tools[0].ID != "a" || m.Runtime.Capabilities[0].ID != "a" {
		t.Fatalf("manifest not normalized: %#v", m)
	}
	if m.Commands[0].Aliases == nil || m.Commands[0].Flags == nil {
		t.Fatal("empty arrays must encode as [] rather than null")
	}
}
