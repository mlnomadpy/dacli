// Package capabilities defines the stable, machine-readable contract between
// an installed dacli binary and agent guidance. The app layer supplies registry
// entries; this package only normalizes, indexes, and compares them.
package capabilities

import (
	"fmt"
	"sort"
)

const (
	ManifestSchema     = 1
	RequirementsSchema = 1
)

type Binary struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`
}

type Flag struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Value         string `json:"value,omitempty"`
	SchemaVersion int    `json:"schema_version"`
}

type Command struct {
	ID            string   `json:"id"`
	Path          string   `json:"path"`
	Aliases       []string `json:"aliases"`
	Flags         []Flag   `json:"flags"`
	JSON          bool     `json:"json"`
	Mutates       bool     `json:"mutates"`
	Usage         string   `json:"usage"`
	SchemaVersion int      `json:"schema_version"`
}

type VersionedCapability struct {
	ID            string `json:"id"`
	SchemaVersion int    `json:"schema_version"`
}

type MCP struct {
	ProtocolVersion string                `json:"protocol_version"`
	ServerVersion   string                `json:"server_version"`
	ToolSchema      int                   `json:"tool_schema_version"`
	Tools           []VersionedCapability `json:"tools"`
}

type Prompt struct {
	Schema  string `json:"schema"`
	Version string `json:"version"`
}

type RuntimeAdapters struct {
	SchemaVersion int                   `json:"schema_version"`
	Capabilities  []VersionedCapability `json:"capabilities"`
}

type Manifest struct {
	SchemaVersion int                   `json:"schema_version"`
	Binary        Binary                `json:"binary"`
	State         []VersionedCapability `json:"state_schemas"`
	Commands      []Command             `json:"commands"`
	MCP           MCP                   `json:"mcp"`
	Prompt        Prompt                `json:"prompt"`
	Runtime       RuntimeAdapters       `json:"runtime_adapters"`
}

// Normalize makes output deterministic even when registries are assembled in
// a different source order.
func (m *Manifest) Normalize() {
	sort.Slice(m.Commands, func(i, j int) bool { return m.Commands[i].ID < m.Commands[j].ID })
	for i := range m.Commands {
		if m.Commands[i].Aliases == nil {
			m.Commands[i].Aliases = []string{}
		}
		if m.Commands[i].Flags == nil {
			m.Commands[i].Flags = []Flag{}
		}
		sort.Slice(m.Commands[i].Flags, func(a, b int) bool { return m.Commands[i].Flags[a].ID < m.Commands[i].Flags[b].ID })
	}
	sort.Slice(m.State, func(i, j int) bool { return m.State[i].ID < m.State[j].ID })
	sort.Slice(m.MCP.Tools, func(i, j int) bool { return m.MCP.Tools[i].ID < m.MCP.Tools[j].ID })
	sort.Slice(m.Runtime.Capabilities, func(i, j int) bool { return m.Runtime.Capabilities[i].ID < m.Runtime.Capabilities[j].ID })
}

type Requirement struct {
	ID            string `json:"id"`
	SchemaVersion int    `json:"schema_version,omitempty"`
	Fallback      string `json:"fallback,omitempty"`
}

type Requirements struct {
	SchemaVersion      int           `json:"schema_version"`
	GeneratedByVersion string        `json:"generated_by_version,omitempty"`
	Required           []Requirement `json:"required"`
	Optional           []Requirement `json:"optional"`
}

type Finding struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Required  int    `json:"required_schema_version,omitempty"`
	Installed int    `json:"installed_schema_version,omitempty"`
	Fallback  string `json:"fallback,omitempty"`
}

type Diagnosis struct {
	SchemaVersion      int       `json:"schema_version"`
	SkillRequirements  string    `json:"skill_requirements"`
	GeneratedByVersion string    `json:"generated_by_version,omitempty"`
	Binary             Binary    `json:"binary"`
	Findings           []Finding `json:"findings"`
	Compatible         bool      `json:"compatible"`
}

// Index returns every stable capability identity advertised by the manifest.
func Index(m Manifest) map[string]int {
	out := map[string]int{
		"manifest.schema":        m.SchemaVersion,
		"mcp.protocol":           1,
		"mcp.tool-schema":        m.MCP.ToolSchema,
		"prompt.schema":          1,
		"runtime.adapter-schema": m.Runtime.SchemaVersion,
	}
	for _, c := range m.Commands {
		out[c.ID] = c.SchemaVersion
		for _, f := range c.Flags {
			out[f.ID] = f.SchemaVersion
		}
	}
	for _, c := range m.State {
		out[c.ID] = c.SchemaVersion
	}
	for _, c := range m.MCP.Tools {
		out[c.ID] = c.SchemaVersion
	}
	for _, c := range m.Runtime.Capabilities {
		out[c.ID] = c.SchemaVersion
	}
	return out
}

func Diagnose(m Manifest, req Requirements, source string) (Diagnosis, error) {
	if req.SchemaVersion != RequirementsSchema {
		return Diagnosis{}, fmt.Errorf("requirements schema %d is incompatible with supported schema %d", req.SchemaVersion, RequirementsSchema)
	}
	d := Diagnosis{SchemaVersion: RequirementsSchema, SkillRequirements: source, GeneratedByVersion: req.GeneratedByVersion, Binary: m.Binary, Compatible: true, Findings: []Finding{}}
	installed := Index(m)
	check := func(kind string, r Requirement) {
		got, ok := installed[r.ID]
		f := Finding{ID: r.ID, Kind: kind, Required: r.SchemaVersion, Installed: got, Fallback: r.Fallback}
		switch {
		case !ok:
			f.Status = kind + "-missing"
			if kind == "required" {
				d.Compatible = false
			}
		case r.SchemaVersion > 0 && got != r.SchemaVersion:
			f.Status = "incompatible-schema"
			if kind == "required" {
				d.Compatible = false
			}
		default:
			f.Status = "supported"
		}
		if kind == "optional" && f.Status != "supported" && f.Fallback == "" {
			f.Fallback = "omit this optional behavior or use guidance generated for the installed manifest"
		}
		d.Findings = append(d.Findings, f)
	}
	for _, r := range req.Required {
		check("required", r)
	}
	for _, r := range req.Optional {
		check("optional", r)
	}
	sort.Slice(d.Findings, func(i, j int) bool { return d.Findings[i].ID < d.Findings[j].ID })
	return d, nil
}
