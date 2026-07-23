package clikit

import "os"

// Palette renders ANSI styling for human-facing output. It is deliberately
// conservative about when color is "on": a real terminal, no NO_COLOR, and
// never --json. Since the MCP executor and every test harness write to a
// bytes.Buffer (never an *os.File), and since machine callers pass --json,
// this one check is the entire mechanism that keeps escape codes out of
// agent-facing output and JSON — no separate "am I an agent" flag needed.
type Palette struct{ on bool }

// NewPalette decides once, from ctx, whether output should be colored.
func NewPalette(ctx *Ctx) Palette {
	if ctx == nil || ctx.JSON {
		return Palette{}
	}
	if os.Getenv("NO_COLOR") != "" {
		return Palette{}
	}
	f, ok := ctx.Stdout.(*os.File)
	if !ok {
		return Palette{}
	}
	fi, err := f.Stat()
	if err != nil {
		return Palette{}
	}
	return Palette{on: fi.Mode()&os.ModeCharDevice != 0}
}

// Enabled reports whether this palette will actually emit color — callers
// that need to pad a string BEFORE coloring it (so escape codes don't count
// toward a fixed column width) can skip the padding work when color is off.
func (p Palette) Enabled() bool { return p.on }

func (p Palette) paint(code, s string) string {
	if !p.on || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (p Palette) Bold(s string) string    { return p.paint("1", s) }
func (p Palette) Dim(s string) string     { return p.paint("2", s) }
func (p Palette) Red(s string) string     { return p.paint("31", s) }
func (p Palette) Green(s string) string   { return p.paint("32", s) }
func (p Palette) Yellow(s string) string  { return p.paint("33", s) }
func (p Palette) Blue(s string) string    { return p.paint("34", s) }
func (p Palette) Magenta(s string) string { return p.paint("35", s) }
func (p Palette) Cyan(s string) string    { return p.paint("36", s) }
